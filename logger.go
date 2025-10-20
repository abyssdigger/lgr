// Docs are based on CoPilot (GPT-5 mini) generation
package lgr

import (
	"errors"
	"io"
	"maps"
	"os"
	"slices"
	"time"
)

/*
Docs are based on CoPilot (GPT-5 mini) generation
logger.go

Contains initialization, lifecycle and configuration helpers for the logger
instance as well as functions to enqueue messages and issue client-level
commands.

Most functions are unexported (package-private) and intended to be used
internally by the package. Error message constants are defined here to keep
consistent error text across the package.
*/

const (
	// Error messages used across logger operations.
	ERROR_MESSAGE_LOGGER_STARTED  = "logger is allready started"
	ERROR_MESSAGE_LOGGER_INACTIVE = "logger is not active"
	ERROR_MESSAGE_CHANNEL_IS_NIL  = "logger channel is nil"
	ERROR_MESSAGE_LOG_MSG_IS_NIL  = "log message is nil"
	ERROR_MESSAGE_CLIENT_IS_ALIEN = "logger client is nil or alien (belongs to another logger or nil)"
	ERROR_MESSAGE_CLIENT_IS_NIL   = "client is nil"
	ERROR_MESSAGE_NON_CLIENT_CMD  = "non-client command"
	ERROR_MESSAGE_CMD_EMPTY_DATA  = "no data in command message"
	ERROR_MESSAGE_CMD_NIL_CLIENT  = "nil client in command message"
)

// InitAndStart creates a logger with default parameters and starts its
// background processing goroutine. 'buffsize' controls the channel buffer size.
// Optional outputs can be passed and will be added to the outputs map.
func InitAndStart(buffsize int, outputs ...outType) (l *logger) {
	l = Init(outputs...)
	l.Start(buffsize)
	return
}

// Init creates a logger with default parameters but does not start the
// processing goroutine. By default stderr is used as fallback.
func Init(outputs ...outType) *logger {
	return InitWithParams(DEFAULT_LOG_LEVEL, os.Stderr, outputs...)
}

// InitWithParams constructs a logger instance with explicit initial settings.
// The returned logger is in STATE_STOPPED and must be Start()ed.
func InitWithParams(level LogLevel, fallback outType, outputs ...outType) *logger {
	l := new(logger)
	l.state = STATE_STOPPED
	l.outputs = outList{}
	l.SetMinLevel(level)
	l.SetFallback(fallback)
	l.AddOutputs(outputs...)
	return l
}

// IsEnabled returns whether an output context is enabled for writes.
func (st *outContext) IsEnabled() bool {
	return st.enabled
}

// Start launches the background goroutine that processes queued messages.
// If the logger is already active an error is returned. The channel is
// created with the provided buffsize (lgr.DEFAULT_MSG_BUFF if negative).
//
// The started goroutine will run l.procced() and is tracked by the internal
// wait group so callers can Wait() for graceful shutdown.
func (l *logger) Start(buffsize int) error {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	if l.IsActive() {
		return errors.New(ERROR_MESSAGE_LOGGER_STARTED)
	}
	if buffsize <= 0 {
		buffsize = DEFAULT_MSG_BUFF
	}
	l.channel = make(chan logMessage, buffsize)
	l.sync.waitEnd.Go(func() { l.procced() })
	l.state = STATE_ACTIVE
	return nil
}

// Stop initiates logger shutdown. It sets STATE_STOPPING and closes the channel
// to stop the background processor. The actual processor goroutine will exit
// once the channel drains, at which point callers should call Wait() to block
// until the goroutine has finished.
func (l *logger) Stop() {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	if l.IsActive() {
		l.state = STATE_STOPPING
		close(l.channel)
	}
}

// Wait blocks until the background processor goroutine has finished.
func (l *logger) Wait() {
	l.sync.waitEnd.Wait()
}

// StopAndWait is a convenience to Stop() and then Wait() for completion.
func (l *logger) StopAndWait() {
	l.Stop()
	l.Wait()
}

// SetMinLevel sets the global minimal level for the logger. Messages below
// this level will be ignored by outputs that also respect the logger's level.
func (l *logger) SetMinLevel(minlevel LogLevel) *logger {
	l.sync.chngMtx.Lock()
	defer l.sync.chngMtx.Unlock()
	l.level = normLevel(minlevel)
	return l
}

// SetFallback sets the fallback output used to report internal errors.
// If f is nil io.Discard is used to silently drop fallback messages.
func (l *logger) SetFallback(f outType) *logger {
	l.sync.fbckMtx.Lock()
	defer l.sync.fbckMtx.Unlock()
	if f != nil {
		l.fallbck = f
	} else {
		l.fallbck = io.Discard
	}
	return l
}

