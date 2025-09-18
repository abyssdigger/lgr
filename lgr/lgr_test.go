package lgr

import (
	"bytes"
	"io"
	"testing"
)

func TestLogger_Start_Stop_Wait(t *testing.T) {
	logger := &Logger{}
	var buf bytes.Buffer
	if err := logger.Start(DEBUG, 4, &buf); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if !logger.IsActive() {
		t.Error("Logger should be active after Start")
	}
	logger.Stop()
	logger.Wait()
	if logger.IsActive() {
		t.Error("Logger should not be active after Stop and Wait")
	}
}

func TestLogger_StartDefault(t *testing.T) {
	logger := &Logger{}
	if err := logger.StartDefault(); err != nil {
		t.Fatalf("StartDefault error: %v", err)
	}
	logger.StopAndWait()
	if logger.IsActive() {
		t.Error("Logger should not be active after StopAndWait")
	}
}

func TestLogger_Log(t *testing.T) {
	logger := &Logger{}
	var buf bytes.Buffer
	if err := logger.Start(DEBUG, 4, &buf); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	defer logger.StopAndWait()
	logger.level = DEBUG
	msg := "test message\n"
	if err := logger.Log(DEBUG, msg); err != nil {
		t.Errorf("Log error: %v", err)
	}
	logger.StopAndWait()
	if !bytes.Contains(buf.Bytes(), []byte(msg)) {
		t.Error("Log did not write expected message")
	}
}

func TestLogger_setState_IsActive(t *testing.T) {
	logger := &Logger{}
	logger.setState(ACTIVE)
	if !logger.IsActive() {
		t.Error("Logger should be active")
	}
	logger.setState(STOPPED)
	if logger.IsActive() {
		t.Error("Logger should not be active")
	}
}

func TestLogger_StopAndWait(t *testing.T) {
	logger := &Logger{}
	if err := logger.StartDefault(); err != nil {
		t.Fatalf("StartDefault error: %v", err)
	}
	logger.StopAndWait()
	if logger.IsActive() {
		t.Error("Logger should not be active after StopAndWait")
	}
}
