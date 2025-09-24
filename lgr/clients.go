package lgr

import "io"

func (l *logger) NewClient(name string, maxLevel logLevel) *logClient {
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

func (lc *logClient) LogE(level logLevel, s string) (err error) {
	if level <= lc.maxLevel {
		return lc.logger.LogC(lc, level, s)
	} else {
		return nil
	}
}

func (lc *logClient) Lvl(level logLevel) *logClient {
	lc.logger.Log(LVL_WARN, "client test")
	return lc
}

func (lc *logClient) Write(p []byte) (n int, err error) {
	err = lc.LogE(lc.curLevel, string(p))
	if err == nil {
		n = len(p)
	}
	return
}

func (lc *logClient) Log(level logLevel, s string) {
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
