package lgr

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testbytes = []byte(testlogstr)

func Test_Logger_proceedMsg(t *testing.T) {
	tests := []struct {
		wantErr bool
		name    string // description of this test case
		msg     logMessage
	}{
		{false, "cmd_dummy", logMessage{msgtype: _MSG_COMMAND, msgdata: testbytes, annex: basetype(_CMD_DUMMY)}},
		{false, "log_msgtype", logMessage{msgtype: _MSG_LOG_TEXT, msgdata: testbytes, annex: basetype(LVL_UNMASKABLE)}},
		{true, "unused_msgtype", logMessage{msgtype: _MSG_MAX_for_checks_only, msgdata: testbytes, annex: basetype(LVL_UNMASKABLE)}},
		{true, "unknown_msgtype", logMessage{msgtype: _MSG_MAX_for_checks_only + 10, msgdata: testbytes, annex: basetype(LVL_UNMASKABLE)}},
		{true, "cmd_ping_with_err", logMessage{msgtype: _MSG_COMMAND, msgdata: testbytes, annex: basetype(_CMD_PING_FALLBACK)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out1 := &FakeWriter{}
			l := InitWithParams(LVL_TRACE, nil, out1)
			l.msgbuf = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
			gotErr := l.proceedMsg(&tt.msg)
			if tt.wantErr {
				assert.Error(t, gotErr, "no expected error")
				assert.Empty(t, out1.buffer, "unexpected write to output on error")
			} else {
				assert.NoError(t, gotErr, "unexpected error")
				if LogLevel(tt.msg.annex) == LVL_TRACE {
					assert.Contains(t, out1.String(), "COMMAND: ")
					assert.Contains(t, out1.String(), testlogstr)
				} else {
					assert.Equal(t, testlogstr+"\n", out1.String())
				}
			}
		})
	}
	t.Run("forbidden_msgtype", func(t *testing.T) {
		l := Init() // any outputs, they are not used in this test
		assert.Panics(t, func() {
			l.proceedMsg(&logMessage{msgtype: _MSG_FORBIDDEN, msgdata: testbytes})
		}, "The code did not panic")
	})
	t.Run("empty_msgtype", func(t *testing.T) {
		l := Init() // any outputs, they are not used in this test
		assert.Panics(t, func() {
			l.proceedMsg(&logMessage{msgdata: testbytes})
		}, "The code did not panic")
	})
}

func Test_Logger_procced(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		s := "Write to 2 outputs"
		l := InitWithParams(LVL_UNKNOWN, nil, out1, out2)
		l.Start(0) // start procced goroutine
		l.channel <- logMessage{msgtype: _MSG_LOG_TEXT, msgdata: []byte(s)}
		l.StopAndWait() // set state to STOPPING,
		assert.Equal(t, s+"\n", out1.String())
		assert.Equal(t, s+"\n", out2.String())
	})
	t.Run("panic_in_procced", func(t *testing.T) {
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		ferr := &FakeWriter{}
		s := "Write to 2 outputs and 1 panic"
		l := InitWithParams(LVL_UNKNOWN, ferr, out1, &PanicWriter{}, out2)
		l.Start(0) // start procced goroutine
		l.channel <- logMessage{msgtype: _MSG_LOG_TEXT, msgdata: []byte(s)}
		l.StopAndWait() // set state to STOPPING,
		assert.Equal(t, s+"\n", out1.String())
		assert.Equal(t, s+"\n", out2.String())
		assert.Contains(t, ferr.String(), "`"+panicStr+"`\n")
	})
	t.Run("procced_unknown_msgtype", func(t *testing.T) {
		ferr := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferr)
		l.Start(0) // start procced goroutine
		l.channel <- logMessage{msgtype: 99, msgdata: testbytes}
		l.StopAndWait() // set state to STOPPING,
		assert.Contains(t, ferr.String(), "unknown message type")
	})
	t.Run("panic_on_empty_msgtype", func(t *testing.T) {
		ferr := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferr)
		l.Start(0) // start procced goroutine
		l.channel <- logMessage{msgdata: testbytes}
		l.StopAndWait() // set state to STOPPING,
		assert.Contains(t, ferr.String(), "panic")
	})
	t.Run("procced_forbidden_msgtype", func(t *testing.T) {
		ferr := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferr)
		l.Start(0) // start procced goroutine
		l.channel <- logMessage{msgtype: _MSG_FORBIDDEN, msgdata: testbytes}
		l.StopAndWait() // set state to STOPPING,
		assert.Contains(t, ferr.String(), "panic on forbidden message type")
	})
}

