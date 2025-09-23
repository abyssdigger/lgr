package lgr

import (
	"io"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger_ClearOutputs(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		outputs []OutType
	}{
		// TODO: Add test cases.
		{"One", []OutType{os.Stdout}},
		{"Two", []OutType{io.Discard, os.Stdout}},
		{"Five", []OutType{io.Discard, os.Stdout, os.Stderr, io.Discard, io.Discard}},
		{"Empty", []OutType{}},
		{"nil", []OutType{nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := InitWithParams(LVL_TRACE, io.Discard, tt.outputs...)
			l.ClearOutputs()
			assert.Equal(t, 0, len(l.outputs))
		})
	}
}

func TestLogger_RemoveOutputs(t *testing.T) {
	tests := []struct {
		wants   int
		name    string // description of this test case
		outputs []OutType
		removes []OutType
	}{
		// BE CAREFULL!!! In tests os.Stdout == os.Stderr
		{1, "1_rem_nil", []OutType{os.Stdout}, nil},
		{1, "1_rem_1nil", []OutType{os.Stdout}, []OutType{nil}},
		{1, "1_rem_empty", []OutType{os.Stdout}, []OutType{}},
		{1, "1_rem_unknown", []OutType{os.Stdout}, []OutType{os.Stdin}},
		{0, "1_rem_1", []OutType{os.Stdout}, []OutType{os.Stdout}},
		{0, "1_rem_2", []OutType{os.Stdout}, []OutType{os.Stdout, os.Stdin}},
		{0, "2_rem_2", []OutType{os.Stdout, os.Stdin}, []OutType{os.Stdout, os.Stdin}},
		{1, "2_rem_1_1unkn", []OutType{os.Stdout, os.Stdin}, []OutType{os.Stdout, io.Discard}},
		{2, "2_rem_0", []OutType{os.Stdout, os.Stdin}, []OutType{}},
		{0, "2_rem_3", []OutType{os.Stdout, os.Stdin}, []OutType{os.Stdout, os.Stdin, io.Discard}},
		{0, "0_rem_1", []OutType{}, []OutType{os.Stdout}},
		{0, "0_rem_0", []OutType{}, []OutType{}},
		{0, "0_rem_nil", []OutType{}, []OutType{nil}},
		{0, "1nil_rem_1", []OutType{nil}, []OutType{os.Stdout}},
		{0, "1nil_rem_0", []OutType{nil}, []OutType{}},
		{0, "1nil_rem_nil", []OutType{nil}, nil},
		{0, "nil_rem_1", nil, []OutType{os.Stdout}},
		{0, "nil_rem_0", nil, []OutType{}},
		{0, "nil_rem_nil", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := InitWithParams(LVL_TRACE, io.Discard, tt.outputs...)
			l.RemoveOutputs(tt.removes...)
			assert.Equal(t, tt.wants, len(l.outputs))
		})
	}
}

func TestLogger_SetFallback(t *testing.T) {
	tests := []struct {
		name     string // description of this test case
		fallback OutType
		wants    OutType
	}{
		// TODO: Add test cases.
		{"Stdout", os.Stdout, os.Stdout},
		{"Discard", io.Discard, io.Discard},
		{"Nil->Discard", nil, io.Discard},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := InitWithParams(LVL_TRACE, io.Discard, tt.fallback, nil)
			l.SetFallback(tt.fallback)
			assert.Equal(t, tt.wants, l.fallbck)
		})
	}
}

func TestLogger_IsActive(t *testing.T) {
	l := Init()
	rng := 256
	t.Run("one_from_255", func(t *testing.T) {
		for i := range rng {
			l.setState(LoggerState(i))
			assert.Equal(t, l.state == STATE_ACTIVE, l.IsActive())
		}
	})
}

func TestLogger_SetLogLevel(t *testing.T) {
	l := Init()
	rng := 255
	for i := range rng {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			l.SetLogLevel(LogLevel(i))
			res := LogLevel(i)
			if res >= _LVL_MAX_FOR_CHECKS_ONLY {
				res = _LVL_MAX_FOR_CHECKS_ONLY - 1
			}
			assert.Equal(t, res, l.level)
		})
	}
}

