package lgr

import "io"

func (l *Logger) NewClient(prefix, postfix string, maxLevel LogLevel) *logClient {
	client := &logClient{
		logger:   l,
		prefix:   prefix,
		postfix:  postfix,
		maxLevel: maxLevel,
		curLevel: LVL_UNKNOWN,
	}
	l.clients[client] = maxLevel
	io.Discard.Write([]byte{0})
	return client
}

func (lc *logClient) Lvl(level LogLevel) *logClient {
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

func (lc *logClient) LogE(level LogLevel, s string) (err error) {
	return lc.logger.LogC(lc, level, s)
}
