package lgr

import (
	"errors"
	"io"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testlogstr = "Test log АБВ こんにちは, 世界!`'\u00e9\"\\\x5A\254\n\a\b\t\f\r\v"
const panicStr = "panic generated in writer"
const errorStr = "error generated in writer"

type PanicWriter struct{}

func (p *PanicWriter) Write(b []byte) (int, error) { panic(panicStr) }

type ErrorWriter struct{}

func (e *ErrorWriter) Write(b []byte) (int, error) { return 0, errors.New(errorStr) }

type FakeWriter struct {
	buffer []byte
}

func (f *FakeWriter) Write(b []byte) (int, error) {
	f.buffer = append(f.buffer, b...)
	return len(b), nil
}
func (f *FakeWriter) String() string { return string(f.buffer) }
func (f *FakeWriter) Clear()         { f.buffer = f.buffer[:0] }

func Test_logger_AddOutputs(t *testing.T) {
	var l *logger
	t.Run("add_1_16", func(t *testing.T) {
		for i := range 16 {
			outs := []outType{}
			for range i + 1 {
				outs = append(outs, &FakeWriter{})
			}
			assert.NotPanics(t, func() {
				l = Init()
				lres := l.AddOutputs(nil, outs...)
				assert.Equal(t, l, lres, "result is another logger")
			})
			assert.Equal(t, len(outs), len(l.outputs), "wrong outputs quantity")
		}
	})
	t.Run("add_3clones_1_16", func(t *testing.T) {
		for i := range 16 {
			outs := []outType{}
			for range i + 1 {
				out := &FakeWriter{}
				outs = append(outs, out, out, out)
			}
			assert.NotPanics(t, func() {
				l = Init()
				lres := l.AddOutputs(nil, outs...)
				assert.Equal(t, l, lres, "result is another logger")
			})
			assert.Equal(t, len(outs)/3, len(l.outputs), "wrong outputs quantity")
		}
	})
	t.Run("add_empties", func(t *testing.T) {
		assert.NotPanics(t, func() {
			l = Init()
			for range 16 {
				lres := l.AddOutputs(nil, []outType{}...)
				assert.Equal(t, l, lres, "result is another logger")
			}
		})
		assert.Empty(t, l.outputs, "outputs exixts")
	})
	t.Run("add_nils", func(t *testing.T) {
		assert.NotPanics(t, func() {
			l = Init()
			for range 16 {
				lres := l.AddOutputs(nil, nil)
				assert.Equal(t, l, lres, "result is another logger")
			}
		})
		assert.Empty(t, l.outputs, "outputs exixts")
	})
}

func TestLogger_ClearOutputs(t *testing.T) {
	tests := []struct {
		name    string // description of this test case
		outputs []outType
	}{
		// TODO: Add test cases.
		{"One", []outType{os.Stdout}},
		{"Two", []outType{io.Discard, os.Stdout}},
		{"Five", []outType{io.Discard, os.Stdout, os.Stderr, io.Discard, io.Discard}},
		{"Empty", []outType{}},
		{"nil", []outType{nil}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := InitWithParams(LVL_TRACE, io.Discard, tt.outputs...)
			lres := l.ClearOutputs()
			assert.Equal(t, 0, len(l.outputs))
			assert.Equal(t, l, lres, "result is another logger")
		})
	}
}

func TestLogger_RemoveOutputs(t *testing.T) {
	tests := []struct {
		wants   int
		name    string // description of this test case
		outputs []outType
		removes []outType
	}{
		// BE CAREFULL!!! In tests os.Stdout == os.Stderr
		{1, "1_rem_nil", []outType{os.Stdout}, nil},
		{1, "1_rem_1nil", []outType{os.Stdout}, []outType{nil}},
		{1, "1_rem_empty", []outType{os.Stdout}, []outType{}},
		{1, "1_rem_unknown", []outType{os.Stdout}, []outType{os.Stdin}},
		{0, "1_rem_1", []outType{os.Stdout}, []outType{os.Stdout}},
		{0, "1_rem_2", []outType{os.Stdout}, []outType{os.Stdout, os.Stdin}},
		{0, "2_rem_2", []outType{os.Stdout, os.Stdin}, []outType{os.Stdout, os.Stdin}},
		{1, "2_rem_1_1unkn", []outType{os.Stdout, os.Stdin}, []outType{os.Stdout, io.Discard}},
		{2, "2_rem_0", []outType{os.Stdout, os.Stdin}, []outType{}},
		{0, "2_rem_3", []outType{os.Stdout, os.Stdin}, []outType{os.Stdout, os.Stdin, io.Discard}},
		{0, "0_rem_1", []outType{}, []outType{os.Stdout}},
		{0, "0_rem_0", []outType{}, []outType{}},
		{0, "0_rem_nil", []outType{}, []outType{nil}},
		{0, "1nil_rem_1", []outType{nil}, []outType{os.Stdout}},
		{0, "1nil_rem_0", []outType{nil}, []outType{}},
		{0, "1nil_rem_nil", []outType{nil}, nil},
		{0, "nil_rem_1", nil, []outType{os.Stdout}},
		{0, "nil_rem_0", nil, []outType{}},
		{0, "nil_rem_nil", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := InitWithParams(LVL_TRACE, io.Discard, tt.outputs...)
			lres := l.RemoveOutputs(tt.removes...)
			assert.Equal(t, tt.wants, len(l.outputs))
			assert.Equal(t, l, lres, "result is another logger")
		})
	}
}

