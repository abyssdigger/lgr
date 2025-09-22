package lgr

import (
	"io"
	"testing"
)

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

func TestLogger_logText(t *testing.T) {
	tests := []struct {
		wantPanic bool
		wantErr   bool
		name      string // description of this test case
		// Named input parameters for target function.
		output OutType
		msg    *logMessage
	}{
		// TODO: Add test cases.
		{false, false, "valid_output", io.Discard, &logMessage{msgtype: MSG_LOG_TEXT, msgtext: "test text"}},
		{false, true, "nil_output", nil, &logMessage{msgtype: MSG_LOG_TEXT, msgtext: "test text"}},
		{true, true, "panic_output", panicWriter(0), &logMessage{msgtype: MSG_LOG_TEXT, msgtext: "test text"}},
		{false, false, "empty_msg", io.Discard, &logMessage{msgtype: MSG_LOG_TEXT, msgtext: ""}},
		{false, false, "nil_msg", io.Discard, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Init()
			got, gotErr := l.logText(tt.output, tt.msg)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("logText() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("logText() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("logText() = %v, want %v", got, tt.want)
			}
		})
	}
}
