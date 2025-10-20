package lgr

/*
Docs are based on CoPilot (GPT-5 mini) generation
common.go

Defines package-wide constants, enums and helper utilities used by the
logger implementation: default sizes, enums for levels/state/message types,
normalization helpers and ANSI/color related constants.
*/

const (
	DEFAULT_LOG_LEVEL = LVL_ERROR
	DEFAULT_MSG_BUFF  = 32  // default buffered messages in channel
	DEFAULT_OUT_BUFF  = 256 // initial buffer size for formatting output
)

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
	// Logger lifecycle states.
	STATE_UNKNOWN lgrState = iota
	STATE_ACTIVE
	STATE_STOPPING
	STATE_STOPPED
	_STATE_MAX_for_checks_only
)

const (
	// Message types that can be enqueued.
	MSG_FORBIDDEN msgType = iota
	MSG_LOG_TEXT
	MSG_COMMAND
	_MSG_MAX_for_checks_only
)

const (
	// Command ID layout. Values are arranged so helper ranges exist for client
	// commands checks.
	CMD_DUMMY, _CMD_MIN_for_checks_only cmdType = iota, iota
	CMD_CLIENT_DUMMY, _CMD_CLIENT_commands_min
	CMD_CLIENT_SET_LEVEL, _
	CMD_CLIENT_SET_NAME, _CMD_CLIENT_commands_max
	CMD_PING_FALLBACK, _CMD_MAX_for_checks_only
)

// normState ensures a provided lgrState is within the valid range and
// returns STATE_UNKNOWN on invalid values.
func normState(state lgrState) lgrState {
	return norm_byte(state, _STATE_MAX_for_checks_only, STATE_UNKNOWN)
}

// normLevel ensures a provided LogLevel is valid and returns LVL_UNKNOWN
// for out of range values.
func normLevel(level LogLevel) LogLevel {
	return norm_byte(level, _LVL_MAX_for_checks_only, LVL_UNKNOWN)
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

const DEFAULT_DELIMITER = ":"

const (
	// ANSI colored text fragments prefix/suffix used when colors are requested.
	// For a colored piece of text the sequence will be:
	// ANSI_COL_PRFX + colorSpec + ANSI_COL_SUFX + text + ANSI_COL_RESET
	ANSI_COL_PRFX  = "\033["
	ANSI_COL_SUFX  = "m"
	ANSI_COL_RESET = ANSI_COL_PRFX + "0" + ANSI_COL_SUFX
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

const UNKNOWN_PANIC_TEXT = "[no panic description]"

// panicDesc converts an arbitrary recovered panic value into a readable
// string used when translating panics into errors or fallback messages.
func panicDesc(panic any) (errtext string) {
	switch v := panic.(type) {
	case string:
		errtext = ": `" + v + "`"
	case error:
		errtext = ": (error) `" + v.Error() + "`"
	default:
		errtext = " " + UNKNOWN_PANIC_TEXT
	}
	return errtext
}
