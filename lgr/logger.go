package lgr

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"time"
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

func (st *outContext) IsEnabled() bool {
	return st.enabled
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

func (l *logger) SetOutputLevelPrefix(output outType, prefixmap *LevelMap, delimiter string) *logger {
	return l.changeOutSettings(output, func(s *outContext) {
		s.SetPrefix(prefixmap)
		s.SetDelimiter(delimiter)
	})
}

func (l *logger) SetOutputLevelColor(output outType, colormap *LevelMap) *logger {
	return l.changeOutSettings(output, func(s *outContext) {
		s.SetColor(colormap)
	})
}

func (l *logger) SetOutputTimeFormat(output outType, format string) *logger {
	return l.changeOutSettings(output, func(s *outContext) {
		s.SetTimeFormat(format)
	})
}

func (l *logger) ShowOutputLevelCode(output outType) *logger {
	return l.changeOutSettings(output, func(s *outContext) {
		s.ShowLevelCode()
	})
}

func (l *logger) SetOutputMinLevel(output outType, minlevel LogLevel) *logger {
	return l.changeOutSettings(output, func(s *outContext) {
		s.SetMinLevel(minlevel)
	})
}

func (l *logger) changeOutSettings(output outType, f func(s *outContext)) *logger {
	if l.outputs[output] != nil {
		l.sync.outsMtx.Lock()
		defer l.sync.outsMtx.Unlock()
		f(l.outputs[output])
	}
	return l
}

// Getters /////////////////////////////////////////////
func (l *logger) Context(output outType) *outContext {
	return l.outputs[output]
}

// Setters /////////////////////////////////////////////
func (st *outContext) SetDelimiter(delimiter string) *outContext {
	st.delimiter = []byte(delimiter)
	return st
}

func (st *outContext) SetPrefix(prefixmap *LevelMap) *outContext {
	st.prefixmap = prefixmap
	return st
}

func (st *outContext) SetColor(colormap *LevelMap) *outContext {
	st.colormap = colormap
	return st
}

func (st *outContext) SetTimeFormat(timeformat string) *outContext {
	st.timefmt = timeformat
	return st
}

func (st *outContext) ShowLevelCode() *outContext {
	st.showlvlid = true
	return st
}

func (st *outContext) SetMinLevel(minlevel LogLevel) *outContext {
	st.minlevel = normLevel(minlevel)
	return st
}

// Messages manipulations ///////////////////////////////

func (l *logger) pushMessage(msg *logMessage) (t time.Time, err error) {
	l.sync.statMtx.RLock()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic [%v]", r)
		}
		l.sync.statMtx.RUnlock()
	}()
	t = time.Now()
	if msg == nil {
		err = fmt.Errorf("log message is nil")
	} else {
		msg.pushed = t
		if !l.IsActive() {
			err = fmt.Errorf("logger is not active")
		} else {
			if l.channel == nil {
				err = fmt.Errorf("logger channel is nil")
			} else {
				// will panic if channel is closed (with recover and setting error)
				l.channel <- *msg
			}
		}
	}
	return t, err
}

func makeTextMessage(lc *logClient, level LogLevel, data []byte) *logMessage {
	return &logMessage{
		msgtype: MSG_LOG_TEXT,
		msgclnt: lc,
		msgdata: data,
		annex:   basetype(level),
	}
}

func makeCmdMessage(lc *logClient, cmd cmdType, data []byte) *logMessage {
	return &logMessage{
		msgtype: MSG_COMMAND,
		msgclnt: lc,
		msgdata: data,
		annex:   basetype(cmd),
	}
}

// Client manipulation ///////////////////////////////

func (l *logger) NewClient(name string, minlevel LogLevel) *logClient {
	client := &logClient{
		logger:   l,
		name:     []byte(name),
		minLevel: normLevel(minlevel),
		curLevel: LVL_UNKNOWN,
	}
	//l.clients[client] = true // For further "disable client"
	return client
}

func (l *logger) PushClientMinLevel(lc *logClient, minlevel LogLevel) (t time.Time, err error) {
	// Change client settings by commands (sent messages has to be printed with previous settings)
	if lc == nil {
		err = fmt.Errorf("client is nil")
	} else if lc.logger != l {
		err = fmt.Errorf("alien client (belongs to nil or another logger)")
	} else {
		t, err = l.pushMessage(makeCmdMessage(lc, CMD_SET_CLIENT_MINLEVEL, []byte{byte(minlevel)}))
	}
	return t, err
}
