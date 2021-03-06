// Hide all the ugly special card logic out of sight.

package main

import (
	"fmt"
	"strings"
)

var _ = fmt.Printf

// Returns a filter function which returns true/false for whether a given
// card can be the target of this one when it is played.
// If the filter returns true when passed nil, it means the card is
// ALWAYS played without a target. It is up to the caller to note that
// minions requiring targets can be played without a target when none exists.
func getPlayCardTargetFilter(card *Card) func(*Card) bool {
	if filter, ok := specialCardTargetFilters[card.JsonCardId]; ok {
		return filter
	}
	// Do we even know how to play this spell?
	if card.Type == "Spell" {
		if _, ok := GlobalCardPlayedActions[card.JsonCardId]; !ok {
			return func(target *Card) bool { return false }
		}
	}
	return func(target *Card) bool { return true }
}

var specialCardTargetFilters = map[string]func(*Card) bool{
	"EX1_603": targetAnyMinion,                                                             // Cruel Taskmaster
	"CS2_108": func(card *Card) bool { return targetEnemyMinion(card) && card.Damage > 0 }, // Execute
	"EX1_607": targetAnyMinion,                                                             // Inner Rage
	"EX1_391": targetAnyMinion,                                                             // Slam
}

func targetEnemyMinion(card *Card) bool {
	return card != nil && card.Type == "Minion" && card.Zone == "OPPOSING PLAY"
}

func targetAnyMinion(card *Card) bool {
	return card != nil && card.Type == "Minion" && strings.Contains(card.Zone, "PLAY")
}

////////////////////

// All functions that we care about/ know about for when a card (key of the map is JsonId)
// is played with optional target `targetCardId`  The action should modify `gs`
// It is up to the caller to call gs.cleanupState afterwards.
var GlobalCardPlayedActions = map[string]func(gs *GameState, params *MoveParams){
	// "EX1_392": // Battle Rage -- TODO how do we handle card draw?
	"EX1_603": taskmasterAction,                                                                 // Cruel Taskmaster
	"CS2_108": func(gs *GameState, params *MoveParams) { params.CardTwo.PendingDestroy = true }, // Execute
	// "CS2_147": // Gnomish Inventor -- TODO how do we handle card draw?
	"EX1_607": taskmasterAction,                                                             // Inner Rage
	"EX1_391": func(gs *GameState, params *MoveParams) { gs.dealDamage(params.CardTwo, 2) }, // Slam -- TODO how do we handle card draw?
	"GAME_005": func(gs *GameState, params *MoveParams) {
		if gs.ManaMax < 10 || gs.ManaUsed > 0 {
			gs.ManaTemp += 1
		}
	}, // The Coin
	"EX1_400": whirlwindAction, // Whirlwind
}

func taskmasterAction(gs *GameState, params *MoveParams) {
	//fmt.Println("DEBUG: taskmasterAction with target: ", params.CardTwo)
	if params.CardTwo != nil {
		params.CardTwo.Attack += 2
		gs.dealDamage(params.CardTwo, 1)
	}
}

func whirlwindAction(gs *GameState, _ *MoveParams) {
	//total := 0
	for minion := range gs.CardsByZone["FRIENDLY PLAY"] {
		//total += 1
		//fmt.Printf("DEBUG: whirlwindAction sees %v friends\n", len(gs.CardsByZone["FRIENDLY PLAY"]))
		gs.dealDamage(minion, 1)
	}
	for minion := range gs.CardsByZone["OPPOSING PLAY"] {
		//total += 1
		gs.dealDamage(minion, 1)
	}
	//fmt.Printf("DEBUG: whirlwindAction just did %v damage\n", total)
}

func getCardPlayedAction(card *Card) func(gs *GameState, params *MoveParams) {
	if action, ok := GlobalCardPlayedActions[card.JsonCardId]; ok {
		return action
	}
	return func(gs *GameState, params *MoveParams) {}
}

// All deathrattle actions we care about.  The action should modify `gs`
var GlobalDeathrattleActions = map[string]func(gs *GameState, params *MoveParams){
	"FP1_021": whirlwindAction, // Death's Bite
	"FP1_024": whirlwindAction, // Unstable Ghoul
}

func getDeathrattleAction(card *Card) func(gs *GameState, params *MoveParams) {
	if action, ok := GlobalDeathrattleActions[card.JsonCardId]; ok {
		return action
	}
	return func(gs *GameState, params *MoveParams) {}
}
