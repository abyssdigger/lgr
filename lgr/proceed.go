package lgr

import (
	"bytes"
	"fmt"
)

func (l *logger) setState(newstate lgrState) {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	l.state = normState(newstate)
}

func (l *logger) procced() {
	l.msgbuf = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(l.fallbck, "panic proceeding log: %v\n", r)
		}
		l.msgbuf = nil
		l.setState(STATE_STOPPED)
	}()
	for {
		msg, opened := <-l.channel
		if !opened {
			break
		}
		err := l.proceedMsg(&msg)
		if err != nil {
			fmt.Fprintf(l.fallbck, "error proceeding message: %v\n", err)
		}
	}
}

func (l *logger) proceedMsg(msg *logMessage) error {
	l.sync.procMtx.RLock()
	defer func() {
		l.sync.procMtx.RUnlock()
	}()
	switch msg.msgtype {
	case MSG_LOG_TEXT:
		l.logTextToOutputs(msg)
	case MSG_COMMAND:
		l.proceedCmd(msg)
	case MSG_FORBIDDEN:
		// For testing purposes only
		panic(fmt.Sprintf("panic on forbidden message type %d", msg.msgtype))
	default:
		return fmt.Errorf("unknown message type %v (data: %v)", msg.msgtype, msg.msgdata)
	}
	return nil
}

func (l *logger) proceedCmd(msg *logMessage) {
	l.sync.clntMtx.RLock()
	defer l.sync.clntMtx.RUnlock()
	errstr := ""
	if msg == nil {
		errstr = "nil command message, nothing to proceed"
	} else {
		switch cmdType(msg.annex) {
		case CMD_SET_CLIENT_MINLEVEL:
			if len(msg.msgdata) < 1 {
				errstr = "no level in command message"
			} else if msg.msgclnt == nil {
				errstr = "nil client in command message"
			} else {
				msg.msgclnt.minLevel = normLevel(LogLevel(msg.msgdata[0]))
			}
		case CMD_DUMMY:
			// do nothing
		case CMD_PING_FALLBACK:
			errstr = "<ping>"
		default:
			errstr = fmt.Sprintf("unknown command %d (data: %v)", msg.annex, msg.msgdata)
		}
	}
	if len(errstr) > 0 {
		l.handleLogWriteError(errstr)
	}
}

func (l *logger) logTextToOutputs(msg *logMessage) {
	for output, settings := range l.outputs {
		if settings.enabled && output != nil {
			panicked, err := l.logData(output, msg)
			if panicked {
				// got panic writing, disable output
				l.outputs[output].enabled = false
			}
			if err != nil {
				l.handleLogWriteError(err.Error())
			}
		}
	}
}

func (l *logger) logData(output outType, msg *logMessage) (panicked bool, err error) {
	// only returns of named result values can be changed by defer:
	// https://bytegoblin.io/blog/golang-magic-modify-return-value-using-deferred-function
	var n int64
	panicked = false
	proceed := true
	err = nil
	defer func() {
		if r := recover(); r != nil {
			panicked = true
			err = fmt.Errorf("panic writing log to output `%v`: %v", output, r)
		}
	}()
	//n, err := output.Write(databuff)
	context := l.outputs[output]
	if context != nil {
		proceed = LogLevel(msg.annex) >= context.minlevel
	}
	if proceed {
		n, err = buildTextMessage(l.msgbuf, msg, context).WriteTo(output)
		if err != nil {
			err = fmt.Errorf("error writing log to output `%v` (%d bytes written): %v", output, n, err)
		}
	}
	return
}

func (l *logger) handleLogWriteError(errormsg string) {
	l.sync.fbckMtx.RLock()
	defer l.sync.fbckMtx.RUnlock()
	if l.fallbck != nil {
		fmt.Fprintln(l.fallbck, errormsg)
	}
}

func buildTextMessage(outBuffer *bytes.Buffer, msg *logMessage, context *outContext) *bytes.Buffer {
	outBuffer.Reset()
	if msg != nil {
		level := normLevel(LogLevel(msg.annex))
		withColor := false
		if context != nil {
			if len(context.timefmt) > 0 {
				outBuffer.WriteString(msg.pushed.Format(context.timefmt))
			}
			if context.showlvlid {
				if _LVL_MAX_for_checks_only <= 10 {
					outBuffer.Write([]byte{'[', '0' + byte(msg.annex), ']'})
				} else {
					fmt.Fprintf(outBuffer, "[%d]", msg.annex)
				}
			}
			if context.prefixmap != nil {
				outBuffer.WriteString(context.prefixmap[level])
				outBuffer.Write(context.delimiter)
			}
			if context.colormap != nil {
				withColor = true
				outBuffer.WriteString(ANSI_COL_PRFX)
				outBuffer.WriteString(context.colormap[level])
				outBuffer.WriteString(ANSI_COL_SUFX)
			}
			if msg.msgclnt != nil {
				outBuffer.Write(msg.msgclnt.name)
				outBuffer.Write(context.delimiter)
			}
		}
		outBuffer.Write(msg.msgdata)
		if withColor {
			outBuffer.WriteString(ANSI_COL_RESET)
		}
		outBuffer.WriteByte('\n')
	}
	return outBuffer
}
