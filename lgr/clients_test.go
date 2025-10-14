package lgr

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_logClient_Log_with_err(t *testing.T) {
	var msg *logMessage
	var err error
	var l *logger
	var lc *logClient
	prep := func(f func(), loglevel LogLevel, msglevel LogLevel, ferr *FakeWriter, outs ...outType) (*logMessage, error) {
		l = Init()
		l.SetFallback(outType(ferr)).ClearOutputs().AddOutputs(outs...)
		lc = l.NewClient("[Testing client name]", LVL_UNKNOWN)
		if f == nil {
			l.Start(0)
		} else {
			f()
		}
		l.level = loglevel
		t, e := lc.Log_with_err(msglevel, testlogstr)
		if f == nil {
			l.StopAndWait()
		}
		if e == nil {
			msg := makeTextMessage(lc, loglevel, []byte(testlogstr))
			msg.pushed = t
			return msg, nil
		}
		return nil, e
	}
	t.Run("panic_on_closed_channel", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg, err = prep(func() {
				l.Start(0)
				close(l.channel)
			}, LVL_WARN, LVL_WARN, ferr, out1)
		}, "Panic on write")
		l.channel = make(chan logMessage)
		l.StopAndWait()
		assert.Error(t, err, "no error on log to stopped logger")
		assert.ErrorContains(t, err, "panic", "Wrong error on log to stopped logger")
		assert.Empty(t, ferr.buffer, "data written to fallback")
		assert.Empty(t, out1.buffer, "data written to output")
	})
	t.Run("not_active", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg, err = prep(func() {}, LVL_WARN, LVL_WARN, ferr, out1)
		}, "Panic on write")
		l.StopAndWait()
		assert.ErrorContains(t, err, "not active", "Wrong error on log to stopped logger")
		assert.Empty(t, ferr.buffer, "data written to fallback")
		assert.Empty(t, out1.buffer, "data written to output")
	})
	t.Run("unrecovered_panic", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		assert.Panics(t, func() {
			err = nil
			msg, err = prep(nil, _LVL_MAX_for_checks_only, LVL_DEBUG, ferr, out1, out2)
		}, "No panic on unmasked panic")
		assert.NoError(t, err, "unexpected error")
		assert.Nil(t, msg, "message not nil")
		assert.Empty(t, out1.buffer, "data written to output 1")
		assert.Empty(t, out2.buffer, "data written to output 2")
		assert.Empty(t, ferr.buffer, "data written to fallback")
	})
	t.Run("error_on_nil_channel", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg, err = prep(func() {
				l.state = STATE_ACTIVE
			}, LVL_INFO, LVL_WARN, ferr, out1)
		}, "Panic on write")
		assert.Error(t, err, "no error on log to stopped logger")
		assert.Nil(t, msg, "message not nil")
		assert.ErrorContains(t, err, "channel is nil", "Wrong error on log to stopped logger")
		assert.Empty(t, ferr.buffer, "data written to fallback")
		assert.Empty(t, out1.buffer, "data written to output")
	})
	t.Run("error_on_nil_logger", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg, err = prep(func() {
				l.Start(0)
				lc.logger = nil
			}, LVL_INFO, LVL_WARN, ferr, out1)
		}, "Panic on write")
		assert.Error(t, err, "no error on log to stopped logger")
		assert.Nil(t, msg, "message not nil")
		assert.ErrorContains(t, err, "logger is nil", "Wrong error on log to stopped logger")
		assert.Empty(t, ferr.buffer, "data written to fallback")
		assert.Empty(t, out1.buffer, "data written to output")
	})
	t.Run("nil_output", func(t *testing.T) {
		ferr := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg, err = prep(nil, LVL_INFO, LVL_WARN, ferr)
			assert.NoError(t, err, "unexpected error")
		}, "Panic on write")
		assert.Empty(t, ferr.buffer, "data written to fallback")
	})
	t.Run("no_outputs", func(t *testing.T) {
		ferr := &FakeWriter{}
		msg, err = prep(nil, LVL_INFO, LVL_WARN, ferr, []outType{}...)
		assert.Empty(t, ferr.buffer, "data written to fallback on write to nil outputs")
	})
	t.Run("2_outputs", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		outBuffer := bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		assert.NotPanics(t, func() {
			msg, err = prep(nil, LVL_INFO, LVL_WARN, ferr, out1, out2)
			assert.NoError(t, err, "unexpected error")
		}, "Panic on write")
		assert.Equal(t, buildTextMessage(outBuffer, msg, l.Context(out1)).Bytes(), out1.buffer)
		assert.Equal(t, buildTextMessage(outBuffer, msg, l.Context(out2)).Bytes(), out2.buffer)
		assert.Empty(t, ferr.buffer, "data written to fallback")
	})
	t.Run("1_error_1_panic_outs", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		outBuffer := bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		assert.NotPanics(t, func() {
			err = nil
			msg, err = prep(nil, LVL_INFO, LVL_WARN, ferr, out1, &ErrorWriter{}, out2, &PanicWriter{})
			assert.NoError(t, err, "unexpected error")
		}, "Panic on write")
		assert.Equal(t, buildTextMessage(outBuffer, msg, l.Context(out1)).Bytes(), out1.buffer)
		assert.Equal(t, buildTextMessage(outBuffer, msg, l.Context(out2)).Bytes(), out2.buffer)
		assert.Contains(t, ferr.String(), panicStr+"\n")
		assert.Contains(t, ferr.String(), errorStr+"\n")
	})
	t.Run("level_masked", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg, err = prep(nil, LVL_INFO, LVL_DEBUG, ferr, out1)
			assert.NoError(t, err, "unexpected error")
		}, "Panic on write")
		assert.Empty(t, ferr.buffer, "data written to fallback")
		assert.Empty(t, out1.buffer, "data written to output on log with level lower than set")
	})
	t.Run("level_logged", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		outBuffer := bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		assert.NotPanics(t, func() {
			msg, err = prep(nil, LVL_WARN, LVL_WARN, ferr, out1)
			assert.NoError(t, err, "unexpected error")
		}, "Panic on write")
		assert.Empty(t, ferr.buffer, "data written to fallback")
		assert.NotEmpty(t, out1.buffer, "no data written to output on log with level higher than set")
		assert.Equal(t, buildTextMessage(outBuffer, msg, l.Context(out1)).Bytes(), out1.buffer)
	})
}

