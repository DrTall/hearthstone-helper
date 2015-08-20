package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ActiveState/tail"
	"time"
)

func prettyPrint(x interface{}) {
	json, _ := json.MarshalIndent(x, "", "  ")
	fmt.Println(string(json))
}

func prettyPrintDecisionTreeNode(node *DecisionTreeNode) {
	if node == nil {
		fmt.Println("nil")
		return
	}
	for i, move := range node.Moves {
		fmt.Printf("%v.  %v\n", i+1, move.Description)
	}
}

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
				go WalkDecisionTree(gs.DeepCopy(), successChan, newAbortChan)
			} else if somethingHappened && abortChan != nil {
				*abortChan <- time.Now()
				abortChan = nil
			}
		case solution := <-successChan:
			var buffer bytes.Buffer
			for _, move := range solution.Moves {
				buffer.WriteString(move.Description + "\n")
			}
			fmt.Println(buffer.String())
		}
	}
}
