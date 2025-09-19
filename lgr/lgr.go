package lgr

import (
	"fmt"
	"io"
	"os"
	"sync"
)

type LoggerState int8

type logNone struct{}

const (
	STOPPED  LoggerState = 0
	ACTIVE   LoggerState = 1
	STOPPING LoggerState = -1
)

type LogLevel uint8

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

const (
	DEFAULT_BUFF_SIZE = 32
	DEFAULT_LOG_LEVEL = INFO
)

type logMessage struct {
	message string
}

type outType io.Writer
type outList map[outType]bool

type Logger struct {
	sync struct {
		statMtx sync.RWMutex
		outsMtx sync.RWMutex
		chngMtx sync.RWMutex
		waitEnd sync.WaitGroup
	}
	//clients map[string]loggerClient
	outputs outList
	fallbck outType
	channel chan logMessage
	state   LoggerState
	level   LogLevel
}

/*type loggerClient struct {
	prefix   string
	postfix  string
	maxLevel LogLevel
}

func (l *Logger) NewClient(prefix, postfix string, maxLevel LogLevel) *loggerClient {
	client := &loggerClient{
		prefix:   prefix,
		postfix:  postfix,
		maxLevel: maxLevel,
	}
	l.chngMtx.Lock()
	defer l.chngMtx.Unlock()

	return client
}

func (l *Logger) RemoveClient(*loggerClient) {
	l.chngMtx.Lock()
	defer l.chngMtx.Unlock()


}
*/

func (l *Logger) handleLogWriteError(errormsg string) {
	l.sync.outsMtx.RLock()
	defer l.sync.outsMtx.RUnlock()
	fmt.Fprintln(l.fallbck, errormsg)
}

func (l *Logger) setState(newstate LoggerState) {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	l.state = newstate
}

func (l *Logger) Log_(level LogLevel, s string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic [%v]", r)
		}
	}()
	if !l.IsActive() {
		return fmt.Errorf("logger is not active")
	}
	if level >= l.level {
		l.sync.statMtx.RLock()
		if l.IsActive() {
			l.channel <- logMessage{s}
		}
		l.sync.statMtx.RUnlock()
	}
	return err
}

func (l *Logger) Log(level LogLevel, s string) {
	err := l.Log_(level, s)
	if err != nil {
		l.handleLogWriteError(err.Error())
	}
}

func (l *Logger) IsActive() bool {
	return l.state == ACTIVE
}

func (l *Logger) SetFallback(f outType) {
	l.sync.outsMtx.Lock()
	defer l.sync.outsMtx.Unlock()
	if f != nil {
		l.fallbck = f
	} else {
		l.fallbck = io.Discard
	}
}

func (l *Logger) SetLogLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) operateOutputs(slice []outType, operation func(m outList, k outType)) {
	if len(slice) == 0 {
		return
	}
	l.sync.outsMtx.Lock()
	defer l.sync.outsMtx.Unlock()
	for _, output := range slice {
		if output != nil {
			operation(l.outputs, output)
		}
	}
}

func (l *Logger) AddOutputs(outputs ...outType) {
	l.operateOutputs(outputs, func(m outList, k outType) { m[k] = true })
}

func (l *Logger) ClearOutputs() {
	l.sync.outsMtx.Lock()
	defer l.sync.outsMtx.Unlock()
	l.outputs = outList{}
	/*for o := range l.outputs {
		delete(l.outputs, o)
	}*/
}

func (l *Logger) RemoveOutputs(outputs ...outType) {
	if len(outputs) == 0 {
		return
	}
	l.sync.outsMtx.Lock()
	defer l.sync.outsMtx.Unlock()
	for _, output := range outputs {
		delete(l.outputs, output)
	}
}

func (l *Logger) Start(level LogLevel, buffsize uint, fallback outType, outputs ...outType) error {
	l.sync.statMtx.Lock()
	defer l.sync.statMtx.Unlock()
	if l.IsActive() {
		return fmt.Errorf("logger is allready started")
	}
	l.SetFallback(fallback)
	l.channel = make(chan logMessage, buffsize)
	l.level = level
	l.ClearOutputs()
	l.AddOutputs(outputs...)
	l.state = ACTIVE
	l.sync.waitEnd.Go(func() { l.procced() })
	return nil
}

func (l *Logger) StartDefault() error {
	return l.Start(DEFAULT_LOG_LEVEL, DEFAULT_BUFF_SIZE, os.Stderr, os.Stdout)
}

func (l *Logger) Stop() {
	l.setState(STOPPING)
	close(l.channel)
}

func (l *Logger) Wait() {
	l.sync.waitEnd.Wait()
}

func (l *Logger) StopAndWait() {
	l.Stop()
	l.Wait()
}

func (l *Logger) writeMsg(msg *logMessage) {
	l.sync.outsMtx.RLock()
	defer l.sync.outsMtx.RUnlock()
	for output, enabled := range l.outputs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					l.handleLogWriteError(fmt.Sprintf("panic writing log to output `%v`: %v\n", output, r))
				}
			}()
			if enabled && output != nil {
				n, err := output.Write([]byte(msg.message))
				if err != nil {
					l.handleLogWriteError(fmt.Sprintf("error writing log to output `%v` (%d bytes written): %v\n", output, n, err))
				}
			}
		}()
	}
}

func (l *Logger) procced() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(l.fallbck, "panic proceeding log: %v\n", r)
		}
	}()
	for {
		msg, ok := <-l.channel
		if !ok {
			l.setState(STOPPED)
			return
		}
		l.writeMsg(&msg)
	}
}
