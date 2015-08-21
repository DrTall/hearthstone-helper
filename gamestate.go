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
	CardsById     map[int32]*Card
	CardsByZone   map[string]map[*Card]interface{}
	ManaMax       int32
	ManaUsed      int32
	ManaTemp      int32
	HighestCardId int32
	Winner        int32
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
	result.ManaMax = gs.ManaMax
	result.ManaUsed = gs.ManaUsed
	result.ManaTemp = gs.ManaTemp
	result.HighestCardId = gs.HighestCardId
	result.Winner = gs.Winner
	return &result
}

func (gs *GameState) resetGameState() {
	gs.CardsById = make(map[int32]*Card)
	gs.CardsByZone = make(map[string]map[*Card]interface{})
	gs.ManaMax = 0
	gs.ManaUsed = 0
	gs.ManaTemp = 0
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
		gs.ManaTemp -= playCard.Cost
		if gs.ManaTemp < 0 {
			gs.ManaUsed -= gs.ManaTemp
			gs.ManaTemp = 0
		}
		switch playCard.Type {
		case "Minion":
			// Warsong Commander
			maybeTriggerWarsongCommander(gs, playCard)
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
			//prettyPrint(friendlyHero)
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

func maybeTriggerWarsongCommander(gs *GameState, card *Card) {
	if card.Attack <= 3 {
		for friendlyMinion := range gs.CardsByZone["FRIENDLY PLAY"] {
			if friendlyMinion.JsonCardId == "EX1_084" && !friendlyMinion.Silenced {
				//fmt.Println("DEBUG: Getting charge from Warsong Commander.")
				card.Charge = true
			}
		}
	}
}

// Run the action out of GlobalCardPlayedActions for a given move.
func runCardPlayedAction(gs *GameState, params *MoveParams) {
	// fmt.Println("DEBUG: running action for move: ", params)
	getCardPlayedAction(params.CardOne)(gs, params)
}

// Run the action out of GlobalDeathrattleActions for a given card.
func runDeathrattleAction(gs *GameState, dyingMinion *Card) {
	if !dyingMinion.Silenced {
		getDeathrattleAction(dyingMinion)(gs, &MoveParams{
			CardOne:     dyingMinion,
			Description: "Dummy move param for deathrattle",
		})
	}
}

func (gs *GameState) CreateNewMinion(jsonId string, zone string) *Card {
	card := gs.getOrCreateCard(jsonId, gs.HighestCardId+1)
	card.Exhausted = true
	maybeTriggerWarsongCommander(gs, card)
	gs.moveCard(card, zone)
	return card
}

// Minion attack or weapon attack (modifies `gs` and the cards in it).
func attack(gs *GameState, params *MoveParams) {
	gs.dealDamage(params.CardOne, params.CardTwo.Attack)
	gs.dealDamage(params.CardTwo, params.CardOne.Attack)
	params.CardOne.NumAttacksThisTurn += 1
}

func (gs *GameState) dealDamage(target *Card, amount int32) {
	if amount <= 0 {
		return
	}

	target.JustTookDamage = true
	target.Armor -= amount
	if target.Armor < 0 {
		target.Damage -= target.Armor
		target.Armor = 0
	}

	if target.Type == "Minion" {
		for friendlyMinion := range gs.CardsByZone["FRIENDLY PLAY"] {
			if friendlyMinion.JsonCardId == "EX1_604" && !friendlyMinion.Silenced { // Frothing Berserker
				friendlyMinion.Attack += 1
				//fmt.Printf("DEBUG: My blade be thirsty! Attack is now %v\n", friendlyMinion.Attack)
			}
		}
		for enemyMinion := range gs.CardsByZone["OPPOSING PLAY"] {
			if enemyMinion.JsonCardId == "EX1_604" && !enemyMinion.Silenced { // Frothing Berserker
				//fmt.Println("DEBUG: Enemy Frothing Berserker triggered.")
				enemyMinion.Attack += 1
			}
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

	friendlyHero.JustTookDamage = false
	enemyHero.JustTookDamage = false

	for minion, _ := range gs.CardsByZone["FRIENDLY PLAY"] {
		if minionNeedsKilling(minion) {
			didAnything = true
			//fmt.Println("Minion should die due to damage: ", minion)
			gs.handleDeath(minion)
		} else if minion.JustTookDamage && minion.JsonCardId == "BRM_019" &&
			!minion.Silenced && len(gs.CardsByZone["FRIENDLY PLAY"]) < 7 {
			// We're not bad people, but we did a bad thing...
			// TODO: Implement some kind of generic listener framework someday.
			//fmt.Println("DEBUG: Everyone! Get in here!")
			didAnything = true
			gs.CreateNewMinion("BRM_019", "FRIENDLY PLAY")
		}
		minion.JustTookDamage = false
	}
	for minion, _ := range gs.CardsByZone["OPPOSING PLAY"] {
		if minionNeedsKilling(minion) {
			didAnything = true
			//fmt.Println("Minion should die due to damage: ", minion)
			gs.handleDeath(minion)
		}
		minion.JustTookDamage = false
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
	InstanceId         int32  // Globally unique.
	JsonCardId         string // Refers to JsonCardData.Id
	Type               string // Refers to JsonCardData.Type
	Name               string // Refers to JsonCardData.Name
	Cost               int32
	Attack             int32
	Health             int32
	Armor              int32
	Damage             int32
	NumAttacksThisTurn int32
	Charge             bool
	Exhausted          bool
	Frozen             bool
	Taunt              bool
	Silenced           bool
	Zone               string
	PendingDestroy     bool // Internal. Should this minion be destroyed in the next cleanup step?
	JustTookDamage     bool // Internal. Did this minion take damage since the last cleanup step?
}

// TODO (dz): this is kinda hacky... Card should really be a struct with just InstanceId + CardInfo,
// but I'm too lazy to change it for now.
// An internal class representing the aspects of a Card that matter to the game
// (everything besides the InstanceId)
type CardInfo struct {
	JsonCardId         string // Refers to JsonCardData.Id
	Type               string // Refers to JsonCardData.Type
	Name               string // Refers to JsonCardData.Name
	Cost               int32
	Attack             int32
	Health             int32
	Armor              int32
	Damage             int32
	NumAttacksThisTurn int32
	Charge             bool
	Exhausted          bool
	Frozen             bool
	Taunt              bool
	Silenced           bool
	Zone               string
	PendingDestroy     bool // Internal. Should this minion be destroyed in the next cleanup step?
}

func (c *Card) getInfo() CardInfo {
	return CardInfo{
		JsonCardId:         c.JsonCardId,
		Type:               c.Type,
		Name:               c.Name,
		Cost:               c.Cost,
		Attack:             c.Attack,
		Health:             c.Health,
		Armor:              c.Armor,
		Damage:             c.Damage,
		NumAttacksThisTurn: c.NumAttacksThisTurn,
		Charge:             c.Charge,
		Exhausted:          c.Exhausted,
		Frozen:             c.Frozen,
		Taunt:              c.Taunt,
		Silenced:           c.Silenced,
		Zone:               c.Zone,
		PendingDestroy:     c.PendingDestroy,
	}
}

// for enemy minions, we don't care about JsonCardId.
func (c *Card) getInfoAsEnemyMinion() CardInfo {
	return CardInfo{
		Type:           c.Type,
		Name:           c.Name,
		Attack:         c.Attack,
		Health:         c.Health,
		Armor:          c.Armor,
		Damage:         c.Damage,
		Taunt:          c.Taunt,
		Zone:           c.Zone,
		PendingDestroy: c.PendingDestroy,
	}
}