func TestLogger_SetFallback(t *testing.T) {
	tests := []struct {
		name     string // description of this test case
		fallback outType
		wants    outType
	}{
		// TODO: Add test cases.
		{"Stdout", os.Stdout, os.Stdout},
		{"Discard", io.Discard, io.Discard},
		{"Nil->Discard", nil, io.Discard},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := InitWithParams(LVL_TRACE, io.Discard, tt.fallback, nil)
			lres := l.SetFallback(tt.fallback)
			assert.Equal(t, tt.wants, l.fallbck)
			assert.Equal(t, l, lres, "result is another logger")
		})
	}
}

func TestLogger_IsActive(t *testing.T) {
	l := Init()
	rng := 256
	t.Run("one_from_255", func(t *testing.T) {
		for i := range rng {
			l.setState(lgrState(i))
			assert.Equal(t, l.state == STATE_ACTIVE, l.IsActive())
		}
	})
}

func TestLogger_SetLogLevel(t *testing.T) {
	l := Init()
	rng := 255
	t.Run("only_valid_from_255", func(t *testing.T) {
		for i := range rng {
			lres := l.SetMinLevel(logLevel(i))
			res := logLevel(i)
			if res >= _LVL_MAX_FOR_CHECKS_ONLY {
				res = _LVL_MAX_FOR_CHECKS_ONLY - 1
			}
			assert.Equal(t, res, l.level)
			assert.Equal(t, l, lres, "result is another logger")
		}
	})
}

func TestLogger_setState(t *testing.T) {
	l := Init()
	rng := 255
	t.Run("only_valid_from_255", func(t *testing.T) {
		for i := range rng {
			l.setState(lgrState(i))
			res := lgrState(i)
			if res >= _STATE_MAX_FOR_CHECKS_ONLY {
				res = STATE_UNKNOWN
			}
			assert.Equal(t, res, l.state)
		}
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
		lc := l.NewClient("fake", LVL_UNMASKABLE)
		for i := range buffsize {
			lc.Log(LVL_UNMASKABLE, strconv.Itoa(i))
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
	t.Run("explicit_params", func(t *testing.T) {
		var l *logger
		out1 := os.Stdout
		assert.NotPanics(t, func() {
			l = Init(out1)
			l.Start(0)
			l.StopAndWait()
		})
		assert.Equal(t, STATE_STOPPED, l.state, "wrong state")
		assert.Equal(t, DEFAULT_LOG_LEVEL, l.level, "wrong log level")
		assert.Equal(t, 1, len(l.outputs), "wrong outputs count")
		assert.Contains(t, l.outputs, out1, "wrong output")
		assert.Equal(t, os.Stderr, l.fallbck, "wrong fallback")
	})
	t.Run("nil_output", func(t *testing.T) {
		var l *logger
		assert.NotPanics(t, func() {
			l = Init(nil)
			l.Start(0)
			l.StopAndWait()
		})
		assert.Empty(t, l.outputs, "outputs exist")
		assert.Equal(t, DEFAULT_LOG_LEVEL, l.level, "wrong log level")
		assert.Equal(t, os.Stderr, l.fallbck, "wrong fallback")
	})
	t.Run("empty_output", func(t *testing.T) {
		var l *logger
		assert.NotPanics(t, func() {
			l = Init([]outType{}...)
			l.Start(0)
			l.StopAndWait()
		})
		assert.Empty(t, l.outputs, "outputs exist")
		assert.Equal(t, DEFAULT_LOG_LEVEL, l.level, "wrong log level")
		assert.Equal(t, os.Stderr, l.fallbck, "wrong fallback")
	})
}

func TestInitAndStart(t *testing.T) {
	t.Run("explicit_params", func(t *testing.T) {
		var l *logger
		out1 := os.Stdout
		assert.NotPanics(t, func() {
			l = InitAndStart(DEFAULT_BUFF_SIZE, out1)
		})
		assert.Equal(t, DEFAULT_BUFF_SIZE, cap(l.channel))
		assert.Equal(t, STATE_ACTIVE, l.state, "wrong active state")
		assert.Equal(t, DEFAULT_LOG_LEVEL, l.level, "wrong log level")
		assert.Equal(t, 1, len(l.outputs), "wrong outputs count")
		assert.Contains(t, l.outputs, out1, "wrong output")
		assert.Equal(t, os.Stderr, l.fallbck, "wrong fallback")
		assert.NotPanics(t, func() {
			l.StopAndWait()
		})
		assert.Equal(t, STATE_STOPPED, l.state, "wrong stopped state")
	})
	t.Run("min_params", func(t *testing.T) {
		var l *logger
		assert.NotPanics(t, func() {
			l = InitAndStart(-1)
		})
		assert.Equal(t, DEFAULT_BUFF_SIZE, cap(l.channel))
		assert.Equal(t, STATE_ACTIVE, l.state, "wrong active state")
		assert.Equal(t, DEFAULT_LOG_LEVEL, l.level, "wrong log level")
		assert.Empty(t, l.outputs, "outputs exist")
		assert.Equal(t, os.Stderr, l.fallbck, "wrong fallback")
		assert.NotPanics(t, func() {
			l.StopAndWait()
		})
		assert.Equal(t, STATE_STOPPED, l.state, "wrong stopped state")
	})
}