func Test_Logger_logData(t *testing.T) {
	foutput := &FakeWriter{}
	tests := []struct {
		wantPnc bool
		wantErr bool
		name    string // description of this test case
		// Named input parameters for target function.
		output OutType
		data   []byte
	}{
		{false, false, "valid_output", foutput, []byte(testlogstr)},
		{false, false, "empty_msg", foutput, []byte{}},
		{false, false, "nil_msg", foutput, nil},
		{false, true, "error_output", OutType(&ErrorWriter{}), []byte(testlogstr)},
		{true, true, "panic_output", OutType(&PanicWriter{}), []byte(testlogstr)},
		{true, true, "nil_output", nil, []byte(testlogstr)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foutput.Clear()
			l := Init()
			l.msgbuf = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
			gotPnc, gotErr := l.logTextData(tt.output, &logMessage{msgtype: 99, msgdata: tt.data})
			assert.True(t, !tt.wantPnc || gotPnc, "did not panic when expected")
			assert.True(t, !tt.wantErr || gotErr != nil, "no error on expected failure")
			assert.False(t, !tt.wantPnc && gotPnc, "unexpected panic")
			assert.False(t, !tt.wantErr && gotErr != nil, fmt.Sprintf("unexpected error: %v", gotErr))
			if !tt.wantPnc && !tt.wantErr {
				assert.Equal(t, string(tt.data)+"\n", foutput.String(), "written data mismatch")
			}
		})
	}
}

func Test_Logger_handleLogWriteError(t *testing.T) {
	foutput := &FakeWriter{}
	tests := []struct {
		name string // description of this test case
		emsg string
	}{
		{"text", "normal text"},
		{"utf8", "нормальный текст"},
		{"none", ""},
		{"escp", testlogstr},
	}
	l := InitWithParams(LVL_TRACE, foutput)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l.handleLogWriteError(tt.emsg)
			assert.Equal(t, tt.emsg+"\n", foutput.String()[len(_FALLBACK_TIME_FORMAT):], "written data mismatch")
			foutput.Clear()
		})
	}
	t.Run("2nil", func(t *testing.T) {
		l := InitWithParams(LVL_TRACE, nil)
		assert.NotPanics(t, func() {
			l.handleLogWriteError("test write to nil fallback")
		}, "Panic on write to nil fallback")
	})
	t.Run("panic", func(t *testing.T) {
		l := InitWithParams(LVL_TRACE, &PanicWriter{})
		assert.Panics(t, func() {
			l.handleLogWriteError("test panic")
		}, "The code did not panic")
	})
}

