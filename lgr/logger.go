package lgr

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
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

func (l *logger) SetMinLevel(minlevel logLevel) *logger {
	l.sync.chngMtx.Lock()
	defer l.sync.chngMtx.Unlock()
	l.level = normLevel(minlevel)
	return l
}

func (l *logger) SetFallback(f outType) *logger {
	l.sync.fbckMtx.Lock()
	defer l.sync.fbckMtx.Unlock()
	if f != nil {
		l.fallbck = f
	} else {
		l.fallbck = io.Discard
	}
	return l
}

func (l *logger) IsActive() bool {
	return l.state == STATE_ACTIVE
}

func (l *logger) AddOutputs(outputs ...outType) *logger {
	l.operateOutputs(outputs, func(m outList, k outType) { m[k] = true })
	return l
}

func (l *logger) RemoveOutputs(outputs ...outType) *logger {
	l.operateOutputs(outputs, func(m outList, k outType) { delete(m, k) })
	return l
}

func (l *logger) ClearOutputs() *logger {
	l.RemoveOutputs(slices.Collect(maps.Keys(l.outputs))...)
	return l
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
