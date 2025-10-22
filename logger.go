// A lightweight, levelled logging package for Go. Provides timestamped, colorized
// and filtered log output with per-client and per-output configuration.
package lgr

/*
Historically, docs are based on CoPilot/GPT5mini generation, but little
remains of the original delusion.
*/

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"time"
)

const (
	// Error messages used across logger operations (used for testing).
	_ERROR_MESSAGE_LOGGER_STARTED  = "logger is allready started"
	_ERROR_MESSAGE_LOGGER_INACTIVE = "logger is not active"
	_ERROR_MESSAGE_CHANNEL_IS_NIL  = "logger channel is nil"
	_ERROR_MESSAGE_LOG_MSG_IS_NIL  = "log message is nil"
	_ERROR_MESSAGE_CLIENT_IS_ALIEN = "logger client is nil or alien (belongs to another logger or nil)"
	_ERROR_MESSAGE_CLIENT_IS_NIL   = "client is nil"
	_ERROR_MESSAGE_NON_CLIENT_CMD  = "non-client command"
	_ERROR_MESSAGE_CMD_EMPTY_DATA  = "no data in command message"
	_ERROR_MESSAGE_CMD_NIL_CLIENT  = "nil client in command message"
	_ERROR_UNKNOWN_PANIC_TEXT      = "[no panic description]"
)

// Creates a logger with default parameters and starts its queue processing goroutine.
//   - buffsize: channel messages buffer capacity ([DEFAULT_MSG_BUFF] is used fore negative values)
//   - outputs: list of log outputs (files, [os.Stdout], [os.Stderr], etc.)
//
// Preferred usage example:
//
//	func main() {
//	    logger := InitAndStart(-1, os.Stdout)
//	    defer logger.StopAndWait()
//	    ...
//	}
func InitAndStart(buffsize int, outputs ...OutType) (l *logger) {
	l = Init(outputs...)
	l.Start(buffsize)
	return
}

// Short form of Init creates a logger with provided outputs, default log level and
// [os.Stderr] as fallback for logging errors (can be changed later with Set methods).
//
// The returned logger is in stopped state and must be started by Start() to proceed
// log messages.
func Init(outputs ...OutType) *logger {
	return InitWithParams(DEFAULT_LOG_LEVEL, os.Stderr, outputs...)
}

// InitWithParams constructs a logger instance with explicit initial settings.
//
// The returned logger is in stopped state and must be started by Start() to proceed
// log messages.
func InitWithParams(level LogLevel, fallback OutType, outputs ...OutType) *logger {
	l := new(logger)
	l.state = _STATE_STOPPED
	l.outputs = outList{}
	l.SetMinLevel(level)
	l.SetFallback(fallback)
	l.AddOutputs(outputs...)
	return l
}

// Start launches the background goroutine that processes queued messages.
// If the logger is already active an error is returned. The channel is
// created with the provided buffsize (DEFAULT_MSG_BUFF for negative).
//
// The started goroutine will run procced() and is tracked by the internal
// wait group so callers can Wait() for graceful shutdown.
func (l *logger) Start(buffsize int) error {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	if l.IsActive() {
		return errors.New(_ERROR_MESSAGE_LOGGER_STARTED)
	}
	if buffsize <= 0 {
		buffsize = DEFAULT_MSG_BUFF
	}
	l.channel = make(chan logMessage, buffsize)
	l.sync.waitEnd.Go(func() { l.procced() })
	l.state = _STATE_ACTIVE
	return nil
}

// Stop initiates logger shutdown. It sets STATE_STOPPING and closes the channel
// to stop the background processor. No new messages will be queued in this state.
// The actual processor goroutine will exit once the channel drains.
//
// Wait() should be called before program exits to prevent the loss of last queued
// messages.
func (l *logger) Stop() {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	if l.IsActive() {
		l.state = _STATE_STOPPING
		close(l.channel)
	}
}

// Wait blocks until the background queue goroutine has finished.
func (l *logger) Wait() {
	l.sync.waitEnd.Wait()
}

// A convenience to Stop() and then Wait() for completion. Useful if logger has
// to be stopped just before program exit.
// Preferred usage example:
//
//	func main() {
//	    logger := InitAndStart(-1, os.Stdout)
//	    defer logger.StopAndWait()
//	    ...
//	}
func (l *logger) StopAndWait() {
	l.Stop()
	l.Wait()
}

