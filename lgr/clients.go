package lgr

import (
	"fmt"
	"time"
)

func (l *logger) NewClient(name string, level LogLevel) *logClient {
	client := &logClient{
		logger:   l,
		name:     []byte(name),
		level:    normLevel(level),
		curLevel: LVL_UNKNOWN,
	}
	//l.clients[client] = true // For further "disable client"
	return client
}

/*func (l *logger) DelClient(lc *logClient) {
	delete(l.clients, lc)
}*/

func (lc *logClient) Log_with_err(level LogLevel, s string) (time.Time, error) {
	return lc.LogBytes_with_err(level, []byte(s))
}

func (lc *logClient) LogBytes_with_err(level LogLevel, data []byte) (t time.Time, err error) {
	t = time.Now()
	l := lc.logger
	if l == nil {
		return t, fmt.Errorf("logger is nil")
	}
	if l.level >= _LVL_MAX_for_checks_only {
		//For testing purposes only, should never happen in real code
		//because SetLogLevel() prevents setting invalid levels
		panic("panic on forbidden log level")
	}
	if level < lc.level || level < l.level {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic [%v]", r)
		}
	}()
	l.sync.statMtx.RLock()
	defer l.sync.statMtx.RUnlock()
	if !l.IsActive() {
		return t, fmt.Errorf("logger is not active")
	} else {
		if l.channel == nil {
			return t, fmt.Errorf("logger channel is nil")
		}
		// will panic if channel is closed (with recover and setting error)
		l.channel <- lc.makeTextMessage(t, level, data)
	}
	return
}

func (lc *logClient) makeTextMessage(time time.Time, level LogLevel, data []byte) logMessage {
	return logMessage{
		msgtype: MSG_LOG_TEXT,
		msgclnt: lc,
		msgtime: time,
		msgdata: data,
		level:   level,
	}
}

func (lc *logClient) Log(level LogLevel, s string) time.Time {
	return lc.LogBytes(level, []byte(s))
}

func (lc *logClient) LogBytes(level LogLevel, data []byte) time.Time {
	t, err := lc.LogBytes_with_err(level, data)
	if err != nil {
		lc.logger.handleLogWriteError(err.Error())
	}
	return t
}

func (lc *logClient) LogTrace(s string) time.Time {
	return lc.LogBytes(LVL_TRACE, []byte(s))
}

func (lc *logClient) LogDebug(s string) time.Time {
	return lc.LogBytes(LVL_DEBUG, []byte(s))
}

func (lc *logClient) LogInfo(s string) time.Time {
	return lc.LogBytes(LVL_INFO, []byte(s))
}

func (lc *logClient) LogWarn(s string) time.Time {
	return lc.LogBytes(LVL_WARN, []byte(s))
}

func (lc *logClient) LogError(s string) time.Time {
	return lc.LogBytes(LVL_ERROR, []byte(s))
}

func (lc *logClient) LogErr(e error) time.Time {
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
	if p == nil {
		return 0, nil
	}
	_, err = lc.LogBytes_with_err(lc.curLevel, p)
	if err == nil {
		n = len(p)
	} else {
		n = 0
	}
	return
}
