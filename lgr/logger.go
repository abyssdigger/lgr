package lgr

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
)

func InitAndStart(buffsize int, outputs ...outType) (l *logger) {
	l = Init(outputs...)
	l.Start(buffsize)
	return
}

func Init(outputs ...outType) *logger {
	return InitWithParams(DEFAULT_LOG_LEVEL, os.Stderr, outputs...)
}

func InitWithParams(level LogLevel, fallback outType, outputs ...outType) *logger {
	l := new(logger)
	l.state = STATE_STOPPED
	l.outputs = outList{}
	//l.clients = clientMap{}
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
		buffsize = DEFAULT_MSG_BUFF
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

func (l *logger) SetMinLevel(minlevel LogLevel) *logger {
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
	l.operateOutputs(outputs, func(m *outList, k outType) {
		(*m)[k] = &outContext{
			enabled: true,
		}
	})
	return l
}

// Outputs //////////////////////////////////////////

func (l *logger) RemoveOutputs(outputs ...outType) *logger {
	l.operateOutputs(outputs, func(m *outList, k outType) { delete(*m, k) })
	return l
}

func (l *logger) ClearOutputs() *logger {
	l.RemoveOutputs(slices.Collect(maps.Keys(l.outputs))...)
	return l
}

func (l *logger) operateOutputs(slice []outType, operation func(m *outList, k outType)) {
	if len(slice) == 0 {
		return
	}
	l.sync.outsMtx.Lock()
	defer l.sync.outsMtx.Unlock()
	for _, output := range slice {
		if output != nil {
			operation(&l.outputs, output)
		}
	}
}

// Settings //////////////////////////////////////////

func (l *logger) SetLevelPrefix(output outType, prefixmap *LevelMap, delimiter string) *logger {
	return l.changeOutSettings(output, func(s *outContext) {
		s.prefixmap = prefixmap
		s.delimiter = []byte(delimiter)
	})
}

func (l *logger) SetLevelColor(output outType, colormap *LevelMap) *logger {
	return l.changeOutSettings(output, func(s *outContext) {
		s.colormap = colormap
	})
}

func (l *logger) SetTimeFormat(output outType, format string) *logger {
	return l.changeOutSettings(output, func(s *outContext) {
		s.timefmt = format
	})
}

func (l *logger) ShowLevelCode(output outType) *logger {
	return l.changeOutSettings(output, func(s *outContext) {
		s.lvlcode = true
	})
}

func (l *logger) changeOutSettings(output outType, f func(s *outContext)) *logger {
	if l.outputs[output] != nil {
		f(l.outputs[output])
	}
	return l
}