// Sets the global minimal level for the logger. Messages below this level will
// be ignored.
//
// The operation is protected by mutex for thread safety.
func (l *logger) SetMinLevel(minlevel LogLevel) *logger {
	l.sync.chngMtx.Lock()
	defer l.sync.chngMtx.Unlock()
	l.level = normLevel(minlevel)
	return l
}

// Sets the fallback output used to report internal errors, io.Discard is used
// instead of nil to silently drop fallback messages.
//
// The operation is protected by mutex for thread safety.
func (l *logger) SetFallback(f OutType) *logger {
	l.sync.fbckMtx.Lock()
	defer l.sync.fbckMtx.Unlock()
	if f != nil {
		l.fallbck = f
	} else {
		l.fallbck = io.Discard
	}
	return l
}

// True if the logger is in active state (i.e. ready to proceed log messages).
func (l *logger) IsActive() bool {
	return l.state == _STATE_ACTIVE
}

// Attaches one or more outputs (io.Writer) to the logger and creates a
// default context for each. Nil outputs are ignored.
//
// The operation is protected by mutex for thread safety.
//
// Changes will be applied immediately (any previously queued messages
// will be directed to the updated set of outputs).
func (l *logger) AddOutputs(outputs ...OutType) *logger {
	l.operateOutputs(outputs, func(m *outList, k OutType) {
		(*m)[k] = &OutContext{
			enabled:   true,
			delimiter: []byte(DEFAULT_DELIMITER),
		}
	})
	return l
}

// Removes the provided outputs from the logger. No errors if there is no
// such output in logger's outputs map.
//
// The operation is protected by mutex for thread safety.
//
// Changes will be applied immediately (any previously queued messages
// will be directed to the updated set of outputs).
func (l *logger) RemoveOutputs(outputs ...OutType) *logger {
	l.operateOutputs(outputs, func(m *outList, k OutType) { delete(*m, k) })
	return l
}

// Removes all outputs from the logger.
//
// The operation is protected by mutex for thread safety.
//
// Changes will be applied immediately (any previously queued message
// will be discarded if no new outputs are added before proceeding).
func (l *logger) ClearOutputs() *logger {
	//The current implementation removes the keys extracted from the map
	// (helper usage for develop/testing purposes).
	l.RemoveOutputs(slices.Collect(maps.Keys(l.outputs))...)
	return l
}

