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
	l.state = STATE_STOPPED
	l.outputs = OutList{}
	l.clients = Clients{}
	l.SetMinLevel(level)
	l.SetFallback(fallback)
	l.AddOutputs(outputs...)
	return l
}

func Init() *Logger {
	return InitWithParams(DEFAULT_LOG_LEVEL, os.Stderr, os.Stdout) //DEFAULT_BUFF_SIZE?
}

func (l *Logger) Start(buffsize int) error {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	if l.IsActive() {
		return fmt.Errorf("logger is allready started")
	}
	if buffsize <= 0 {
		buffsize = DEFAULT_BUFF_SIZE
	}
	l.channel = make(chan logMessage, buffsize)
	l.sync.waitEnd.Go(func() { l.procced() })
	l.state = STATE_ACTIVE
	return nil
}

func (l *Logger) Stop() {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	if l.IsActive() {
		l.state = STATE_STOPPING
		close(l.channel)
	}
}

func (l *Logger) Wait() {
	l.sync.waitEnd.Wait()
}

func (l *Logger) StopAndWait() {
	l.Stop()
	l.Wait()
}

func (l *Logger) Log(level LogLevel, s string) {
	err := l.LogE(level, s)
	if err != nil {
		l.handleLogWriteError(err.Error())
	}
}

func (l *Logger) LogC(lc *logClient, level LogLevel, s string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic [%v]", r)
		}
	}()
	if level < l.level {
		return
	}
	if l.level >= _LVL_MAX_FOR_CHECKS_ONLY {
		//For testing purposes only, should never happen in real code
		//because SetLogLevel() prevents setting invalid levels
		panic("panic on forbidden log level")
	}
	l.sync.statMtx.RLock()
	defer l.sync.statMtx.RUnlock()
	if !l.IsActive() {
		return fmt.Errorf("logger is not active")
	} else {
		if l.channel == nil {
			return fmt.Errorf("logger channel is nil")
		}
		// will panic if channel is closed
		l.channel <- logMessage{msgclnt: lc, msgtype: MSG_LOG_TEXT, msgdata: s}
	}
	return
}

func (l *Logger) LogE(level LogLevel, s string) (err error) {
	return l.LogC(nil, level, s)
}

func (l *Logger) setState(newstate LgrState) {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	l.state = normState(newstate)
}

func (l *Logger) SetMinLevel(minlevel LogLevel) {
	l.sync.chngMtx.Lock()
	defer l.sync.chngMtx.Unlock()
	l.level = normLevel(minlevel)
}

func (l *Logger) IsActive() bool {
	return l.state == STATE_ACTIVE
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
