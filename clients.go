package lgr

import (
	"errors"
	"fmt"
	"time"
)

/*
Docs are based on CoPilot (GPT-5 mini) generation
clients.go

Lightweight client-side helpers for producing log messages. A logClient is a
thin handle that records per-client settings (name, minLevel, enabled) and
forwards textual data to the owning logger instance. The functions below
provide both error-returning variants (suffixed with _with_err) and
convenience wrappers that surface write failures to the logger's fallback
writer.

Concurrency notes:
 - logClient methods are typically called by application goroutines.
 - pushMessage enqueues into the logger channel and performs the necessary
   state checks; it recovers panics caused by sending on a closed channel and
   converts them into an error value.
 - Client fields (name, minLevel, enabled) can be mutated by commands
   processed by the background worker (see runClientCommand/proceedCmd). The
   logger enforces ordering by applying changes via queued command messages.

Log_with_err writes a textual message at the provided level and returns the
time the message was queued and any error encountered while attempting to
enqueue the message. This wrapper simply converts the string to bytes and
delegates to LogBytes_with_err.
*/

func (lc *logClient) Log_with_err(level LogLevel, s string) (time.Time, error) {
	return lc.LogBytes_with_err(level, []byte(s))
}

// LogBytes_with_err enqueues a raw byte payload as a log message at the given
// level. It returns the push timestamp and an error if the logger is nil,
// inactive, the channel is unavailable, or a panic occurred while sending.
//
// Filtering behavior:
//   - If the client is disabled, or the message level is below the client's
//     minLevel, or below the global logger level, the call is a no-op and
//     returns zero time + nil error (message intentionally ignored).
//
// Note: There is a test-only check that panics if logger.level is invalid; in
// normal code SetMinLevel/normLevel should prevent invalid values.
func (lc *logClient) LogBytes_with_err(level LogLevel, data []byte) (t time.Time, err error) {
	if lc.logger == nil {
		return t, fmt.Errorf("logger is nil")
	}
	if lc.logger.level >= _LVL_MAX_for_checks_only {
		// For testing purposes only — exercising panic recovery paths.
		panic(errors.New("panic on forbidden log level"))
	}
	// Apply per-client and global filtering before enqueuing.
	if !lc.enabled || level < lc.minLevel || level < lc.logger.level {
		// intentionally ignore message; caller treats nil error as success/ignore
		return
	}
	t, err = lc.logger.pushMessage(makeTextMessage(lc, level, data))
	return t, err
}

// Log is a convenience wrapper that takes a string and sends it without
// returning write errors. Use Log_with_err when callers need to react to
// delivery problems.
func (lc *logClient) Log(level LogLevel, s string) time.Time {
	return lc.LogBytes(level, []byte(s))
}

// LogBytes is the bytes variant of Log. If an underlying enqueue/write error
// occurs, the logger's fallback error handler is invoked so the program can
// still observe the error via the fallback writer.
func (lc *logClient) LogBytes(level LogLevel, data []byte) time.Time {
	t, err := lc.LogBytes_with_err(level, data)
	if err != nil {
		// Report the write/enqueue error to the logger fallback. This keeps the
		// simple Log* API ergonomic while still surfacing failures.
		lc.logger.handleLogWriteError(err.Error())
	}
	return t
}

// Convenience level-specific helpers for common log levels.
// These are thin wrappers around LogBytes that provide inline hints in
// editors and documentation tools.
//
// All of these helpers behave like LogBytes: they do not return an error.
// If an enqueue/write error occurs it will be reported to the logger's
// fallback writer (via handleLogWriteError).

// LogTrace logs a textual message at TRACE level.
//
// Use this for very verbose diagnostic information. Does not accept an error
// value and does not return enqueue/write errors — failures are forwarded to
// the logger fallback writer.
func (lc *logClient) LogTrace(s string) time.Time {
	return lc.LogBytes(LVL_TRACE, []byte(s))
}

// LogDebug logs a textual message at DEBUG level.
//
// Intended for developer-focused debugging output. Like other convenience
// helpers it does not return an error; write failures are reported to the
// fallback writer.
func (lc *logClient) LogDebug(s string) time.Time {
	return lc.LogBytes(LVL_DEBUG, []byte(s))
}

// LogInfo logs an informational message at INFO level.
//
// Use for normal operational messages. This helper mirrors LogBytes semantics:
// it does not return write errors and forwards any failures to the fallback.
func (lc *logClient) LogInfo(s string) time.Time {
	return lc.LogBytes(LVL_INFO, []byte(s))
}

// LogWarn logs a warning message at WARN level.
//
// Use for recoverable or noteworthy conditions that deserve attention. As with
// the other helpers, it does not return enqueue/write errors; failures are
// reported to the logger fallback writer.
func (lc *logClient) LogWarn(s string) time.Time {
	return lc.LogBytes(LVL_WARN, []byte(s))
}

// LogError logs an arbitrary textual message at ERROR level.
//
// Use this when you have a formatted or constructed string that represents
// an error condition. The function accepts a string and writes it at
// LVL_ERROR. It does NOT accept an error value and does not return a send
// error — write failures are forwarded to the logger fallback writer.
func (lc *logClient) LogError(s string) time.Time {
	return lc.LogBytes(LVL_ERROR, []byte(s))
}

// LogErr logs an error.Value at ERROR level.
//
// This is a convenience specifically for error values: it calls Error() on the
// provided error and logs that string at LVL_ERROR. Semantically equivalent to
// calling LogError(err.Error()) but clearer at call sites when you already
// have an error object. Like LogError, it does not return enqueue/write errors.
func (lc *logClient) LogErr(e error) time.Time {
	return lc.LogBytes(LVL_ERROR, []byte(e.Error()))
}

/////////////////////////////////////////////////////////////////////////////////////
// io.Writer interface implementation
//
// The logClient implements io.Writer so it can be used with fmt.Fprintf and
// other formatting helpers. The semantics are:
//  - Lvl(level) sets the current level used by subsequent Write calls.
//  - Write(p) enqueues the bytes at the currently set curLevel and returns
//    len(p) on success, 0 and a non-nil error on failure.
//
// This allows patterns like:
//   fmt.Fprintf(client.Lvl(LVL_WARN), "disk low: %d%%", percent)
// But remember that fmt is not thread-safe!

// Lvl sets the client's current level (used by Write/fmt.Fprintf) and returns
// the same client for convenient chaining.
func (lc *logClient) Lvl(level LogLevel) *logClient {
	lc.curLevel = normLevel(level)
	return lc
}

// Write implements io.Writer. It forwards the provided bytes as a log message
// at the client's curLevel. On success it returns n=len(p) and err==nil.
// If the payload is nil it is treated as a zero-length write with no error.
func (lc *logClient) Write(p []byte) (n int, err error) {
	if p == nil {
		return 0, nil
	}
	_, err = lc.LogBytes_with_err(lc.curLevel, p)
	if err == nil {
		n = len(p)
	} else {
		n = 0
	}
	return
}
