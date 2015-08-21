package main

import (
	//"fmt"
	"testing"
	"time"
)

func getAllCardsInZone(gs *GameState, zone string) []*Card {
	result := make([]*Card, 0)
	for card, _ := range gs.CardsByZone[zone] {
		result = append(result, card)
	}
	return result
}

func SetupNoPruning() {
	GlobalPruningOpts = PruningOpts{
		getCardsFromFriendlyZone: getAllCardsInZone,
		getCardsInOpposingPlay:   func(gs *GameState) []*Card { return getAllCardsInZone(gs, "OPPOSING PLAY") },
	}
}

func BenchmarkAllToFaceNoPruning(t *testing.B) {
	resetGlobalPruningOpts()
	SetupNoPruning()
	AllToFaceTest()
}

func BenchmarkAllToFace(t *testing.B) {
	resetGlobalPruningOpts()
	AllToFaceTest()
}

func BenchmarkComboInHand(t *testing.B) {
	resetGlobalPruningOpts()
	ComboInHandTest()
}

func BenchmarkComboInHandNoPruning(t *testing.B) {
	resetGlobalPruningOpts()
	SetupNoPruning()
	ComboInHandTest()
}

func AllToFaceTest() {
	abortChan := make(chan time.Time, 1)
	gs := createEmptyGameState()
	enemyHero := getSingletonFromZone(&gs, "OPPOSING PLAY (Hero)", true)
	enemyHero.Damage = 12
	gs.CreateNewMinion("EX1_084", "FRIENDLY PLAY") // Warsong Commander
	// Grim Patrons
	gs.CreateNewMinion("BRM_019", "FRIENDLY PLAY")
	gs.CreateNewMinion("BRM_019", "FRIENDLY PLAY")
	gs.CreateNewMinion("BRM_019", "FRIENDLY PLAY")
	gs.CreateNewMinion("BRM_019", "FRIENDLY PLAY")
	gs.CreateNewMinion("BRM_019", "FRIENDLY PLAY")
	gs.CreateNewMinion("BRM_019", "FRIENDLY PLAY")

	solutionChan := make(chan *DecisionTreeNode)
	go WalkDecisionTree(&gs, solutionChan, abortChan)
	solution := <-solutionChan
	abortChan <- time.Now()
	prettyPrintDecisionTreeNode(solution)
}

func ComboInHandTest() {
	abortChan := make(chan time.Time, 1)
	gs := createEmptyGameState()
	gs.ManaMax = 10
	gs.CreateNewMinion("EX1_084", "FRIENDLY HAND") // Warsong Commander
	//gs.CreateNewMinion("BRM_019", "FRIENDLY HAND") // Grim Patron
	patron := gs.CreateNewMinion("BRM_019", "FRIENDLY HAND") // Grim Patron
	patron.Cost = 0
	gs.CreateNewMinion("EX1_604", "FRIENDLY HAND") // Frothing Berserker
	//frothing.Exhausted = false
	gs.CreateNewMinion("EX1_400", "FRIENDLY HAND") // Whirlwind
	gs.CreateNewMinion("EX1_400", "FRIENDLY HAND") // Whirlwind

	gs.CreateNewMinion("GVG_060", "OPPOSING PLAY")         // Quartermaster
	gs.CreateNewMinion("GVG_122", "OPPOSING PLAY")         // Wee Spellstopper
	gs.CreateNewMinion("DS1_178", "OPPOSING PLAY")         // Tundra Rhino
	gs.CreateNewMinion("BRM_016", "OPPOSING PLAY")         // Axe Flinger
	woat := gs.CreateNewMinion("CS2_052", "OPPOSING PLAY") // Wrath of Air Totem as 2/5
	woat.Attack = 2
	woat.Health = 7
	woat.Damage = 2
	gs.CreateNewMinion("EX1_584", "OPPOSING PLAY")            // Ancient Mage
	maexxna := gs.CreateNewMinion("FP1_010", "OPPOSING PLAY") // Maexxna (easy for her to end up as a 2/5)
	maexxna.Silenced = true

	solutionChan := make(chan *DecisionTreeNode)
	go WalkDecisionTree(&gs, solutionChan, nil)
	solution := <-solutionChan
	abortChan <- time.Now()
	prettyPrintDecisionTreeNode(solution)
}