func Test_logClient_Log(t *testing.T) {
	var l *logger
	var lc *logClient
	prep := func(f func(), loglevel LogLevel, msglevel LogLevel, ferr outType, outs ...outType) {
		l = Init()
		l.SetFallback(ferr).ClearOutputs().AddOutputs(outs...)
		lc = l.NewClient("[Testing client name]", LVL_UNKNOWN)
		//lc.minLevel = _LVL_MAX_for_checks_only
		if f == nil {
			l.Start(0)
		} else {
			f()
		}
		l.level = loglevel
		lc.Log(msglevel, testlogstr)
		if f == nil {
			l.StopAndWait()
		}
	}
	t.Run("not_active", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			prep(func() {}, LVL_WARN, LVL_WARN, ferr, out1)
		}, "Panic on write")
		l.StopAndWait()
		assert.NotEmpty(t, ferr.buffer, "err not written to fallback")
		assert.Contains(t, ferr.String(), "not active")
		assert.Empty(t, out1.buffer, "data written to output")
	})
}
func Test_logClient_LogLevels(t *testing.T) {
	l := Init()
	out1 := &FakeWriter{}
	outBuffer := bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
	l.AddOutputs(out1)
	l.SetOutputLevelColor(out1, LevelColorOnBlackMap)
	l.SetOutputLevelPrefix(out1, LevelFullNames, " !delimiter! ")
	l.SetMinLevel(LVL_UNKNOWN)
	lc := l.NewClient("", LVL_UNKNOWN)
	l.Start(0)
	tests := []struct {
		level LogLevel
		fn    func(string) time.Time
	}{
		{LVL_TRACE, lc.LogTrace},
		{LVL_DEBUG, lc.LogDebug},
		{LVL_INFO, lc.LogInfo},
		{LVL_WARN, lc.LogWarn},
		{LVL_ERROR, lc.LogError},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("level_%d", tt.level), func(t *testing.T) {
			out1.Clear()
			l.Start(0)
			msg := makeTextMessage(lc, tt.level, []byte(testlogstr))
			msg.pushed = tt.fn(testlogstr)
			l.StopAndWait()
			assert.Equal(t, buildTextMessage(outBuffer, msg, l.outputs[out1]).Bytes(), out1.buffer, "wrong output")
		})
	}
	t.Run("error_write", func(t *testing.T) {
		out1.Clear()
		outBuffer := bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		l.Start(0)
		msg := makeTextMessage(lc, LVL_ERROR, []byte(testlogstr))
		msg.pushed = lc.LogErr(errors.New(testlogstr))
		l.StopAndWait()
		assert.Equal(t, buildTextMessage(outBuffer, msg, l.outputs[out1]).Bytes(), out1.buffer, "wrong output")
	})
}