// Helper that applies the operation for each non-nil output from the provided slice.
//
// The operation is performed with the outputs mutex held to ensure thread-safety.
func (l *logger) operateOutputs(slice []OutType, operation func(m *outList, k OutType)) {
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

// Returns whether a specified output is added to logger
func (l *logger) IsOutputExists(out OutType) bool {
	return l.outputs[out] != nil
}

// Returns whether an output is enabled for writes (false if output doesn't exist)
func (l *logger) IsOutputEnabled(out OutType) bool {
	c := l.outputs[out]
	if c != nil {
		return c.enabled
	}
	return false
}

// The next set of functions change per-output settings by delegating to
// changeOutSettings which takes a closure and runs it while holding the
// outputs mutex.

// Sets the prefix map (per-level prefix) and the delimiter for a specific output.
func (l *logger) SetOutputLevelPrefix(output OutType, prefixmap *LevelMap, delimiter string) *logger {
	return l.changeOutSettings(output, func(c *OutContext) {
		c.prefixmap = prefixmap
		c.delimiter = []byte(delimiter)
	})
}

// Assigns a color map (ANSI fragments) used when building messages for the specified output.
func (l *logger) SetOutputLevelColor(output OutType, colormap *LevelMap) *logger {
	return l.changeOutSettings(output, func(c *OutContext) {
		c.colormap = colormap
	})
}

// Sets the time.Format string used to prefix messages for the specified output. If empty
// no timestamp is written.
//
// More about time format layouts at https://pkg.go.dev/time#Layout. Example:
//
//	"2006-01-02 15:04:05"
func (l *logger) SetOutputTimeFormat(output OutType, format, delimiter string) *logger {
	return l.changeOutSettings(output, func(c *OutContext) {
		c.timefmt = format + delimiter
	})
}

// Enables printing a level id (like "[3]") after time and before any oter info and decorations.
// May be useful for debugging or log filtering.
func (l *logger) ShowOutputLevelCode(output OutType) *logger {
	return l.changeOutSettings(output, func(c *OutContext) {
		c.showlvlid = true
	})
}

// Sets the minimal level to log for the specified output.
//
// Used in addition to logger and client minimal levels.
func (l *logger) SetOutputMinLevel(output OutType, minlevel LogLevel) *logger {
	return l.changeOutSettings(output, func(c *OutContext) {
		c.minlevel = normLevel(minlevel)
	})
}

// Safely modifies a context with a given function for the given output (if it exists).
func (l *logger) changeOutSettings(output OutType, f func(*OutContext)) *logger {
	if l.outputs[output] != nil {
		l.sync.outsMtx.Lock()
		defer l.sync.outsMtx.Unlock()
		f(l.outputs[output])
	}
	return l
}

/////////////////////////////////////////////////////////////////////////////////////////

// Attempts to enqueue a logMessage into the logger's channel. It returns the
// timestamp (t) that represents the push time and an error if the message could
// not be enqueued. Catches any panics (including writing to the closed channel)
// and converts them to errors.
func (l *logger) pushMessage(msg *logMessage) (t time.Time, err error) {
	l.sync.statMtx.RLock()
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("panic" + panicDesc(r))
		}
		l.sync.statMtx.RUnlock()
	}()
	t1 := time.Now()
	if msg == nil {
		err = errors.New(_ERROR_MESSAGE_LOG_MSG_IS_NIL)
	} else {
		if !l.IsActive() {
			err = errors.New(_ERROR_MESSAGE_LOGGER_INACTIVE)
		} else {
			if l.channel == nil {
				err = errors.New(_ERROR_MESSAGE_CHANNEL_IS_NIL)
			} else {
				// will panic if channel is closed (with recover and setting error)
				msg.pushed = t1
				l.channel <- *msg
				t = t1
			}
		}
	}
	return t, err
}

// Helper to build a logMessage representing a textual log entry
func makeTextMessage(lc *LogClient, level LogLevel, data []byte) *logMessage {
	return &logMessage{
		msgtype: _MSG_LOG_TEXT,
		msgclnt: lc,
		msgdata: data,
		annex:   basetype(level),
	}
}

// Helper to build a command message (used to change something in queued order to prevent
// already queued messages from beeng affected).
func makeCmdMessage(lc *LogClient, cmd cmdType, data []byte) *logMessage {
	return &logMessage{
		msgtype: _MSG_COMMAND,
		msgclnt: lc,
		msgdata: data,
		annex:   basetype(cmd),
	}
}

/////////////////////////////////////////////////////////////////////////////////////////
/*
Logger client is an abstraction for program part, goroutine, module etc that
can have its own name, minimal log level and which logs can enabled/disabled
regardless of other logger clients. These parameters can be changed only by
it's parent logger, not by client itself. Implemented as `logClient` type.

All logs are written by logger clients, not by the logger istelf. For a simple
single-part program it can be the only one logger client.

Lightweight client-side helpers are provided for producing log messages and
have both error-returning variants (suffixed with _with_err) and convenience
wrappers that write failures to the logger's fallback writer.

Concurrency notes:
 - logClient methods are thread-safe and can be called by application goroutines.
 - pushMessage enqueues into the logger channel and performs the necessary
   state checks; it recovers panics caused by sending on a closed channel and
   converts them into an error value.
 - Client main fields (name, minLevel) are changed by queued commands (see
   runClientCommand/proceedCmd) to prevent already queued messages from changes.
   Client enable affects only newly createds messages so can be changed directly.

The basic LogBytes_with_err writes a []byte at the provided level and returns
the message queue time or any error encountered while attempting to enqueue the
message. All other client Log* methods are just wrappers on it.

Special thanks to the CoPilot for this mess of letters.
*/

