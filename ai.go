// Working from a game state to try to win.

package main

import (
	"fmt"
	"strings"
	"time"
)

// parameters that apply to all moves.  CardTwo is optional.  When it exists, it is the target (of an attack, spell, etc)
type MoveParams struct {
	CardOne     *Card
	CardTwo     *Card
	Description string
}

type DecisionTreeNode struct {
	Gs                 *GameState
	Moves              []*MoveParams
	SuccessProbability float32
}

func getPrettyCardDesc(card *Card, careAboutCost bool) string {
	changes := make([]string, 0)

	jsonCard := GlobalCardJsonData[card.JsonCardId]
	switch card.Type {
	case "Minion":
		if jsonCard.Health != card.Health || jsonCard.Attack != card.Attack || card.Damage != 0 {
			life := card.Health - card.Damage
			changes = append(changes, fmt.Sprintf("now a %v/%v", card.Attack, life))
		}
	case "Hero":
		if card.Damage != 0 || card.Armor != 0 {
			life := card.Health - card.Damage
			if card.Armor != 0 {
				changes = append(changes, fmt.Sprintf("at %v life with %v armor", life, card.Armor))
			} else {
				changes = append(changes, fmt.Sprintf("at %v life", life))
			}
		}
	}
	if careAboutCost {
		if jsonCard.Cost > card.Cost {
			changes = append(changes, fmt.Sprintf("cost reduced to %v", card.Cost))
		} else if jsonCard.Cost < card.Cost {
			changes = append(changes, fmt.Sprintf("cost increased to %v", card.Cost))
		}
	}
	if card.Charge {
		changes = append(changes, "with charge")
	}
	if len(changes) > 0 {
		return fmt.Sprintf("%v (%v)", card.Name, strings.Join(changes, " "))
	}
	return card.Name
}

// Hack around the fact that you have to iterate to get a map key.
func getSingletonFromZone(gs *GameState, zone string, mustExist bool) (result *Card) {
	numFound := 0
	for card := range gs.CardsByZone[zone] {
		numFound += 1
		if numFound > 1 {
			panic(fmt.Sprintf("ERROR: Multiple cards found in singleton zone: %v", zone))
		}
		result = card
	}
	if numFound == 0 && mustExist {
		fmt.Printf("ERROR: Zero cards found in singleton zone: %v\n", zone)
	}
	return
}

func translateMoveToGs(gs *GameState, move *MoveParams) {
	if move.CardOne != nil {
		if card, ok := gs.CardsById[move.CardOne.InstanceId]; ok {
			move.CardOne = card
		} else {
			panic("There is some kind of serious bug in DeepCopy")
		}
	}
	if move.CardTwo != nil {
		if card, ok := gs.CardsById[move.CardTwo.InstanceId]; ok {
			move.CardTwo = card
		} else {
			panic("There is some kind of serious bug in DeepCopy")
		}
	}
}

func generateNode(node *DecisionTreeNode, move *MoveParams) *DecisionTreeNode {
	//fmt.Println("DEBUG: Testing out a move:", move.Description)
	newGs := node.Gs.DeepCopy()
	translateMoveToGs(newGs, move)
	useCard(newGs, move)
	newMoves := make([]*MoveParams, len(node.Moves)+1)
	copy(newMoves, node.Moves)
	newMoves[len(newMoves)-1] = move
	return &DecisionTreeNode{
		Gs:                 newGs,
		Moves:              newMoves,
		SuccessProbability: 1.0, // TODO: Actually think about this.
	}
}

func canCardAttack(card *Card) bool {
	return !(card.NumAttacksThisTurn > 0 || (card.Exhausted && !card.Charge) || card.Frozen || card.Attack == 0)
}

