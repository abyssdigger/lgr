package lgr

import (
	"errors"
	"io"
	"maps"
	"os"
	"slices"
	"time"
)

const (
	ERROR_MESSAGE_LOGGER_STARTED  = "logger is allready started"
	ERROR_MESSAGE_LOGGER_INACTIVE = "logger is not active"
	ERROR_MESSAGE_CHANNEL_IS_NIL  = "logger channel is nil"
	ERROR_MESSAGE_LOG_MSG_IS_NIL  = "log message is nil"
	ERROR_MESSAGE_CLIENT_IS_ALIEN = "logger client is nil or alien (belongs to another logger or nil)"
	ERROR_MESSAGE_CLIENT_IS_NIL   = "client is nil"
	ERROR_MESSAGE_NON_CLIENT_CMD  = "non-client command"
	ERROR_MESSAGE_CMD_EMPTY_DATA  = "no data in command message"
	ERROR_MESSAGE_CMD_NIL_CLIENT  = "nil client in command message"
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
		return errors.New(ERROR_MESSAGE_LOGGER_STARTED)
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
			err = errors.New("panic" + panicDesc(r))
		}
		l.sync.statMtx.RUnlock()
	}()
	t = time.Now()
	if msg == nil {
		err = errors.New(ERROR_MESSAGE_LOG_MSG_IS_NIL)
	} else {
		msg.pushed = t
		if !l.IsActive() {
			err = errors.New(ERROR_MESSAGE_LOGGER_INACTIVE)
		} else {
			if l.channel == nil {
				err = errors.New(ERROR_MESSAGE_CHANNEL_IS_NIL)
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
		enabled:  true,
	}
	//l.clients[client] = true // For further "disable client"
	return client
}

func (l *logger) checkClient(lc *logClient) (err error) {
	if lc == nil {
		err = errors.New(ERROR_MESSAGE_CLIENT_IS_NIL)
	} else if lc.logger != l {
		err = errors.New(ERROR_MESSAGE_CLIENT_IS_ALIEN)
	}
	return
}

func (l *logger) SetClientEnabled(lc *logClient, enabled bool) (err error) {
	if err = l.checkClient(lc); err == nil {
		lc.enabled = enabled
	}
	return
}

// Level and name changes must not apply to allready queued messages so implemented as queued command
func (l *logger) SetClientMinLevel(lc *logClient, minlevel LogLevel) (t time.Time, err error) {
	return l.runClientCommand(lc, CMD_CLIENT_SET_LEVEL, []byte{byte(minlevel)})
}

func (l *logger) SetClientName(lc *logClient, newname string) (time.Time, error) {
	return l.runClientCommand(lc, CMD_CLIENT_SET_NAME, []byte(newname))
}

func (l *logger) runClientCommand(lc *logClient, cmd cmdType, data []byte) (t time.Time, err error) {
	// Change client settings by commands (sent messages has to be printed with previous settings)
	err = l.checkClient(lc)
	if err == nil {
		if cmd < _CMD_CLIENT_commands_min || cmd > _CMD_CLIENT_commands_max {
			err = errors.New(ERROR_MESSAGE_NON_CLIENT_CMD)
		} else {
			t, err = l.pushMessage(makeCmdMessage(lc, cmd, data))
		}
	}
	return t, err
}

func clientChangeFromCmdMsg(msg *logMessage, f func(*logClient, []byte)) (errstr string) {
	if len(msg.msgdata) < 1 {
		errstr = ERROR_MESSAGE_CMD_EMPTY_DATA
	} else if msg.msgclnt == nil {
		errstr = ERROR_MESSAGE_CMD_NIL_CLIENT
	} else {
		f(msg.msgclnt, msg.msgdata)
	}
	return
}
