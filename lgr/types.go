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
	level    LogLevel
	curLevel LogLevel
}

type outContext struct {
	colormap  *LevelMap // logLevel-associated ANSI terminal color
	prefixmap *LevelMap // logLevel-associated prefix
	delimiter []byte    // added after client name and prefix (usualy ":")
	timefmt   string    // as in time.Format(), no timestamp on ""
	showlvlid bool      // show [<msg.level>] (after time)
	enabled   bool      // enable write message to output if set
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
