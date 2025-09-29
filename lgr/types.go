package lgr

import (
	"io"
	"sync"
	"time"
)

type logMessage struct {
	msgclnt *logClient
	msgtime time.Time
	msgdata string
	msgtype msgType
	level   LogLevel
}

type logClient struct {
	logger   *logger
	name     string
	maxLevel LogLevel
	curLevel LogLevel
}

type outType io.Writer
type outDecoration struct {
	ansicolormap *LevelMap // logLevel-associated ANSI terminal color
	lvlprefixmap *LevelMap // logLevel-associated prefix
	delimiter    string    // to be added after prefix (usualy used ":")
	timeformat   string    // no timestamp on ""
	showlvlnum   bool      // Show [<msg.level>] (after time)
	enabled      bool
}
type outList map[outType]*outDecoration

type clients map[*logClient]LogLevel

type logger struct {
	sync struct {
		statMtx sync.RWMutex
		fbckMtx sync.RWMutex
		outsMtx sync.RWMutex
		chngMtx sync.RWMutex
		clntMtx sync.RWMutex
		waitEnd sync.WaitGroup
	}
	clients clients
	outputs outList
	fallbck outType
	channel chan logMessage
	state   lgrState
	level   LogLevel
}

type DefaultShrtLvlNames interface {
	outType
}

type DefaultFullLvlNames outType
type DefaultColorOnBlack outType

type Writerval interface{ io.Writer }
