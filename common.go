package lgr

import (
	"bytes"
	"io"
	"sync"
	"time"
)

/*
Defines the core data types used by the logger:
 - basetype and a small set of typed aliases for clarity
 - logMessage: internal representation of queued items
 - logClient: lightweight client handle that callers obtain to write logs
 - outContext: per-output settings used when formatting messages
 - logger: the central state object that coordinates message queuing,
   output management and the processing goroutine.

Also defines package-wide constants, enums and helper utilities used by the logger:
 - default sizes and values
 - ANSI/color related constants
 - enums for levels/state/message types
 - normalization helpers
*/

// basetype is the underlying byte-sized representation used for enums.
type basetype byte

// Strongly-typed aliases over basetype for clarity and type-safety.
type LogLevel basetype
type lgrState basetype
type msgType basetype
type cmdType basetype

// outType is an alias for io.Writer to represent logger outputs.
type outType io.Writer

// outList maps output writers to their per-output context (settings).
type outList map[outType]*outContext

// logMessage is the unit enqueued into the logger channel. It may represent
// a textual log entry (MSG_LOG_TEXT) or a command (MSG_COMMAND). The annex
// field stores either a LogLevel or a cmdType (encoded via basetype).
type logMessage struct {
	pushed  time.Time  // timestamp when message was queued
	msgclnt *logClient // originating client (may be nil for some internal messages)
	msgdata []byte     // payload (text or command data)
	msgtype msgType    // message type enum
	annex   basetype   // extra byte-sized value (level or command id)
}

// logClient represents a producer of log messages. Clients are lightweight
// and intended to be created by logger.NewClient.
type logClient struct {
	logger   *logger  // owning logger instance
	name     []byte   // client name used in output (raw bytes for efficiency)
	minLevel LogLevel // per-client minimal level to accept
	curLevel LogLevel // current level used by Write / fmt.Fprintf helpers
	enabled  bool     // whether the client may submit messages
}

// outContext holds formatting and filtering options for a specific output.
type outContext struct {
	colormap  *LevelMap // logLevel-associated ANSI terminal color fragments
	prefixmap *LevelMap // per-level textual prefix
	delimiter []byte    // separator after prefix/client name (usually ":")
	timefmt   string    // time.Format string; if empty, no timestamp is written
	showlvlid bool      // whether to include numeric level id like "[3]"
	enabled   bool      // whether this output is enabled for writing
	minlevel  LogLevel  // minimal level accepted by this output
}

// logger is the central state holder. It contains synchronization primitives,
// the outputs map, fallback writer, message channel and buffer used while
// formatting writes.
type logger struct {
	sync struct {
		statMtx sync.RWMutex   // guards state and channel checks
		fbckMtx sync.RWMutex   // guards access to fallback writer
		outsMtx sync.RWMutex   // guards outputs map
		chngMtx sync.RWMutex   // guards general configuration changes
		clntMtx sync.RWMutex   // guards client mutations while processing commands
		procMtx sync.RWMutex   // guards message processing (read lock used during procced)
		waitEnd sync.WaitGroup // tracks background goroutine lifecycle
	}
	outputs outList // map of outputs and per-output contexts
	fallbck outType // fallback writer used to report internal errors
	channel chan logMessage
	msgbuf  *bytes.Buffer // buffer reused while building formatted output
	state   lgrState
	level   LogLevel // global minimal level for the logger
}

/////////////////////////////////////////////////////////////////////////////////////////

const (
	// Log level values. The trailing _LVL_MAX_for_checks_only is used as an
	// exclusive upper bound for normalization checks.
	LVL_UNKNOWN LogLevel = iota
	LVL_TRACE
	LVL_DEBUG
	LVL_INFO
	LVL_WARN
	LVL_ERROR
	LVL_FATAL
	LVL_UNMASKABLE
	_LVL_MAX_for_checks_only
)

const (
	// ANSI colored text fragments prefix/suffix used when colors are requested.
	// For a colored piece of text the sequence will be:
	// ANSI_COL_PRFX + colorSpec + ANSI_COL_SUFX + text + ANSI_COL_RESET
	ANSI_COL_PRFX  = "\033["
	ANSI_COL_SUFX  = "m"
	ANSI_COL_RESET = ANSI_COL_PRFX + "0" + ANSI_COL_SUFX
)

