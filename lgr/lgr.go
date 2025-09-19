package lgr

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
)

func Init(level LogLevel, fallback outType, outputs ...outType) *Logger {
	l := new(Logger)
	print(l.outputs)
	delete(l.outputs, nil) // just in case
	l.state = STOPPED
	l.level = level
	l.outputs = outList{}
	l.AddOutputs(outputs...)
	l.SetFallback(fallback)
	return l
}

func InitDefault() *Logger {
	return Init(DEFAULT_LOG_LEVEL, os.Stderr, os.Stdout) //DEFAULT_BUFF_SIZE?
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

func (l *Logger) setState(newstate LoggerState) {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
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
		l.sync.statMtx.RLock()
		if l.IsActive() {
			l.channel <- logMessage{s}
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

func (l *Logger) IsActive() bool {
	return l.state == ACTIVE
}

func (l *Logger) SetFallback(f outType) {
	l.sync.outsMtx.Lock()
	defer l.sync.outsMtx.Unlock()
	if f != nil {
		l.fallbck = f
	} else {
		l.fallbck = io.Discard
	}
}

func (l *Logger) SetLogLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) operateOutputs(slice []outType, operation func(m outList, k outType)) {
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

func (l *Logger) AddOutputs(outputs ...outType) {
	l.operateOutputs(outputs, func(m outList, k outType) { m[k] = true })
}

func (l *Logger) RemoveOutputs(outputs ...outType) {
	l.operateOutputs(outputs, func(m outList, k outType) { delete(m, k) })
}

func (l *Logger) ClearOutputs() {
	l.RemoveOutputs(slices.Collect(maps.Keys(l.outputs))...)
}
