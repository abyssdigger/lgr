// Docs are based on CoPilot (GPT-5 mini) generation
package lgr

// never use fmt in threads!

import (
	"bytes"
	"errors"
	"strconv"
)

/*
Docs are based on CoPilot (GPT-5 mini) generation
proceed.go

Contains the background processing loop and the logic that converts queued
messages into writes to configured outputs. Responsible for:
 - running the processor goroutine that reads from the logger channel
 - dispatching command messages (client setting changes)
 - formatting textual messages according to per-output settings and writing
   to outputs
 - error reporting to the fallback writer
*/

// fbckWriteln writes a single-line message to the fallback writer.
// Used to report internal errors encountered in the background goroutine.
func (l *logger) fbckWriteln(s string) {
	l.fallbck.Write([]byte(s + "\n"))
}

// msgDescStr returns a concise one-line description of a logMessage used in
// debugging/error strings.
func msgDescStr(m *logMessage) string {
	return "type=" + strconv.Itoa(int(m.msgtype)) +
		" annex=" + strconv.Itoa(int(m.annex)) +
		" data=`" + string(m.msgdata) + "`"
}

// setState sets the logger state with write locking; normalizes the provided
// state before assignment.
func (l *logger) setState(newstate lgrState) {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	l.state = normState(newstate)
}

// procced is the background message processing loop. It reads messages from
// the channel until the channel is closed. For each message it calls
// proceedMsg to perform the appropriate action.
//
// The function recovers panics to ensure the background goroutine doesn't die
// silently; recover triggers a fallback write and ensures state is moved to
// STATE_STOPPED before returning.
func (l *logger) procced() {
	l.msgbuf = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
	defer func() {
		if r := recover(); r != nil {
			l.fbckWriteln("panic proceeding log" + panicDesc(r))
		}
		l.msgbuf = nil
		l.setState(STATE_STOPPED)
	}()
	for {
		msg, opened := <-l.channel
		if !opened {
			break
		}
		if err := l.proceedMsg(&msg); err != nil {
			l.fbckWriteln("error proceeding message: " + err.Error())
		}
	}
}

// proceedMsg dispatches a single message. Commands are executed (proceedCmd)
// and then converted to a TRACE text message (so commands are visible in the
// log stream). Text messages are forwarded to outputs. Unknown or forbidden
// message types produce errors or panics (the latter used for testing).
func (l *logger) proceedMsg(msg *logMessage) error {
	l.sync.procMtx.RLock()
	defer func() {
		l.sync.procMtx.RUnlock()
	}()
	switch msg.msgtype {
	case MSG_COMMAND:
		if l.proceedCmd(msg) != nil {
			break
		}
		// convert successfully executed command message to text with level TRACE
		msg.msgtype = MSG_LOG_TEXT
		msg.annex = basetype(LVL_TRACE)
		msg.msgdata = []byte("<COMMAND: " + msgDescStr(msg) + ">")
		fallthrough
	case MSG_LOG_TEXT:
		l.logTextToOutputs(msg)
	case MSG_FORBIDDEN:
		// For testing purposes only — panic to exercise panic handling
		panic("panic on forbidden message type: " + msgDescStr(msg))
	default:
		return errors.New("unknown message type: " + msgDescStr(msg))
	}
	return nil
}

const COMMAND_PING_MESSAGE = "<ping>"

// proceedCmd executes a command message. Supported client commands mutate the
// client settings in-place; any errors or invalid payloads are reported to the
// fallback writer via handleLogWriteError and returned as an error.
func (l *logger) proceedCmd(msg *logMessage) (err error) {
	l.sync.clntMtx.RLock()
	defer l.sync.clntMtx.RUnlock()
	errstr := ""
	if msg == nil {
		errstr = "nil command message, nothing to proceed"
	} else {
		switch cmdType(msg.annex) {
		case CMD_CLIENT_SET_LEVEL:
			// Expect at least one byte with the new level
			errstr = clientChangeFromCmdMsg(msg, func(lc *logClient, data []byte) {
				lc.minLevel = normLevel(LogLevel(data[0]))
			})
		case CMD_CLIENT_SET_NAME:
			// Replace client name with provided bytes
			errstr = clientChangeFromCmdMsg(msg, func(lc *logClient, data []byte) {
				lc.name = data
			})
		case CMD_DUMMY, CMD_CLIENT_DUMMY:
			// No-op placeholder commands.
		case CMD_PING_FALLBACK:
			// ping fallback writes a fixed message to indicate the fallback is reachable
			errstr = COMMAND_PING_MESSAGE
		default:
			errstr = "unknown command: " + msgDescStr(msg)
		}
	}
	if len(errstr) > 0 {
		l.handleLogWriteError(errstr)
		err = errors.New(errstr)
	}
	return
}

