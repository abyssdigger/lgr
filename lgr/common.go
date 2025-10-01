package lgr

const (
	DEFAULT_LOG_LEVEL = LVL_ERROR
	DEFAULT_MSG_BUFF  = 32
	DEFAULT_OUT_BUFF  = 256
)

const (
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
	STATE_UNKNOWN lgrState = iota
	STATE_ACTIVE
	STATE_STOPPING
	STATE_STOPPED
	_STATE_MAX_FOR_CHECKS_ONLY
)

const (
	MSG_FORBIDDEN msgType = iota
	MSG_LOG_TEXT
	MSG_CHG_LEVEL
	_MSG_MAX_FOR_CHECKS_ONLY
)

func normState(state lgrState) lgrState {
	return norm_byte(state, _STATE_MAX_FOR_CHECKS_ONLY, STATE_UNKNOWN)
}

func normLevel(level LogLevel) LogLevel {
	return norm_byte(level, _LVL_MAX_for_checks_only, LVL_UNKNOWN)
}

func norm_byte[T ~byte](val, overlimit, def T) T {
	if val < overlimit {
		return val
	} else {
		return def
	}
}

const DEFAULT_DELIMITER = ":"

const (
	// ANSI colored text is a string like \033[38;2;⟨r⟩;⟨g⟩;⟨b⟩mSome_colored_text\033[0m
	ANSI_COL_PRFX  = "\033["
	ANSI_COL_SUFX  = "m"
	ANSI_COL_RESET = ANSI_COL_PRFX + "0" + ANSI_COL_SUFX
)

type LevelMap [_LVL_MAX_for_checks_only]string

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
