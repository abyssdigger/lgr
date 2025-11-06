package lgr

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const testlogstr = "Test log АБВ こんにちは, 世界`'\u00e9\"\\\x5A\254\n\a\b\t\f\r\vи други глупости!"
const panicStr = "panic generated in writer"
const errorStr = "error generated in writer"

// Returns the outContext for a given output writer by pointer so it can be changed directly
// outside logger functions (such changes can have thread-unsafe side effects in queue
// proceedeng so use with care).
//
// Use for test purposes only.
func (l *Logger) getContext(output OutType) *outContext {
	return l.outputs[output]
}

type PanicWriter struct{}

func (p *PanicWriter) Write(b []byte) (int, error) { panic(panicStr) }

type NilPanicWriter struct{}

func (p *NilPanicWriter) Write(b []byte) (int, error) { panic(&runtime.PanicNilError{}) }

// &runtime.PanicNilError{} instead of nil to prevent VSC problem "panic with nil value"

type ZeroPanicWriter struct{}

func (p *ZeroPanicWriter) Write(b []byte) (int, error) { panic(0) }

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

var globalFerr = &FakeWriter{}
var globalOutp = &FakeWriter{}

func Test_JustVisualTest(t *testing.T) {
	var logger = InitWithParams(LVL_UNKNOWN, os.Stderr, nil) //...Default()
	var alter1 = *os.Stdout
	var alter2 = *os.Stdout
	res1 := &alter1
	res2 := &alter2
	outs := [...]io.Writer{nil, res1, os.Stdout, res2, os.Stderr}
	for i := 1; i <= len(outs); i++ {
		t.Run("Stage #"+strconv.Itoa(i), func(t *testing.T) {
			logger.Start(32)
			logger.AddOutputs(outs[i-1])
			logger.SetOutputLevelPrefix(os.Stderr, LevelShortNames, "\t")
			logger.SetOutputLevelPrefix(res1, LevelFullNames, " --> ")
			logger.SetOutputLevelColor(res1, LevelColorOnBlackMap)
			logger.SetOutputLevelPrefix(res2, LevelShortNames, "|")
			logger.SetOutputLevelColor(os.Stdout, LevelColorOnBlackMap)
			logger.SetOutputTimeFormat(res1, "2006-01-02 15:04:05", " ")
			logger.SetOutputTimeFormat(os.Stderr, "2006-01-02 15:04:05", " ")
			logger.ShowOutputLevelCode(os.Stderr)
			lclient1 := logger.NewClientWithLevel("<Όνομα δοκιμής TST>", LVL_UNKNOWN)
			lclient2 := logger.NewClientWithLevel("^chinese 你好 друг^", LVL_UNMASKABLE)
			for j := range LogLevel(LVL_UNMASKABLE + 10) {
				_, err := lclient1.Log_with_err(j, "LOG! #"+fmt.Sprint(j+1))
				if err != nil {
					fmt.Printf("Error on lvl%2d: %s\n", j, err.Error())
				} else {
					_, err := lclient2.Log_with_err(j, "ЛОГ? №"+fmt.Sprint(j+1))
					if err != nil {
						fmt.Println("Error1:", err)
						assert.NoError(t, err, "unexpected error '"+err.Error()+"'")
					}
				}
			}
			fmt.Println("Stopping logger...")
			logger.StopAndWait()
			logger.ClearOutputs()
			fmt.Println("*** FINITA LA COMEDIA #", i, "***")
			time.Sleep(100 * time.Millisecond)
		})
	}
}

func Test_Logger_AddOutputs(t *testing.T) {
	var l *Logger
	t.Run("add_1_16", func(t *testing.T) {
		for i := range 16 {
			outs := []OutType{}
			for range i + 1 {
				outs = append(outs, &FakeWriter{})
			}
			assert.NotPanics(t, func() {
				l = Init()
				lres := l.AddOutputs(outs...)
				assert.Equal(t, l, lres, "result is another logger")
			})
			assert.Equal(t, len(outs), len(l.outputs), "wrong outputs quantity")
		}
	})
	t.Run("add_3clones_1_16", func(t *testing.T) {
		for i := range 16 {
			outs := []OutType{}
			for range i + 1 {
				out := &FakeWriter{}
				outs = append(outs, out, out, out)
			}
			assert.NotPanics(t, func() {
				l = Init()
				lres := l.AddOutputs(outs...)
				assert.Equal(t, l, lres, "result is another logger")
			})
			assert.Equal(t, len(outs)/3, len(l.outputs), "wrong outputs quantity")
		}
	})
	t.Run("add_empties", func(t *testing.T) {
		assert.NotPanics(t, func() {
			l = Init()
			for range 16 {
				lres := l.AddOutputs([]OutType{}...)
				assert.Equal(t, l, lres, "result is another logger")
			}
		})
		assert.Empty(t, l.outputs, "outputs exixts")
	})
	t.Run("add_nils", func(t *testing.T) {
		assert.NotPanics(t, func() {
			l = Init()
			for range 16 {
				lres := l.AddOutputs(nil)
				assert.Equal(t, l, lres, "result is another logger")
			}
		})
		assert.Empty(t, l.outputs, "outputs exixts")
	})
}

func Test_Logger_ClearOutputs(t *testing.T) {
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
			lres := l.ClearOutputs()
			assert.Equal(t, 0, len(l.outputs))
			assert.Equal(t, l, lres, "result is another logger")
		})
	}
}

func Test_Logger_RemoveOutputs(t *testing.T) {
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
			lres := l.RemoveOutputs(tt.removes...)
			assert.Equal(t, tt.wants, len(l.outputs))
			assert.Equal(t, l, lres, "result is another logger")
		})
	}
}

