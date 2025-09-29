package lgr

import (
	"bytes"
	"fmt"
)

var outBuffer = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))

func (l *logger) procced() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(l.fallbck, "panic proceeding log: %v\n", r)
		}
	}()
	for {
		msg, opened := <-l.channel
		if !opened {
			l.setState(STATE_STOPPED)
			break
		}
		err := l.proceedMsg(&msg)
		if err != nil {
			fmt.Fprintf(l.fallbck, "error proceeding message: %v\n", err)
		}
	}
}

func (l *logger) proceedMsg(msg *logMessage) error {
	switch msg.msgtype {
	case MSG_LOG_TEXT:
		l.logTextToOutputs(msg)
	case MSG_FORBIDDEN:
		// For testing purposes only
		panic(fmt.Sprintf("panic on forbidden message type %d", msg.msgtype))
	default:
		return fmt.Errorf("unknown message type %v (data: %v)", msg.msgtype, msg.msgdata)
	}
	return nil
}

func (l *logger) logTextToOutputs(msg *logMessage) {
	l.sync.outsMtx.RLock()
	defer l.sync.outsMtx.RUnlock()
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
	panicked = false
	err = nil
	defer func() {
		if r := recover(); r != nil {
			panicked = true
			err = fmt.Errorf("panic writing log to output `%v`: %v", output, r)
		}
	}()
	//n, err := output.Write(databuff)
	n, err := buildTextMessage(msg, l.outputs[output]).WriteTo(output)
	if err != nil {
		err = fmt.Errorf("error writing log to output `%v` (%d bytes written): %v", output, n, err)
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

func buildTextMessage(msg *logMessage, deco *outDecoration) *bytes.Buffer {
	// переделать под bytes.Buffer + sync.Pool
	outBuffer.Reset()
	level := normLevel(msg.level)
	withColor := deco != nil && deco.ansicolormap != nil
	withPrefix := deco != nil && deco.lvlprefixmap != nil
	if len(deco.timeformat) > 0 {
		outBuffer.WriteString(msg.msgtime.Format(deco.timeformat))
	}
	if deco.showlvlnum {
		fmt.Fprintf(outBuffer, "[%d]", msg.level)
	}
	if withPrefix {
		outBuffer.WriteString(deco.lvlprefixmap[level])
		outBuffer.WriteString(deco.delimiter)
	}
	if withColor {
		outBuffer.WriteString(ANSI_COLOR_PREFIX)
		outBuffer.WriteString(deco.ansicolormap[level])
		outBuffer.WriteString(ANSI_COLOR_SUFFIX)
	}
	if msg.msgclnt != nil && len(msg.msgclnt.name) > 0 {
		outBuffer.WriteString(msg.msgclnt.name)
		outBuffer.WriteString(deco.delimiter)
	}
	outBuffer.WriteString(msg.msgdata)
	if withColor {
		outBuffer.WriteString(ANSI_COLOR_RESET)
	}
	outBuffer.WriteByte('\n')
	return outBuffer
}
