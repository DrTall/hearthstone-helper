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
// This applies only to spells & enchantments (TODO (dz): verify)
var GlobalCardPlayedActions = map[string]func(gs *GameState, params *MoveParams){
// TODO fill in
}

// All deathrattle actions we care about.  The action should modify `gs`
var GlobalDeathrattleActions = map[string]func(gs *GameState, params *MoveParams){
// TODO fill in
}
