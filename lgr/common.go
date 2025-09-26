package lgr

const (
	DEFAULT_LOG_LEVEL = LVL_ERROR
	DEFAULT_BUFF_SIZE = 32
)

type logLevel uint8

const (
	LVL_UNKNOWN logLevel = iota
	LVL_TRACE
	LVL_DEBUG
	LVL_INFO
	LVL_WARN
	LVL_ERROR
	LVL_FATAL
	LVL_UNMASKABLE
	_LVL_MAX_FOR_CHECKS_ONLY
)

type lgrState uint8

const (
	STATE_UNKNOWN lgrState = iota
	STATE_ACTIVE
	STATE_STOPPING
	STATE_STOPPED
	_STATE_MAX_FOR_CHECKS_ONLY
)

type msgType uint8

const (
	MSG_FORBIDDEN msgType = iota
	MSG_LOG_TEXT
	MSG_CHG_LEVEL
	_MSG_MAX_FOR_CHECKS_ONLY
)

func normState(state lgrState) lgrState {
	return norm_uint8(state, _STATE_MAX_FOR_CHECKS_ONLY, STATE_UNKNOWN)
}

func normLevel(level logLevel) logLevel {
	return norm_uint8(level, _LVL_MAX_FOR_CHECKS_ONLY, _LVL_MAX_FOR_CHECKS_ONLY-1)
}

func norm_uint8[T ~uint8](val, overlimit, def T) T {
	if val < overlimit {
		return val
	} else {
		return def
	}
}

const (
	// ANSI colored text is a string like `ESC`[38;2;⟨r⟩;⟨g⟩;⟨b⟩mSome_text`ESC`[0m (`ESC`=\033)
	ANSI_COLOR_PREFIX = "\033["
	ANSI_COLOR_SUFFIX = "m"
	ANSI_COLOR_RESET  = ANSI_COLOR_PREFIX + "0" + ANSI_COLOR_SUFFIX
)

//////////////////////////////////////////////////////////////////////////////////////////
//const logTermReset = "\033[0m"

type logLevelDesc struct {
	Short string
	Long  string
	color string
}

var defaultShortPrefixes *lvlStringMap

func PrefixesShort() *lvlStringMap {
	if defaultShortPrefixes == nil {
		m := make(lvlStringMap)
		m[LVL_UNKNOWN] = "???"
		m[LVL_TRACE] = "TRC"
		m[LVL_DEBUG] = "DBG"
		m[LVL_INFO] = "INF"
		m[LVL_WARN] = "WRN"
		m[LVL_ERROR] = "ERR"
		m[LVL_FATAL] = "FTL"
		m[LVL_UNMASKABLE] = "!!!"
		defaultShortPrefixes = &m
	}
	return defaultShortPrefixes
}

/*
"UNKNOWN"
"TRACE"
"DEBUG"
"INFO"
"WARN"
"ERROR"
"FATAL"
"UNMASKABLE"
*/
var LogLevelDesc map[logLevel]*logLevelDesc

func init() {
	LogLevelDesc = make(map[logLevel]*logLevelDesc)
	LogLevelDesc[LVL_UNKNOWN] = &logLevelDesc{Short: "???", Long: "UNKNOWN"}
	LogLevelDesc[LVL_TRACE] = &logLevelDesc{Short: "TRC", Long: "TRACE"}
	LogLevelDesc[LVL_DEBUG] = &logLevelDesc{Short: "DBG", Long: "DEBUG"}
	LogLevelDesc[LVL_INFO] = &logLevelDesc{Short: "INF", Long: "INFO"}
	LogLevelDesc[LVL_WARN] = &logLevelDesc{Short: "WRN", Long: "WARN"}
	LogLevelDesc[LVL_ERROR] = &logLevelDesc{Short: "ERR", Long: "ERROR"}
	LogLevelDesc[LVL_FATAL] = &logLevelDesc{Short: "FTL", Long: "FATAL"}
	LogLevelDesc[LVL_UNMASKABLE] = &logLevelDesc{Short: "!!!", Long: "UNMASKABLE"}
	//https://habr.com/ru/companies/first/articles/672464/?ysclid=mfy8zz61fw842674829
	LogLevelDesc[LVL_UNKNOWN].color = "\033[9;90m"
	LogLevelDesc[LVL_TRACE].color = "\033[2;90m"
	LogLevelDesc[LVL_DEBUG].color = "\033[0;90m"
	LogLevelDesc[LVL_INFO].color = "\033[0;97m"
	LogLevelDesc[LVL_WARN].color = "\033[0;33m"
	LogLevelDesc[LVL_ERROR].color = "\033[0;91m"
	LogLevelDesc[LVL_FATAL].color = "\033[101m\033[1;33m"
	LogLevelDesc[LVL_UNMASKABLE].color = "\033[107m\033[1;31m"
}

/*func logstr(format, prefix, text string) string {
	return fmt.Sprintf(format, prefix, text)
}*/
