package lgr

import (
	"fmt"
	"io"
	"os"
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
	errout  io.Writer
	channel chan logMessage
	statMtx sync.RWMutex
	waitEnd sync.WaitGroup
	state   LoggerState
	level   LogLevel
}

func (l *Logger) Log(level LogLevel, s string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic [%v]", r)
		}
	}()
	if !l.IsActive() {
		return fmt.Errorf("logger is not active")
	}
	if level <= l.level {
		l.statMtx.RLock()
		if l.IsActive() {
			l.channel <- logMessage{s}
		}
		l.statMtx.RUnlock()
	}
	return err
}

func (l *Logger) setState(newstate LoggerState) {
	l.statMtx.Lock()
	l.state = newstate
	l.statMtx.Unlock()
}

func (l *Logger) IsActive() bool {
	return l.state == ACTIVE
}

func (l *Logger) Start(level LogLevel, buffsize uint, outputs ...io.Writer) error {
	l.statMtx.Lock()
	defer l.statMtx.Unlock()
	if l.IsActive() {
		return fmt.Errorf("logger is allready started")
	}
	l.channel = make(chan logMessage, buffsize)
	l.level = level
	l.outputs = outputs
	l.errout = os.Stderr
	l.state = ACTIVE
	l.waitEnd.Go(func() { l.procced() })
	return nil
}

func (l *Logger) StartDefault() error {
	return l.Start(DEFAULT_LOG_LEVEL, DEFAULT_BUFF_SIZE, os.Stdout)
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

func (l *Logger) procced() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(l.errout, "error proceeding log: %v", r)
		}
	}()
	for {
		msg, ok := <-l.channel
		if !ok {
			l.setState(STOPPED)
			return
		}
		n, err := io.MultiWriter(l.outputs...).Write([]byte(msg.message))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing log (%d bytes written): %v\n", n, err)
		}
	}
}