// IsActive reports whether the logger is in STATE_ACTIVE.
func (l *logger) IsActive() bool {
	return l.state == STATE_ACTIVE
}

// AddOutputs attaches one or more outputs (io.Writer) to the logger and
// creates a default outContext for each. The operation is protected by the
// outputs mutex. Nil outputs are ignored.
func (l *logger) AddOutputs(outputs ...outType) *logger {
	l.operateOutputs(outputs, func(m *outList, k outType) {
		(*m)[k] = &outContext{
			enabled:   true,
			delimiter: []byte(DEFAULT_DELIMITER),
		}
	})
	return l
}

// RemoveOutputs removes the provided outputs from the logger.
func (l *logger) RemoveOutputs(outputs ...outType) *logger {
	l.operateOutputs(outputs, func(m *outList, k outType) { delete(*m, k) })
	return l
}

// ClearOutputs removes all outputs from the logger. The current
// implementation removes the keys extracted from the map (helper usage).
func (l *logger) ClearOutputs() *logger {
	l.RemoveOutputs(slices.Collect(maps.Keys(l.outputs))...)
	return l
}

// operateOutputs iterates the provided slice and applies the operation
// for each non-nil outType. The operation is performed with the outputs mutex
// held to ensure thread-safety.
func (l *logger) operateOutputs(slice []outType, operation func(m *outList, k outType)) {
	if len(slice) == 0 {
		return
	}
	l.sync.outsMtx.Lock()
	defer l.sync.outsMtx.Unlock()
	for _, output := range slice {
		if output != nil {
			operation(&l.outputs, output)
		}
	}
}

// The next set of functions change per-output settings by delegating to
// changeOutSettings which takes a closure and runs it while holding the
// outputs mutex.

// SetOutputLevelPrefix sets the prefix map (per-level prefix) and delimiter
// for a specific output.
func (l *logger) SetOutputLevelPrefix(output outType, prefixmap *LevelMap, delimiter string) *logger {
	return l.changeOutSettings(output, func(s *outContext) {
		s.SetPrefix(prefixmap)
		s.SetDelimiter(delimiter)
	})
}

// SetOutputLevelColor assigns a color map (ANSI fragments) used when building
// messages for the specified output.
func (l *logger) SetOutputLevelColor(output outType, colormap *LevelMap) *logger {
	return l.changeOutSettings(output, func(s *outContext) {
		s.SetColor(colormap)
	})
}

// SetOutputTimeFormat sets the time.Format string used to prefix messages for
// the specified output. If empty, no timestamp is written. Example string:
// "2006-01-02 15:04:05 " (remember about a delimiter at the end - space in this case)
func (l *logger) SetOutputTimeFormat(output outType, format string) *logger {
	return l.changeOutSettings(output, func(s *outContext) {
		s.SetTimeFormat(format)
	})
}

// ShowOutputLevelCode enables printing a level id (like "[3]") on the output.
func (l *logger) ShowOutputLevelCode(output outType) *logger {
	return l.changeOutSettings(output, func(s *outContext) {
		s.ShowLevelCode()
	})
}

// SetOutputMinLevel sets the minimal level to log for the specified output.
func (l *logger) SetOutputMinLevel(output outType, minlevel LogLevel) *logger {
	return l.changeOutSettings(output, func(s *outContext) {
		s.SetMinLevel(minlevel)
	})
}

// changeOutSettings safely modifies an outContext for the given output if it
// exists. The provided function runs while holding the outputs mutex.
func (l *logger) changeOutSettings(output outType, f func(s *outContext)) *logger {
	if l.outputs[output] != nil {
		l.sync.outsMtx.Lock()
		defer l.sync.outsMtx.Unlock()
		f(l.outputs[output])
	}
	return l
}

// Context returns the outContext for a given output writer.
func (l *logger) Context(output outType) *outContext {
	return l.outputs[output]
}

// The following methods manipulate outContext fields in a fluent style.

// SetDelimiter sets the delimiter bytes written after client name/prefix.
func (st *outContext) SetDelimiter(delimiter string) *outContext {
	st.delimiter = []byte(delimiter)
	return st
}

// SetPrefix assigns the prefix map for the outContext.
func (st *outContext) SetPrefix(prefixmap *LevelMap) *outContext {
	st.prefixmap = prefixmap
	return st
}

// SetColor assigns the color map for the outContext.
func (st *outContext) SetColor(colormap *LevelMap) *outContext {
	st.colormap = colormap
	return st
}

// SetTimeFormat assigns the time format string used when building messages.
func (st *outContext) SetTimeFormat(timeformat string) *outContext {
	st.timefmt = timeformat
	return st
}

// ShowLevelCode configures the outContext to include a numeric level id.
func (st *outContext) ShowLevelCode() *outContext {
	st.showlvlid = true
	return st
}

