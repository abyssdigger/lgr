package lgr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type PanicWriter struct{}

func (p *PanicWriter) Write(b []byte) (int, error) { panic("generated panic in writer") }

type ErrorWriter struct{}

func (e *ErrorWriter) Write(b []byte) (int, error) { return 0, fmt.Errorf("generated error in writer") }

type FakeWriter struct {
	buffer []byte
}

func (f *FakeWriter) Write(b []byte) (int, error) {
	f.buffer = append(f.buffer, b...)
	return len(b), nil
}
func (f *FakeWriter) String() string { return string(f.buffer) }
func (f *FakeWriter) Clear()         { f.buffer = f.buffer[:0] }

func TestLogger_proceedMsg(t *testing.T) {
	tests := []struct {
		wantErr bool
		name    string // description of this test case
		// Named input parameters for target function.
		msg logMessage
	}{
		// TODO: Add test cases.
		{false, "log_msgtype", logMessage{msgtype: MSG_LOG_TEXT, msgtext: "test text"}},
		{true, "unused_msgtype", logMessage{msgtype: MSG_CHG_LEVEL, msgtext: "test text"}},
		{true, "unknown_msgtype", logMessage{msgtype: 99, msgtext: "test text"}},
		{true, "empty_msgtype", logMessage{msgtext: "test text"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Init() // any outputs, they are not used in this test
			gotErr := l.proceedMsg(&tt.msg)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("proceedCmd() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("proceedCmd() succeeded unexpectedly")
			}
		})
	}
}

func TestLogger_logData(t *testing.T) {
	foutput := &FakeWriter{}
	tests := []struct {
		wantPnc bool
		wantErr bool
		name    string // description of this test case
		// Named input parameters for target function.
		output OutType
		data   []byte
	}{
		{false, false, "valid_output", foutput, []byte("normal text")},
		{false, false, "empty_msg", foutput, []byte{}},
		{false, false, "nil_msg", foutput, nil},
		{false, true, "error_output", OutType(&ErrorWriter{}), []byte("test text")},
		{true, true, "panic_output", OutType(&PanicWriter{}), []byte("test text")},
		{true, true, "nil_output", nil, []byte("test to nil")},
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
		{"escp", "\n\t\r\f\v\b'\"\\`"},
	}
	l := InitWithParams(TRACE, foutput)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l.handleLogWriteError(tt.emsg)
			assert.Equal(t, tt.emsg+"\n", foutput.String(), "written data mismatch")
			foutput.Clear()
		})
	}
	t.Run("2nil", func(t *testing.T) {
		l := InitWithParams(TRACE, nil)
		assert.NotPanics(t, func() {
			l.handleLogWriteError("test write to nil fallback")
		}, "Panic on write to nil fallback")
	})
	t.Run("panic", func(t *testing.T) {
		l := InitWithParams(TRACE, &PanicWriter{})
		assert.Panics(t, func() {
			l.handleLogWriteError("test panic")
		}, "The code did not panic")
	})
}

func TestLogger_logTextToOutputs(t *testing.T) {
	var s string
	out1 := &FakeWriter{}
	//out2 := &FakeWriter{}
	t.Run("one", func(t *testing.T) {
		l := InitWithParams(TRACE, nil, out1)
		s = "Write to 1 output"
		l.logTextToOutputs(s)
		assert.Equal(t, s+"\n", out1.String())
	})
}