func Test_Logger_logTextToOutputs(t *testing.T) {
	msg := &logMessage{msgdata: testbytes}
	out1 := &FakeWriter{}
	out2 := &FakeWriter{}
	ferr := &FakeWriter{}
	t.Run("one_out", func(t *testing.T) {
		out1.Clear()
		l := InitWithParams(LVL_UNKNOWN, nil, out1)
		l.msgbuf = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		l.logTextToOutputs(msg)
		assert.Equal(t, string(msg.msgdata)+"\n", out1.String())
	})
	t.Run("two_outs", func(t *testing.T) {
		out1.Clear()
		out2.Clear()
		l := InitWithParams(LVL_UNKNOWN, nil, out1, out2)
		l.msgbuf = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		l.logTextToOutputs(msg)
		assert.Equal(t, string(msg.msgdata)+"\n", out1.String())
		assert.Equal(t, string(msg.msgdata)+"\n", out2.String())
	})
	t.Run("two_outs_one_write", func(t *testing.T) {
		out1.Clear()
		out2.Clear()
		l := InitWithParams(LVL_UNKNOWN, nil, out1, out2)
		l.outputs[out2].minlevel = LVL_TRACE
		l.msgbuf = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		l.logTextToOutputs(msg)
		assert.Equal(t, string(msg.msgdata)+"\n", out1.String())
		assert.Empty(t, out2.buffer, "unexpected write to outpus with lower level: ["+out2.String()+"], len="+strconv.Itoa(len(out2.buffer)))
	})
	t.Run("no_outputs_no_fallback", func(t *testing.T) {
		l := InitWithParams(LVL_UNKNOWN, nil)
		l.msgbuf = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		assert.NotPanics(t, func() {
			l.logTextToOutputs(msg)
		}, "Panic on write to nil outputs and nil fallback")
	})
	t.Run("no_outputs", func(t *testing.T) {
		ferr.Clear()
		l := InitWithParams(LVL_UNKNOWN, ferr)
		assert.NotPanics(t, func() {
			l.logTextToOutputs(msg)
		}, "Panic on write to nil outputs")
		assert.Equal(t, "", ferr.String())
	})
	t.Run("nil_outs", func(t *testing.T) {
		l := InitWithParams(LVL_UNKNOWN, ferr, nil, nil)
		assert.NotPanics(t, func() {
			l.logTextToOutputs(msg)
		}, "Panic on write to nil outputs")
	})
	t.Run("with_panic", func(t *testing.T) {
		out1.Clear()
		out2.Clear()
		ferr.Clear()
		l := InitWithParams(LVL_UNKNOWN, ferr, out1, &PanicWriter{}, out2)
		l.msgbuf = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		l.logTextToOutputs(msg)
		assert.Equal(t, string(msg.msgdata)+"\n", out1.String())
		assert.Equal(t, string(msg.msgdata)+"\n", out2.String())
		assert.Contains(t, ferr.String(), panicStr+"`\n")
	})
	t.Run("with_error", func(t *testing.T) {
		out1.Clear()
		out2.Clear()
		ferr.Clear()
		l := InitWithParams(LVL_UNKNOWN, ferr, out1, &ErrorWriter{}, out2)
		l.msgbuf = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		l.logTextToOutputs(msg)
		assert.Equal(t, string(msg.msgdata)+"\n", out1.String())
		assert.Equal(t, string(msg.msgdata)+"\n", out2.String())
		assert.Contains(t, ferr.String(), errorStr+"\n")
	})
	t.Run("with_both", func(t *testing.T) {
		out1.Clear()
		out2.Clear()
		ferr.Clear()
		l := InitWithParams(LVL_UNKNOWN, ferr, out1, &ErrorWriter{}, &PanicWriter{}, out2)
		l.msgbuf = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		l.logTextToOutputs(msg)
		assert.Equal(t, string(msg.msgdata)+"\n", out1.String())
		assert.Equal(t, string(msg.msgdata)+"\n", out2.String())
		assert.Contains(t, ferr.String(), errorStr+"\n")
		assert.Contains(t, ferr.String(), panicStr+"`\n")
	})
	t.Run("with_both_no_fallback", func(t *testing.T) {
		out1.Clear()
		out2.Clear()
		l := InitWithParams(LVL_UNKNOWN, nil, out1, &ErrorWriter{}, &PanicWriter{}, out2)
		l.msgbuf = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		assert.NotPanics(t, func() {
			l.logTextToOutputs(msg)
		}, "Panic on write to nil fallback")
		assert.Equal(t, string(msg.msgdata)+"\n", out1.String())
		assert.Equal(t, string(msg.msgdata)+"\n", out2.String())
	})
	t.Run("all_disabled", func(t *testing.T) {
		out1.Clear()
		out2.Clear()
		ferr.Clear()
		l := InitWithParams(LVL_UNKNOWN, ferr, out1, out2)
		l.msgbuf = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		l.outputs[out1].enabled = false
		l.outputs[out2].enabled = false
		l.logTextToOutputs(msg)
		assert.Equal(t, "", out1.String())
		assert.Equal(t, "", out2.String())
		assert.Equal(t, "", ferr.String())
	})
	t.Run("one_enabled", func(t *testing.T) {
		out1.Clear()
		out2.Clear()
		ferr.Clear()
		l := InitWithParams(LVL_UNKNOWN, ferr, out1, out2)
		l.msgbuf = bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		l.outputs[out1].enabled = true
		l.outputs[out2].enabled = false
		l.logTextToOutputs(msg)
		assert.Equal(t, string(msg.msgdata)+"\n", out1.String())
		assert.Equal(t, "", out2.String())
		assert.Equal(t, "", ferr.String())
	})
}

