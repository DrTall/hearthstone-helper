// Game state tracking, from either the parser or hypotheticals.

package main

import "fmt"

// Who won this game?
const (
	NO_VICTORY = iota
	FRIENDLY_VICTORY
	OPPOSING_VICTORY_OR_DRAW
)

type GameState struct {
	CardsById            map[int32]*Card
	CardsByZone          map[string]map[*Card]interface{}
	Mana                 int32
	LastManaAdjustPlayer string
	HighestCardId        int32
	Winner               int32
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
	result.HighestCardId = gs.HighestCardId
	result.Winner = gs.Winner
	return &result
}

func (gs *GameState) resetGameState() {
	gs.CardsById = make(map[int32]*Card)
	gs.CardsByZone = make(map[string]map[*Card]interface{})
	gs.Mana = 0
	gs.LastManaAdjustPlayer = "NOBODY?!"
	gs.HighestCardId = 0
	gs.Winner = NO_VICTORY
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
			delete(oldZoneCards, card)
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
func useCard(gs *GameState, params *MoveParams) {
	playCard := params.CardOne
	switch playCard.Zone {
	// If played from hand
	case "FRIENDLY HAND":
		gs.Mana -= playCard.Cost
		switch playCard.Type {
		case "Minion":
			// Warsong Commander
			if playCard.Attack <= 3 {
				for friendlyMinion := range gs.CardsByZone["FRIENDLY PLAY"] {
					if friendlyMinion.JsonCardId == "EX1_084" && !friendlyMinion.Silenced {
						//fmt.Println("DEBUG: Getting charge from Warsong Commander.")
						playCard.Charge = true
					}
				}
			}
			// minion comes into play
			gs.moveCard(playCard, "FRIENDLY PLAY")
			playCard.Exhausted = !playCard.Charge
			// battlecry effects, if any
			runCardPlayedAction(gs, params)
		case "Spell":
			// execute spell
			runCardPlayedAction(gs, params)
			gs.moveCard(playCard, "FRIENDLY GRAVEYARD")
		case "FRIENDLY PLAY (Hero Power)":
			// if using hero power
			runCardPlayedAction(gs, params)
			// hero power is now exhausted
			playCard.Exhausted = true
		case "Weapon":
			// remove anything currently in weapon zone
			friendlyHero := getSingletonFromZone(gs, "FRIENDLY PLAY (Hero)", true)
			prettyPrint(friendlyHero)
			if oldWeapon := getSingletonFromZone(gs, "FRIENDLY PLAY (Weapon)", false); oldWeapon != nil {
				// Yes, really destroy the weapon now: http://hearthstone.gamepedia.com/Advanced_rulebook#Instant_weapon_destruction
				gs.handleDeath(oldWeapon)
				friendlyHero.Attack -= oldWeapon.Attack
			}
			// new weapon to weapon zone
			gs.moveCard(playCard, "FRIENDLY PLAY (Weapon)")
			friendlyHero.Attack += playCard.Attack
			// Assert we now have exactly one weapon.
			getSingletonFromZone(gs, "FRIENDLY PLAY (Weapon)", true)
			// TODO (dz): other card types (Enchantment?)
		}
	// If on battlefield, then this is a minion attack.
	case "FRIENDLY PLAY":
		// TODO (dz): is it easier for nextMoves to call use or attack?
		//fmt.Println("`useCard` called with a card on the field, using `attack` instead.")
		attack(gs, params)
	// if using hero attack
	case "FRIENDLY PLAY (Hero)":
		//fmt.Println("`useCard` called with hero card, using `attack` instead.")
		attack(gs, params)
	default:
		fmt.Println("Unrecognized Zone to play a card from: ", playCard.Zone)
		panic("Unrecognized Zone to play a card from!")
	}
	gs.cleanupState()
}

// Run the action out of GlobalCardPlayedActions for a given move.
func runCardPlayedAction(gs *GameState, params *MoveParams) {
	// fmt.Println("DEBUG: running action for move: ", params)
	getCardPlayedAction(params.CardOne)(gs, params)
}

// Run the action out of GlobalDeathrattleActions for a given card.
func runDeathrattleAction(gs *GameState, dyingMinion *Card) {
	getDeathrattleAction(dyingMinion)(gs, &MoveParams{
		CardOne:     dyingMinion,
		Description: "Dummy move param for deathrattle",
	})
}

func (gs *GameState) CreateNewMinion(jsonId string, zone string) {
	card := gs.getOrCreateCard(jsonId, gs.HighestCardId+1)
	card.Exhausted = true
	gs.moveCard(card, zone)
}

// Minion attack or weapon attack (modifies `gs` and the cards in it).
func attack(gs *GameState, params *MoveParams) {
	gs.dealDamage(params.CardOne, params.CardTwo.Attack)
	gs.dealDamage(params.CardTwo, params.CardOne.Attack)
	params.CardOne.Exhausted = true
}

func (gs *GameState) dealDamage(target *Card, amount int32) {
	if amount <= 0 {
		return
	}

	target.Armor -= amount
	if target.Armor < 0 {
		target.Damage -= target.Armor
		target.Armor = 0
	}

	// We're not bad people, but we did a bad thing...
	// TODO: Implement some kind of generic listener framework someday.
	if target.JsonCardId == "BRM_019" && !target.Silenced &&
		target.Damage < target.Health && len(gs.CardsByZone[target.Zone]) < 7 { // Grim Patron
		//fmt.Println("DEBUG: Everyone! Get in here!")
		gs.CreateNewMinion("BRM_019", target.Zone)
	}
	for friendlyMinion := range gs.CardsByZone["FRIENDLY PLAY"] {
		if friendlyMinion.JsonCardId == "EX1_604" && !friendlyMinion.Silenced { // Frothing Berserker
			//fmt.Println("DEBUG: My blade be thirsty!")
			friendlyMinion.Attack += 1
		}
	}
	for enemyMinion := range gs.CardsByZone["OPPOSING PLAY"] {
		if enemyMinion.JsonCardId == "EX1_604" && !enemyMinion.Silenced { // Frothing Berserker
			//fmt.Println("DEBUG: Enemy Frothing Berserker triggered.")
			enemyMinion.Attack += 1
		}
	}
}

func minionNeedsKilling(card *Card) bool {
	return card.PendingDestroy || card.Damage >= card.Health
}

// Clean up the gs state, moving cards to their zones, executing deathrattles, etc
func (gs *GameState) cleanupState() {
	// check for PendingDestroy or lethal damage on minions
	didAnything := false
	friendlyHero := getSingletonFromZone(gs, "FRIENDLY PLAY (Hero)", true)
	enemyHero := getSingletonFromZone(gs, "OPPOSING PLAY (Hero)", true)
	if minionNeedsKilling(friendlyHero) {
		//fmt.Println("DEBUG: Oops, we died.")
		gs.Winner = OPPOSING_VICTORY_OR_DRAW
	} else if minionNeedsKilling(enemyHero) && gs.Winner == NO_VICTORY {
		//fmt.Println("DEBUG: Victory!")
		gs.Winner = FRIENDLY_VICTORY
	}

	for minion, _ := range gs.CardsByZone["FRIENDLY PLAY"] {
		if minion.PendingDestroy || (minion.Damage >= minion.Health) {
			didAnything = true
			//fmt.Println("Minion should die due to damage: ", minion)
			gs.handleDeath(minion)
		}
	}
	for minion, _ := range gs.CardsByZone["OPPOSING PLAY"] {
		if minion.PendingDestroy || (minion.Damage >= minion.Health) {
			didAnything = true
			//fmt.Println("Minion should die due to damage: ", minion)
			gs.handleDeath(minion)
		}
	}
	if didAnything {
		gs.cleanupState()
	}
}

func (gs *GameState) handleDeath(minion *Card) {
	// Execute deathrattle
	runDeathrattleAction(gs, minion)
	// Move to graveyard
	gs.moveCard(minion, "FRIENDLY GRAVEYARD")
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