// Enumerate all of the possible next moves from the given GameState.
// TODO things this function does not currently consider:
//  Immune enemies (these are rare).
//  Hero power (this is not due to complexity but mostly because it probably won't help us win).
func generateNextNodes(node *DecisionTreeNode, workChan chan<- *DecisionTreeNode) {
	// Pre-compute some useful stuff.
	friendlyHero := getSingletonFromZone(node.Gs, "FRIENDLY PLAY (Hero)", true)
	if friendlyHero == nil {
		return
	}
	enemyHero := getSingletonFromZone(node.Gs, "OPPOSING PLAY (Hero)", true)
	enemyTauntExists := false
	for enemyMinion := range node.Gs.CardsByZone["OPPOSING PLAY"] {
		if enemyMinion.Taunt {
			enemyTauntExists = true
			break
		}
	}

	// Minions can attack minions or face.
	for friendlyMinion := range node.Gs.CardsByZone["FRIENDLY PLAY"] {
		if !canCardAttack(friendlyMinion) {
			// This minion can't attack.
			//fmt.Printf("DEBUG: %v is in play but can't attack for some reason.\n", friendlyMinion.Name)
			continue
		}
		for enemyMinion := range node.Gs.CardsByZone["OPPOSING PLAY"] {
			if enemyTauntExists && !enemyMinion.Taunt {
				// This minion can't be attacked.
				//fmt.Printf("DEBUG: %v is protected by a taunt minion.\n", enemyMinion.Name)
				continue
			}
			// Attack minion
			desc := fmt.Sprintf("%v attacks %v", getPrettyCardDesc(friendlyMinion, false), getPrettyCardDesc(enemyMinion, false))
			workChan <- generateNode(node, &MoveParams{CardOne: friendlyMinion, CardTwo: enemyMinion, Description: desc})
		}
		if !enemyTauntExists {
			// Attack face
			desc := fmt.Sprintf("%v attacks face (%v)", getPrettyCardDesc(friendlyMinion, false), getPrettyCardDesc(enemyHero, false))
			workChan <- generateNode(node, &MoveParams{CardOne: friendlyMinion, CardTwo: enemyHero, Description: desc})
		}
	}

	// Hero can attack minions or face with a weapon.
	if canCardAttack(friendlyHero) {
		for enemyMinion := range node.Gs.CardsByZone["OPPOSING PLAY"] {
			if enemyTauntExists && !enemyMinion.Taunt {
				// This minion can't be attacked.
				//fmt.Printf("DEBUG: %v is protected by a taunt minion.\n", getPrettyCardDesc(enemyMinion)
				continue
			}
			desc := fmt.Sprintf("You (%v) attack %v", getPrettyCardDesc(friendlyHero, false), getPrettyCardDesc(enemyMinion, false))
			workChan <- generateNode(node, &MoveParams{CardOne: friendlyHero, CardTwo: enemyMinion, Description: desc})
		}
		if !enemyTauntExists {
			// Attack face
			desc := fmt.Sprintf("You (%v) attack face (%v)", getPrettyCardDesc(friendlyHero, false), getPrettyCardDesc(enemyHero, false))
			workChan <- generateNode(node, &MoveParams{CardOne: friendlyHero, CardTwo: enemyHero, Description: desc})
		}
	}

	// Spells, Minions, and Weapons can be played including targets maybe.
	numFriendlyMinions := len(node.Gs.CardsByZone["FRIENDLY PLAY"])
	availableMana := node.Gs.ManaMax - node.Gs.ManaUsed + node.Gs.ManaTemp
	for cardInHand := range node.Gs.CardsByZone["FRIENDLY HAND"] {
		if cardInHand.Cost > availableMana {
			// Too expensive.
			//fmt.Printf("DEBUG: %v is too expensive to play.\n", getPrettyCardDesc(cardInHand)
			continue
		}
		var descPrefix string
		switch cardInHand.Type {
		case "Spell":
			descPrefix = fmt.Sprintf("Cast %v", getPrettyCardDesc(cardInHand, true))
		case "Weapon":
			descPrefix = fmt.Sprintf("Equip %v", getPrettyCardDesc(cardInHand, true))
		case "Minion":
			if numFriendlyMinions >= 7 {
				//fmt.Printf("DEBUG: No space on the board to play %v.\n", getPrettyCardDesc(cardInHand)
				continue
			}
			descPrefix = fmt.Sprintf("Play %v", getPrettyCardDesc(cardInHand, true))
		}
		filter := getPlayCardTargetFilter(cardInHand)
		if filter(nil) {
			workChan <- generateNode(node, &MoveParams{CardOne: cardInHand, CardTwo: nil, Description: descPrefix})
		} else {
			couldTargetAny := false
			for _, target := range node.Gs.CardsById {
				if filter(target) {
					couldTargetAny = true
					desc := fmt.Sprintf("%v on %v", descPrefix, getPrettyCardDesc(target, false))
					workChan <- generateNode(node, &MoveParams{CardOne: cardInHand, CardTwo: target, Description: desc})
				}
			}
			if !couldTargetAny {
				if cardInHand.Type == "Minion" {
					//fmt.Printf("DEBUG: Allowing %v to be played without a target since none exist.\n", getPrettyCardDesc(cardInHand)
					workChan <- generateNode(node, &MoveParams{CardOne: cardInHand, CardTwo: nil, Description: descPrefix})
				} else {
					//fmt.Printf("DEBUG: No valid targets for %v.\n", getPrettyCardDesc(cardInHand)
				}
			}
		}
	}
}

