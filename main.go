package main

import (
	"fmt"
	"io"
	"logger/lgr"
	"os"
	"time"
)

func st1() {
	var logger = lgr.InitWithParams(lgr.LVL_DEBUG, os.Stderr, nil) //...Default()
	outs := [...]io.Writer{nil, os.Stdout, nil, os.Stderr, os.Stdout}
	for i := 1; i <= len(outs); i++ {
		logger.Start(32)
		logger.AddOutputs(outs[i-1])
		lclient := logger.NewClient("", lgr.LVL_UNMASKABLE)
		for j := 0; j < 10; j++ {
			err := lclient.LogE(lgr.LVL_DEBUG, "LOG! #"+fmt.Sprint(j+1))
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

func st2() {
	var logger = lgr.InitAndStart(0) //...Default()
	logger.SetMinLevel(lgr.LVL_UNKNOWN)
	defer logger.StopAndWait()
	c := logger.NewClient("A", lgr.LVL_UNMASKABLE)
	for i := lgr.LVL_UNKNOWN; i <= lgr.LVL_UNMASKABLE; i++ {
		c.Log(i, "<test>")
	}
	c.LogWarn("<test>")
}

const stage = 2

func main() {
	switch stage {
	case 1:
		st1()
	case 2:
		st2()
	}
	fmt.Println("\033[1m", "bold", "\033[0m", "\033[9m", "strike", "\033[0m", "\033[3m", "italic", "\033[0m")

	fmt.Println("\033[37m", "white", "\033[1m", "\033[0;33m", "cYellow", "\033[0m", "\033[3;90m", "Gray", "\033[0m", "\033[1;90m", "Gray", "\033[0m")
}
