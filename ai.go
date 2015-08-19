// Working from a game state to try to win.

package main

import (
	"fmt"
	"time"
)

// parameters that apply to all moves.  IdTwo is optional.  When it exists, it is the target (of an attack, spell, etc)
type MoveParams struct {
	IdOne       int32
	IdTwo       int32
	Description string
}

type Move struct {
	ApplyMove func(gs *GameState, params *MoveParams) *GameState // Returns a copy of the gamestate
	params    MoveParams
}

type DecisionTreeNode struct {
	Gs                 *GameState
	Moves              []Move
	SuccessProbability float32
}

func getNextMoves(node DecisionTreeNode) []Move {
	return nil
}

func WalkDecisionTree(successChan <-chan *DecisionTreeNode, abortChan <-chan time.Time) {
	fmt.Println("DEBUG: Beginning decision tree walk.")
	workChan := make(chan DecisionTreeNode)
	timeoutChan := time.After(time.Second * 70)
	for {
		select {
		case <-abortChan:
			fmt.Println("DEBUG: Decision tree walk aborting...")
			return
		case <-timeoutChan:
			fmt.Println("DEBUG: Decision tree walk timing out...")
			return
		case node := <-workChan:
			go func() {
				nextMoves := getNextMoves(node) // Does not modify node.
				for _, move := range nextMoves {
					move = move // Long story.
					go func() {
						// Deep copy node.gs
						// Apply move to gs to make new node.
						// If we win, put node into successChan.
						// Else put node into workChan.
					}()
				}
			}()
		}
	}
}
