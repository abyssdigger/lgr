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
	level   logLevel
}

type logClient struct {
	logger   *logger
	name     string
	maxLevel logLevel
	curLevel logLevel
}

type outType io.Writer
type outDecoration struct {
	ansicolormap *lvlStringMap // logLevel-associated ANSI terminal color
	lvlprefixmap *lvlStringMap // logLevel-associated prefix
	delimiter    string        // to be added after prefix (usualy used ":")
	timeformat   string        // no timestamp on ""
	enabled      bool
}
type outList map[outType]*outDecoration
type lvlStringMap map[logLevel]string //use [nil] as default value

type clients map[*logClient]logLevel

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
	level   logLevel
}
