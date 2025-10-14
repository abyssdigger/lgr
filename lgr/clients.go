package lgr

import (
	"fmt"
	"time"
)

// Log message (basic is Log() )

func (lc *logClient) Log_with_err(level LogLevel, s string) (time.Time, error) {
	return lc.LogBytes_with_err(level, []byte(s))
}

func (lc *logClient) LogBytes_with_err(level LogLevel, data []byte) (t time.Time, err error) {
	if lc.logger == nil {
		return t, fmt.Errorf("logger is nil")
	}
	if lc.logger.level >= _LVL_MAX_for_checks_only {
		//For testing purposes only, should never happen in real code
		//because SetLogLevel() prevents setting invalid levels
		panic("panic on forbidden log level")
	}
	if level < lc.minLevel || level < lc.logger.level {
		return
	}
	t, err = lc.logger.pushMessage(makeTextMessage(lc, level, data))
	return t, err
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

/////////////////////////////////////////////////////////////////////////////////////
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
