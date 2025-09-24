package lgr

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"time"
)

func InitAndStart(buffsize int) (l *logger) {
	l = Init()
	l.Start(buffsize)
	return
}

func Init() *logger {
	return InitWithParams(DEFAULT_LOG_LEVEL, os.Stderr, os.Stdout) //DEFAULT_BUFF_SIZE?
}

func InitWithParams(level logLevel, fallback outType, outputs ...outType) *logger {
	l := new(logger)
	l.state = STATE_STOPPED
	l.outputs = outList{}
	l.clients = clients{}
	l.SetMinLevel(level)
	l.SetFallback(fallback)
	l.AddOutputs(outputs...)
	return l
}

func (l *logger) Start(buffsize int) error {
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

func (l *logger) Stop() {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	if l.IsActive() {
		l.state = STATE_STOPPING
		close(l.channel)
	}
}

func (l *logger) Wait() {
	l.sync.waitEnd.Wait()
}

func (l *logger) StopAndWait() {
	l.Stop()
	l.Wait()
}

func (l *logger) Log(level logLevel, s string) {
	err := l.LogE(level, s)
	if err != nil {
		l.handleLogWriteError(err.Error())
	}
}

func (l *logger) LogC(lc *logClient, level logLevel, s string) (err error) {
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
		l.channel <- logMessage{level: level, msgclnt: lc, msgtime: time.Now(), msgtype: MSG_LOG_TEXT, msgdata: s}
	}
	return
}

func (l *logger) LogE(level logLevel, s string) (err error) {
	return l.LogC(nil, level, s)
}

func (l *logger) SetMinLevel(minlevel logLevel) {
	l.sync.chngMtx.Lock()
	defer l.sync.chngMtx.Unlock()
	l.level = normLevel(minlevel)
}

func (l *logger) SetTimeFormat(s string) {
	l.sync.chngMtx.Lock()
	defer l.sync.chngMtx.Unlock()
	l.timefmt = s
}

func (l *logger) SetFallback(f outType) {
	l.sync.outsMtx.Lock()
	defer l.sync.outsMtx.Unlock()
	if f != nil {
		l.fallbck = f
	} else {
		l.fallbck = io.Discard
	}
}

func (l *logger) IsActive() bool {
	return l.state == STATE_ACTIVE
}

func (l *logger) AddOutputs(outputs ...outType) {
	l.operateOutputs(outputs, func(m outList, k outType) { m[k] = true })
}

func (l *logger) RemoveOutputs(outputs ...outType) {
	l.operateOutputs(outputs, func(m outList, k outType) { delete(m, k) })
}

func (l *logger) ClearOutputs() {
	l.RemoveOutputs(slices.Collect(maps.Keys(l.outputs))...)
}

func (l *logger) operateOutputs(slice []outType, operation func(m outList, k outType)) {
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

func (l *logger) setState(newstate lgrState) {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	l.state = normState(newstate)
}
