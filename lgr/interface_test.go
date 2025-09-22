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
			l := InitWithParams(TRACE, io.Discard, tt.outputs...)
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
			l := InitWithParams(TRACE, io.Discard, tt.outputs...)
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
			l := InitWithParams(TRACE, io.Discard, tt.fallback, nil)
			l.SetFallback(tt.fallback)
			assert.Equal(t, tt.wants, l.fallbck)
		})
	}
}

func TestLogger_IsActive(t *testing.T) {
	l := Init()
	rng := 128
	for i := range rng * 2 {
		t.Run(strconv.Itoa(i-rng), func(t *testing.T) {
			l.setState(LoggerState(i - rng))
			assert.Equal(t, l.state == ACTIVE, l.IsActive())
		})
	}
}

func TestLogger_SetLogLevel(t *testing.T) {
	l := Init()
	rng := 255
	for i := range rng {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			l.SetLogLevel(LogLevel(i))
			var res int
			if i < int(_MAX_FOR_CHECKS_ONLY) {
				res = i
			} else {
				res = int(_MAX_FOR_CHECKS_ONLY) - 1
			}
			assert.Equal(t, LogLevel(res), l.level)
		})
	}
}

func FuzzLogger_SetLogLevel(f *testing.F) {
	l := Init()
	f.Fuzz(func(t *testing.T, b uint8) {
		l.SetLogLevel(LogLevel(b))
		res := LogLevel(b)
		if res >= _MAX_FOR_CHECKS_ONLY {
			res = _MAX_FOR_CHECKS_ONLY - 1
		}
		assert.Equal(t, res, l.level)
	})
}
