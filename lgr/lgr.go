package lgr

import (
	"fmt"
	"io"
	"os"
	"slices"
	"sync"
)

type LoggerState int8

const (
	STOPPED  LoggerState = 0
	ACTIVE   LoggerState = 1
	STOPPING LoggerState = -1
)

type LogLevel uint8

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

const (
	DEFAULT_BUFF_SIZE = 32
	DEFAULT_LOG_LEVEL = INFO
)

type logMessage struct {
	message string
}

type Logger struct {
	outputs []io.Writer
	fallbck io.Writer
	channel chan logMessage
	statMtx sync.RWMutex
	chngMtx sync.RWMutex
	waitEnd sync.WaitGroup
	state   LoggerState
	level   LogLevel
}

func (l *Logger) handleLogWriteError(errormsg string) {
	l.chngMtx.RLock()
	defer l.chngMtx.RUnlock()
	fmt.Fprintln(l.fallbck, errormsg)
}

func (l *Logger) setState(newstate LoggerState) {
	l.statMtx.Lock()
	defer l.statMtx.Unlock()
	l.state = newstate
}

func (l *Logger) Log_(level LogLevel, s string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic [%v]", r)
		}
	}()
	if !l.IsActive() {
		return fmt.Errorf("logger is not active")
	}
	if level >= l.level {
		l.statMtx.RLock()
		if l.IsActive() {
			l.channel <- logMessage{s}
		}
		l.statMtx.RUnlock()
	}
	return err
}

func (l *Logger) Log(level LogLevel, s string) {
	err := l.Log_(level, s)
	if err != nil {
		l.handleLogWriteError(err.Error())
	}
}

func (l *Logger) IsActive() bool {
	return l.state == ACTIVE
}

func (l *Logger) SetFallback(w io.Writer) {
	if w != nil {
		l.chngMtx.Lock()
		l.fallbck = w
		l.chngMtx.Unlock()
	}
}

func (l *Logger) AddOutput(w io.Writer) {
	if w == nil {
		return
	}
	l.chngMtx.Lock()
	defer l.chngMtx.Unlock()
	if !slices.Contains(l.outputs, w) {
		l.outputs = append(l.outputs, w)
	}
}

func (l *Logger) RemoveOutput(w io.Writer) {
	l.chngMtx.Lock()
	defer l.chngMtx.Unlock()
	newOutputs := l.outputs[:0]
	for _, out := range l.outputs {
		if out != w {
			newOutputs = append(newOutputs, out)
		}
	}
	l.outputs = newOutputs
}

func (l *Logger) Start(level LogLevel, buffsize uint, fallback io.Writer, outputs ...io.Writer) error {
	l.statMtx.Lock()
	defer l.statMtx.Unlock()
	if l.IsActive() {
		return fmt.Errorf("logger is allready started")
	}
	l.channel = make(chan logMessage, buffsize)
	l.level = level
	l.outputs = outputs
	if fallback != nil {
		l.fallbck = fallback
	} else {
		l.fallbck = io.Discard
	}
	l.state = ACTIVE
	l.waitEnd.Go(func() { l.procced() })
	return nil
}

func (l *Logger) StartDefault() error {
	return l.Start(DEFAULT_LOG_LEVEL, DEFAULT_BUFF_SIZE, os.Stderr, os.Stdout)
}

func (l *Logger) Stop() {
	l.setState(STOPPING)
	close(l.channel)
}

func (l *Logger) Wait() {
	l.waitEnd.Wait()
}

func (l *Logger) StopAndWait() {
	l.Stop()
	l.Wait()
}

func (l *Logger) writeMsg(msg *logMessage) {
	l.chngMtx.RLock()
	defer l.chngMtx.RUnlock()
	for idx, output := range l.outputs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					l.handleLogWriteError(fmt.Sprintf("panic writing log to output #%d: %v\n", idx, r))
				}
			}()
			if output != nil {
				n, err := output.Write([]byte(msg.message))
				if err != nil {
					l.handleLogWriteError(fmt.Sprintf("error writing log to output #%d (%d bytes written): %v\n", idx, n, err))
				}
			}
		}()
	}
}

func (l *Logger) procced() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(l.fallbck, "panic proceeding log: %v\n", r)
		}
	}()
	for {
		msg, ok := <-l.channel
		if !ok {
			l.setState(STOPPED)
			return
		}
		l.writeMsg(&msg)
	}
}