func Test_Logger_SetFallback(t *testing.T) {
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
			lres := l.SetFallback(tt.fallback)
			assert.Equal(t, tt.wants, l.fallbck)
			assert.Equal(t, l, lres, "result is another logger")
		})
	}
}

func Test_Logger_IsActive(t *testing.T) {
	l := Init()
	rng := 256
	t.Run("one_from_255", func(t *testing.T) {
		for i := range rng {
			l.setState(lgrState(i))
			assert.Equal(t, l.state == _STATE_ACTIVE, l.IsActive())
		}
	})
}

func Test_Logger_SetLogLevel(t *testing.T) {
	l := Init()
	rng := 255
	t.Run("only_valid_from_255", func(t *testing.T) {
		for i := range rng {
			lres := l.SetMinLevel(LogLevel(i))
			res := LogLevel(i)
			if res >= _LVL_MAX_for_checks_only {
				res = LVL_UNKNOWN
			}
			assert.Equal(t, res, l.level)
			assert.Equal(t, l, lres, "result is another logger")
		}
	})
}

func Test_Logger_setState(t *testing.T) {
	l := Init()
	rng := 255
	t.Run("only_valid_from_255", func(t *testing.T) {
		for i := range rng {
			l.setState(lgrState(i))
			res := lgrState(i)
			if res >= _STATE_MAX_for_checks_only {
				res = _STATE_UNKNOWN
			}
			assert.Equal(t, res, l.state)
		}
	})
}

func Test_Logger_Start(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on normal start")
		assert.Equal(t, _STATE_ACTIVE, l.state, "wrong state after normal start")
		l.StopAndWait()
	})
	t.Run("double", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on first start")
		err = l.Start(0)
		assert.NotNil(t, err, "no error on double start")
		assert.EqualError(t, err, _ERROR_MESSAGE_LOGGER_STARTED, "wrong error on double start")
		assert.Equal(t, _STATE_ACTIVE, l.state, "wrong state after double start")
		l.StopAndWait()
	})
	t.Run("negative_buffsize", func(t *testing.T) {
		l := Init()
		err := l.Start(-10)
		assert.Nil(t, err, "error with negative buffsize")
		assert.Equal(t, cap(l.channel), DEFAULT_MSG_BUFF, "wrong channel capacity")
		l.StopAndWait()
	})
}
func Test_Logger_Stop(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on start")
		l.Stop()
		assert.Equal(t, _STATE_STOPPING, l.state, "wrong state after stop")
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
		assert.Equal(t, _STATE_STOPPED, l.state, "wrong state after stop without start")
	})
}

func Test_Logger_Wait(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on start")
		l.Stop()
		assert.NotPanics(t, func() { l.Wait() }, "Panic on wait after stop")
		assert.Equal(t, _STATE_STOPPED, l.state, "wrong state after wait")
	})
	t.Run("without_start", func(t *testing.T) {
		l := Init()
		assert.NotPanics(t, func() { l.Wait() }, "Panic on wait without start")
		assert.Equal(t, _STATE_STOPPED, l.state, "wrong state after wait without start")
	})
	t.Run("double_wait", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on start")
		l.Stop()
		assert.NotPanics(t, func() { l.Wait() }, "Panic on first wait")
		assert.NotPanics(t, func() { l.Wait() }, "Panic on second wait")
		assert.Equal(t, _STATE_STOPPED, l.state, "wrong state after double wait")
	})
	t.Run("wait_long_buff", func(t *testing.T) {
		buffsize := 8192
		l := InitWithParams(DEFAULT_LOG_LEVEL, nil)
		err := l.Start(buffsize)
		assert.Nil(t, err, "error on start")
		lc := l.NewClient("fake")
		for i := range buffsize {
			lc.Log(LVL_UNMASKABLE, strconv.Itoa(i))
		}
		l.Stop()
		l.Wait()
		assert.Empty(t, l.channel, "channel is not empty")
	})
}

func Test_Logger_StopAndWait(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on start")
		assert.NotPanics(t, func() { l.StopAndWait() }, "Panic on StopAndWait")
		assert.Equal(t, _STATE_STOPPED, l.state, "wrong state after StopAndWait")
	})
	t.Run("without_start", func(t *testing.T) {
		l := Init()
		assert.NotPanics(t, func() { l.StopAndWait() }, "Panic on StopAndWait without start")
		assert.Equal(t, _STATE_STOPPED, l.state, "wrong state after StopAndWait without start")
	})
	t.Run("double_StopAndWait", func(t *testing.T) {
		l := Init()
		err := l.Start(0)
		assert.Nil(t, err, "error on start")
		assert.NotPanics(t, func() { l.StopAndWait() }, "Panic on first StopAndWait")
		assert.NotPanics(t, func() { l.StopAndWait() }, "Panic on second StopAndWait")
		assert.Equal(t, _STATE_STOPPED, l.state, "wrong state after double StopAndWait")
	})
}

