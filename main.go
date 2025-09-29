package main

import (
	"fmt"
	"io"
	"logger/lgr"
	"os"
	"time"
)

func st1() {
	var logger = lgr.InitWithParams(lgr.LVL_UNKNOWN, os.Stderr, nil) //...Default()
	var alter = *os.Stdout
	res := lgr.Writerval(&alter)
	outs := [...]io.Writer{nil, res, os.Stdout, nil, os.Stderr}
	for i := 1; i <= len(outs); i++ {
		logger.Start(32)
		logger.AddOutputs(outs[i-1])
		logger.SetLevelPrefix(os.Stderr, &lgr.LevelShortNames, ": ")
		logger.SetLevelPrefix(res, &lgr.LevelFullNames, " --> ")
		logger.SetLevelColor(os.Stdout, &lgr.ColorOnBlackMap)
		logger.SetTimeFormat(res, "2006-01-02 15:04:05 ")
		logger.SetTimeFormat(os.Stderr, "2006-01-02 15:04:05 ")
		logger.SetShowLevelNum(os.Stderr)
		lclient := logger.NewClient("", lgr.LVL_UNMASKABLE+1)
		for j := range lgr.LogLevel(lgr.LVL_UNMASKABLE + 1 + 1) {
			err := lclient.LogE(j, "LOG! #"+fmt.Sprint(j+1))
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

const stage = 1

func main() {
	switch stage {
	case 1:
		st1()
	case 2:
		st2()
	}
}
