package lgr

import (
	"bytes"
	"io"
	"sync"
	"time"
)

type basetype byte

type LogLevel basetype
type lgrState basetype
type msgType basetype
type cmdType basetype
type outType io.Writer
type outList map[outType]*outContext

type logMessage struct {
	pushed  time.Time
	msgclnt *logClient
	msgdata []byte
	msgtype msgType
	annex   basetype
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
	timefmt   string    // as in time.Format(), if "" then no timestamp
	showlvlid bool      // show [<msg.level>] (after time)
	enabled   bool      // enable write message to output if set
	minlevel  LogLevel  // minimal level to log
}

type logger struct {
	sync struct {
		statMtx sync.RWMutex
		fbckMtx sync.RWMutex
		outsMtx sync.RWMutex
		chngMtx sync.RWMutex
		clntMtx sync.RWMutex
		procMtx sync.RWMutex
		waitEnd sync.WaitGroup
	}
	//clients clientMap
	outputs outList
	fallbck outType
	channel chan logMessage
	msgbuf  *bytes.Buffer
	state   lgrState
	level   LogLevel
}