// SetMinLevel configures the minimal level accepted by this outContext.
func (st *outContext) SetMinLevel(minlevel LogLevel) *outContext {
	st.minlevel = normLevel(minlevel)
	return st
}

// pushMessage attempts to enqueue a logMessage into the logger's channel.
// It returns the timestamp (t) that represents the push time and an error
// if the message could not be enqueued. panics while writing to the closed
// channel are recovered and translated into errors.
func (l *logger) pushMessage(msg *logMessage) (t time.Time, err error) {
	l.sync.statMtx.RLock()
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("panic" + panicDesc(r))
		}
		l.sync.statMtx.RUnlock()
	}()
	t = time.Now()
	if msg == nil {
		err = errors.New(ERROR_MESSAGE_LOG_MSG_IS_NIL)
	} else {
		msg.pushed = t
		if !l.IsActive() {
			err = errors.New(ERROR_MESSAGE_LOGGER_INACTIVE)
		} else {
			if l.channel == nil {
				err = errors.New(ERROR_MESSAGE_CHANNEL_IS_NIL)
			} else {
				// will panic if channel is closed (with recover and setting error)
				l.channel <- *msg
			}
		}
	}
	return t, err
}

// makeTextMessage builds a logMessage representing a textual log entry.
func makeTextMessage(lc *logClient, level LogLevel, data []byte) *logMessage {
	return &logMessage{
		msgtype: MSG_LOG_TEXT,
		msgclnt: lc,
		msgdata: data,
		annex:   basetype(level),
	}
}

// makeCmdMessage builds a command message used to mutate client settings
// in-order (queued) so that changes do not affect already queued messages.
func makeCmdMessage(lc *logClient, cmd cmdType, data []byte) *logMessage {
	return &logMessage{
		msgtype: MSG_COMMAND,
		msgclnt: lc,
		msgdata: data,
		annex:   basetype(cmd),
	}
}

// NewClient constructs a new logClient associated with this logger. The
// client carries its own minimal level and an initial name.
func (l *logger) NewClient(name string, minlevel LogLevel) *logClient {
	client := &logClient{
		logger:   l,
		name:     []byte(name),
		minLevel: normLevel(minlevel),
		curLevel: LVL_UNKNOWN,
		enabled:  true,
	}
	//l.clients[client] = true // For further "disable client"
	return client
}

// checkClient validates that lc belongs to this logger and is not nil.
func (l *logger) checkClient(lc *logClient) (err error) {
	if lc == nil {
		err = errors.New(ERROR_MESSAGE_CLIENT_IS_NIL)
	} else if lc.logger != l {
		err = errors.New(ERROR_MESSAGE_CLIENT_IS_ALIEN)
	}
	return
}

// SetClientEnabled toggles whether a client may enqueue messages.
func (l *logger) SetClientEnabled(lc *logClient, enabled bool) (err error) {
	if err = l.checkClient(lc); err == nil {
		lc.enabled = enabled
	}
	return
}

// SetClientMinLevel enqueues a client-level change as a command message so
// the change takes effect only after previously queued messages are processed.
func (l *logger) SetClientMinLevel(lc *logClient, minlevel LogLevel) (t time.Time, err error) {
	return l.runClientCommand(lc, CMD_CLIENT_SET_LEVEL, []byte{byte(minlevel)})
}

// SetClientName enqueues a name-change command for a client.
func (l *logger) SetClientName(lc *logClient, newname string) (time.Time, error) {
	return l.runClientCommand(lc, CMD_CLIENT_SET_NAME, []byte(newname))
}

// runClientCommand performs validation and enqueues a command message to
// mutate a client. Commands are processed in-order by the background worker.
func (l *logger) runClientCommand(lc *logClient, cmd cmdType, data []byte) (t time.Time, err error) {
	// Change client settings by commands (sent messages has to be printed with previous settings)
	err = l.checkClient(lc)
	if err == nil {
		if cmd < _CMD_CLIENT_commands_min || cmd > _CMD_CLIENT_commands_max {
			err = errors.New(ERROR_MESSAGE_NON_CLIENT_CMD)
		} else {
			t, err = l.pushMessage(makeCmdMessage(lc, cmd, data))
		}
	}
	return t, err
}

// clientChangeFromCmdMsg is a small helper that validates command message
// payload and invokes f to perform the actual client mutation.
func clientChangeFromCmdMsg(msg *logMessage, f func(*logClient, []byte)) (errstr string) {
	if len(msg.msgdata) < 1 {
		errstr = ERROR_MESSAGE_CMD_EMPTY_DATA
	} else if msg.msgclnt == nil {
		errstr = ERROR_MESSAGE_CMD_NIL_CLIENT
	} else {
		f(msg.msgclnt, msg.msgdata)
	}
	return
}
