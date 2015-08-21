package main

// Options for benchmarking.
type PruningOpts struct {
	getCardsFromFriendlyZone func(gs *GameState, zone string) []*Card
	getCardsInOpposingPlay   func(gs *GameState) []*Card
	isNodeHighPriority       func(node *DecisionTreeNode) bool
	// Whether to optimize The Coin (you basically always should)
	useCoinOptimization bool
}

var GlobalPruningOpts PruningOpts

func resetGlobalPruningOpts() {
	GlobalPruningOpts = PruningOpts{
		getCardsFromFriendlyZone: uniqueCardsInZone,
		getCardsInOpposingPlay:   uniqueCardsInOpposingPlay,
		isNodeHighPriority:       isFrothingBerserkerReady,
		useCoinOptimization:      true,
	}
}

func init() {
	resetGlobalPruningOpts()
}

// Returns a slices of "unique" *Cards, per CardInfo
func uniqueCardsInZone(gs *GameState, zone string) []*Card {
	result := make([]*Card, 0)
	allMinions := gs.CardsByZone[zone]
	uniqueMinionInfo := make(map[CardInfo]*Card)
	for minion := range allMinions {
		if _, exists := uniqueMinionInfo[minion.getInfo()]; !exists {
			result = append(result, minion)
			uniqueMinionInfo[minion.getInfo()] = minion
		}
	}
	return result
}

// Returns a slices of "unique" *Cards, per CardInfo for the zone
// "OPPOSING PLAY", where we don't care about certain attributes of the card.
func uniqueCardsInOpposingPlay(gs *GameState) []*Card {
	result := make([]*Card, 0)
	zone := "OPPOSING PLAY"
	allMinions := gs.CardsByZone[zone]
	uniqueMinionInfo := make(map[CardInfo]*Card)
	for minion := range allMinions {
		if _, exists := uniqueMinionInfo[minion.getInfoAsEnemyMinion()]; !exists {
			result = append(result, minion)
			uniqueMinionInfo[minion.getInfoAsEnemyMinion()] = minion
		}
	}
	return result
}

// Does this GameState have a Frothing Berserker ready to attack?
func isFrothingBerserkerReady(node *DecisionTreeNode) bool {
	for friendlyMinion, _ := range node.Gs.CardsByZone["FRIENDLY PLAY"] {
		if friendlyMinion.JsonCardId == "EX1_604" && !friendlyMinion.Silenced && canCardAttack(friendlyMinion) {
			return true
		}
	}
	return false
}
