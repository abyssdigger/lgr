package lgr

import (
	"io"
	"sync"
)

type LoggerState int8

const (
	UNKNOWN  LoggerState = 0
	STOPPED  LoggerState = -1
	ACTIVE   LoggerState = 1
	STOPPING LoggerState = -2
)

type LogLevel uint8

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
	_MAX_FOR_CHECKS_ONLY
)

const (
	DEFAULT_BUFF_SIZE = 32
	DEFAULT_LOG_LEVEL = INFO
)

type msgType uint8

const (
	MSG_FORBIDDEN = iota
	MSG_LOG_TEXT
	MSG_CHG_LEVEL
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
