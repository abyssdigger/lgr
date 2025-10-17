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
	var alter1 = *os.Stdout
	var alter2 = *os.Stdout
	res1 := &alter1
	res2 := &alter2
	outs := [...]io.Writer{nil, res1, os.Stdout, res2, os.Stderr}
	for i := 1; i <= len(outs); i++ {
		logger.Start(32)
		logger.AddOutputs(outs[i-1])
		logger.SetOutputLevelPrefix(os.Stderr, lgr.LevelShortNames, "\t")
		logger.SetOutputLevelPrefix(res1, lgr.LevelFullNames, " --> ")
		logger.SetOutputLevelColor(res1, lgr.LevelColorOnBlackMap)
		logger.SetOutputLevelPrefix(res2, lgr.LevelShortNames, "|")
		logger.SetOutputLevelColor(os.Stdout, lgr.LevelColorOnBlackMap)
		logger.SetOutputTimeFormat(res1, "2006-01-02 15:04:05 ")
		logger.SetOutputTimeFormat(os.Stderr, "2006-01-02 15:04:05 ")
		logger.ShowOutputLevelCode(os.Stderr)
		lclient1 := logger.NewClient("<Тестовое имя Name>", lgr.LVL_UNMASKABLE+1)
		lclient2 := logger.NewClient("^china 你好 прочая^", lgr.LVL_UNMASKABLE+1)
		for j := range lgr.LogLevel(lgr.LVL_UNMASKABLE + 1 + 1) {
			//_, err := lclient1.Log_with_err(j, "LOG! #"+fmt.Sprint(j+1))
			_, err := fmt.Fprint(lclient1.Lvl(j), "LOG! #"+fmt.Sprint(j+1))
			if err != nil {
				fmt.Println("Error1:", err)
			} else {
				_, err := lclient2.Log_with_err(j, "ЛОГ? №"+fmt.Sprint(j+1))
				if err != nil {
					fmt.Println("Error1:", err)
				} else {
					fmt.Println("Logged #")
				}
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