// Constructs a new logClient associated with this logger. The client carries its own
// minimal level, an initial name and can be disabled separately from other clients.
func (l *logger) NewClientWithLevel(name string, minlevel LogLevel) *LogClient {
	client := &LogClient{
		logger:   l,
		name:     []byte(name),
		minLevel: normLevel(minlevel),
		curLevel: LVL_UNKNOWN, // Used only for io.Writer usage
		enabled:  true,
	}
	return client
}

// Constructs a new logClient owned by this logger with default properties.
//
// Client properties can be changed with logger SetClient...() setters but not by
// client functions to centralize and secure log management.
func (l *logger) NewClient(name string) *LogClient {
	return l.NewClientWithLevel(name, LVL_UNKNOWN)
}

// Validates that logger client belongs to this logger
func (l *logger) IsOwnClient(lc *LogClient) bool {
	return lc != nil && lc.logger == l
}

// Validates that logger client belongs to this logger with extended error text
func (l *logger) checkClient(lc *LogClient) (err error) {
	if lc == nil {
		err = errors.New(_ERROR_MESSAGE_CLIENT_IS_NIL)
	} else if lc.logger != l {
		err = errors.New(_ERROR_MESSAGE_CLIENT_IS_ALIEN)
	}
	return
}

// Toggles whether a client's log messages should be proceeded
func (l *logger) SetClientEnabled(lc *LogClient, enabled bool) error {
	err := l.checkClient(lc)
	if err == nil {
		lc.enabled = enabled
	}
	return err
}

// Enqueues a client minimum level change as a command message so the change takes
// effect only after previously queued messages are processed
func (l *logger) SetClientMinLevel(lc *LogClient, minlevel LogLevel) (t time.Time, err error) {
	return l.runClientCommand(lc, _CMD_CLIENT_SET_LEVEL, []byte{byte(minlevel)})
}

// Enqueues a client name change  as a command message so the change takes
// effect only after previously queued messages are processed
func (l *logger) SetClientName(lc *LogClient, newname string) (time.Time, error) {
	return l.runClientCommand(lc, _CMD_CLIENT_SET_NAME, []byte(newname))
}

// Performs validation and enqueues a command message to change client properties.
//
// Commands are processed in-order by the background worker so changes will not affect
// messages queued (logged) before this command.
func (l *logger) runClientCommand(lc *LogClient, cmd cmdType, data []byte) (t time.Time, err error) {
	// Change client settings by commands (sent messages has to be printed with previous settings)
	err = l.checkClient(lc)
	if err == nil {
		if cmd < _CMD_CLIENT_commands_min || cmd > _CMD_CLIENT_commands_max {
			err = errors.New(_ERROR_MESSAGE_NON_CLIENT_CMD)
		} else {
			t, err = l.pushMessage(makeCmdMessage(lc, cmd, data))
		}
	}
	return t, err
}

// Validates command message payload and performs the specified client changes
func clientChangeFromCmdMsg(msg *logMessage, f func(*LogClient, []byte)) (errstr string) {
	if len(msg.msgdata) < 1 {
		errstr = _ERROR_MESSAGE_CMD_EMPTY_DATA
	} else if msg.msgclnt == nil {
		errstr = _ERROR_MESSAGE_CMD_NIL_CLIENT
	} else {
		f(msg.msgclnt, msg.msgdata)
	}
	return
}

// LogBytes_with_err enqueues a raw byte payload as a log message at the given
// level. It returns the push timestamp and an error if the logger is nil,
// inactive, the channel is unavailable, or a panic occurred while sending.
//
// Filtering behavior: the call is a no-op and returns zero time + nil error
// (message intentionally ignored), if
//   - the client is disabled, or
//   - the message level is below the client's minLevel, or
//   - the message level is below the global logger level.
//
// Note: There is a test-only check that panics if logger.level is invalid; in
// normal code SetMinLevel/normLevel should prevent invalid level values.
func (lc *LogClient) LogBytes_with_err(level LogLevel, data []byte) (t time.Time, err error) {
	if lc.logger == nil {
		return t, fmt.Errorf("logger is nil")
	}
	if lc.logger.level > _LVL_MAX_for_checks_only {
		// For testing purposes only — exercising panic recovery paths.
		panic(errors.New("panic on forbidden log level (for testing purposes)"))
	}
	// Apply per-client and global filtering before enqueuing.
	if lc.enabled && level >= lc.minLevel && level < _LVL_MAX_for_checks_only && level >= lc.logger.level {
		t, err = lc.logger.pushMessage(makeTextMessage(lc, level, data))
	}
	return t, err
}

