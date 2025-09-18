package main

import (
	"fmt"
	"logger/lgr"
)

func main() {
	var logger lgr.Logger
	for i := 1; i <= 2; i++ {
		logger.StartDefault()
		for j := 0; j < 10; j++ {
			err := logger.Log(lgr.DEBUG, "LOG! #"+fmt.Sprint(j+1)+"\n")
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