func Test_Logger_InitWithParams(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		fallbck := io.Discard
		out1 := io.Discard
		out2 := os.Stdout
		level := LVL_DEBUG
		l := InitWithParams(level, fallbck, out1, out2)
		assert.Equal(t, _STATE_STOPPED, l.state, "wrong state after init")
		assert.Equal(t, level, l.level, "wrong level after init")
		assert.Equal(t, 2, len(l.outputs), "wrong outputs count after init")
		assert.Contains(t, l.outputs, out1, "missing output1 after init")
		assert.Contains(t, l.outputs, out2, "missing output2 after init")
		assert.Equal(t, fallbck, l.fallbck, "wrong fallback after init")
	})
	t.Run("corrections", func(t *testing.T) {
		out1 := os.Stdout
		out2 := io.Discard
		level := _LVL_MAX_for_checks_only + 10
		l := InitWithParams(level, nil, nil, out1, nil, out2)
		assert.Equal(t, _STATE_STOPPED, l.state, "wrong state after init")
		assert.Equal(t, LVL_UNKNOWN, l.level, "wrong level after init")
		assert.Equal(t, 2, len(l.outputs), "wrong outputs count after init")
		assert.Contains(t, l.outputs, out1, "missing output1 after init")
		assert.Contains(t, l.outputs, out2, "missing output2 after init")
		assert.NotContains(t, l.outputs, nil, "nil output after init")
		assert.Equal(t, io.Discard, l.fallbck, "wrong fallback after init")
	})
}

func Test_Logger_Init(t *testing.T) {
	t.Run("explicit_params", func(t *testing.T) {
		var l *Logger
		out1 := os.Stdout
		assert.NotPanics(t, func() {
			l = Init(out1)
			l.Start(0)
			l.StopAndWait()
		})
		assert.Equal(t, _STATE_STOPPED, l.state, "wrong state")
		assert.Equal(t, DEFAULT_LOG_LEVEL, l.level, "wrong log level")
		assert.Equal(t, 1, len(l.outputs), "wrong outputs count")
		assert.Contains(t, l.outputs, out1, "wrong output")
		assert.Equal(t, os.Stderr, l.fallbck, "wrong fallback")
	})
	t.Run("nil_output", func(t *testing.T) {
		var l *Logger
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
		var l *Logger
		assert.NotPanics(t, func() {
			l = Init([]OutType{}...)
			l.Start(0)
			l.StopAndWait()
		})
		assert.Empty(t, l.outputs, "outputs exist")
		assert.Equal(t, DEFAULT_LOG_LEVEL, l.level, "wrong log level")
		assert.Equal(t, os.Stderr, l.fallbck, "wrong fallback")
	})
}

func Test_Logger_InitAndStart(t *testing.T) {
	t.Run("explicit_params", func(t *testing.T) {
		var l *Logger
		out1 := os.Stdout
		assert.NotPanics(t, func() {
			l = InitAndStart(DEFAULT_MSG_BUFF, out1)
		})
		assert.Equal(t, DEFAULT_MSG_BUFF, cap(l.channel))
		assert.Equal(t, _STATE_ACTIVE, l.state, "wrong active state")
		assert.Equal(t, DEFAULT_LOG_LEVEL, l.level, "wrong log level")
		assert.Equal(t, 1, len(l.outputs), "wrong outputs count")
		assert.Contains(t, l.outputs, out1, "wrong output")
		assert.Equal(t, os.Stderr, l.fallbck, "wrong fallback")
		assert.NotPanics(t, func() {
			l.StopAndWait()
		})
		assert.Equal(t, _STATE_STOPPED, l.state, "wrong stopped state")
	})
	t.Run("min_params", func(t *testing.T) {
		var l *Logger
		assert.NotPanics(t, func() {
			l = InitAndStart(-1)
		})
		assert.Equal(t, DEFAULT_MSG_BUFF, cap(l.channel))
		assert.Equal(t, _STATE_ACTIVE, l.state, "wrong active state")
		assert.Equal(t, DEFAULT_LOG_LEVEL, l.level, "wrong log level")
		assert.Empty(t, l.outputs, "outputs exist")
		assert.Equal(t, os.Stderr, l.fallbck, "wrong fallback")
		assert.NotPanics(t, func() {
			l.StopAndWait()
		})
		assert.Equal(t, _STATE_STOPPED, l.state, "wrong stopped state")
	})
}

func Test_Logger_SetLevelPrefix(t *testing.T) {
	out1 := &FakeWriter{}
	l := Init(out1)
	tests := []struct {
		name      string // description of this test case
		prefixmap *LevelMap
		delimiter string
	}{
		{"nil_map_no_delim", nil, ""},
		{"short_map_short_delim", LevelShortNames, "!short!"},
		{"long_map_long_delim", LevelShortNames, testlogstr},
		{"empty", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := l.SetOutputLevelPrefix(out1, tt.prefixmap, tt.delimiter)
			assert.Equal(t, l, got, "wrong return (must be self)")
			assert.Equal(t, tt.prefixmap, l.outputs[out1].prefixmap, "wrong prefixmap assignment")
			assert.Equal(t, tt.delimiter, string(l.outputs[out1].delimiter), "wrong delimiter assignment")
		})
	}
}

func Test_Logger_SetLevelColor(t *testing.T) {
	out1 := &FakeWriter{}
	l := Init(out1)
	tests := []struct {
		name     string // description of this test case
		colormap *LevelMap
	}{
		{"nil", nil},
		{"map", LevelColorOnBlackMap},
		{"empty", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := l.SetOutputLevelColor(out1, tt.colormap)
			assert.Equal(t, l, got, "wrong return (must be self)")
			assert.Equal(t, tt.colormap, l.outputs[out1].colormap, "wrong colormap assignment")
		})
	}
}

func Test_Logger_SetMinLevel(t *testing.T) {
	out1 := &FakeWriter{}
	l := Init(out1)
	tests := []struct {
		name string // description of this test case
		val  LogLevel
	}{
		{"normal", LVL_INFO},
		{"zero", LVL_UNKNOWN},
		{"overmax", _LVL_MAX_for_checks_only},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := l.SetOutputMinLevel(out1, tt.val)
			assert.Equal(t, l, got, "wrong return (must be self)")
			assert.Equal(t, normLevel(tt.val), l.outputs[out1].minlevel, "wrong level assignment")
		})
	}
}

