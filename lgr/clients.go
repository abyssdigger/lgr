package lgr

import (
	"fmt"
	"time"
)

func (l *logger) NewClient(name string, level LogLevel) *logClient {
	client := &logClient{
		logger:   l,
		name:     []byte(name),
		minLevel: normLevel(level),
		curLevel: LVL_UNKNOWN,
	}
	//l.clients[client] = true // For further "disable client"
	return client
}

/*func (l *logger) DelClient(lc *logClient) {
	delete(l.clients, lc)
}*/

func (lc *logClient) Log_with_err(level LogLevel, s string) (*logMessage, error) {
	return lc.LogBytes_with_err(level, []byte(s))
}

func (lc *logClient) LogBytes_with_err(level LogLevel, data []byte) (msg *logMessage, err error) {
	t := time.Now()
	l := lc.logger
	if l == nil {
		return nil, fmt.Errorf("logger is nil")
	}
	if l.level >= _LVL_MAX_for_checks_only {
		//For testing purposes only, should never happen in real code
		//because SetLogLevel() prevents setting invalid levels
		panic("panic on forbidden log level")
	}
	if level < lc.minLevel || level < l.level {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			msg = nil
			err = fmt.Errorf("panic [%v]", r)
		}
	}()
	l.sync.statMtx.RLock()
	defer l.sync.statMtx.RUnlock()
	if !l.IsActive() {
		return nil, fmt.Errorf("logger is not active")
	} else {
		if l.channel == nil {
			return nil, fmt.Errorf("logger channel is nil")
		}
		msg = &logMessage{
			level:   level,
			msgclnt: lc,
			msgtime: t,
			msgtype: MSG_LOG_TEXT,
			msgdata: data,
		}
		// will panic if channel is closed (with recover and setting error)
		l.channel <- *msg
	}
	return
}

func (lc *logClient) Log(level LogLevel, s string) *logMessage {
	return lc.LogBytes(level, []byte(s))
}

func (lc *logClient) LogBytes(level LogLevel, data []byte) *logMessage {
	msg, err := lc.LogBytes_with_err(level, data)
	if err != nil {
		lc.logger.handleLogWriteError(err.Error())
	}
	return msg
}

func (lc *logClient) LogDebug(s string) *logMessage {
	return lc.LogBytes(LVL_DEBUG, []byte(s))
}

func (lc *logClient) LogInfo(s string) *logMessage {
	return lc.LogBytes(LVL_INFO, []byte(s))
}

func (lc *logClient) LogWarn(s string) *logMessage {
	return lc.LogBytes(LVL_WARN, []byte(s))
}

func (lc *logClient) LogError(s string) *logMessage {
	return lc.LogBytes(LVL_ERROR, []byte(s))
}

func (lc *logClient) LogErr(e error) *logMessage {
	return lc.LogBytes(LVL_ERROR, []byte(e.Error()))
}

// io.Writer interface implementation (just to exist)
// * common usage: fmt.Fprintf(C.Lvl(LVL_WARN), "warning: %s happened in module %s", text, modulename)
// * with preset curLevel: fmt.Fprintf(C, "something %s in %s", text, somewhere)

func (lc *logClient) Lvl(level LogLevel) *logClient {
	lc.curLevel = normLevel(level)
	return lc
}

func (lc *logClient) Write(p []byte) (n int, err error) {
	_, err = lc.LogBytes_with_err(lc.curLevel, p)
	if err == nil {
		n = len(p)
	}
	return
}