const (
	// Default values
	_DEFAULT_LOG_LEVEL = LVL_ERROR
	_DEFAULT_MSG_BUFF  = 32  // default buffer size of messages channel
	_DEFAULT_OUT_BUFF  = 256 // initial buffer size for log output text
	_DEFAULT_DELIMITER = ":" // default delimiter between log fields (except time)
)

const (
	// Logger lifecycle states.
	_STATE_UNKNOWN lgrState = iota
	_STATE_ACTIVE
	_STATE_STOPPING
	_STATE_STOPPED
	_STATE_MAX_for_checks_only
)

const (
	// Message types that can be enqueued.
	_MSG_FORBIDDEN msgType = iota // only for testing panic in proceed
	_MSG_LOG_TEXT
	_MSG_COMMAND
	_MSG_MAX_for_checks_only
)

const (
	// Command ID layout. Values are arranged so helper ranges exist for client
	// commands checks.
	_CMD_DUMMY, _CMD_MIN_for_checks_only cmdType = iota, iota
	_CMD_CLIENT_DUMMY, _CMD_CLIENT_commands_min
	_CMD_CLIENT_SET_LEVEL, _
	_CMD_CLIENT_SET_NAME, _CMD_CLIENT_commands_max
	_CMD_PING_FALLBACK, _CMD_MAX_for_checks_only
)

// LevelMap is a fixed-size array with one entry per log level. Using an array
// instead of a map avoids allocations and simplifies indexing by LogLevel.
type LevelMap [_LVL_MAX_for_checks_only]string

// Predefined name and color maps for convenience. They are pointers to a LevelMap
// so they can be passed as nil or referenced directly by outContext settings.
var LevelShortNames = &LevelMap{
	"???", //LVL_UNKNOWN
	"TRC", //LVL_TRACE
	"DBG", //LVL_DEBUG
	"INF", //LVL_INFO
	"WRN", //LVL_WARN
	"ERR", //LVL_ERROR
	"FTL", //LVL_FATAL
	"!!!", //LVL_UNMASKABLE
}

var LevelFullNames = &LevelMap{
	"UNKNOWN",    //LVL_UNKNOWN
	"TRACE",      //LVL_TRACE
	"DEBUG",      //LVL_DEBUG
	"INFO",       //LVL_INFO
	"WARN",       //LVL_WARN
	"ERROR",      //LVL_ERROR
	"FATAL",      //LVL_FATAL
	"UNMASKABLE", //LVL_UNMASKABLE
}

var LevelColorOnBlackMap = &LevelMap{
	"9;90",     //LVL_UNKNOWN
	"2;90",     //LVL_TRACE
	"0;90",     //LVL_DEBUG
	"0;97",     //LVL_INFO
	"0;33",     //LVL_WARN
	"0;91",     //LVL_ERROR
	"101;1;33", //LVL_FATAL
	"107;1;31", //LVL_UNMASKABLE
}

// Generic byte normalization helper used by normState/normLevel.
// The type parameter T must be a byte-like alias.
func norm_byte[T ~byte](val, overlimit, def T) T {
	if val < overlimit {
		return val
	} else {
		return def
	}
}

// normState ensures a provided lgrState is within the valid range and
// returns STATE_UNKNOWN on invalid values.
func normState(state lgrState) lgrState {
	return norm_byte(state, _STATE_MAX_for_checks_only, _STATE_UNKNOWN)
}

// normLevel ensures a provided LogLevel is valid and returns LVL_UNKNOWN
// for out of range values.
func normLevel(level LogLevel) LogLevel {
	return norm_byte(level, _LVL_MAX_for_checks_only, LVL_UNKNOWN)
}

// panicDesc converts an arbitrary recovered panic value into a readable
// string used when translating panics into errors or fallback messages.
func panicDesc(panic any) (errtext string) {
	switch v := panic.(type) {
	case string:
		errtext = ": `" + v + "`"
	case error:
		errtext = ": (error) `" + v.Error() + "`"
	default:
		errtext = " " + _ERROR_UNKNOWN_PANIC_TEXT
	}
	return errtext
}