// Same as LogBytes_with_err() but underlying enqueue/write error is written to
// logger fallback. Returns zero time on error.
func (lc *LogClient) LogBytes(level LogLevel, data []byte) time.Time {
	t, err := lc.LogBytes_with_err(level, data)
	if err != nil {
		// Report the write/enqueue error to the logger fallback. This keeps the
		// simple Log* API ergonomic while still surfacing failures.
		lc.logger.handleLogWriteError(err.Error())
	}
	return t
}

// Writes a string as log message at the provided level. Returns the time
// the message was queued or an error encountered while attempting to enqueue
// the message.
//
// If no special error processing needed use
//
//	Log()
//
// instead.
func (lc *LogClient) Log_with_err(level LogLevel, s string) (time.Time, error) {
	return lc.LogBytes_with_err(level, []byte(s))
}

// Writes a string as log message at the provided level. Returns the time
// the message was queued or zero value on error. Any error encountered while
// attempting to enqueue the message will be written as a string to the logger
// fallback.
//
// Use
//
//	Log_with_err()
//
// when callers need to react to delivery problems.
func (lc *LogClient) Log(level LogLevel, s string) time.Time {
	return lc.LogBytes(level, []byte(s))
}

/////////////////////////////////////////////////////////////////////////////////////////
/*
Convenience level-specific helpers for common log levels.
These are thin wrappers around LogBytes that provide inline hints in
editors and documentation tools.

All of these helpers behave like LogBytes: they do not return an error.
If an enqueue/write error occurs it will be reported to the logger's
fallback writer (via handleLogWriteError).
*/

// Logs a textual message at TRACE level.
//
// Use this for very verbose diagnostic information. Does not accept an error
// value and does not return enqueue/write errors — failures are forwarded to
// the logger fallback writer.
//
// Logger commands are written to log with this level.
func (lc *LogClient) LogTrace(s string) time.Time {
	return lc.LogBytes(LVL_TRACE, []byte(s))
}

// Logs a textual message at DEBUG level. Returns the time the message was queued
// or zero value on error. Any error encountered while attempting to enqueue the
// message will be written as a string to the logger fallback.
//
// Intended for developer-focused debugging output.
func (lc *LogClient) LogDebug(s string) time.Time {
	return lc.LogBytes(LVL_DEBUG, []byte(s))
}

// Logs an informational message at INFO level. Returns the time the message was queued
// or zero value on error. Any error encountered while attempting to enqueue the
// message will be written as a string to the logger fallback.
//
// Use for normal operational messages.
func (lc *LogClient) LogInfo(s string) time.Time {
	return lc.LogBytes(LVL_INFO, []byte(s))
}

// LogWarn logs a warning message at WARN level. Returns the time the message was queued
// or zero value on error. Any error encountered while attempting to enqueue the
// message will be written as a string to the logger fallback.
//
// Use for recoverable or noteworthy conditions that deserve attention.
func (lc *LogClient) LogWarn(s string) time.Time {
	return lc.LogBytes(LVL_WARN, []byte(s))
}

// LogError logs an error-level message. Returns the time the message was queued
// or zero value on error. Any error encountered while attempting to enqueue the
// message will be written as a string to the logger fallback.
//
// Use this when you have a formatted or constructed string that represents
// an error condition. Use
//
//	LogErr(e error)
//
// to log error instead of string.
func (lc *LogClient) LogError(s string) time.Time {
	return lc.LogBytes(LVL_ERROR, []byte(s))
}

// LogErr logs an error.Value at ERROR level. Returns the time the message was queued
// or zero value on error. Any error encountered while attempting to enqueue the
// message will be written as a string to the logger fallback.
//
// This is a convenience specifically for error values: it calls Error() on the
// provided error and logs that string at LVL_ERROR. Semantically equivalent to
// calling
//
//	LogError(err.Error())
//
// but clearer at call sites when you already have an error object.
func (lc *LogClient) LogErr(e error) time.Time {
	return lc.LogBytes(LVL_ERROR, []byte(e.Error()))
}
