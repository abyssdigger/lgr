package lgr

import (
	"fmt"
)

func (l *Logger) procced() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(l.fallbck, "panic proceeding log: %v\n", r)
		}
	}()
	for {
		msg, opened := <-l.channel
		if !opened {
			l.setState(STOPPED)
			break
		}
		err := l.proceedMsg(&msg)
		if err != nil {
			fmt.Fprintf(l.fallbck, "error proceeding command %s: %v\n", msg.msgtext, err)
		}
	}
}

func (l *Logger) proceedMsg(msg *logMessage) error {
	switch msg.msgtype {
	case MSG_LOG_TEXT:
		l.logTextToOutputs(msg)
	default:
		return fmt.Errorf("unknown message type %v (text: %s)", msg.msgtype, msg.msgtext)
	}
	return nil
}

func (l *Logger) logTextToOutputs(msg *logMessage) {
	l.sync.outsMtx.RLock()
	defer l.sync.outsMtx.RUnlock()
	for output, enabled := range l.outputs {
		if enabled && output != nil && msg != nil {
			panicked, err := l.logData(output, []byte(msg.msgtext))
			if panicked {
				// got panic writing, disable output
				l.outputs[output] = false
			}
			if err != nil {
				l.handleLogWriteError(err.Error())
			}
		}
	}
}

func (l *Logger) logData(output OutType, data []byte) (panicked bool, err error) {
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
	n, err := output.Write(data)
	if err != nil {
		err = fmt.Errorf("error writing log to output `%v` (%d bytes written): %v", output, n, err)
	}
	return
}

func (l *Logger) handleLogWriteError(errormsg string) {
	l.sync.outsMtx.RLock()
	defer l.sync.outsMtx.RUnlock()
	fmt.Fprintln(l.fallbck, errormsg)
}
