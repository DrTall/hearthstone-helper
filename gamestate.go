// Game state tracking, from either the parser or hypotheticals.

package main

type GameState struct {
	CardsById            map[int32]*Card
	CardsByZone          map[string]map[*Card]interface{}
	Mana                 int32
	LastManaAdjustPlayer string
}

func (gs *GameState) resetGameState() {
	gs.CardsById = make(map[int32]*Card)
	gs.CardsByZone = make(map[string]map[*Card]interface{})
	gs.Mana = 0
	gs.LastManaAdjustPlayer = "NOBODY?!"
}

func (gs *GameState) getOrCreateCard(jsonCardId string, instanceId int32) *Card {
	if card, ok := gs.CardsById[instanceId]; ok {
		//fmt.Println("DEBUG: Already knew about card: ", gs.cardsById)
		return card
	}
	//fmt.Println("DEBUG: Creating new card for instance: ", instanceId)
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
	if _, ok := gs.CardsByZone[newZone]; !ok {
		gs.CardsByZone[newZone] = make(map[*Card]interface{})
	}
	card.Zone = newZone
	gs.CardsByZone[card.Zone][card] = nil
	//fmt.Println("BEFORE HELLO!!!", gs)
	//prettyPrint(gs)
}

// A particular instance of a card in the game.
type Card struct {
	InstanceId int32  // Globally unique.
	JsonCardId string // Refers to JsonCardData.Id
	Name       string // Refers to JsonCardData.Name
	Cost       int32
	Attack     int32
	Health     int32
	Armor      int32
	Damage     int32
	Exhausted  bool
	Frozen     bool
	Taunt      bool
	Silenced   bool
	Zone       string
}