func Test_Logger_SetTimeFormat(t *testing.T) {
	out1 := &FakeWriter{}
	l := Init(out1)
	tests := []struct {
		name    string // description of this test case
		timefmt string
	}{
		{"empty", ""},
		{"any", time.RFC1123},
		{"other", time.Stamp},
		{"fake", testlogstr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := l.SetOutputTimeFormat(out1, tt.timefmt, "")
			assert.Equal(t, l, got, "wrong return (must be self)")
			assert.Equal(t, tt.timefmt, l.outputs[out1].timefmt, "wrong time format assignment")
		})
	}
}

func Test_Logger_ShowLevelCode(t *testing.T) {
	out1 := &FakeWriter{}
	l := Init(out1)
	tests := []struct {
		name string // description of this test case
	}{
		{"set"},
		{"again"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := l.ShowOutputLevelCode(out1)
			assert.Equal(t, l, got, "wrong return (must be self)")
			assert.True(t, l.outputs[out1].showlvlid, "showlvlid is unset")
		})
	}
}

func Test_Logger_IsOutputEnabled(t *testing.T) {
	l := Init(io.Discard)
	t.Run("20_times", func(t *testing.T) {
		b := false
		for i := range 20 {
			b = i%3 == 1
			l.outputs[io.Discard].enabled = b
			//assert.Equal(t, b, l.outputs[io.Discard].IsEnabled())
			assert.Equal(t, b, l.IsOutputEnabled(io.Discard))
		}
	})
	t.Run("unexisting", func(t *testing.T) {
		assert.False(t, l.IsOutputEnabled(os.Stderr))
	})
}

func Test_Logger_IsOutputExists(t *testing.T) {
	w0 := &FakeWriter{}
	w1 := &FakeWriter{}
	l := Init(io.Discard, os.Stdout, w0)
	t.Run("existing", func(t *testing.T) {
		assert.True(t, l.IsOutputExists(io.Discard))
		assert.True(t, l.IsOutputExists(os.Stdout))
		assert.True(t, l.IsOutputExists(w0))
	})
	t.Run("unexisting", func(t *testing.T) {
		assert.False(t, l.IsOutputExists(w1))
		assert.False(t, l.IsOutputExists(nil))
	})
}

