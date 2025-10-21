package lgr

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_logClient_Lvl(t *testing.T) {
	t.Run("for_255", func(t *testing.T) {
		var lc logClient
		for level := range LogLevel(255) {
			assert.Equal(t, normLevel(level), lc.Lvl(level).curLevel, fmt.Sprintf("Fail on %d", level))
		}
	})
}

func Test_logClient_Write(t *testing.T) {
	outBuffer := bytes.NewBuffer(make([]byte, _DEFAULT_OUT_BUFF))
	ferr := &FakeWriter{}
	out1 := &FakeWriter{}
	l := Init()
	l.SetFallback(ferr)
	lc := l.NewClient(testlogstr, LVL_UNKNOWN)

	prep := func() {
		outBuffer.Reset()
		out1.Clear()
		ferr.Clear()
		l.ClearOutputs()
		l.AddOutputs(out1)
		l.SetOutputLevelColor(out1, LevelColorOnBlackMap).SetOutputLevelPrefix(out1, LevelFullNames, " !delimiter! ")
	}

	for range 5 {
		t.Run("error_out", func(t *testing.T) {
			prep()
			short := "!test!"
			l.AddOutputs(&ErrorWriter{}, &PanicWriter{}, &NilPanicWriter{}, &ZeroPanicWriter{})
			l.SetOutputLevelColor(os.Stdout, LevelColorOnBlackMap).SetOutputLevelPrefix(out1, LevelFullNames, " !delimiter! ")
			l.Start(0)
			n, err := fmt.Fprint(lc.Lvl(LVL_UNMASKABLE), short)
			//n, err := lc.Lvl(LVL_UNMASKABLE).Write([]byte(short))
			assert.NoError(t, err)
			assert.Equal(t, n, len(short))
			l.StopAndWait()
			msg := makeTextMessage(lc, LVL_UNMASKABLE, []byte(short))
			assert.Contains(t, ferr.String(), errorStr+"\n")
			assert.Contains(t, ferr.String(), "`"+panicStr+"`\n")
			assert.Equal(t, 1, strings.Count(ferr.String(), _ERROR_UNKNOWN_PANIC_TEXT+"\n"))
			assert.Equal(t, buildTextMessage(outBuffer, msg, l.outputs[out1]).Bytes(), out1.buffer)
			assert.Equal(t, buildTextMessage(outBuffer, msg, l.outputs[out1]).String(), out1.String())
		})
		t.Run("nil_message", func(t *testing.T) {
			prep()
			l.Start(0)
			lc.Lvl(LVL_UNMASKABLE)
			n, err := lc.Write(nil)
			assert.NoError(t, err)
			assert.Zero(t, n)
			l.StopAndWait()
			assert.Empty(t, ferr.buffer)
			assert.Empty(t, out1.buffer)
		})
		t.Run("full_message", func(t *testing.T) {
			prep()
			l.Start(0)
			n, err := fmt.Fprint(lc.Lvl(LVL_UNMASKABLE), testlogstr)
			assert.NoError(t, err)
			assert.Equal(t, n, len(testbytes))
			l.StopAndWait()
			msg := makeTextMessage(lc, LVL_UNMASKABLE, []byte(testlogstr))
			assert.Empty(t, ferr.buffer)
			assert.Equal(t, buildTextMessage(outBuffer, msg, l.outputs[out1]).Bytes(), out1.buffer)
		})
		t.Run("not_active", func(t *testing.T) {
			prep()
			n, err := fmt.Fprint(lc.Lvl(LVL_UNMASKABLE), testlogstr)
			assert.ErrorContains(t, err, "not active")
			l.Start(0)
			l.StopAndWait()
			assert.Empty(t, out1.buffer)
			assert.Empty(t, ferr.buffer)
			assert.Zero(t, n)
		})
	}
}
