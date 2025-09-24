package lgr

import (
	"fmt"
	"io"
	"sync"
	"time"
)

type logLevel uint8

const (
	LVL_UNKNOWN logLevel = iota
	LVL_TRACE
	LVL_DEBUG
	LVL_INFO
	LVL_WARN
	LVL_ERROR
	LVL_FATAL
	LVL_UNMASKABLE
	_LVL_MAX_FOR_CHECKS_ONLY
)

type llDesc struct {
	Short string
	Long  string
	color string
}

var LogLevelDesc map[logLevel]*llDesc

const logTermReset = "\033[0m"

func init() {
	LogLevelDesc = make(map[logLevel]*llDesc)
	LogLevelDesc[LVL_UNKNOWN] = &llDesc{Short: "???", Long: "UNKNOWN"}
	LogLevelDesc[LVL_TRACE] = &llDesc{Short: "TRC", Long: "TRACE"}
	LogLevelDesc[LVL_DEBUG] = &llDesc{Short: "DBG", Long: "DEBUG"}
	LogLevelDesc[LVL_INFO] = &llDesc{Short: "INF", Long: "INFO"}
	LogLevelDesc[LVL_WARN] = &llDesc{Short: "WRN", Long: "WARN"}
	LogLevelDesc[LVL_ERROR] = &llDesc{Short: "ERR", Long: "ERROR"}
	LogLevelDesc[LVL_FATAL] = &llDesc{Short: "FTL", Long: "FATAL"}
	LogLevelDesc[LVL_UNMASKABLE] = &llDesc{Short: "!!!", Long: "UNMASKABLE"}
	//https://habr.com/ru/companies/first/articles/672464/?ysclid=mfy8zz61fw842674829
	LogLevelDesc[LVL_UNKNOWN].color = "\033[9;90m"
	LogLevelDesc[LVL_TRACE].color = "\033[2;90m"
	LogLevelDesc[LVL_DEBUG].color = "\033[0;90m"
	LogLevelDesc[LVL_INFO].color = "\033[0;97m"
	LogLevelDesc[LVL_WARN].color = "\033[0;33m"
	LogLevelDesc[LVL_ERROR].color = "\033[0;91m"
	LogLevelDesc[LVL_FATAL].color = "\033[101m\033[1;33m"
	LogLevelDesc[LVL_UNMASKABLE].color = "\033[107m\033[1;31m"
}

type lgrState uint8

const (
	STATE_UNKNOWN lgrState = iota
	STATE_ACTIVE
	STATE_STOPPING
	STATE_STOPPED
	_STATE_MAX_FOR_CHECKS_ONLY
)

type msgType uint8

const (
	MSG_FORBIDDEN msgType = iota
	MSG_LOG_TEXT
	MSG_CHG_LEVEL
	_MSG_MAX_FOR_CHECKS_ONLY
)

const (
	DEFAULT_LOG_LEVEL = LVL_ERROR
	DEFAULT_BUFF_SIZE = 32
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
type outList map[outType]bool
type clients map[*logClient]logLevel

type logger struct {
	sync struct {
		statMtx sync.RWMutex
		outsMtx sync.RWMutex
		chngMtx sync.RWMutex
		clntMtx sync.RWMutex
		waitEnd sync.WaitGroup
	}
	clients clients
	outputs outList
	fallbck outType
	channel chan logMessage
	timefmt string
	state   lgrState
	level   logLevel
}

func norm_uint8[T ~uint8](val, overlimit, def T) T {
	if val < overlimit {
		return val
	} else {
		return def
	}
}

func normState(state lgrState) lgrState {
	return norm_uint8(state, _STATE_MAX_FOR_CHECKS_ONLY, STATE_UNKNOWN)
}

func normLevel(level logLevel) logLevel {
	return norm_uint8(level, _LVL_MAX_FOR_CHECKS_ONLY, _LVL_MAX_FOR_CHECKS_ONLY-1)
}

func logstr(format, prefix, text string) string {
	return fmt.Sprintf(format, prefix, text)
}