// logTextToOutputs walks the outputs map and attempts to write the provided
// message to each enabled output. If a write panics the output is disabled to
// avoid repeated panics; write errors are passed to the fallback writer.
func (l *logger) logTextToOutputs(msg *logMessage) {
	for output, settings := range l.outputs {
		if output != nil && settings != nil && settings.enabled {
			panicked, err := l.logTextData(output, msg)
			if panicked {
				// got panic writing, disable output for further writes
				l.outputs[output].enabled = false
			}
			if err != nil {
				l.handleLogWriteError(err.Error())
			}
		}
	}
}

// logTextData formats the message for a single output and writes it. It
// returns two values: panicked (true if a panic occurred while writing) and
// err for any write-related error. The deferred recover sets panicked and
// converts the panic into an error.
func (l *logger) logTextData(output outType, msg *logMessage) (panicked bool, err error) {
	// only returns of named result values can be changed by defer:
	// https://bytegoblin.io/blog/golang-magic-modify-return-value-using-deferred-function
	panicked = false
	proceed := true
	defer func() {
		if r := recover(); r != nil {
			panicked = true
			err = errors.New("panic writing log to output" + panicDesc(r))
		}
	}()
	level := LogLevel(msg.annex)
	context := l.outputs[output]
	if context != nil {
		proceed = level >= context.minlevel && level >= l.level
	}
	if proceed {
		buildTextMessage(l.msgbuf, msg, context)
		n, e := l.msgbuf.WriteTo(output)
		if e != nil {
			err = errors.New("error writing log to output (" + strconv.FormatInt(n, 10) + " bytes written): " + e.Error())
		}
	}
	return
}

// handleLogWriteError writes a human-readable error message to the fallback
// writer. A read lock is used since we only need consistent access to fallbck.
func (l *logger) handleLogWriteError(errormsg string) {
	l.sync.fbckMtx.RLock()
	defer l.sync.fbckMtx.RUnlock()
	if l.fallbck != nil {
		l.fallbck.Write([]byte(errormsg + "\n"))
	}
}

// buildTextMessage constructs the textual representation for a message using
// the provided outContext. The message is appended into outBuffer and the
// same buffer is returned. The buffer is expected to be reset by the caller
// before writing to the output.
func buildTextMessage(outBuffer *bytes.Buffer, msg *logMessage, context *outContext) *bytes.Buffer {
	outBuffer.Reset()
	if msg != nil {
		level := normLevel(LogLevel(msg.annex))
		withColor := false
		if context != nil {
			// optional time prefix
			if len(context.timefmt) > 0 {
				outBuffer.Write([]byte(msg.pushed.Format(context.timefmt)))
			}
			// optional numeric level id (compact path for small max)
			if context.showlvlid {
				if _LVL_MAX_for_checks_only <= 10 {
					outBuffer.Write([]byte{'[', '0' + byte(msg.annex), ']'})
				} else {
					outBuffer.Write([]byte("[" + strconv.FormatUint(uint64(msg.annex), 10) + "]"))
				}
			}
			// optional prefix map + delimiter
			if context.prefixmap != nil {
				outBuffer.Write([]byte(context.prefixmap[level]))
				outBuffer.Write(context.delimiter)
			}
			// optional color prefix (ANSI)
			if context.colormap != nil {
				withColor = true
				outBuffer.Write([]byte(ANSI_COL_PRFX))
				outBuffer.Write([]byte(context.colormap[level]))
				outBuffer.Write([]byte(ANSI_COL_SUFX))
			}
			// client name and delimiter if present
			if msg.msgclnt != nil {
				outBuffer.Write(msg.msgclnt.name)
				outBuffer.Write(context.delimiter)
			}
		}
		// the actual log text
		outBuffer.Write(msg.msgdata)
		if withColor {
			// append reset sequence if color was used
			outBuffer.Write([]byte(ANSI_COL_RESET))
		}
		// terminate line
		outBuffer.Write([]byte{'\n'})
	}
	return outBuffer
}