func Test_Logger_NewClient(t *testing.T) {
	var l *Logger
	var lc *LogClient
	prep := func(lvl LogLevel, name string) {
		l = Init()
		assert.NotPanics(t, func() {
			lc = l.NewClientWithLevel(name, lvl)
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

func Test_Logger_pushMessage(t *testing.T) {
	ferr := &FakeWriter{}
	out1 := &FakeWriter{}
	textmsg := &logMessage{msgtype: _MSG_LOG_TEXT, msgdata: testbytes, annex: basetype(LVL_UNMASKABLE)}
	tests := []struct {
		name    string // description of this test case
		msg     *logMessage
		started bool
		nilchan bool
		state   lgrState
		wantErr string
	}{
		{"msg_txt", textmsg, true, false, _STATE_ACTIVE, ""},
		{"msg_nil", nil, true, false, _STATE_ACTIVE, _ERROR_MESSAGE_LOG_MSG_IS_NIL},
		{"not-started", textmsg, false, false, _STATE_UNKNOWN, _ERROR_MESSAGE_LOGGER_INACTIVE},
		{"channel_nil", textmsg, true, true, _STATE_ACTIVE, _ERROR_MESSAGE_CHANNEL_IS_NIL},
		{"msg_cmd", &logMessage{msgtype: _MSG_COMMAND, annex: basetype(_CMD_DUMMY), msgdata: testbytes}, true, false, _STATE_ACTIVE, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Init(out1)
			l.SetFallback(ferr)
			if tt.started {
				l.Start(0)
			}
			if tt.nilchan {
				l.channel = nil
			}
			l.state = tt.state
			t0 := time.Now()
			tm, gotErr := l.pushMessage(tt.msg)
			t1 := time.Now()
			if tt.started && !tt.nilchan {
				l.StopAndWait()
			}
			res := ""
			if gotErr != nil {
				res = gotErr.Error()
			}
			if tt.wantErr == "" {
				assert.WithinRange(t, tm, t0, t1)
				assert.NoError(t, gotErr, "unexpected error")
				assert.Contains(t, out1.String(), testlogstr, "wrong output")
				assert.Empty(t, ferr, "fallback is not empty: "+ferr.String())
			} else {
				assert.Contains(t, res, tt.wantErr, "wrong error text")
				assert.Zero(t, tm, "non-zero time on error")
			}
		})
	}
}

func Test_Logger_runClientCommand(t *testing.T) {
	ferr1 := &FakeWriter{}
	ferr2 := &FakeWriter{}
	out1 := &FakeWriter{}
	out2 := &FakeWriter{}
	l1 := Init(out1)
	l1.SetFallback(ferr1)
	l1.state = _STATE_ACTIVE
	lc1 := l1.NewClientWithLevel("lc1", LVL_UNKNOWN)
	l2 := Init(out2)
	l2.SetFallback(ferr2)
	lc2 := l2.NewClientWithLevel("lc2", LVL_UNMASKABLE)
	lcnil := l2.NewClientWithLevel("lc2", LVL_UNMASKABLE)
	lcnil.logger = nil

	type testType struct {
		name    string // description of this test case
		lc      *LogClient
		cmd     cmdType
		wantMsg bool
		wantErr string
	}
	tests := []testType{
		{"nil_client", nil, _CMD_CLIENT_DUMMY, false, _ERROR_MESSAGE_CLIENT_IS_NIL},
		{"alien_client", lc2, _CMD_CLIENT_DUMMY, false, _ERROR_MESSAGE_CLIENT_IS_ALIEN},
		{"orphan_client", lcnil, _CMD_CLIENT_DUMMY, false, _ERROR_MESSAGE_CLIENT_IS_ALIEN},
		{"wrong_command", lc1, _CMD_MAX_for_checks_only + 10, false, _ERROR_MESSAGE_NON_CLIENT_CMD},
	}
	for i := _CMD_CLIENT_commands_min; i <= _CMD_CLIENT_commands_max; i++ {
		tests = append(tests, testType{"cmd_" + strconv.Itoa(int(i)), lc1, i, true, ""})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ferr1.Clear()
			ferr2.Clear()
			out1.Clear()
			out2.Clear()
			l2.Start(0)
			l1.channel = make(chan logMessage, 1)
			defer close(l1.channel)
			t0 := time.Now()
			got, gotErr := l1.runClientCommand(tt.lc, tt.cmd, []byte(testlogstr))
			t1 := time.Now()
			select {
			case msg := <-l1.channel:
				assert.True(t, tt.wantMsg, "forbidden sending to channel")
				assert.ElementsMatch(t, []byte(testlogstr), msg.msgdata, "command data changed after channel")
				if tt.wantErr == "" {
					assert.NoError(t, gotErr, "unexpected error")
					assert.WithinRange(t, got, t0, t1, "wrong timestamp")
				} else {
					assert.Zero(t, got, "non-zero time returned on error")
					assert.NotNil(t, gotErr, "no error")
					assert.ErrorContains(t, gotErr, tt.wantErr, "wrong error")
				}
			default:
				assert.False(t, tt.wantMsg, "message is not sent to channel")
			}
			l2.StopAndWait()
			assert.Empty(t, out1.String(), "output1 is not empty")
			assert.Empty(t, out2.String(), "output2 is not empty")
			assert.Empty(t, ferr1.String(), "fallback1 is not empty")
			assert.Empty(t, ferr2.String(), "fallback1 is not empty")
		})
	}
}

func Test_Logger_SetClientMinLevel(t *testing.T) {
	ferr1 := &FakeWriter{}
	ferr2 := &FakeWriter{}
	out1 := &FakeWriter{}
	out2 := &FakeWriter{}
	l1 := Init(out1)
	l1.SetFallback(ferr1)
	lc1 := l1.NewClientWithLevel("lc1", LVL_UNKNOWN)
	l2 := Init(out2)
	l2.SetFallback(ferr2)
	lc2 := l2.NewClientWithLevel("lc2", LVL_UNMASKABLE)
	l2.Start(0)
	lcnil := l2.NewClientWithLevel("lc2", LVL_UNMASKABLE)
	lcnil.logger = nil

	tests := []struct {
		name     string // description of this test case
		lc       *LogClient
		minlevel LogLevel
		wantErr  string
	}{
		{"normal", lc1, LVL_UNMASKABLE, ""},
		{"unknown_level", lc1, _LVL_MAX_for_checks_only + 10, ""},
		{"nil_client", nil, LVL_UNMASKABLE, _ERROR_MESSAGE_CLIENT_IS_NIL},
		{"alien_client", lc2, LVL_UNMASKABLE, _ERROR_MESSAGE_CLIENT_IS_ALIEN},
		{"orphan_client", lcnil, LVL_UNMASKABLE, _ERROR_MESSAGE_CLIENT_IS_ALIEN},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ferr1.Clear()
			ferr2.Clear()
			out1.Clear()
			out2.Clear()
			l1.Start(0)
			got, gotErr := l1.SetClientMinLevel(tt.lc, tt.minlevel)
			l1.StopAndWait()
			if tt.wantErr == "" {
				res := ""
				if gotErr != nil {
					res = gotErr.Error()
				}
				assert.Empty(t, gotErr, "unexpected error: '"+res+"'")
				assert.WithinDuration(t, time.Now(), got, time.Second*5, "wrong timestamp")
				assert.Equal(t, normLevel(tt.minlevel), tt.lc.minLevel, "wrong level setting")
			} else {
				assert.NotNil(t, gotErr, "no error")
				assert.ErrorContains(t, gotErr, tt.wantErr, "wrong error")
			}
			assert.Empty(t, out1.String(), "output1 is not empty")
			assert.Empty(t, out2.String(), "output2 is not empty")
			assert.Empty(t, ferr1.String(), "fallback1 is not empty")
			assert.Empty(t, ferr2.String(), "fallback2 is not empty")
		})
	}
}
func Test_LogClient_LogBytes_with_err(t *testing.T) {
	var msg *logMessage
	var err error
	var l *Logger
	var lc *LogClient
	prep := func(logdata []byte, loglevel LogLevel, msglevel LogLevel, ferr *FakeWriter, outs ...OutType) (*logMessage, error) {
		l = Init()
		l.SetFallback(OutType(ferr)).ClearOutputs().AddOutputs(outs...)
		lc = l.NewClientWithLevel("[Testing client name]", LVL_UNKNOWN)
		l.Start(0)
		l.level = loglevel
		t, e := lc.LogBytes_with_err(msglevel, logdata)
		l.StopAndWait()
		if e == nil {
			msg := makeTextMessage(lc, loglevel, logdata)
			msg.pushed = t
			return msg, nil // message for comparison
		}
		return nil, e
	}
	t.Run("unrecovered_panic", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		assert.PanicsWithError(t, _ERROR_MESSAGE_TEST_PANIC_TEXT, func() {
			msg, err = prep(testbytes, _LVL_MAX_for_checks_only+1, LVL_DEBUG, ferr, out1, out2)
		}, "No panic on unmasked panic")
		assert.NoError(t, err, "unexpected error")
		assert.Nil(t, msg, "message not nil")
		assert.Empty(t, out1.buffer, "data written to output 1")
		assert.Empty(t, out2.buffer, "data written to output 2")
		assert.Empty(t, ferr.buffer, "data written to fallback")
	})
	t.Run("level_masked", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		for l := range _LVL_MAX_for_checks_only {
			assert.NotPanics(t, func() {
				msg, err = prep(testbytes, _LVL_MAX_for_checks_only, l, ferr, out1)
				assert.NoError(t, err, "unexpected error")
			}, "Panic on write")
			assert.Empty(t, ferr.buffer, "data written to fallback")
			assert.Empty(t, out1.buffer, "data written to output on log with level lower than set")
		}
	})
	t.Run("level_out_of_range", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		for l := _LVL_MAX_for_checks_only; l < _LVL_MAX_for_checks_only+100; l++ {
			assert.NotPanics(t, func() {
				msg, err = prep(testbytes, _LVL_MAX_for_checks_only, l, ferr, out1)
				assert.EqualError(t, err, _ERROR_MESSAGE_LOG_LEVEL_RANGE)
			}, "Panic on write")
			assert.Empty(t, ferr.buffer, "data written to fallback")
			assert.Empty(t, out1.buffer, "data written to output on log with level lower than set")
		}
	})
	t.Run("empty_message", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg, err = prep([]byte{}, LVL_WARN, LVL_WARN, ferr, out1)
		}, "Panic on write")
		assert.NoError(t, err, "error on empty message")
		assert.Zero(t, msg.pushed, "non-zero pushed time returned on empty message")
		assert.Empty(t, ferr.buffer, "data written to fallback")
		assert.Empty(t, out1.buffer, "data written to output on empty message")
	})
	t.Run("level_logged", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		outBuffer := bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		assert.NotPanics(t, func() {
			msg, err = prep(testbytes, LVL_WARN, LVL_WARN, ferr, out1)
			assert.NoError(t, err, "unexpected error")
		}, "Panic on write")
		assert.Empty(t, ferr.buffer, "data written to fallback")
		assert.NotEmpty(t, out1.buffer, "no data written to output on log with level higher than set")
		assert.Equal(t, buildTextMessage(outBuffer, msg, l.getContext(out1)).Bytes(), out1.buffer)
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
		msg, err = prep(testbytes, LVL_INFO, LVL_WARN, ferr, []OutType{}...)
		assert.Empty(t, ferr.buffer, "data written to fallback on write to nil outputs")
	})
	t.Run("2_outputs", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		outBuffer := bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		assert.NotPanics(t, func() {
			msg, err = prep(testbytes, LVL_INFO, LVL_WARN, ferr, out1, out2)
			assert.NoError(t, err, "unexpected error")
		}, "Panic on write")
		assert.Equal(t, buildTextMessage(outBuffer, msg, l.getContext(out1)).Bytes(), out1.buffer)
		assert.Equal(t, buildTextMessage(outBuffer, msg, l.getContext(out2)).Bytes(), out2.buffer)
		assert.Empty(t, ferr.buffer, "data written to fallback")
	})
	t.Run("1_error_1_panic_outs", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		out2 := &FakeWriter{}
		outBuffer := bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		assert.NotPanics(t, func() {
			msg, err = prep(testbytes, LVL_INFO, LVL_WARN, ferr, out1, &ErrorWriter{}, out2, &PanicWriter{})
			assert.NoError(t, err, "unexpected error")
		}, "Panic on write")
		assert.Equal(t, buildTextMessage(outBuffer, msg, l.getContext(out1)).Bytes(), out1.buffer)
		assert.Equal(t, buildTextMessage(outBuffer, msg, l.getContext(out2)).Bytes(), out2.buffer)
		assert.Contains(t, ferr.String(), panicStr+"`\n")
		assert.Contains(t, ferr.String(), errorStr+"\n")
	})
}
func Test_LogClient_LogBytes(t *testing.T) {
	t.Run("log_with_error_and_nil_logger", func(t *testing.T) {
		lc := &LogClient{}
		assert.Zero(t, lc.LogBytes(LVL_UNMASKABLE, testbytes), "non-zero push time on error")
	})
	t.Run("log_with_error_and_real_logger", func(t *testing.T) {
		ferr := &FakeWriter{}
		l := InitWithParams(LVL_UNKNOWN, ferr)
		lc := l.NewClient("test")
		tm := lc.LogBytes(_LVL_MAX_for_checks_only+10, testbytes)
		assert.Zero(t, tm, "non-zero push time on error")
		assert.Equal(t, _ERROR_MESSAGE_LOG_LEVEL_RANGE+"\n", string(ferr.buffer[len(_FALLBACK_TIME_FORMAT):]))
	})
}

