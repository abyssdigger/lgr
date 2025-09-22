package lgr

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
)

func InitWithParams(level LogLevel, fallback OutType, outputs ...OutType) *Logger {
	l := new(Logger)
	delete(l.outputs, nil) // just in case
	l.state = STOPPED
	l.level = level
	l.outputs = OutList{}
	l.AddOutputs(outputs...)
	l.SetFallback(fallback)
	return l
}

func Init() *Logger {
	return InitWithParams(DEFAULT_LOG_LEVEL, os.Stderr, os.Stdout) //DEFAULT_BUFF_SIZE?
}

func (l *Logger) Start(buffsize uint) error {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	if l.IsActive() {
		return fmt.Errorf("logger is allready started")
	}
	l.channel = make(chan logMessage, buffsize)
	l.sync.waitEnd.Go(func() { l.procced() })
	l.state = ACTIVE
	return nil
}

func (l *Logger) Stop() {
	l.setState(STOPPING)
	close(l.channel)
}

func (l *Logger) Wait() {
	l.sync.waitEnd.Wait()
}

func (l *Logger) StopAndWait() {
	l.Stop()
	l.Wait()
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
		l.sync.statMtx.RLock()
		if l.IsActive() {
			l.channel <- logMessage{msgtype: MSG_LOG_TEXT, msgtext: s}
		}
		l.sync.statMtx.RUnlock()
	}
	return err
}

func (l *Logger) Log(level LogLevel, s string) {
	err := l.Log_(level, s)
	if err != nil {
		l.handleLogWriteError(err.Error())
	}
}

func (l *Logger) setState(newstate LoggerState) {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	l.state = newstate
}

func (l *Logger) SetLogLevel(level LogLevel) {
	if level < _MAX_FOR_CHECKS_ONLY {
		l.level = level
	} else {
		l.level = _MAX_FOR_CHECKS_ONLY - 1
	}
}

func (l *Logger) IsActive() bool {
	return l.state == ACTIVE
}

func (l *Logger) SetFallback(f OutType) {
	l.sync.outsMtx.Lock()
	defer l.sync.outsMtx.Unlock()
	if f != nil {
		l.fallbck = f
	} else {
		l.fallbck = io.Discard
	}
}

func (l *Logger) AddOutputs(outputs ...OutType) {
	l.operateOutputs(outputs, func(m OutList, k OutType) { m[k] = true })
}

func (l *Logger) RemoveOutputs(outputs ...OutType) {
	l.operateOutputs(outputs, func(m OutList, k OutType) { delete(m, k) })
}

func (l *Logger) ClearOutputs() {
	l.RemoveOutputs(slices.Collect(maps.Keys(l.outputs))...)
}

func (l *Logger) operateOutputs(slice []OutType, operation func(m OutList, k OutType)) {
	if len(slice) == 0 {
		return
	}
	l.sync.outsMtx.Lock()
	defer l.sync.outsMtx.Unlock()
	for _, output := range slice {
		if output != nil {
			operation(l.outputs, output)
		}
	}
}
