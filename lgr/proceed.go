package lgr

// never use fmt in threads!

import (
	"bytes"
	"errors"
	"strconv"
)

func (l *logger) fbckWriteln(s string) {
	l.fallbck.Write([]byte(s + "\n"))
}

func msgDescStr(m *logMessage) string {
	return "type=" + strconv.Itoa(int(m.msgtype)) +
		" annex=" + strconv.Itoa(int(m.annex)) +
		" data=`" + string(m.msgdata) + "`"
}

func (l *logger) setState(newstate lgrState) {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	l.state = normState(newstate)
}

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
		//convert successfully executed command message to text with level TRACE
		msg.msgtype = MSG_LOG_TEXT
		msg.annex = basetype(LVL_TRACE)
		msg.msgdata = []byte("<COMMAND: " + msgDescStr(msg) + ">")
		fallthrough
	case MSG_LOG_TEXT:
		l.logTextToOutputs(msg)
	case MSG_FORBIDDEN:
		// For testing purposes only
		panic("panic on forbidden message type: " + msgDescStr(msg))
	default:
		return errors.New("unknown message type: " + msgDescStr(msg))
	}
	return nil
}

const COMMAND_PING_MESSAGE = "<ping>"

func (l *logger) proceedCmd(msg *logMessage) (err error) {
	l.sync.clntMtx.RLock()
	defer l.sync.clntMtx.RUnlock()
	errstr := ""
	if msg == nil {
		errstr = "nil command message, nothing to proceed"
	} else {
		switch cmdType(msg.annex) {
		case CMD_CLIENT_SET_LEVEL:
			errstr = clientChangeFromCmdMsg(msg, func(lc *logClient, data []byte) {
				lc.minLevel = normLevel(LogLevel(data[0]))
			})
		case CMD_CLIENT_SET_NAME:
			errstr = clientChangeFromCmdMsg(msg, func(lc *logClient, data []byte) {
				lc.name = data
			})
		case CMD_DUMMY, CMD_CLIENT_DUMMY:
			// do nothing
		case CMD_PING_FALLBACK:
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

func (l *logger) handleLogWriteError(errormsg string) {
	l.sync.fbckMtx.RLock()
	defer l.sync.fbckMtx.RUnlock()
	if l.fallbck != nil {
		l.fallbck.Write([]byte(errormsg + "\n"))
	}
}

func buildTextMessage(outBuffer *bytes.Buffer, msg *logMessage, context *outContext) *bytes.Buffer {
	outBuffer.Reset()
	if msg != nil {
		level := normLevel(LogLevel(msg.annex))
		withColor := false
		if context != nil {
			if len(context.timefmt) > 0 {
				outBuffer.Write([]byte(msg.pushed.Format(context.timefmt)))
			}
			if context.showlvlid {
				if _LVL_MAX_for_checks_only <= 10 {
					outBuffer.Write([]byte{'[', '0' + byte(msg.annex), ']'})
				} else {
					outBuffer.Write([]byte("[" + strconv.FormatUint(uint64(msg.annex), 10) + "]"))
				}
			}
			if context.prefixmap != nil {
				outBuffer.Write([]byte(context.prefixmap[level]))
				outBuffer.Write(context.delimiter)
			}
			if context.colormap != nil {
				withColor = true
				outBuffer.Write([]byte(ANSI_COL_PRFX))
				outBuffer.Write([]byte(context.colormap[level]))
				outBuffer.Write([]byte(ANSI_COL_SUFX))
			}
			if msg.msgclnt != nil {
				outBuffer.Write(msg.msgclnt.name)
				outBuffer.Write(context.delimiter)
			}
		}
		outBuffer.Write(msg.msgdata)
		if withColor {
			outBuffer.Write([]byte(ANSI_COL_RESET))
		}
		outBuffer.Write([]byte{'\n'})
	}
	return outBuffer
}