func TestLogger_setState(t *testing.T) {
	l := Init()
	rng := 255
	for i := range rng {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			l.setState(LoggerState(i))
			res := LoggerState(i)
			if res >= _STATE_MAX_FOR_CHECKS_ONLY {
				res = STATE_UNKNOWN
			}
			assert.Equal(t, res, l.state)
		})
	}
}

func TestLogger_LogE(t *testing.T) {
	logstr := "Test log message"
	t.Run("nil_output", func(t *testing.T) {
		ferror := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferror)
		l.Start(0)
		err := l.LogE(LVL_INFO, logstr)
		assert.Nil(t, err, "error on log to nil outputs")
		l.StopAndWait()
		assert.Empty(t, ferror, "data written to fallback")
	})
	t.Run("not_active", func(t *testing.T) {
		ferror := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferror)
		err := l.LogE(LVL_INFO, logstr)
		assert.NotNil(t, err, "no error on log to stopped logger")
		assert.Contains(t, err.Error(), "not active", "Wrong error on log to stopped logger")
		assert.Empty(t, ferror, "data written to fallback")
	})
	t.Run("level_masked", func(t *testing.T) {
		ferror := &FakeWriter{}
		out_1 := &FakeWriter{}
		l := InitWithParams(LVL_WARN, ferror, out_1)
		l.Start(0)
		err := l.LogE(LVL_INFO, logstr)
		assert.Nil(t, err, "error on log with level lower than set")
		l.StopAndWait()
		assert.Empty(t, ferror, "data written to fallback on log with level lower than set")
		assert.Empty(t, out_1, "data written to output on log with level lower than set")
	})
	t.Run("level_logged", func(t *testing.T) {
		ferror := &FakeWriter{}
		out_1 := &FakeWriter{}
		l := InitWithParams(LVL_WARN, ferror, out_1)
		l.Start(0)
		err := l.LogE(LVL_WARN, logstr)
		assert.Nil(t, err, "error on log with level higher than set")
		l.StopAndWait()
		assert.Empty(t, ferror, "data written to fallback on log with level higher than set")
		assert.NotEmpty(t, out_1, "no data written to output on log with level higher than set")
		assert.Equal(t, logstr+"\n", out_1.String(), "wrong data written to output on log with level higher than set")
	})
	t.Run("panic_on_closed_channel", func(t *testing.T) {
		ferror := &FakeWriter{}
		out_1 := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferror, out_1)
		l.Start(0)
		close(l.channel)
		err := l.LogE(LVL_TRACE, logstr)
		assert.NotNil(t, err, "no error on log with closed channel")
		assert.Contains(t, err.Error(), "panic", "wrong error on log with closed channel")
		l.channel = make(chan logMessage)
		l.StopAndWait()
		assert.Empty(t, ferror, "data written to fallback on log with closed channel")
		assert.Empty(t, out_1, "data written to output on log with closed channel")
	})
	t.Run("error_on_nil_channel", func(t *testing.T) {
		ferror := &FakeWriter{}
		out_1 := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferror, out_1)
		l.state = STATE_ACTIVE
		err := l.LogE(LVL_TRACE, logstr)
		assert.NotNil(t, err, "no error on log with nil channel")
		assert.Contains(t, err.Error(), "is nil", "wrong error on log with nil channel")
		assert.Empty(t, ferror, "data written to fallback on log with nil channel")
		assert.Empty(t, out_1, "data written to output on log with nil channel")
	})
	t.Run("panic_on_forbidden_level", func(t *testing.T) {
		ferror := &FakeWriter{}
		out_1 := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferror, out_1)
		l.Start(0)
		l.level = _LVL_MAX_FOR_CHECKS_ONLY
		// simulate wrong level set by SetLogLevel
		// to test panic in Log_()
		err := l.LogE(_LVL_MAX_FOR_CHECKS_ONLY, logstr)
		assert.NotNil(t, err, "no error on log with forbidden level")
		assert.Contains(t, err.Error(), "panic", "wrong error on log with forbidden level")
		l.StopAndWait()
		assert.Empty(t, ferror, "data written to fallback on log with forbidden level")
		assert.Empty(t, out_1, "data written to output on log with forbidden level")
	})
	t.Run("multiple_outputs", func(t *testing.T) {
		ferror := &FakeWriter{}
		out_1 := &FakeWriter{}
		out_2 := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferror, out_1, out_2)
		l.Start(0)
		err := l.LogE(LVL_INFO, logstr)
		assert.Nil(t, err, "error on log with level higher than set")
		l.StopAndWait()
		assert.Empty(t, ferror, "data written to fallback on log with level higher than set")
		assert.NotEmpty(t, out_1, "no data written to output1 on log with level higher than set")
		assert.Equal(t, logstr+"\n", out_1.String(), "wrong data written to output1 on log with level higher than set")
		assert.NotEmpty(t, out_2, "no data written to output2 on log with level higher than set")
		assert.Equal(t, logstr+"\n", out_2.String(), "wrong data written to output2 on log with level higher than set")
	})
}