func WalkDecisionTree(gs *GameState, solutionChan chan<- *DecisionTreeNode, abortChan <-chan time.Time) {
	workChan := make(chan *DecisionTreeNode, 1000)
	softTimeoutChan := time.After(time.Second * 70)
	timeoutChan := time.After(time.Second * 300)

	// Sleep briefly before kicking off the work, since it will get cancelled
	// very quickly in turns where the human operator knows there's no hope.
	go func() {
		time.Sleep(time.Second)
		workChan <- &DecisionTreeNode{
			Gs:                 gs,
			Moves:              make([]*MoveParams, 0),
			SuccessProbability: 1.0,
		}
	}()
	var totalNodes, maxDepth int
	var deepestNode *DecisionTreeNode
	anySolution := false
	defer func() {
		fmt.Printf("INFO: WalkDecisionTree exited after considering %v nodes with maxDepth %v.\n", totalNodes, maxDepth)
		if !anySolution && totalNodes > 0 {
			fmt.Println("Sorry you didn't win. Here is the deepest node discovered:")
			prettyPrintDecisionTreeNode(deepestNode)
		}
	}()
	for {
		select {
		case <-abortChan:
			//fmt.Println("DEBUG: Decision tree walk aborting...")
			return
		case <-timeoutChan:
			fmt.Println("DEBUG: Decision tree walk timing out...")
			return
		case <-softTimeoutChan:
			fmt.Println("WARN: It's been 70 seconds now.")
		case node := <-workChan:
			if totalNodes == 0 {
				fmt.Println("DEBUG: Beginning decision tree walk.")
			}
			totalNodes += 1
			if totalNodes%100000 == 0 {
				fmt.Printf("DEBUG: Seen %v nodes so far.\n", totalNodes)
			}
			depth := len(node.Moves)
			if depth > maxDepth {
				fmt.Printf("DEBUG: New depth reached: %v. Seen %v nodes so far.\n", depth, totalNodes)
				maxDepth = depth
				deepestNode = node
			}
			//fmt.Printf("DEBUG: Working on a new node. It is %v levels deep.\n", len(node.Moves))
			/*if len(node.Moves) > 100 {
			  out := ""
			  for _, move := range node.Moves {
			    out += move.Description + "~"
			  }
			  fmt.Printf("FATAL: %v", out)
			  return
			}*/
			switch node.Gs.Winner {
			case FRIENDLY_VICTORY:
				anySolution = true
				solutionChan <- node
			case NO_VICTORY:
				go generateNextNodes(node, workChan)
			case OPPOSING_VICTORY_OR_DRAW:
				// Do nothing
			default:
				panic("Unknown Winner state")
			}
		case <-time.After(5 * time.Second):
			fmt.Println("INFO: Analysis complete.")
			return
		}
	}
}
