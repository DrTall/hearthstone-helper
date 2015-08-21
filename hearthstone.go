package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ActiveState/tail"
	"strings"
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
	hsUsername := flag.String("username", "no-username-specified", "Your battlenet ID (without the #1234).")

	flag.Parse()

	createManaUpdateParser(*hsUsername)
	log, _ := tail.TailFile(*hsLogFile, tail.Config{Follow: true})

	gs := GameState{}
	gs.resetGameState()
	solutionChan := make(chan *DecisionTreeNode)
	seenUsername := false
	var deepestSolution, shortestSolution *DecisionTreeNode
	var abortChan *chan time.Time
	for {
		select {
		case line := <-log.Lines:
			if !seenUsername && strings.Contains(line.Text, *hsUsername) {
				seenUsername = true
			}
			if turnStart, somethingHappened := ParseHearthstoneLogLine(line.Text, &gs); turnStart || somethingHappened {
				if !seenUsername {
					fmt.Println("WARN: Waiting to see --username before looking for solutions.")
					continue
				}
				//fmt.Println("It is the start of turn for:", gs.LastManaAdjustPlayer)
				if abortChan != nil {
					*abortChan <- time.Now()
					abortChan = nil
					deepestSolution = nil
					shortestSolution = nil
				}
				newAbortChan := make(chan time.Time, 1)
				abortChan = &newAbortChan
				go WalkDecisionTree(gs.DeepCopy(), solutionChan, newAbortChan)
			}
		case solution := <-solutionChan:
			if deepestSolution == nil {
				deepestSolution = solution
				shortestSolution = solution
				fmt.Println("INFO: Solution found")
				prettyPrintDecisionTreeNode(solution)
			}
			if len(deepestSolution.Moves) < len(solution.Moves) {
				deepestSolution = solution
				fmt.Println("INFO: Another solution with more BM:")
				prettyPrintDecisionTreeNode(solution)
			}
			if len(shortestSolution.Moves) > len(solution.Moves) {
				shortestSolution = solution
				fmt.Println("INFO: Another solution with fewer steps:")
				prettyPrintDecisionTreeNode(solution)
			}
		}
	}
}