func Test_LogClient_Log_with_err(t *testing.T) {
	var msg *logMessage
	var err error
	var l *Logger
	var lc *LogClient
	prep := func(f func(), loglevel LogLevel, msglevel LogLevel, ferr *FakeWriter, outs ...OutType) (*logMessage, error) {
		l = Init()
		l.SetFallback(OutType(ferr)).ClearOutputs().AddOutputs(outs...)
		lc = l.NewClientWithLevel("[Testing client name]", LVL_UNKNOWN)
		f()
		l.level = loglevel
		t, e := lc.Log_with_err(msglevel, testlogstr)
		if e == nil {
			msg := makeTextMessage(lc, loglevel, []byte(testlogstr))
			msg.pushed = t
			return msg, nil
		}
		return nil, e
	}
	t.Run("disabled_client", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg, err = prep(func() {
				l.SetClientEnabled(lc, false)
				l.Start(0)
				close(l.channel)
			}, LVL_WARN, LVL_WARN, ferr, out1)
		}, "Panic on write")
		l.channel = make(chan logMessage)
		l.StopAndWait()
		assert.NoError(t, err, "error on log to disabled client")
		assert.Empty(t, ferr.buffer, "data written to fallback")
		assert.Empty(t, out1.buffer, "data written to output")
	})
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
	t.Run("error_on_nil_channel", func(t *testing.T) {
		ferr := &FakeWriter{}
		out1 := &FakeWriter{}
		assert.NotPanics(t, func() {
			msg, err = prep(func() {
				l.state = _STATE_ACTIVE
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
		assert.EqualError(t, err, _ERROR_MESSAGE_LOGGER_IS_NIL, "Wrong error on log to stopped logger")
		assert.Empty(t, ferr.buffer, "data written to fallback")
		assert.Empty(t, out1.buffer, "data written to output")
	})
}
func Test_LogClient_Log(t *testing.T) {
	const (
		clname = "[Testing client name]"
		format = "2006-01-02 15:04:05"
		delim1 = "#"
		delim2 = "@"
		level  = LVL_UNMASKABLE
	)
	out1 := &FakeWriter{}
	ferr := &FakeWriter{}
	l := InitWithParams(LVL_UNKNOWN, ferr, out1)
	l.SetOutputLevelPrefix(out1, nil, delim2)
	tests := []struct {
		name string
		prfx string
		sufx string
		fn   func(*Logger)
	}{
		// prfx and sufx are cumulative!!!
		{"time", "", "", nil},
		{"level_index", "[" + strconv.Itoa(int(level)) + "]" + delim2, "", func(l *Logger) { l.ShowOutputLevelCode(out1) }},
		{"level_name", LevelShortNames[level] + delim2, "", func(l *Logger) { l.SetOutputLevelPrefix(out1, LevelShortNames, delim2) }},
		{"color_map", ANSI_COL_PRFX + LevelColorOnBlackMap[level] + ANSI_COL_SUFX, ANSI_COL_RESET, func(l *Logger) { l.SetOutputLevelColor(out1, LevelColorOnBlackMap) }},
	}
	prefix := ""
	suffix := "\n"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ferr.Clear()
			out1.Clear()
			l.SetOutputTimeFormat(out1, format, delim1)
			if tt.fn != nil {
				tt.fn(l)
			}
			l.Start(0)
			lc := l.NewClient(clname)
			t0 := lc.Log(level, testlogstr)
			tstr := t0.Format(format) + delim1
			//////////////////////
			prefix = prefix + tt.prfx
			suffix = tt.sufx + suffix
			resstr := tstr + prefix + clname + delim2 + testlogstr + suffix
			l.StopAndWait()
			assert.Empty(t, ferr.buffer, "err written to fallback")
			assert.Equal(t, resstr, out1.String())
		})
	}
}
func Test_LogClient_LogLevels(t *testing.T) {
	l := Init()
	out1 := &FakeWriter{}
	outBuffer := bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
	l.AddOutputs(out1)
	l.SetOutputLevelColor(out1, LevelColorOnBlackMap)
	l.SetOutputLevelPrefix(out1, LevelFullNames, " !delimiter! ")
	l.SetMinLevel(LVL_UNKNOWN)
	lc := l.NewClientWithLevel("", LVL_UNKNOWN)
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
	t.Run("fatal_write", func(t *testing.T) {
		out1.Clear()
		outBuffer := bytes.NewBuffer(make([]byte, DEFAULT_OUT_BUFF))
		l.Start(0)
		msg := makeTextMessage(lc, LVL_FATAL, []byte(testlogstr))
		msg.pushed = lc.LogFatal(errors.New(testlogstr))
		l.StopAndWait()
		assert.Equal(t, buildTextMessage(outBuffer, msg, l.outputs[out1]).Bytes(), out1.buffer, "wrong output")
	})
}
func Test_Logger_SetClientEnabled(t *testing.T) {
	t.Run("nil_client", func(t *testing.T) {
		l := Init()
		err := l.SetClientEnabled(nil, false)
		if assert.Error(t, err) {
			assert.ErrorContains(t, err, _ERROR_MESSAGE_CLIENT_IS_NIL)
		}
	})

	t.Run("alien_client", func(t *testing.T) {
		l1 := Init()
		l2 := Init()
		lc := l2.NewClientWithLevel("alien", LVL_UNKNOWN)
		// initial state must be true
		assert.True(t, lc.enabled)
		err := l1.SetClientEnabled(lc, false)
		if assert.Error(t, err) {
			assert.ErrorContains(t, err, _ERROR_MESSAGE_CLIENT_IS_ALIEN)
		}
		// ensure alien client's enabled flag was not changed by other logger
		assert.True(t, lc.enabled)
	})

	t.Run("disable_enable_toggle", func(t *testing.T) {
		l := Init()
		lc := l.NewClientWithLevel("local", LVL_UNKNOWN)
		// initially enabled
		assert.True(t, lc.enabled)

		// disable
		err := l.SetClientEnabled(lc, false)
		assert.NoError(t, err)
		assert.False(t, lc.enabled)

		// enable again
		err = l.SetClientEnabled(lc, true)
		assert.NoError(t, err)
		assert.True(t, lc.enabled)

		// idempotent calls
		err = l.SetClientEnabled(lc, true)
		assert.NoError(t, err)
		assert.True(t, lc.enabled)
	})
}
func Test_Logger_SetClientName(t *testing.T) {
	ferr1 := &FakeWriter{}
	ferr2 := &FakeWriter{}
	out1 := &FakeWriter{}
	out2 := &FakeWriter{}
	l1 := Init(out1)
	l1.SetFallback(ferr1)
	l1.state = _STATE_ACTIVE
	lc1 := l1.NewClientWithLevel("lc1", LVL_UNKNOWN)
	l2 := Init(out2)
	l2.SetFallback(ferr2)
	lc2 := l2.NewClientWithLevel("lc2", LVL_UNMASKABLE)
	lcnil := l2.NewClientWithLevel("lc2", LVL_UNMASKABLE)
	lcnil.logger = nil

	type testType struct {
		name    string
		lc      *LogClient
		wantMsg bool
		wantErr string
	}
	tests := []testType{
		{"nil_client", nil, false, _ERROR_MESSAGE_CLIENT_IS_NIL},
		{"alien_client", lc2, false, _ERROR_MESSAGE_CLIENT_IS_ALIEN},
		{"orphan_client", lcnil, false, _ERROR_MESSAGE_CLIENT_IS_ALIEN},
		{"normal", lc1, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ferr1.Clear()
			ferr2.Clear()
			out1.Clear()
			out2.Clear()
			l2.Start(0)
			l1.channel = make(chan logMessage, 1)
			defer close(l1.channel)
			t0 := time.Now()
			got, gotErr := l1.SetClientName(tt.lc, testlogstr)
			t1 := time.Now()
			select {
			case msg := <-l1.channel:
				assert.True(t, tt.wantMsg, "forbidden sending to channel")
				assert.ElementsMatch(t, []byte(testlogstr), msg.msgdata, "command data changed after channel")
				if tt.wantErr == "" {
					assert.NoError(t, gotErr, "unexpected error")
					assert.WithinRange(t, got, t0, t1, "wrong timestamp")
				} else {
					assert.Zero(t, got, "non-zero time returned on error")
					assert.NotNil(t, gotErr, "no error")
					assert.ErrorContains(t, gotErr, tt.wantErr, "wrong error")
				}
			default:
				assert.False(t, tt.wantMsg, "message is not sent to channel")
				if tt.wantErr == "" {
					// expected a message but none was sent
					assert.Fail(t, "expected message in channel but none received")
				} else {
					// error cases: ensure error returned
					if assert.Error(t, gotErr) {
						assert.ErrorContains(t, gotErr, tt.wantErr)
					}
				}
			}
			l2.StopAndWait()
			assert.Empty(t, out1.String(), "output1 is not empty")
			assert.Empty(t, out2.String(), "output2 is not empty")
			assert.Empty(t, ferr1.String(), "fallback1 is not empty")
			assert.Empty(t, ferr2.String(), "fallback2 is not empty")
		})
	}
}

