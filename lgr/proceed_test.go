package lgr

/*
import (
	"testing"

	"github.com/stretchr/testify/assert"
)


func TestLogger_proceedMsg(t *testing.T) {
	tests := []struct {
		wantErr bool
		name    string // description of this test case
		// Named input parameters for target function.
		msg logMessage
	}{
		// TODO: Add test cases.
		{false, "log_msgtype", logMessage{msgtype: MSG_LOG_TEXT, msgdata: testlogstr}},
		{true, "unused_msgtype", logMessage{msgtype: MSG_CHG_LEVEL, msgdata: testlogstr}},
		{true, "unknown_msgtype", logMessage{msgtype: 99, msgdata: testlogstr}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out_1 := &FakeWriter{}
			l := InitWithParams(DEFAULT_LOG_LEVEL, nil, out_1)
			gotErr := l.proceedMsg(&tt.msg)
			if tt.wantErr {
				assert.Error(t, gotErr, "no expected error")
				assert.Empty(t, out_1)
			} else {
				assert.Nil(t, gotErr, "unexpected error")
				assert.Equal(t, testlogstr+"\n", out_1.String())
			}
		})
	}
	t.Run("forbidden_msgtype", func(t *testing.T) {
		l := Init() // any outputs, they are not used in this test
		assert.Panics(t, func() {
			l.proceedMsg(&logMessage{msgtype: MSG_FORBIDDEN, msgdata: testlogstr})
		}, "The code did not panic")
	})
	t.Run("empty_msgtype", func(t *testing.T) {
		l := Init() // any outputs, they are not used in this test
		assert.Panics(t, func() {
			l.proceedMsg(&logMessage{msgdata: testlogstr})
		}, "The code did not panic")
	})
}

func TestLogger_logData(t *testing.T) {
	foutput := &FakeWriter{}
	tests := []struct {
		wantPnc bool
		wantErr bool
		name    string // description of this test case
		// Named input parameters for target function.
		output outType
		data   []byte
	}{
		{false, false, "valid_output", foutput, []byte(testlogstr)},
		{false, false, "empty_msg", foutput, []byte{}},
		{false, false, "nil_msg", foutput, nil},
		{false, true, "error_output", outType(&ErrorWriter{}), []byte(testlogstr)},
		{true, true, "panic_output", outType(&PanicWriter{}), []byte(testlogstr)},
		{true, true, "nil_output", nil, []byte(testlogstr)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			foutput.Clear()
			l := Init()
			gotPnc, gotErr := l.logData(tt.output, tt.data)
			assert.True(t, !tt.wantPnc || gotPnc, "did not panic when expected")
			assert.True(t, !tt.wantErr || gotErr != nil, "no error on expected failure")
			assert.False(t, !tt.wantPnc && gotPnc, "unexpected panic")
			assert.False(t, !tt.wantErr && gotErr != nil, fmt.Sprintf("unexpected error: %v", gotErr))
			if !tt.wantPnc && !tt.wantErr {
				assert.Equal(t, string(tt.data), foutput.String(), "written data mismatch")
			}
		})
	}
}

func TestLogger_handleLogWriteError(t *testing.T) {
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
			assert.Equal(t, tt.emsg+"\n", foutput.String(), "written data mismatch")
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

func TestLogger_logTextToOutputs(t *testing.T) {
	msg := &logMessage{msgdata: "Test data"}
	out1 := &FakeWriter{}
	out2 := &FakeWriter{}
	ferr := &FakeWriter{}
	t.Run("one_out", func(t *testing.T) {
		out1.Clear()
		l := InitWithParams(LVL_TRACE, nil, out1)
		l.logTextToOutputs(msg)
		assert.Equal(t, msg.msgdata+"\n", out1.String())
	})
	t.Run("two_outs", func(t *testing.T) {
		out1.Clear()
		out2.Clear()
		l := InitWithParams(LVL_TRACE, nil, out1, out2)
		l.logTextToOutputs(msg)
		assert.Equal(t, msg.msgdata+"\n", out1.String())
		assert.Equal(t, msg.msgdata+"\n", out2.String())
	})
	t.Run("no_outputs_no_fallback", func(t *testing.T) {
		l := InitWithParams(LVL_TRACE, nil)
		assert.NotPanics(t, func() {
			l.logTextToOutputs(msg)
		}, "Panic on write to nil outputs and nil fallback")
	})
	t.Run("no_outputs", func(t *testing.T) {
		ferr.Clear()
		l := InitWithParams(LVL_TRACE, ferr)
		assert.NotPanics(t, func() {
			l.logTextToOutputs(msg)
		}, "Panic on write to nil outputs")
		assert.Equal(t, "", ferr.String())
	})
	t.Run("nil_outs", func(t *testing.T) {
		l := InitWithParams(LVL_TRACE, ferr, nil, nil)
		assert.NotPanics(t, func() {
			l.logTextToOutputs(msg)
		}, "Panic on write to nil outputs")
	})
	t.Run("with_panic", func(t *testing.T) {
		out1.Clear()
		out2.Clear()
		ferr.Clear()
		l := InitWithParams(LVL_TRACE, ferr, out1, &PanicWriter{}, out2)
		l.logTextToOutputs(msg)
		assert.Equal(t, msg.msgdata+"\n", out1.String())
		assert.Equal(t, msg.msgdata+"\n", out2.String())
		assert.Contains(t, ferr.String(), panicStr+"\n")
	})
	t.Run("with_error", func(t *testing.T) {
		out1.Clear()
		out2.Clear()
		ferr.Clear()
		l := InitWithParams(LVL_TRACE, ferr, out1, &ErrorWriter{}, out2)
		l.logTextToOutputs(msg)
		assert.Equal(t, msg.msgdata+"\n", out1.String())
		assert.Equal(t, msg.msgdata+"\n", out2.String())
		assert.Contains(t, ferr.String(), errorStr+"\n")
	})
	t.Run("with_both", func(t *testing.T) {
		out1.Clear()
		out2.Clear()
		ferr.Clear()
		l := InitWithParams(LVL_TRACE, ferr, out1, &ErrorWriter{}, &PanicWriter{}, out2)
		l.logTextToOutputs(msg)
		assert.Equal(t, msg.msgdata+"\n", out1.String())
		assert.Equal(t, msg.msgdata+"\n", out2.String())
		assert.Contains(t, ferr.String(), errorStr+"\n")
		assert.Contains(t, ferr.String(), panicStr+"\n")
	})
	t.Run("with_both_no_fallback", func(t *testing.T) {
		out1.Clear()
		out2.Clear()
		l := InitWithParams(LVL_TRACE, nil, out1, &ErrorWriter{}, &PanicWriter{}, out2)
		assert.NotPanics(t, func() {
			l.logTextToOutputs(msg)
		}, "Panic on write to nil fallback")
		assert.Equal(t, msg.msgdata+"\n", out1.String())
		assert.Equal(t, msg.msgdata+"\n", out2.String())
	})
	t.Run("all_disabled", func(t *testing.T) {
		out1.Clear()
		out2.Clear()
		ferr.Clear()
		l := InitWithParams(LVL_TRACE, ferr, out1, out2)
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
		l := InitWithParams(LVL_TRACE, ferr, out1, out2)
		l.outputs[out1].enabled = true
		l.outputs[out2].enabled = false
		l.logTextToOutputs(msg)
		assert.Equal(t, msg.msgdata+"\n", out1.String())
		assert.Equal(t, "", out2.String())
		assert.Equal(t, "", ferr.String())
	})
}

func TestLogger_procced(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		s := "Write to 2 outputs"
		l := InitWithParams(LVL_TRACE, nil, out1, out2)
		l.Start(0) // start procced goroutine
		l.channel <- logMessage{msgtype: MSG_LOG_TEXT, msgdata: s}
		l.StopAndWait() // set state to STOPPING,
		assert.Equal(t, s+"\n", out1.String())
		assert.Equal(t, s+"\n", out2.String())
	})
	t.Run("panic_in_procced", func(t *testing.T) {
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		ferr := &FakeWriter{}
		s := "Write to 2 outputs and 1 panic"
		l := InitWithParams(LVL_TRACE, ferr, out1, &PanicWriter{}, out2)
		l.Start(0) // start procced goroutine
		l.channel <- logMessage{msgtype: MSG_LOG_TEXT, msgdata: s}
		l.StopAndWait() // set state to STOPPING,
		assert.Equal(t, s+"\n", out1.String())
		assert.Equal(t, s+"\n", out2.String())
		assert.Contains(t, ferr.String(), panicStr+"\n")
	})
	t.Run("procced_unknown_msgtype", func(t *testing.T) {
		ferr := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferr)
		l.Start(0) // start procced goroutine
		l.channel <- logMessage{msgtype: 99, msgdata: "test text"}
		l.StopAndWait() // set state to STOPPING,
		assert.Contains(t, ferr.String(), "unknown message type")
	})
	t.Run("panic_on_empty_msgtype", func(t *testing.T) {
		ferr := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferr)
		l.Start(0) // start procced goroutine
		l.channel <- logMessage{msgdata: "test text"}
		l.StopAndWait() // set state to STOPPING,
		assert.Contains(t, ferr.String(), "panic")
	})
	t.Run("procced_forbidden_msgtype", func(t *testing.T) {
		ferr := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferr)
		l.Start(0) // start procced goroutine
		l.channel <- logMessage{msgtype: MSG_FORBIDDEN, msgdata: "test text"}
		l.StopAndWait() // set state to STOPPING,
		assert.Contains(t, ferr.String(), "panic on forbidden message type")
	})
}
*/
