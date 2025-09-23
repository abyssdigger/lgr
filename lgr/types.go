package lgr

import (
	"io"
	"sync"
)

type LoggerState uint8

const (
	STATE_UNKNOWN LoggerState = iota
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
	msgtext string
	msgtype msgType
}

type OutType io.Writer
type OutList map[OutType]bool

type Logger struct {
	sync struct {
		statMtx sync.RWMutex
		outsMtx sync.RWMutex
		chngMtx sync.RWMutex
		waitEnd sync.WaitGroup
	}
	//clients map[string]loggerClient
	outputs OutList
	fallbck OutType
	channel chan logMessage
	state   LoggerState
	level   LogLevel
}
