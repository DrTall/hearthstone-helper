// Working from a game state to try to win.

package main

import (
	"fmt"
	"time"
)

// parameters that apply to all moves.  CardTwo is optional.  When it exists, it is the target (of an attack, spell, etc)
type MoveParams struct {
	CardOne     *Card
	CardTwo     *Card
	Description string
}

type Move struct {
	ApplyMove func(gs *GameState, params *MoveParams) // DeepCopy before calling.
	Params    MoveParams
}

type DecisionTreeNode struct {
	Gs                 *GameState
	Moves              []Move
	SuccessProbability float32
}

// Hack around the fact that you have to iterate to get a map key.
func getSingletonFromZone(zone string, node *DecisionTreeNode) (result *Card) {
	numFound := 0
	for card := range node.Gs.CardsByZone[zone] {
		numFound += 1
		if numFound > 1 {
			fmt.Println("ERROR: Multiple cards found in singleton zone!", zone)
		}
		result = card
	}
	if numFound == 0 {
		fmt.Println("ERROR: Zero cards found in singleton zone!", zone)
	}
	return
}

// Enumerate all of the possible next moves from the given GameState.
// TODO things this function does not currently consider:
//  Immune enemies (these are rare).
//  Hero power (this is not due to complexity but mostly because it probably won't help us win).
func getNextMoves(node *DecisionTreeNode, resultChan chan<- *Move) {
	// Pre-compute some useful stuff.
	friendlyHero := getSingletonFromZone("FRIENDLY PLAY (Hero)", node)
	enemyHero := getSingletonFromZone("OPPOSING PLAY (Hero)", node)
	enemyTauntExists := false
	for enemyMinion := range node.Gs.CardsByZone["OPPOSING PLAY"] {
		if enemyMinion.Taunt {
			enemyTauntExists = true
			break
		}
	}

	// Minions can attack minions or face.
	for friendlyMinion := range node.Gs.CardsByZone["FRIENDLY PLAY"] {
		if friendlyMinion.Exhausted || friendlyMinion.Frozen || friendlyMinion.Attack == 0 {
			// This minion can't attack.
			fmt.Printf("DEBUG: %v is in play but can't attack for some reason.\n", friendlyMinion.Name)
			continue
		}
		for enemyMinion := range node.Gs.CardsByZone["OPPOSING PLAY"] {
			if enemyTauntExists && !enemyMinion.Taunt {
				// This minion can't be attacked.
				fmt.Printf("DEBUG: %v is protected by a taunt minion.\n", enemyMinion.Name)
				continue
			}
			// Attack minion
			desc := fmt.Sprintf("%v attacking %v", friendlyMinion.Name, enemyMinion.Name)
			resultChan <- &Move{nil, MoveParams{CardOne: friendlyMinion, CardTwo: enemyMinion, Description: desc}}
		}
		if !enemyTauntExists {
			// Attack face
			desc := fmt.Sprintf("%v attacking face (%v)", friendlyMinion.Name, enemyHero.Name)
			resultChan <- &Move{nil, MoveParams{CardOne: friendlyMinion, CardTwo: enemyHero, Description: desc}}
		}
	}

	// Hero can attack minions or face with a weapon.
	if friendlyHero.Attack > 0 && !friendlyHero.Exhausted {
		for enemyMinion := range node.Gs.CardsByZone["OPPOSING PLAY"] {
			if enemyTauntExists && !enemyMinion.Taunt {
				// This minion can't be attacked.
				fmt.Printf("DEBUG: %v is protected by a taunt minion.\n", enemyMinion.Name)
				continue
			}
			desc := fmt.Sprintf("You (%v) attacking %v", friendlyHero.Name, enemyMinion.Name)
			resultChan <- &Move{nil, MoveParams{CardOne: friendlyHero, CardTwo: enemyMinion, Description: desc}}
		}
		if !enemyTauntExists {
			// Attack face
			desc := fmt.Sprintf("You (%v) attacking face (%v)", friendlyHero.Name, enemyHero.Name)
			resultChan <- &Move{nil, MoveParams{CardOne: friendlyHero, CardTwo: enemyHero, Description: desc}}
		}
	}

	// Spells, Minions, and Weapons can be played including targets maybe.
	numFriendlyMinions := len(node.Gs.CardsByZone["FRIENDLY PLAY"])
	for cardInHand := range node.Gs.CardsByZone["FRIENDLY HAND"] {
		if cardInHand.Cost > node.Gs.Mana {
			// Too expensive.
			fmt.Printf("DEBUG: %v is too expensive to play.\n", cardInHand.Name)
			continue
		}
		var descPrefix string
		switch cardInHand.Type {
		case "Spell":
			descPrefix = fmt.Sprintf("Cast %v", cardInHand.Name)
		case "Weapon":
			descPrefix = fmt.Sprintf("Equip %v", cardInHand.Name)
		case "Minion":
			if numFriendlyMinions >= 7 {
				fmt.Printf("DEBUG: No space on the board to play %v.\n", cardInHand.Name)
				continue
			}
			descPrefix = fmt.Sprintf("Play %v", cardInHand.Name)
		}
		filter := getPlayCardTargetFilter(cardInHand)
		if filter(nil) {
			resultChan <- &Move{nil, MoveParams{CardOne: cardInHand, CardTwo: nil, Description: descPrefix}}
		} else {
			couldTargetAny := false
			for _, target := range node.Gs.CardsById {
				if filter(target) {
					couldTargetAny = true
					desc := fmt.Sprintf("%v on %v", descPrefix, target.Name)
					resultChan <- &Move{nil, MoveParams{CardOne: cardInHand, CardTwo: nil, Description: desc}}
				}
			}
			if !couldTargetAny {
				if cardInHand.Type == "Minion" {
					fmt.Printf("DEBUG: Allowing %v to be played without a target since none exist.\n", cardInHand.Name)
					resultChan <- &Move{nil, MoveParams{CardOne: cardInHand, CardTwo: nil, Description: descPrefix}}
				} else {
					fmt.Printf("DEBUG: No valid targets for %v.\n", cardInHand.Name)
				}
			}
		}
	}
}

func WalkDecisionTree(gs *GameState, successChan <-chan *DecisionTreeNode, abortChan <-chan time.Time) {
	fmt.Println("DEBUG: Beginning decision tree walk.")
	workChan := make(chan *DecisionTreeNode, 1000)
	timeoutChan := time.After(time.Second * 70)

	// Sleep briefly before kicking off the work, since it will get cancelled
	// very quickly in turns where the human operator knows there's no hope.
	go func() {
		time.Sleep(time.Second)
		workChan <- &DecisionTreeNode{
			Gs:                 gs,
			Moves:              make([]Move, 0),
			SuccessProbability: 1.0,
		}
	}()
	for {
		select {
		case <-abortChan:
			fmt.Println("DEBUG: Decision tree walk aborting...")
			return
		case <-timeoutChan:
			fmt.Println("DEBUG: Decision tree walk timing out...")
			return
		case node := <-workChan:
			movesChan := make(chan *Move, 1000)
			go func() {
				for {
					move := <-movesChan
					fmt.Println("DEBUG: Here's a move: ", move.Params.Description)
					// Deep copy node.Gs
					// Apply move to gs to make new node.
					// If we win, put node into successChan.
					// Else put node into workChan.
				}
			}()
			getNextMoves(node, movesChan) // Does not modify node.
		case <-time.After(5 * time.Second):
			fmt.Println("INFO: Analysis complete. You cannot win this turn.")
			return
		}
	}
}
