package lgr

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Parallel_Multithreading(t *testing.T) {
	const (
		_MAXDATALEN_ = 1000 // Max len of message to be logged
		_DATACOUNT_  = 1000 // Number of messages every goroutine/client has to log
		_GOROUTINES_ = 1000 // Number of simultaneous goroutines/clients logging
	)
	type jobType struct {
		clnt *LogClient
		task [_DATACOUNT_]int
		curr int
	}
	type dataType struct {
		byteArr []byte
	}
	var strings [_DATACOUNT_]dataType
	var workers [_GOROUTINES_]jobType
	var wg sync.WaitGroup
	hold := make(chan int)

	//Rand := rand.New(rand.NewSource(0)) // repeatable results
	Rand := rand.New(rand.NewSource(time.Now().UnixNano())) // stochastic

	// Count the size of logger client/worker name (digits in GOROUTINES)
	namesize := 0
	for i := _GOROUTINES_; i > 0; i /= 10 {
		namesize += 1
	}

	// Generate random log data and count total planned output size
	plantotal := 0
	for i := range _DATACOUNT_ {
		//datalen := maxdatalen ---------------------------------------------------------------
		datalen := Rand.Intn(_MAXDATALEN_) + 1 // next string length (no zero-length for better output analysis)
		plantotal += namesize + datalen + 1    // planned log output length (<client_name> + <data> + '\n')
		strings[i].byteArr = make([]byte, datalen)
		for j := range datalen {
			const first, last = 0, 255 // printable: 33..126, letters: 97..122, digits: 48..57, byte-wide: 0..255
			// random code from first to last
			strings[i].byteArr[j] = byte(Rand.Intn(last+1-first)) + first
		}
		//println(fmt.Sprintf("[%2d] %s", i, string(logStrings[i].byteArr)))
	}
	plantotal *= _GOROUTINES_ // Each goroutine/worker logs all strings

	ferr := &FakeWriter{} // fallback - has to be clear after job done
	//out1 := &FakeWriter{} // output with total planned capacity (to awoid slow slice extends)
	out1 := &FakeWriter{make([]byte, 0, plantotal)} // output with total planned capacity (to awoid slice extends, ~5x faster)
	l1 := InitWithParams(LVL_UNKNOWN, ferr, out1)   // create logger with minimal level and desired fallback and output
	l1.SetOutputLevelPrefix(out1, nil, "")          // no level names and delimiters - it's better to test them in other tests

	// Create clients for each goroutine and shaffle log strings order in goroutines/workers jobs
	for i := range _GOROUTINES_ {
		// Each goroutine/worker has own logger client with it's number as the name and minimal log level
		workers[i].clnt = l1.NewClientWithLevel(fmt.Sprintf("%0"+strconv.Itoa(namesize)+"d", i), LVL_UNKNOWN)
		for j, s := range Rand.Perm(_DATACOUNT_) { // shuffle strings order (on step i worker will log strings[task[i]])
			workers[i].task[j] = s // task #j is to log string #s
		}
	}

	// Goroutines
	goWorker := func(n int) {
		defer wg.Done()
		for range hold { // wait until channel is closed (to start all together)
		}
		for i := range _DATACOUNT_ {
			data := &strings[workers[n].task[i]]                   // get data by index from current task
			workers[n].clnt.LogBytes(LVL_UNMASKABLE, data.byteArr) // short log with std delimiter (name:text)
		}
	}
	for i := range _GOROUTINES_ {
		go goWorker(i)
		wg.Add(1)
	}
	l1.Start(_DATACOUNT_ * _GOROUTINES_) // start logger processing with buffer for all expected messages to prevent locks
	close(hold)                          // unhold all goroutines
	wg.Wait()                            // wait all workers/goroutines finished
	l1.StopAndWait()                     // wait for logger have processed all messages

	// Check results
	realtotal := len(out1.buffer)
	assert.Equal(t, plantotal, realtotal, "wrong output total length") // total size of all messages has to be equal to planned
	assert.Empty(t, ferr.buffer, "unexpected fallback errors writes")  // no errors have to be written to fallback

	// Check all log messages are delivered in correct per-client order
	pos := 0
	var name string        // client name (i.e. worker number)
	var workerId int       //worker number
	var taskData *dataType // data had to be written
	var err error

	for pos < realtotal {
		// Get client name (i.e. worker number)
		name = string(out1.buffer[pos : pos+namesize])
		workerId, err = strconv.Atoi(name)
		if err != nil {
			err = fmt.Errorf("Pos %d: client name convertion error (string %s, error %s)", pos, name, err.Error())
			break
		}
		pos += namesize

		// Compare data in next worker's task and output
		worker := &workers[workerId]                    // which client/worker has written this line
		taskData = &(strings[worker.task[worker.curr]]) // which data had to be written according to worker's next task
		tasklen := len(taskData.byteArr)
		if pos+tasklen+1 > realtotal { // current position + data length + \n
			err = fmt.Errorf("Pos %d: no enough data left (client %s, task %d)",
				pos, name, workers[workerId].curr)
			break
		}
		if !bytes.Equal(taskData.byteArr, out1.buffer[pos:pos+tasklen]) {
			err = fmt.Errorf("Pos %d: data not equal (client %s, task %d):\nwanted: %s\ngot%s",
				pos, name, workers[workerId].curr, taskData.byteArr, out1.buffer[pos:pos+tasklen])
			break
		}
		pos += tasklen
		if out1.buffer[pos] != '\n' {
			err = fmt.Errorf("Pos %d: no \\n an the end (client %s, task %d)",
				pos, name, workers[workerId].curr)
			break
		}
		pos += 1
		workers[workerId].curr += 1
	}
	assert.NoError(t, err, "error parsing output")
}
