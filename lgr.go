package main

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const LOG_BUFFFER_SIZE = 10

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

func NewLogger(logLevel LogLevel, outputs ...io.Writer) *Logger {
	l := new(Logger)
	l.level = logLevel
	l.outputs = outputs
	l.errout = os.Stderr
	return l
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
	if l.state != newstate {
		l.statMtx.Lock()
		l.state = newstate
		l.statMtx.Unlock()
	}
}

func (l *Logger) IsActive() bool {
	return l.state == ACTIVE
}

func (l *Logger) Start() error {
	l.statMtx.Lock()
	defer l.statMtx.Unlock()
	if l.IsActive() {
		return fmt.Errorf("logger is not stopped")
	}
	l.channel = make(chan logMessage, LOG_BUFFFER_SIZE)
	l.state = ACTIVE
	l.waitEnd.Go(func() { l.procced() })
	print("Logger started\n")
	return nil
}

func (l *Logger) Stop() {
	l.setState(STOPPING)
	close(l.channel)
}

func (l *Logger) Wait() {
	l.waitEnd.Wait()
}

func (l *Logger) procced() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(l.errout, "error proceeding log: %v", r)
		}
		print("EXIT")
	}()
	for {
		msg, ok := <-l.channel
		if !ok {
			l.setState(STOPPED)
			return
		}
		n, err := io.MultiWriter(l.outputs...).Write([]byte(msg.message))
		time.Sleep(time.Millisecond * 100) // Simulate some delay
		if err != nil {
			fmt.Fprintf(os.Stderr, "error writing log message (%d bytes written): %v\n", n, err)
		}
	}
}

// ///////////////////////////////////////////////////////////////////////
func main() {
	logger := NewLogger(INFO, os.Stdout, os.Stderr)
	for i := 0; i < 2; i++ {
		logger.Start()
		for i := 0; i < 10; i++ {
			err := logger.Log(DEBUG, "LOG! #"+fmt.Sprint(i+1)+"\n")
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Logged #", i+1)
			}
		}
		//logger.chanCmd <- "STOP"
		fmt.Println("Stopping logger...")
		logger.Stop()
		//time.Sleep(time.Second)
		logger.Wait()
		fmt.Println("Finita la comedia #", i)
	}
}
