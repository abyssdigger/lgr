package main

import (
	"fmt"
	"io"
	"logger/lgr"
	"os"
	"time"
)

func main() {
	var logger = lgr.InitWithParams(lgr.LVL_DEBUG, os.Stderr, nil) //...Default()
	outs := [...]io.Writer{nil, os.Stdout, nil, os.Stderr, os.Stdout}
	for i := 1; i <= len(outs); i++ {
		logger.Start(32)
		logger.AddOutputs(outs[i-1])
		for j := 0; j < 10; j++ {
			err := logger.LogE(lgr.LVL_DEBUG, "LOG! #"+fmt.Sprint(j+1))
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Logged #")
			}
		}
		fmt.Println("Stopping logger...")
		logger.StopAndWait()
		logger.ClearOutputs()
		fmt.Println("*** FINITA LA COMEDIA #", i, "***")
		time.Sleep(100 * time.Millisecond)
	}
}