func TestLogger_Log(t *testing.T) {
	logstr := "Test log message"
	t.Run("no_outputs", func(t *testing.T) {
		ferr := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferr)
		assert.NotPanics(t, func() {
			l.Start(0)
			l.Log(LVL_INFO, logstr)
			l.StopAndWait()
		}, "Panic on write to nil fallback")
		assert.Empty(t, ferr.String(), "data written to fallback on write to nil outputs")
	})
	t.Run("2_outputs", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferr, out1, out2)
		assert.NotPanics(t, func() {
			l.Start(0)
			l.Log(LVL_INFO, logstr)
			l.StopAndWait()
		}, "Panic on write to 2 outputs")
		assert.Equal(t, logstr+"\n", out1.String())
		assert.Equal(t, logstr+"\n", out2.String())
		assert.Empty(t, ferr.String(), "data written to fallback on write to 2 outputs")
	})
	t.Run("1_error_1_panic", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferr, out1, &ErrorWriter{}, out2, &PanicWriter{})
		assert.NotPanics(t, func() {
			l.Start(0)
			l.Log(LVL_INFO, logstr)
			l.StopAndWait()
		}, "Panic on write to nil fallback")
		assert.Equal(t, logstr+"\n", out1.String())
		assert.Equal(t, logstr+"\n", out2.String())
		assert.Contains(t, ferr.String(), panicStr+"\n")
		assert.Contains(t, ferr.String(), errorStr+"\n")
	})
	t.Run("panic_in_LogE", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		l := InitWithParams(LVL_TRACE, ferr, out1, out2)
		l.level = _LVL_MAX_FOR_CHECKS_ONLY
		// simulate wrong level set by SetLogLevel
		// to test panic in Log_()
		assert.NotPanics(t, func() {
			l.Start(0)
			l.Log(_LVL_MAX_FOR_CHECKS_ONLY, logstr)
			l.StopAndWait()
		}, "Panic in LogE")
		assert.Empty(t, out1.String())
		assert.Empty(t, out2.String())
		assert.Contains(t, ferr.String(), "panic")
	})
}

func TestLogger_Start(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on normal start")
		assert.Equal(t, STATE_ACTIVE, l.state, "wrong state after normal start")
		l.StopAndWait()
	})
	t.Run("double", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on first start")
		err = l.Start(0)
		assert.NotNil(t, err, "no error on double start")
		assert.Contains(t, err.Error(), "allready started", "wrong error on double start")
		assert.Equal(t, STATE_ACTIVE, l.state, "wrong state after double start")
		l.StopAndWait()
	})
	t.Run("negative_buffsize", func(t *testing.T) {
		l := Init()
		err := l.Start(-10)
		assert.Nil(t, err, "error with negative buffsize")
		assert.Equal(t, cap(l.channel), DEFAULT_BUFF_SIZE, "wrong channel capacity")
		l.StopAndWait()
	})
}
func TestLogger_Stop(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on start")
		l.Stop()
		assert.Equal(t, STATE_STOPPING, l.state, "wrong state after stop")
	})
	t.Run("double", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on start")
		l.Stop()
		assert.NotPanics(t, func() { l.Stop() })
	})
	t.Run("without_start", func(t *testing.T) {
		l := Init()
		l.Stop()
		assert.Equal(t, STATE_STOPPED, l.state, "wrong state after stop without start")
	})
}

