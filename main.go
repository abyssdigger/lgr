package main

import (
	"fmt"
	"io"
	"logger/lgr"
	"os"
)

func main() {
	var logger lgr.Logger
	outs := [...]io.Writer{os.Stdout, nil, os.Stderr}
	for i := 1; i <= 3; i++ {
		logger.Start(lgr.DEBUG, 32, os.Stderr, nil) //Default()
		logger.AddOutput(outs[i-1])
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
		fmt.Println("*** FINITA LA COMEDIA #", i, "***")
	}
}
