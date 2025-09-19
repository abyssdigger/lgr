package main

import (
	"fmt"
	"io"
	"logger/lgr"
	"os"
)

func main() {
	var logger = lgr.Init(lgr.DEBUG, os.Stderr, nil) //...Default()
	outs := [...]io.Writer{os.Stdout, nil, os.Stderr}
	for i := 1; i <= 3; i++ {
		logger.Start(32)
		logger.AddOutputs(outs[i-1])
		for j := 0; j < 10; j++ {
			err := logger.Log_(lgr.DEBUG, "LOG! #"+fmt.Sprint(j+1)+"\n")
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
	}
}
