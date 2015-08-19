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
func getNextMoves(node *DecisionTreeNode) []Move {
	result := make([]Move, 0)

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
			result = append(result,
				Move{nil, MoveParams{IdOne: friendlyMinion.InstanceId, IdTwo: enemyMinion.InstanceId, Description: desc}})
		}
		if !enemyTauntExists {
			// Attack face
			desc := fmt.Sprintf("%v attacking face (%v)", friendlyMinion.Name, enemyHero.Name)
			result = append(result,
				Move{nil, MoveParams{IdOne: friendlyMinion.InstanceId, IdTwo: enemyHero.InstanceId, Description: desc}})
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
			result = append(result,
				Move{nil, MoveParams{IdOne: friendlyHero.InstanceId, IdTwo: enemyMinion.InstanceId, Description: desc}})
		}
		if !enemyTauntExists {
			// Attack face
			desc := fmt.Sprintf("You (%v) attacking face (%v)", friendlyHero.Name, enemyHero.Name)
			result = append(result,
				Move{nil, MoveParams{IdOne: friendlyHero.InstanceId, IdTwo: enemyHero.InstanceId, Description: desc}})
		}
	}

	// Spells, Minions, and Weapons can be played including targets maybe.
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
			descPrefix = fmt.Sprintf("Play %v", cardInHand.Name)
		}
		filter := getPlayCardTargetFilter(cardInHand)
		if filter(nil) {
			result = append(result,
				Move{nil, MoveParams{IdOne: cardInHand.InstanceId, IdTwo: 0, Description: descPrefix}})
		} else {
			couldTargetAny := false
			for _, target := range node.Gs.CardsById {
				if filter(target) {
					couldTargetAny = true
					desc := fmt.Sprintf("%v on %v", descPrefix, target.Name)
					result = append(result,
						Move{nil, MoveParams{IdOne: cardInHand.InstanceId, IdTwo: 0, Description: desc}})
				}
			}
			if !couldTargetAny {
				if cardInHand.Type == "Minion" {
					fmt.Printf("DEBUG: Allowing %v to be played without a target since none exist.\n", cardInHand.Name)
					result = append(result,
						Move{nil, MoveParams{IdOne: cardInHand.InstanceId, IdTwo: 0, Description: descPrefix}})
				} else {
					fmt.Printf("DEBUG: No valid targets for %v.\n", cardInHand.Name)
				}
			}
		}
	}
	return result
}

func WalkDecisionTree(gs *GameState, successChan <-chan *DecisionTreeNode, abortChan <-chan time.Time) {
	fmt.Println("DEBUG: Beginning decision tree walk.")
	workChan := make(chan *DecisionTreeNode, 1000)
	timeoutChan := time.After(time.Second * 70)
	workChan <- &DecisionTreeNode{
		Gs:                 gs,
		Moves:              make([]Move, 0),
		SuccessProbability: 1.0,
	}
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
					localMove := move // Long story.
					go func() {
						fmt.Println("DEBUG: In theory this move is possible...")
						//fmt.Println(move)
						prettyPrint(localMove.Params)
						// Deep copy node.Gs
						// Apply move to gs to make new node.
						// If we win, put node into successChan.
						// Else put node into workChan.
					}()
				}
			}()
		case <-time.After(1 * time.Second):
			fmt.Println("INFO: Analysis complete. You cannot win this turn.")
			return
		}
	}
}
