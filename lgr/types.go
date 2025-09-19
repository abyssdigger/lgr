package lgr

import (
	"io"
	"sync"
)

type LoggerState int8

const (
	STOPPED  LoggerState = 0
	ACTIVE   LoggerState = 1
	STOPPING LoggerState = -1
)

type LogLevel uint8

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

const (
	DEFAULT_BUFF_SIZE = 32
	DEFAULT_LOG_LEVEL = INFO
)

type logMessage struct {
	message string
}

type outType io.Writer
type outList map[outType]bool

type Logger struct {
	sync struct {
		statMtx sync.RWMutex
		outsMtx sync.RWMutex
		chngMtx sync.RWMutex
		waitEnd sync.WaitGroup
	}
	//clients map[string]loggerClient
	outputs outList
	fallbck outType
	channel chan logMessage
	state   LoggerState
	level   LogLevel
}
