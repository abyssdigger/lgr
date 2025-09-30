package lgr

import (
	"io"
	"sync"
	"time"
)

type LogLevel byte
type lgrState byte
type msgType byte
type outType io.Writer
type outList map[outType]*outContext

//type clientMap map[*logClient]bool

type logMessage struct {
	msgclnt *logClient
	msgtime time.Time
	msgdata []byte
	msgtype msgType
	level   LogLevel
}

type logClient struct {
	logger   *logger
	name     []byte
	minLevel LogLevel
	curLevel LogLevel
}

type outContext struct {
	colormap  *LevelMap // logLevel-associated ANSI terminal color
	prefixmap *LevelMap // logLevel-associated prefix
	delimiter []byte    // added after client name and prefix (usualy ":")
	timefmt   string    // as in time.Format(), no timestamp on ""
	lvlcode   bool      // Show [<msg.level>] (after time)
	enabled   bool
}

type logger struct {
	sync struct {
		statMtx sync.RWMutex
		fbckMtx sync.RWMutex
		outsMtx sync.RWMutex
		chngMtx sync.RWMutex
		clntMtx sync.RWMutex
		waitEnd sync.WaitGroup
	}
	//clients clientMap
	outputs outList
	fallbck outType
	channel chan logMessage
	state   lgrState
	level   LogLevel
}

// Getters /////////////////////////////////////////////
func (l *logger) Context(output outType) *outContext {
	return l.outputs[output]
}

func (st *outContext) IsEnabled() bool {
	return st.enabled
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
	st.lvlcode = true
	return st
}
