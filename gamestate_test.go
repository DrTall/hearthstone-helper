package main

import (
	"fmt"
	"testing"
)

func createEmptyGameState() GameState {
	gs := GameState{}
	gs.resetGameState()
	hero1 := gs.getOrCreateCard("HERO_01", 1)
	hero2 := gs.getOrCreateCard("HERO_02", 2)
	gs.moveCard(hero1, "FRIENDLY PLAY (Hero)")
	gs.moveCard(hero2, "OPPOSING PLAY (Hero)")
	return gs
}

func TestDeepCopy(t *testing.T) {
	gs := createEmptyGameState()
	c := gs.getOrCreateCard("GVG_112", 42)
	gs.moveCard(c, "MY_ZONE")
	c.Health = 2 // Mutate c so we know that it is getting copied.
	prettyPrint(c)
	if c.Cost != 6 || c.Attack != 7 || c.Health != 2 || c.Zone != "MY_ZONE" {
		t.Error()
	}
	gs2 := gs.DeepCopy()
	c2 := gs2.getOrCreateCard("GVG_112", 42)
	prettyPrint(c2)
	if c2.Cost != 6 || c2.Attack != 7 || c2.Health != 2 || c.Zone != "MY_ZONE" {
		t.Error()
	}
	if _, ok := gs2.CardsByZone["MY_ZONE"][c2]; !ok {
		t.Error()
	}
	c.Health = 4 // Mutate c again. c2 had better not get mutated.
	gs.moveCard(c, "ANOTHER_ZONE")
	if _, ok := gs2.CardsByZone["MY_ZONE"][c2]; !ok {
		t.Error()
	}
	if _, ok := gs2.CardsByZone["ANOTHER_ZONE"][c2]; ok {
		t.Error()
	}
	if c2.Health != 2 || c2.Zone != "MY_ZONE" {
		t.Error()
	}
}

func TestMove(t *testing.T) {
	gs := createEmptyGameState()
	c := gs.getOrCreateCard("GVG_112", 42)
	gs.moveCard(c, "FRIENDLY HAND")

	// create a move to test
	moveParam := MoveParams{
		CardOne:     c,
		CardTwo:     nil,
		Description: "test move",
	}
	useCard(&gs, &moveParam)
	fmt.Println(gs)

	// Card should no longer be in hand
	if hand, _ := gs.CardsByZone["FRIENDLY HAND"]; len(hand) != 0 {
		t.Error()
	}
	// Card should now be in play
	if cardsInPlay, _ := gs.CardsByZone["FRIENDLY PLAY"]; len(cardsInPlay) != 1 {
		t.Error()
	}
	if c.Zone != "FRIENDLY PLAY" {
		t.Error()
	}
}

func TestBattlecry(t *testing.T) {
	fmt.Println("---- TestBattlecry starting")
	gs := createEmptyGameState()
	c := gs.getOrCreateCard("EX1_603", 42)
	murloc := gs.getOrCreateCard("EX1_506", 43)
	gs.moveCard(c, "FRIENDLY HAND")
	gs.moveCard(murloc, "FRIENDLY PLAY")

	// create a move to test
	moveParam := MoveParams{
		CardOne:     c,
		CardTwo:     murloc,
		Description: "play CruelTaskmaster from hand, targeting Gnomish Inventor with battlecry",
	}
	useCard(&gs, &moveParam)
	fmt.Println(gs)

	// Card should no longer be in hand
	if hand, _ := gs.CardsByZone["FRIENDLY HAND"]; len(hand) != 0 {
		t.Error("Cards in hand: ", len(hand))
	}
	// Card should now be in play, but the Gnomish Inventor should be dead.
	if cardsInPlay, _ := gs.CardsByZone["FRIENDLY PLAY"]; len(cardsInPlay) != 1 {
		t.Error("Cards in play: ", len(cardsInPlay))
	}
	if c.Zone != "FRIENDLY PLAY" {
		t.Error()
	}
	if murloc.Zone != "FRIENDLY GRAVEYARD" {
		t.Error()
	}
}

func TestDedup(t *testing.T) {
	fmt.Println("---- TestDedup starting")
	gs := createEmptyGameState()
	c1 := gs.getOrCreateCard("EX1_603", 42)
	c2 := gs.getOrCreateCard("EX1_603", 43)
	murlocScout := gs.getOrCreateCard("EX1_506a", 44)
	murlocScout2 := gs.getOrCreateCard("EX1_506a", 45)

	gs.moveCard(c1, "FRIENDLY PLAY")
	gs.moveCard(c2, "FRIENDLY PLAY")
	gs.moveCard(murlocScout, "OPPOSING PLAY")
	gs.moveCard(murlocScout2, "OPPOSING PLAY")

	if len(uniqueCardsInZone(&gs, "FRIENDLY PLAY")) != 1 {
		t.Error()
	}

	// Deal damage to c1, so they have different amounts of damage
	gs.dealDamage(c1, 1)
	if len(uniqueCardsInZone(&gs, "FRIENDLY PLAY")) != 2 {
		t.Error()
	}

	if len(uniqueCardsInZone(&gs, "OPPOSING PLAY")) != 1 {
		t.Error()
	}

	murlocScout.Frozen = true
	if len(uniqueCardsInZone(&gs, "OPPOSING PLAY")) != 2 {
		t.Error()
	}

}