func TestLogger_Wait(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on start")
		l.Stop()
		assert.NotPanics(t, func() { l.Wait() }, "Panic on wait after stop")
		assert.Equal(t, STATE_STOPPED, l.state, "wrong state after wait")
	})
	t.Run("without_start", func(t *testing.T) {
		l := Init()
		assert.NotPanics(t, func() { l.Wait() }, "Panic on wait without start")
		assert.Equal(t, STATE_STOPPED, l.state, "wrong state after wait without start")
	})
	t.Run("double_wait", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on start")
		l.Stop()
		assert.NotPanics(t, func() { l.Wait() }, "Panic on first wait")
		assert.NotPanics(t, func() { l.Wait() }, "Panic on second wait")
		assert.Equal(t, STATE_STOPPED, l.state, "wrong state after double wait")
	})
	t.Run("wait_long_buff", func(t *testing.T) {
		buffsize := 8192
		l := InitWithParams(DEFAULT_LOG_LEVEL, nil)
		err := l.Start(buffsize)
		assert.Nil(t, err, "error on start")
		for i := range buffsize {
			l.Log(LVL_UNMASKABLE, strconv.Itoa(i))
		}
		l.Stop()
		l.Wait()
		assert.Empty(t, l.channel, "channel is not empty")
	})
}

func TestLogger_StopAndWait(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on start")
		assert.NotPanics(t, func() { l.StopAndWait() }, "Panic on StopAndWait")
		assert.Equal(t, STATE_STOPPED, l.state, "wrong state after StopAndWait")
	})
	t.Run("without_start", func(t *testing.T) {
		l := Init()
		assert.NotPanics(t, func() { l.StopAndWait() }, "Panic on StopAndWait without start")
		assert.Equal(t, STATE_STOPPED, l.state, "wrong state after StopAndWait without start")
	})
	t.Run("double_StopAndWait", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on start")
		assert.NotPanics(t, func() { l.StopAndWait() }, "Panic on first StopAndWait")
		assert.NotPanics(t, func() { l.StopAndWait() }, "Panic on second StopAndWait")
		assert.Equal(t, STATE_STOPPED, l.state, "wrong state after double StopAndWait")
	})
}

func TestInitWithParams(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		fallbck := io.Discard
		out1 := io.Discard
		out2 := os.Stdout
		level := LVL_DEBUG
		l := InitWithParams(level, fallbck, out1, out2)
		assert.Equal(t, STATE_STOPPED, l.state, "wrong state after init")
		assert.Equal(t, level, l.level, "wrong level after init")
		assert.Equal(t, 2, len(l.outputs), "wrong outputs count after init")
		assert.Contains(t, l.outputs, out1, "missing output1 after init")
		assert.Contains(t, l.outputs, out2, "missing output2 after init")
		assert.Equal(t, fallbck, l.fallbck, "wrong fallback after init")
	})
	t.Run("corrections", func(t *testing.T) {
		out1 := os.Stdout
		out2 := io.Discard
		level := _LVL_MAX_FOR_CHECKS_ONLY + 10
		l := InitWithParams(level, nil, nil, out1, nil, out2)
		assert.Equal(t, STATE_STOPPED, l.state, "wrong state after init")
		assert.Equal(t, _LVL_MAX_FOR_CHECKS_ONLY-1, l.level, "wrong level after init")
		assert.Equal(t, 2, len(l.outputs), "wrong outputs count after init")
		assert.Contains(t, l.outputs, out1, "missing output1 after init")
		assert.Contains(t, l.outputs, out2, "missing output2 after init")
		assert.NotContains(t, l.outputs, nil, "nil output after init")
		assert.Equal(t, io.Discard, l.fallbck, "wrong fallback after init")
	})
}

func TestInit(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		l := Init()
		assert.Equal(t, STATE_STOPPED, l.state, "wrong state after init")
		assert.Equal(t, DEFAULT_LOG_LEVEL, l.level, "wrong level after init")
		assert.Equal(t, 1, len(l.outputs), "wrong outputs count after init")
		assert.Contains(t, l.outputs, os.Stdout, "missing default output after init")
		assert.Equal(t, os.Stderr, l.fallbck, "wrong fallback after init")
	})
}
