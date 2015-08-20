// Game state tracking, from either the parser or hypotheticals.

package main

import "fmt"

type GameState struct {
	CardsById            map[int32]*Card
	CardsByZone          map[string]map[*Card]interface{}
	Mana                 int32
	LastManaAdjustPlayer string
	HighestCardId        int32
}

// Can't just use deepcopy.Copy because of CardsByZone's pointer keys.
func (gs *GameState) DeepCopy() *GameState {
	result := GameState{}
	result.resetGameState()
	for id, cardPtr := range gs.CardsById {
		cardCopy := *cardPtr
		result.CardsById[id] = &cardCopy
		result.moveCard(&cardCopy, cardCopy.Zone) // Populate CardsByZone
	}
	result.Mana = gs.Mana
	result.LastManaAdjustPlayer = gs.LastManaAdjustPlayer
	return &result
}

func (gs *GameState) resetGameState() {
	gs.CardsById = make(map[int32]*Card)
	gs.CardsByZone = make(map[string]map[*Card]interface{})
	gs.Mana = 0
	gs.LastManaAdjustPlayer = "NOBODY?!"
	gs.HighestCardId = 0
}

func (gs *GameState) getOrCreateCard(jsonCardId string, instanceId int32) *Card {
	if card, ok := gs.CardsById[instanceId]; ok {
		//fmt.Println("DEBUG: Already knew about card: ", gs.cardsById)
		return card
	}
	//fmt.Println("DEBUG: Creating new card for instance: ", instanceId)
	if instanceId > gs.HighestCardId {
		gs.HighestCardId = instanceId
	}
	result := newCardFromJson(jsonCardId, instanceId)
	gs.CardsById[instanceId] = &result
	return &result
}

// Update the GameState to note that the given card is in a new zone.
// This function does not apply any logic like deathrattles, it simply
// updates the maps on GameState and card.Zone.
func (gs *GameState) moveCard(card *Card, newZone string) {
	if oldZoneCards, ok := gs.CardsByZone[card.Zone]; ok {
		if _, ok := oldZoneCards[card]; ok {
			//fmt.Println("DEBUG: removing card from old zone: ", card.Zone)
			delete(oldZoneCards, card)
		}
	}

	// Warsong Commander
	if newZone == "FRIENDLY PLAY" && card.Attack <= 3 {
		for friendlyMinion := range gs.CardsByZone["FRIENDLY PLAY"] {
			if friendlyMinion.JsonCardId == "EX1_084" && !friendlyMinion.Silenced {
				fmt.Println("DEBUG: Getting charge from Warsong Commander.")
				card.Charge = true
				card.Exhausted = false
				break
			}
		}
	}

	if _, ok := gs.CardsByZone[newZone]; !ok {
		gs.CardsByZone[newZone] = make(map[*Card]interface{})
	}
	card.Zone = newZone
	gs.CardsByZone[card.Zone][card] = nil

	//fmt.Println("BEFORE HELLO!!!", gs)
	//prettyPrint(gs)
}

// -------------------
// "hypothetical" operations on GameState that modify it.
// These can be used as the function `applyMove` in `Move`.
// -------------------

// use a card: either playing from hand, or using hero power
func (gs *GameState) useCard(params *MoveParams) {
	playCard := params.CardOne
	switch playCard.Zone {
	// If played from hand
	case "FRIENDLY HAND":
		switch playCard.Type {
		case "Minion":
			// play minion
			gs.moveCard(playCard, "FRIENDLY PLAY")
			// TODO (dz): execute battlecry
		case "Spell":
			// execute spell
			// TODO (dz): execute spell effect
			gs.moveCard(playCard, "FRIENDLY GRAVEYARD")
		case "Weapon":
			// remove anything currently in weapon zone
			if weapons, exists := gs.CardsByZone["FRIENDLY PLAY (Weapon)"]; exists {
				if len(weapons) > 1 {
					fmt.Println("more than one weapon in play??", weapons)
				}
				for oldWeapon, _ := range weapons {
					gs.moveCard(oldWeapon, "FRIENDLY GRAVEYARD")
				}
			}
			// new weapon to weapon zone
			gs.moveCard(playCard, "FRIENDLY PLAY (Weapon)")
			// TODO (dz): other card types (Enchantment?)
		}
	// If on battlefield, then this is a minion attack.
	case "FRIENDLY PLAY":
		// TODO (dz): is it easier for nextMoves to call use or attack?
		fmt.Println("`useCard` called with a card on the field, using `attack` instead.")
		gs.attack(params)
	// if using hero attack
	case "FRIENDLY PLAY (Hero)":
		fmt.Println("`useCard` called with hero card, using `attack` instead.")
		gs.attack(params)
	// if using hero power
	default:
		fmt.Println("Unrecognized Zone to play a card from: ", playCard.Zone)
	}
}

func (gs *GameState) CreateNewMinion(jsonId string, zone string) {
	card := gs.getOrCreateCard(jsonId, gs.HighestCardId+1)
	gs.moveCard(card, zone)
}

// Minion attack or weapon attack (modifies `gs` and the cards in it).
func (gs *GameState) attack(params *MoveParams) {
	gs.dealDamage(params.CardOne, params.CardTwo.Attack)
	gs.dealDamage(params.CardTwo, params.CardOne.Attack)
	gs.cleanupState()
}

func (gs *GameState) dealDamage(target *Card, amount int32) {
	target.Damage += amount

	// We're not bad people, but we did a bad thing...
	// TODO: Implement some kind of generic listener framework someday.
	if target.JsonCardId == "BRM_019" && !target.Silenced &&
		target.Damage < target.Health && len(gs.CardsByZone[target.Zone]) < 7 { // Grim Patron
		fmt.Println("DEBUG: Everyone! Get in here!")
		gs.CreateNewMinion(target.Zone, "BRM_019")
	}
	for friendlyMinion := range gs.CardsByZone["FRIENDLY PLAY"] {
		if friendlyMinion.JsonCardId == "EX1_604" && !friendlyMinion.Silenced { // Frothing Berserker
			fmt.Println("DEBUG: My blade be thirsty!")
			friendlyMinion.Attack += 1
		}
	}
	for enemyMinion := range gs.CardsByZone["OPPOSING PLAY"] {
		if enemyMinion.JsonCardId == "EX1_604" && !enemyMinion.Silenced { // Frothing Berserker
			fmt.Println("DEBUG: Enemy Frothing Berserker triggered.")
			enemyMinion.Attack += 1
		}
	}
}

// Clean up the gs state, moving cards to their zones, executing deathrattles, etc
func (gs *GameState) cleanupState() {
	// TODO (dz)
}

// A particular instance of a card in the game.
type Card struct {
	InstanceId     int32  // Globally unique.
	JsonCardId     string // Refers to JsonCardData.Id
	Type           string // Refers to JsonCardData.Type
	Name           string // Refers to JsonCardData.Name
	Cost           int32
	Attack         int32
	Health         int32
	Armor          int32
	Damage         int32
	Charge         bool
	Exhausted      bool
	Frozen         bool
	Taunt          bool
	Silenced       bool
	Zone           string
	PendingDestroy bool // Internal. Should this minion be destroyed in the next cleanup step?
}
