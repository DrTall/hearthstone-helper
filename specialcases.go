// Hide all the ugly special card logic out of sight.

package main

import "strings"

// Returns a filter function which returns true/false for whether a given
// card can be the target of this one when it is played.
// If the filter returns true when passed nil, it means the card is
// ALWAYS played without a target. It is up to the caller to note that
// minions requiring targets can be played without a target when none exists.
func getPlayCardTargetFilter(card *Card) func(*Card) bool {
	if filter, ok := specialCardTargetFilters[card.JsonCardId]; ok {
		return filter
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
	"EX1_603": func(gs *GameState, params *MoveParams) {
		if params.CardTwo != nil {
			params.CardTwo.Attack += 2
			gs.dealDamage(params.CardTwo, 1)
		}
	}, // Cruel Taskmaster
	"CS2_108": func(gs *GameState, params *MoveParams) { params.CardOne.PendingDestroy = true }, // Execute
	// "CS2_147": // Gnomish Inventor -- TODO how do we handle card draw?
	"EX1_607": func(gs *GameState, params *MoveParams) { params.CardOne.Attack += 2; gs.dealDamage(params.CardOne, 1) }, // Inner Rage
	"EX1_391": func(gs *GameState, params *MoveParams) { gs.dealDamage(params.CardOne, 2) },                             // Slam -- TODO how do we handle card draw?
	"EX1_400": whirlwindAction,                                                                                          // Whirlwind
}

func whirlwindAction(gs *GameState, _ *MoveParams) {
	for minion := range gs.CardsByZone["FRIENDLY PLAY"] {
		gs.dealDamage(minion, 1)
	}
	for minion := range gs.CardsByZone["OPPOSING PLAY"] {
		gs.dealDamage(minion, 1)
	}
}

func getCardPlayedAction(card *Card) func(gs *GameState, params *MoveParams) {
	if action, ok := GlobalCardPlayedActions[card.JsonCardId]; ok {
		return action
	}
	return func(gs *GameState, params *MoveParams) {}
}

// All deathrattle actions we care about.  The action should modify `gs`
var GlobalDeathrattleActions = map[string]func(gs *GameState, params *MoveParams){
// TODO fill in
}

func getDeathrattleAction(card *Card) func(gs *GameState, params *MoveParams) {
	if action, ok := GlobalDeathrattleActions[card.JsonCardId]; ok {
		return action
	}
	return func(gs *GameState, params *MoveParams) {}
}
