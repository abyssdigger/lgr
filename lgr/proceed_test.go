package lgr

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type panicWriter struct{}

func (p *panicWriter) Write(b []byte) (int, error) { panic("generated panic in writer") }

type errorWriter struct{}

func (p *errorWriter) Write(b []byte) (int, error) { return 0, fmt.Errorf("generated error in writer") }

func TestLogger_proceedCmd(t *testing.T) {
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
	tests := []struct {
		wantPnc bool
		wantErr bool
		name    string // description of this test case
		// Named input parameters for target function.
		output OutType
		data   []byte
	}{
		// TODO: Add test cases.
		{false, false, "valid_output", io.Discard, []byte("normal text")},
		{false, false, "empty_msg", io.Discard, []byte{}},
		{false, false, "nil_msg", io.Discard, nil},
		{false, true, "error_output", OutType(&errorWriter{}), []byte("test text")},
		{true, true, "panic_output", OutType(&panicWriter{}), []byte("test text")},
		{true, true, "nil_output", nil, []byte("test to nil")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Init()
			gotPnc, gotErr := l.logData(tt.output, tt.data)
			assert.True(t, !tt.wantPnc || gotPnc, "did not panic when expected")
			assert.True(t, !tt.wantErr || gotErr != nil, "no error on expected failure")
			assert.False(t, !tt.wantPnc && gotPnc, "unexpected panic")
			assert.False(t, !tt.wantErr && gotErr != nil, fmt.Sprintf("unexpected error: %v", gotErr))
		})
	}
}
