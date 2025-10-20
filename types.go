// Docs are based on CoPilot (GPT-5 mini) generation
package lgr

import (
	"bytes"
	"io"
	"sync"
	"time"
)

/*
Docs are based on CoPilot (GPT-5 mini) generation
types.go

Defines the core data structures used by the logger:
 - basetype and a small set of typed aliases for clarity
 - logMessage: internal representation of queued items
 - logClient: lightweight client handle that callers obtain to write logs
 - outContext: per-output settings used when formatting messages
 - logger: the central state object that coordinates message queuing,
   output management and the processing goroutine.
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
