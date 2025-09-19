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
		l.proceedMsg(&msg)
	}
}

func (l *Logger) proceedMsg(msg *logMessage) {
	l.sync.outsMtx.RLock()
	defer l.sync.outsMtx.RUnlock()
	for output, enabled := range l.outputs {
		if enabled && output != nil {
			if !l.writeMsg(output, msg) {
				l.outputs[output] = false
			}
		}
	}
}

func (l *Logger) writeMsg(output outType, msg *logMessage) (result bool) {
	// only returns of named result values can be changed by defer:
	// https://bytegoblin.io/blog/golang-magic-modify-return-value-using-deferred-function
	result = true
	defer func() {
		if r := recover(); r != nil {
			l.handleLogWriteError(
				fmt.Sprintf("panic writing log to output `%v`: %v\n", output, r))
			result = false
		}
	}()
	n, err := output.Write([]byte(msg.message))
	if err != nil {
		l.handleLogWriteError(
			fmt.Sprintf("error writing log to output `%v` (%d bytes written): %v\n", output, n, err))
	}
	return result
}

func (l *Logger) handleLogWriteError(errormsg string) {
	l.sync.outsMtx.RLock()
	defer l.sync.outsMtx.RUnlock()
	fmt.Fprintln(l.fallbck, errormsg)
}
