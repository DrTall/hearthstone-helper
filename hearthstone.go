package main

import (
	"flag"
	"fmt"
	"github.com/ActiveState/tail"
	"time"
)

func main() {
	hsLogFile := flag.String("log", "no-log-file-specified", "The file path to the Hearthstone log file.")

	flag.Parse()

	log, _ := tail.TailFile(*hsLogFile, tail.Config{Follow: true})

	gs := GameState{}
	gs.resetGameState()
	var abortChan *chan time.Time
	successChan := make(chan *DecisionTreeNode)
	for {
		select {
		case line := <-log.Lines:
			if turnStart, somethingHappened := ParseHearthstoneLogLine(line.Text, &gs); turnStart && abortChan == nil {
				fmt.Println("It is the start of turn for:", gs.LastManaAdjustPlayer)
				newAbortChan := make(chan time.Time, 1)
				abortChan = &newAbortChan
				go WalkDecisionTree(successChan, newAbortChan)
			} else if somethingHappened && abortChan != nil {
				*abortChan <- time.Now()
				abortChan = nil
			}
		case solution := <-successChan:
			fmt.Println("DEBUG: Found a solution!")
			prettyPrint(*solution)
		}
	}
}
