package lgr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_logClient_LogE(t *testing.T) {
	var err error
	var l *logger
	var lc *logClient
	prep := func(f func(), loglevel logLevel, msglevel logLevel, ferr outType, outs ...outType) error {
		err = nil
		l = Init()
		l.SetFallback(ferr).ClearOutputs().AddOutputs(nil, outs...)
		lc = l.NewClient("Testing", LVL_UNKNOWN)
		lc.maxLevel = _LVL_MAX_FOR_CHECKS_ONLY
		if f == nil {
			l.Start(0)
		} else {
			f()
		}
		l.level = loglevel
		err := lc.LogE(msglevel, testlogstr)
		if f == nil {
			l.StopAndWait()
		}
		return err
	}
	t.Run("not_active", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			err = prep(func() {}, LVL_INFO, LVL_WARN, ferr)
		}, "Panic on write")
		l.StopAndWait()
		assert.Error(t, err, "no error on log to stopped logger")
		assert.Contains(t, err.Error(), "not active", "Wrong error on log to stopped logger")
		assert.Empty(t, ferr, "data written to fallback")
		assert.Empty(t, out1, "data written to output")
	})
	t.Run("panic_on_closed_channel", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			err = prep(func() {
				l.Start(0)
				close(l.channel)
			}, LVL_INFO, LVL_WARN, ferr)
		}, "Panic on write")
		l.channel = make(chan logMessage)
		l.StopAndWait()
		assert.Error(t, err, "no error on log to stopped logger")
		assert.Contains(t, err.Error(), "panic", "Wrong error on log to stopped logger")
		assert.Empty(t, ferr, "data written to fallback")
		assert.Empty(t, out1, "data written to output")
	})
	t.Run("unrecovered_panic", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		assert.Panics(t, func() {
			err = prep(nil, _LVL_MAX_FOR_CHECKS_ONLY, LVL_DEBUG, ferr, out1, out2)
		}, "No panic on unmasked panic")
		assert.NoError(t, err, "unexpected error")
		assert.Empty(t, out1)
		assert.Empty(t, out2)
		assert.Empty(t, ferr)
	})
	t.Run("error_on_nil_channel", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			err = prep(func() {
				l.state = STATE_ACTIVE
			}, LVL_INFO, LVL_WARN, ferr)
		}, "Panic on write")
		assert.Error(t, err, "no error on log to stopped logger")
		assert.Contains(t, err.Error(), "channel is nil", "Wrong error on log to stopped logger")
		assert.Empty(t, ferr, "data written to fallback")
		assert.Empty(t, out1, "data written to output")
	})
	t.Run("error_on_nil_logger", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			err = prep(func() {
				l.Start(0)
				lc.logger = nil
			}, LVL_INFO, LVL_WARN, ferr)
		}, "Panic on write")
		assert.Error(t, err, "no error on log to stopped logger")
		assert.Contains(t, err.Error(), "logger is nil", "Wrong error on log to stopped logger")
		assert.Empty(t, ferr, "data written to fallback")
		assert.Empty(t, out1, "data written to output")
	})
	t.Run("nil_output", func(t *testing.T) {
		ferr := &FakeWriter{}
		assert.NotPanics(t, func() {
			assert.NoError(t,
				prep(nil, LVL_INFO, LVL_WARN, ferr),
				"unexpected error")
		}, "Panic on write")
		assert.Empty(t, ferr, "data written to fallback")
	})
	t.Run("no_outputs", func(t *testing.T) {
		ferr := &FakeWriter{}
		prep(nil, LVL_INFO, LVL_WARN, ferr, []outType{}...)
		assert.Empty(t, ferr, "data written to fallback on write to nil outputs")
	})
	t.Run("2_outputs", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		assert.NotPanics(t, func() {
			assert.NoError(t,
				prep(nil, LVL_INFO, LVL_WARN, ferr, out1, out2),
				"unexpected error")
		}, "Panic on write")
		assert.Equal(t, testlogstr+"\n", out1.String())
		assert.Equal(t, testlogstr+"\n", out2.String())
		assert.Empty(t, ferr, "data written to fallback")
	})
	t.Run("1_error_1_panic_outs", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		assert.NotPanics(t, func() {
			assert.NoError(t,
				prep(nil, LVL_INFO, LVL_WARN, ferr, out1, &ErrorWriter{}, out2, &PanicWriter{}),
				"unexpected error")
		}, "Panic on write")
		assert.Equal(t, testlogstr+"\n", out1.String())
		assert.Equal(t, testlogstr+"\n", out2.String())
		assert.Contains(t, ferr.String(), panicStr+"\n")
		assert.Contains(t, ferr.String(), errorStr+"\n")
	})
	t.Run("level_masked", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			assert.NoError(t,
				prep(nil, LVL_INFO, LVL_DEBUG, ferr, out1),
				"unexpected error")
		}, "Panic on write")
		assert.Empty(t, ferr, "data written to fallback")
		assert.Empty(t, out1, "data written to output on log with level lower than set")
	})
	t.Run("level_logged", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			assert.NoError(t,
				prep(nil, LVL_WARN, LVL_WARN, ferr, out1),
				"unexpected error")
		}, "Panic on write")
		assert.Empty(t, ferr, "data written to fallback")
		assert.NotEmpty(t, out1, "no data written to output on log with level higher than set")
		assert.Equal(t, testlogstr+"\n", out1.String(), "wrong data written to output on log with level higher than set")
	})
}

func Test_logClient_Log(t *testing.T) {
	// To be added... somewhen
}