func Test_Logger_buildTextMessage1(t *testing.T) {
	ti := time.Now()
	outBuffer := bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
	msg := &logMessage{
		pushed:  ti,
		msgdata: testbytes,
		msgtype: _MSG_LOG_TEXT,
		annex:   basetype(LVL_UNMASKABLE),
	}
	t.Run("with_time", func(t *testing.T) {
		ctx := &outContext{}
		ctx.timefmt = time.RFC1123
		buff := buildTextMessage(outBuffer, msg, ctx)
		assert.Regexp(t, "^"+ti.Format(ctx.timefmt)+".*", buff.String())
	})
	t.Run("with_time", func(t *testing.T) {
		context := &outContext{}
		context.showlvlid = true
		buff := buildTextMessage(outBuffer, msg, context)
		assert.Regexp(t, "^"+ti.Format(context.timefmt)+".*", buff.String())
	})
}

func Test_Logger_buildTextMessage(t *testing.T) {
	outBuffer := bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
	lcName := "Logger client name [" + testlogstr + "]"
	l := Init()
	lc := l.NewClientWithLevel(lcName, LVL_UNKNOWN)
	ti := time.Now()
	dm := " -+\033F:" + testlogstr + "\n"
	lvl := LVL_UNMASKABLE
	msg := &logMessage{
		pushed:  ti,
		msgdata: []byte(testlogstr),
		msgtype: _MSG_LOG_TEXT,
		annex:   basetype(lvl),
	}
	tests := []struct {
		name    string // description of this test case
		result  string
		context *outContext
		client  *LogClient
	}{
		{"only_message",
			testlogstr,
			&outContext{},
			nil,
		},
		{"time",
			ti.Format(time.RFC1123) + testlogstr,
			&outContext{timefmt: time.RFC1123},
			nil,
		},
		{"time_with_delim",
			ti.Format(time.RFC1123) + testlogstr,
			&outContext{timefmt: time.RFC1123, delimiter: []byte(dm)},
			nil,
		},
		{"lvl_id",
			"[" + strconv.Itoa(int(lvl)) + "]" + testlogstr,
			&outContext{showlvlid: true},
			nil,
		},
		{"short_prefix",
			LevelShortNames[lvl] + testlogstr,
			&outContext{prefixmap: LevelShortNames},
			nil,
		},
		{"short_prefix_with_delim",
			LevelShortNames[lvl] + dm + testlogstr,
			&outContext{prefixmap: LevelShortNames, delimiter: []byte(dm)},
			nil,
		},
		{"colors",
			ANSI_COL_PRFX + LevelColorOnBlackMap[lvl] + ANSI_COL_SUFX + testlogstr + ANSI_COL_RESET,
			&outContext{colormap: LevelColorOnBlackMap},
			nil,
		},
		{"colors_with_delim",
			ANSI_COL_PRFX + LevelColorOnBlackMap[lvl] + ANSI_COL_SUFX + testlogstr + ANSI_COL_RESET,
			&outContext{colormap: LevelColorOnBlackMap, delimiter: []byte(dm)},
			nil,
		},
		{"client_name",
			lcName + testlogstr,
			&outContext{},
			lc,
		},
		{"client_name_with_delim",
			lcName + dm + testlogstr,
			&outContext{delimiter: []byte(dm)},
			lc,
		},
		{"all_together",
			"" +
				ti.Format(time.RFC1123) +
				"[" + strconv.Itoa(int(lvl)) + "]" + dm +
				LevelShortNames[lvl] + dm +
				ANSI_COL_PRFX + LevelColorOnBlackMap[lvl] + ANSI_COL_SUFX + testlogstr + ANSI_COL_RESET,
			&outContext{
				timefmt:   time.RFC1123,
				showlvlid: true,
				prefixmap: LevelShortNames,
				colormap:  LevelColorOnBlackMap,
				delimiter: []byte(dm),
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg.msgclnt = tt.client
			s := buildTextMessage(outBuffer, msg, tt.context).String()
			assert.Equal(t, tt.result+"\n", s)
		})
	}
	t.Run("nil_msg", func(t *testing.T) {
		s := buildTextMessage(outBuffer, nil, nil).String()
		assert.Equal(t, "", s)
	})
	t.Run("empty_msg", func(t *testing.T) {
		s := buildTextMessage(outBuffer, &logMessage{}, nil).String()
		assert.Equal(t, "\n", s)
	})
}

