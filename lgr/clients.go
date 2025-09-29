package lgr

import (
	"fmt"
	"io"
	"time"
)

func (l *logger) NewClient(name string, maxLevel LogLevel) *logClient {
	client := &logClient{
		logger:   l,
		name:     name,
		maxLevel: maxLevel,
		curLevel: LVL_UNKNOWN,
	}
	l.clients[client] = maxLevel
	io.Discard.Write([]byte{0})
	return client
}

func (l *logger) DelClient(lc *logClient) {
	delete(l.clients, lc)
}

func (lc *logClient) LogE(level LogLevel, s string) (err error) {
	l := lc.logger
	if l == nil {
		return fmt.Errorf("logger is nil")
	}
	if l.level >= _LVL_MAX_FOR_CHECKS_ONLY {
		//For testing purposes only, should never happen in real code
		//because SetLogLevel() prevents setting invalid levels
		panic("panic on forbidden log level")
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic [%v]", r)
		}
	}()
	if level > lc.maxLevel || level < l.level {
		return
	}
	t := time.Now()
	l.sync.statMtx.RLock()
	defer l.sync.statMtx.RUnlock()
	if !l.IsActive() {
		return fmt.Errorf("logger is not active")
	} else {
		if l.channel == nil {
			return fmt.Errorf("logger channel is nil")
		}
		// will panic if channel is closed (with recover and setting error)
		l.channel <- logMessage{
			level:   level,
			msgclnt: lc,
			msgtime: t,
			msgtype: MSG_LOG_TEXT,
			msgdata: s,
		}
	}
	return
}

func (lc *logClient) Log(level LogLevel, s string) {
	err := lc.LogE(level, s)
	if err != nil {
		lc.logger.handleLogWriteError(err.Error())
	}
}

func (lc *logClient) LogDebug(s string) {
	lc.Log(LVL_DEBUG, s)
}

func (lc *logClient) LogInfo(s string) {
	lc.Log(LVL_INFO, s)
}

func (lc *logClient) LogWarn(s string) {
	lc.Log(LVL_WARN, s)
}

func (lc *logClient) LogError(s string) {
	lc.Log(LVL_ERROR, s)
}

func (lc *logClient) LogErr(e error) {
	lc.Log(LVL_ERROR, e.Error())
}

// io.Writer interface implementation (just to exist)
// * common usage: fmt.Fprintf(C.Lvl(LVL_WARN), "warning: %s happened in module %s", text, modulename)
// * with preset curLevel: fmt.Fprintf(C, "something %s in %s", text, somewhere)

func (lc *logClient) Lvl(level LogLevel) *logClient {
	lc.curLevel = level
	return lc
}

func (lc *logClient) Write(p []byte) (n int, err error) {
	err = lc.LogE(lc.curLevel, string(p))
	if err == nil {
		n = len(p)
	}
	return
}
