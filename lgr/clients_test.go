package lgr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_logClient_LogE(t *testing.T) {
	var msg *logMessage
	var err error
	var l *logger
	var lc *logClient
	prep := func(f func(), loglevel LogLevel, msglevel LogLevel, ferr outType, outs ...outType) (*logMessage, error) {
		err = nil
		l = Init()
		l.SetFallback(ferr).ClearOutputs().AddOutputs(outs...)
		lc = l.NewClient("[Testing client name]", LVL_UNKNOWN)
		if f == nil {
			l.Start(0)
		} else {
			f()
		}
		l.level = loglevel
		msg, err := lc.Log_with_err(msglevel, testlogstr)
		if f == nil {
			l.StopAndWait()
		}
		return msg, err
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
		assert.Contains(t, err.Error(), "panic", "Wrong error on log to stopped logger")
		assert.Empty(t, ferr, "data written to fallback")
		assert.Empty(t, out1, "data written to output")
	})
	t.Run("not_active", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg, err = prep(func() {}, LVL_WARN, LVL_WARN, ferr, out1)
		}, "Panic on write")
		l.StopAndWait()
		assert.Error(t, err, "no error on log to stopped logger")
		assert.Nil(t, msg, "message not nil")
		assert.Contains(t, err.Error(), "not active", "Wrong error on log to stopped logger")
		assert.Empty(t, ferr, "data written to fallback")
		assert.Empty(t, out1, "data written to output")
	})
	t.Run("unrecovered_panic", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		assert.Panics(t, func() {
			msg, err = prep(nil, _LVL_MAX_for_checks_only, LVL_DEBUG, ferr, out1, out2)
		}, "No panic on unmasked panic")
		assert.NoError(t, err, "unexpected error")
		assert.Nil(t, msg, "message not nil")
		assert.Empty(t, out1)
		assert.Empty(t, out2)
		assert.Empty(t, ferr)
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
		assert.Contains(t, err.Error(), "channel is nil", "Wrong error on log to stopped logger")
		assert.Empty(t, ferr, "data written to fallback")
		assert.Empty(t, out1, "data written to output")
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
		assert.Contains(t, err.Error(), "logger is nil", "Wrong error on log to stopped logger")
		assert.Empty(t, ferr, "data written to fallback")
		assert.Empty(t, out1, "data written to output")
	})
	t.Run("nil_output", func(t *testing.T) {
		ferr := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg, err = prep(nil, LVL_INFO, LVL_WARN, ferr)
			assert.NoError(t, err, "unexpected error")
		}, "Panic on write")
		assert.Empty(t, ferr, "data written to fallback")
	})
	t.Run("no_outputs", func(t *testing.T) {
		ferr := &FakeWriter{}
		msg, err = prep(nil, LVL_INFO, LVL_WARN, ferr, []outType{}...)
		assert.Empty(t, ferr, "data written to fallback on write to nil outputs")
	})
	t.Run("2_outputs", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg, err = prep(nil, LVL_INFO, LVL_WARN, ferr, out1, out2)
			assert.NoError(t, err, "unexpected error")
		}, "Panic on write")
		assert.Equal(t, buildTextMessage(msg, l.Context(out1)).String(), out1.String())
		assert.Equal(t, buildTextMessage(msg, l.Context(out2)).String(), out2.String())
		assert.Empty(t, ferr, "data written to fallback")
	})
	t.Run("1_error_1_panic_outs", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg, err = prep(nil, LVL_INFO, LVL_WARN, ferr, out1, &ErrorWriter{}, out2, &PanicWriter{})
			assert.NoError(t, err, "unexpected error")
		}, "Panic on write")
		assert.Equal(t, buildTextMessage(msg, l.Context(out1)).String(), out1.String())
		assert.Equal(t, buildTextMessage(msg, l.Context(out2)).String(), out2.String())
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
		assert.Empty(t, ferr, "data written to fallback")
		assert.Empty(t, out1, "data written to output on log with level lower than set")
	})
	t.Run("level_logged", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg, err = prep(nil, LVL_WARN, LVL_WARN, ferr, out1)
			assert.NoError(t, err, "unexpected error")
		}, "Panic on write")
		assert.Empty(t, ferr, "data written to fallback")
		assert.NotEmpty(t, out1, "no data written to output on log with level higher than set")
		assert.Equal(t, buildTextMessage(msg, l.Context(out1)).String(), out1.String())
	})
}

func Test_logClient_Log(t *testing.T) {
	var msg *logMessage
	var l *logger
	var lc *logClient
	prep := func(f func(), loglevel LogLevel, msglevel LogLevel, ferr outType, outs ...outType) *logMessage {
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
		msg := lc.Log(msglevel, testlogstr)
		if f == nil {
			l.StopAndWait()
		}
		return msg
	}
	t.Run("not_active", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg = prep(func() {}, LVL_WARN, LVL_WARN, ferr, out1)
		}, "Panic on write")
		l.StopAndWait()
		assert.Nil(t, msg, "message not nil")
		assert.NotEmpty(t, ferr, "err not written to fallback")
		assert.Contains(t, ferr.String(), "not active")
		assert.Empty(t, out1, "data written to output")
	})
}

func Test_logger_NewClient(t *testing.T) {
	var l *logger
	var lc *logClient
	prep := func(lvl LogLevel, name string) {
		l = Init()
		assert.NotPanics(t, func() {
			lc = l.NewClient(name, lvl)
		}, "Panic new client")
		assert.NotNil(t, lc, "nil client")
		assert.Equal(t, []byte(name), lc.name, "wrong name")
	}
	t.Run("new_client_correct_level", func(t *testing.T) {
		for lvl := range LogLevel(255) {
			prep(lvl, "[Testing client name]")
			assert.Equal(t, normLevel(lvl), lc.minLevel, "wrong max level")
		}
	})
}