func Test_logClient_Lvl(t *testing.T) {
	t.Run("for_255", func(t *testing.T) {
		var lc logClient
		for level := range LogLevel(255) {
			assert.Equal(t, normLevel(level), lc.Lvl(level).curLevel, fmt.Sprintf("Fail on %d", level))
		}
	})
}

func Test_logClient_Write(t *testing.T) {
	ferr := &FakeWriter{}
	out1 := &FakeWriter{}
	l := Init(out1)
	l.SetOutputLevelColor(out1, LevelColorOnBlackMap).SetOutputLevelPrefix(out1, LevelFullNames, " !delimiter! ")
	l.SetFallback(ferr)
	lc := l.NewClient(testlogstr, LVL_UNKNOWN)
	t.Run("nil_message", func(t *testing.T) {
		out1.Clear()
		ferr.Clear()
		l.Start(0)
		lc.Lvl(LVL_UNMASKABLE)
		n, err := lc.Write(nil)
		assert.NoError(t, err)
		assert.Zero(t, n)
		l.StopAndWait()
		assert.Empty(t, ferr.buffer)
		assert.Empty(t, out1.buffer)
	})
	t.Run("full_message", func(t *testing.T) {
		outBuffer := bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		out1.Clear()
		ferr.Clear()
		l.Start(0)
		n, err := fmt.Fprint(lc.Lvl(LVL_UNMASKABLE), testlogstr)
		assert.NoError(t, err)
		assert.Equal(t, n, len(testbytes))
		l.StopAndWait()
		msg := makeTextMessage(lc, LVL_UNMASKABLE, []byte(testlogstr))
		assert.Empty(t, ferr.buffer)
		assert.Equal(t, buildTextMessage(outBuffer, msg, l.outputs[out1]).Bytes(), out1.buffer)
	})
	t.Run("not_active", func(t *testing.T) {
		out1.Clear()
		ferr.Clear()
		n, err := fmt.Fprint(lc.Lvl(LVL_UNMASKABLE), testlogstr)
		assert.ErrorContains(t, err, "not active")
		l.Start(0)
		l.StopAndWait()
		assert.Empty(t, out1.buffer)
		assert.Empty(t, ferr.buffer)
		assert.Zero(t, n)
	})
	t.Run("error_out", func(t *testing.T) {
		outBuffer := bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		out1.Clear()
		ferr.Clear()
		short := "!"
		l.AddOutputs(&ErrorWriter{}, &PanicWriter{})
		l.Start(0)
		n, err := fmt.Fprint(lc.Lvl(LVL_UNMASKABLE), short)
		assert.NoError(t, err)
		assert.Equal(t, n, len(short))
		l.StopAndWait()
		msg := makeTextMessage(lc, LVL_UNMASKABLE, []byte(short))
		assert.Contains(t, ferr.String(), errorStr)
		assert.Contains(t, ferr.String(), panicStr)
		assert.Equal(t, buildTextMessage(outBuffer, msg, l.outputs[out1]).Bytes(), out1.buffer)
	})
}