func Test_Logger_proceedCmd(t *testing.T) {
	const testname = "Test Client Name"
	ferr := &FakeWriter{}
	l1 := Init()
	l1.SetFallback(ferr)
	lc1 := l1.NewClientWithLevel(testname, LVL_UNKNOWN)
	tests := []struct {
		name    string // description of this test case
		cmd     cmdType
		lc      *LogClient
		data    []byte
		wantErr string
	}{
		{"ping", _CMD_PING_FALLBACK, nil, []byte{}, "<ping>"},
		{"dummy", _CMD_DUMMY, nil, []byte{}, ""},
		{"unknown", _CMD_MAX_for_checks_only + 10, nil, []byte{}, "unknown command"},
		{"min_level", _CMD_CLIENT_SET_LEVEL, lc1, []byte{byte(LVL_FATAL)}, ""},
		{"min_level_no_data", _CMD_CLIENT_SET_LEVEL, lc1, []byte{}, "no data"},
		{"min_level_nil_client", _CMD_CLIENT_SET_LEVEL, nil, []byte{byte(LVL_FATAL)}, "nil client"},
		{"new_name", _CMD_CLIENT_SET_NAME, lc1, []byte{byte(LVL_FATAL)}, ""},
		{"new_name_no_data", _CMD_CLIENT_SET_NAME, lc1, []byte{}, "no data"},
		{"new_name_nil_client", _CMD_CLIENT_SET_NAME, nil, []byte{byte(LVL_FATAL)}, "nil client"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ferr.Clear()
			msg := &logMessage{
				msgtype: _MSG_COMMAND,
				msgclnt: tt.lc,
				msgdata: tt.data,
				annex:   basetype(tt.cmd),
			}
			assert.NotPanics(t, func() {
				l1.proceedCmd(msg)
			})
			res := ferr.String()
			if tt.wantErr == "" {
				assert.Empty(t, ferr.buffer, "unexpected error: '"+res+"'")
			} else {
				assert.Contains(t, ferr.String(), tt.wantErr, "wrong error text")
			}
		})
	}
	t.Run("empty", func(t *testing.T) {
		ferr.Clear()
		var msg *logMessage
		assert.NotPanics(t, func() {
			l1.proceedCmd(msg)
		})
		assert.Contains(t, ferr.String(), "nil command", "wrong error text")
	})
}