func TestLogger_IsOwnClient(t *testing.T) {
	t.Run("own", func(t *testing.T) {
		l := Init(nil)
		lc1 := l.NewClient("")
		assert.True(t, l.IsOwnClient(lc1))
	})
	t.Run("alien", func(t *testing.T) {
		l := Init(nil)
		l1 := Init(nil)
		lc1 := l1.NewClient("")
		assert.False(t, l.IsOwnClient(lc1))
	})
	t.Run("orphan", func(t *testing.T) {
		l := Init(nil)
		lc1 := l.NewClient("")
		lc1.logger = nil
		assert.False(t, l.IsOwnClient(lc1))
	})
}

func TestLogger_Fatal_Normal(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		out1 := &FakeWriter{}
		ferr := &FakeWriter{}
		lgr := InitWithParams(LVL_UNKNOWN, ferr, out1)
		lgr.Start(0)
		Fatal(lgr, errors.New(errorStr))
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run="+"TestLogger_Fatal_Normal")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	e, ok := err.(*exec.ExitError)
	assert.True(t, ok, "not ok on calling err.(*exec.ExitError)")
	assert.False(t, e.Success(), "normal exit instead of error code")
	assert.Equal(t, 1, e.ExitCode(), "wrong exit code")
}

func TestLogger_Fatal_Inactive(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		out1 := &FakeWriter{}
		ferr := &FakeWriter{}
		lgr := InitWithParams(LVL_UNKNOWN, ferr, out1)
		Fatal(lgr, errors.New(errorStr))
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run="+"TestLogger_Fatal_Inactive")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	e, ok := err.(*exec.ExitError)
	assert.True(t, ok, "not ok on calling err.(*exec.ExitError)")
	assert.False(t, e.Success(), "normal exit instead of error code")
	assert.Equal(t, 1, e.ExitCode(), "wrong exit code")
}

func TestLogger_Fatal_Nils(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		Fatal(nil, nil)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run="+"TestLogger_Fatal_Nils")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	e, ok := err.(*exec.ExitError)
	assert.True(t, ok, "not ok on calling err.(*exec.ExitError)")
	assert.False(t, e.Success(), "normal exit instead of error code")
	assert.Equal(t, 1, e.ExitCode(), "wrong exit code")
}

// tests := []struct {
// tests := []struct {
// 	name string // description of this test case
// 	l    *Logger
// 	o    []OutType
// 	e    error
// }{
// 	{"nils", nil, nil, nil},
// 	{"normal", lgr, []OutType{out1}, errors.New(errorStr)},
// }
// for _, tt := range tests {
// 	t.Run(tt.name, func(t *testing.T) {
// 		err := cmd.Run()
// 		if e, ok := err.(*exec.ExitError); ok && !e.Success() {
// 			return
// 		}
// 		Fatal(tt.l, tt.e)
// 	})
// }
