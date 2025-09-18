package lgr

import (
	"bytes"
	"io"
	"testing"
)

type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (int, error) { return 0, io.ErrClosedPipe }

type panicWriter struct{}

func (p *panicWriter) Write(b []byte) (int, error) { panic("panic in writer") }

func TestLogger_AddOutputs(t *testing.T) {
	logger := &Logger{}
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}
	// Добавление обычных выходов
	logger.AddOutputs(buf1, buf2)
	if len(logger.outputs) != 2 {
		t.Errorf("Expected 2 outputs, got %d", len(logger.outputs))
	}
	// Попытка добавить повторно
	logger.AddOutputs(buf1)
	if len(logger.outputs) != 2 {
		t.Errorf("Duplicate output should not be added")
	}
	// Попытка добавить nil
	logger.AddOutputs(nil)
	if len(logger.outputs) != 2 {
		t.Errorf("nil output should not be added")
	}
	// Попытка добавить только nil
	logger.AddOutputs(nil, nil)
	if len(logger.outputs) != 2 {
		t.Errorf("Only nil outputs should not be added")
	}
	// Попытка добавить nil и новый
	buf3 := &bytes.Buffer{}
	logger.AddOutputs(nil, buf3)
	if len(logger.outputs) != 3 || logger.outputs[2] != buf3 {
		t.Errorf("Output buf3 should be added, nil ignored")
	}
}

func TestLogger_ClearOutputs(t *testing.T) {
	logger := &Logger{}
	buf := &bytes.Buffer{}
	logger.AddOutputs(buf)
	logger.ClearOutputs()
	if len(logger.outputs) != 0 {
		t.Errorf("Outputs not cleared")
	}
	// Очистка уже пустого
	logger.ClearOutputs()
	if len(logger.outputs) != 0 {
		t.Errorf("ClearOutputs on empty should keep outputs empty")
	}
}

func TestLogger_RemoveOutputs(t *testing.T) {
	logger := &Logger{}
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}
	buf3 := &bytes.Buffer{}
	logger.AddOutputs(buf1, buf2, buf3)
	// Удаление одного
	logger.RemoveOutputs(buf1)
	if len(logger.outputs) != 2 || logger.outputs[0] != buf2 || logger.outputs[1] != buf3 {
		t.Errorf("RemoveOutputs failed, outputs: %v", logger.outputs)
	}
	// Удаление нескольких
	logger.RemoveOutputs(buf2, buf3)
	if len(logger.outputs) != 0 {
		t.Errorf("RemoveOutputs (multiple) failed, outputs: %v", logger.outputs)
	}
	// Удаление из пустого
	logger.RemoveOutputs(buf1)
	if len(logger.outputs) != 0 {
		t.Errorf("RemoveOutputs on empty should keep outputs empty")
	}
	// Удаление nil
	logger.AddOutputs(buf1)
	logger.RemoveOutputs(nil)
	if len(logger.outputs) != 1 {
		t.Errorf("RemoveOutputs(nil) should not remove anything")
	}
}

func TestLogger_Start(t *testing.T) {
	logger := &Logger{}
	buf := &bytes.Buffer{}
	var errBuf bytes.Buffer
	err := logger.Start(DEBUG, 4, &errBuf, buf)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if !logger.IsActive() {
		t.Error("Logger should be active after Start")
	}
	// Проверка повторного запуска
	err2 := logger.Start(DEBUG, 4, &errBuf, buf)
	if err2 == nil {
		t.Error("Expected error when starting already active logger")
	}
	logger.StopAndWait()
	if logger.IsActive() {
		t.Error("Logger should not be active after StopAndWait")
	}
}

func TestLogger_StartDefault(t *testing.T) {
	logger := &Logger{}
	err := logger.StartDefault()
	if err != nil {
		t.Fatalf("StartDefault error: %v", err)
	}
	logger.StopAndWait()
	if logger.IsActive() {
		t.Error("Logger should not be active after StopAndWait")
	}
}

func TestLogger_SetFallback(t *testing.T) {
	logger := &Logger{}
	buf := &bytes.Buffer{}
	logger.SetFallback(buf)
	logger.handleLogWriteError("fallback test")
	if !bytes.Contains(buf.Bytes(), []byte("fallback test")) {
		t.Error("SetFallback or handleLogWriteError failed")
	}
}

func TestLogger_Log_WriteToAllOutputs(t *testing.T) {
	logger := &Logger{}
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}
	var errBuf bytes.Buffer
	err := logger.Start(DEBUG, 4, &errBuf, buf1, buf2)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	logger.Log(DEBUG, "test message\n")
	logger.StopAndWait()
	if !bytes.Contains(buf1.Bytes(), []byte("test message")) || !bytes.Contains(buf2.Bytes(), []byte("test message")) {
		t.Error("Log not written to all outputs")
	}
}

func TestLogger_Log_ErrorAndPanicHandling(t *testing.T) {
	logger := &Logger{}
	var errBuf bytes.Buffer
	goodBuf := &bytes.Buffer{}
	err := logger.Start(DEBUG, 4, &errBuf, &errorWriter{}, &panicWriter{}, goodBuf)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	logger.Log(DEBUG, "errpanic test\n")
	logger.StopAndWait()
	if !bytes.Contains(goodBuf.Bytes(), []byte("errpanic test")) {
		t.Error("Log not written to working output")
	}
	if !bytes.Contains(errBuf.Bytes(), []byte("error writing log to output")) {
		t.Error("Expected error message in fallback output")
	}
	if !bytes.Contains(errBuf.Bytes(), []byte("panic writing log to output")) {
		t.Error("Expected panic message in fallback output")
	}
}

func TestLogger_Log_Inactive(t *testing.T) {
	logger := &Logger{}
	var errBuf bytes.Buffer
	logger.SetFallback(&errBuf)
	logger.Log(DEBUG, "should not log")
	if !bytes.Contains(errBuf.Bytes(), []byte("logger is not active")) {
		t.Error("Expected error for inactive logger")
	}
}

func TestLogger_SetLogLevel(t *testing.T) {
	logger := &Logger{}
	buf := &bytes.Buffer{}
	var errBuf bytes.Buffer
	err := logger.Start(DEBUG, 4, &errBuf, buf)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	logger.SetLogLevel(ERROR)
	logger.Log(DEBUG, "debug message\n")
	logger.Log(ERROR, "error message\n")
	logger.StopAndWait()
	if bytes.Contains(buf.Bytes(), []byte("debug message")) {
		t.Error("Debug message should not be logged at ERROR level")
	}
	if !bytes.Contains(buf.Bytes(), []byte("error message")) {
		t.Error("Error message should be logged at ERROR level")
	}
}

func TestLogger_ConcurrentLogging(t *testing.T) {
	logger := &Logger{}
	buf := &bytes.Buffer{}
	var errBuf bytes.Buffer
	err := logger.Start(DEBUG, 10, &errBuf, buf)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	const goroutines = 5
	const messagesPerGoroutine = 20
	done := make(chan struct{})
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Log(DEBUG, "msg from goroutine "+string(rune('A'+id))+"\n")
			}
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < goroutines; i++ {
		<-done
	}
	logger.StopAndWait()
	expectedMessages := goroutines * messagesPerGoroutine
	actualMessages := bytes.Count(buf.Bytes(), []byte("msg from goroutine"))
	if actualMessages != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, actualMessages)
	}
}
