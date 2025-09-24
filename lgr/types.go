package lgr

import (
	"fmt"
	"io"
	"sync"
)

type LgrState uint8

const (
	STATE_UNKNOWN LgrState = iota
	STATE_ACTIVE
	STATE_STOPPING
	STATE_STOPPED
	_STATE_MAX_FOR_CHECKS_ONLY
)

type LogLevel uint8

const (
	LVL_UNKNOWN LogLevel = iota
	LVL_TRACE
	LVL_DEBUG
	LVL_INFO
	LVL_WARN
	LVL_ERROR
	LVL_FATAL
	LVL_UNMASKABLE
	_LVL_MAX_FOR_CHECKS_ONLY
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
	msgdata string
	msgtype msgType
}

type logClient struct {
	logger   *Logger
	prefix   string
	postfix  string
	maxLevel LogLevel
	curLevel LogLevel
}

type OutType io.Writer
type OutList map[OutType]bool
type Clients map[*logClient]LogLevel

type Logger struct {
	sync struct {
		statMtx sync.RWMutex
		outsMtx sync.RWMutex
		chngMtx sync.RWMutex
		clntMtx sync.RWMutex
		waitEnd sync.WaitGroup
	}
	clients Clients
	outputs OutList
	fallbck OutType
	channel chan logMessage
	state   LgrState
	level   LogLevel
}

func norm_uint8[T ~uint8](val, overlimit, def T) T {
	if val < overlimit {
		return val
	} else {
		return def
	}
}

func normState(state LgrState) LgrState {
	return norm_uint8(state, _STATE_MAX_FOR_CHECKS_ONLY, STATE_UNKNOWN)
}

func normLevel(level LogLevel) LogLevel {
	return norm_uint8(level, _LVL_MAX_FOR_CHECKS_ONLY, _LVL_MAX_FOR_CHECKS_ONLY-1)
}

func logstr(format, prefix, message, postfix string) string {
	return fmt.Sprintf(format, prefix, message, postfix)
}
