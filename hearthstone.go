package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ActiveState/tail"
	"os"
	"regexp"
	"strconv"
)

type LineParser struct {
	applyFunc func(args *LineParserApplyArgs)
	pattern   *regexp.Regexp
}

type LineParserApplyArgs struct {
	gs    *GameState
	line  string
	match map[string]string
}

// LINE PARSER FUNCTIONS

func applyDebugWriteLine(args *LineParserApplyArgs) {
	if len(args.match) > 0 {
		prettyPrint(args.match)
	} else {
		fmt.Println("DEBUG:", args.line)
	}
}

func prettyPrint(x interface{}) {
	json, _ := json.MarshalIndent(x, "", "  ")
	fmt.Println(string(json))
}

func applyNewGame(args *LineParserApplyArgs) {
	args.gs.resetGameState()
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	applyDebugWriteLine(args)
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
	fmt.Println("~~~~~~~~~~~~~~~~")
}

func applyCreatePlayer(args *LineParserApplyArgs) {
	applyDebugWriteLine(args)
}

func applyZoneChange(args *LineParserApplyArgs) {
	applyDebugWriteLine(args)
	instance_id, _ := strconv.ParseInt(args.match["instance_id"], 10, 32)
	card := args.gs.getOrCreateCard(args.match["class_id"], int32(instance_id))
	args.gs.moveCard(card, args.match["zone_to"])
	prettyPrint(*card)
}

func applyTagChange(args *LineParserApplyArgs) {
	applyDebugWriteLine(args)
	instance_id, _ := strconv.ParseInt(args.match["instance_id"], 10, 32)
	card := args.gs.getOrCreateCard("SHOULD_NOT_HAPPEN", int32(instance_id))
	//ATK|COST|DAMAGE|EXHAUSTED|FROZEN|HEALTH|TAUNT|SILENCED
	tag_value_str, _ := strconv.ParseInt(args.match["tag_value"], 10, 32)
	tag_value := int32(tag_value_str)
	switch args.match["tag_name"] {
	case "ATK":
		card.Attack = tag_value
	case "COST":
		card.Cost = tag_value
	case "DAMAGE":
		card.Damage = tag_value
	case "EXHAUSTED":
		card.Exhausted = tag_value == 1
	case "HEALTH":
		card.Health = tag_value
	case "TAUNT":
		card.Taunt = tag_value == 1
	case "SILENCED":
		card.Silenced = tag_value == 1
	}
	prettyPrint(*card)
}

func applyManaUpdate(args *LineParserApplyArgs) {
	applyDebugWriteLine(args)
	mana_str, _ := strconv.ParseInt(args.match["mana"], 10, 32)
	args.gs.Mana = int32(mana_str)
	args.gs.LastManaAdjustPlayer = args.match["name"]
}

type GameState struct {
	CardsById            map[int32]*Card
	CardsByZone          map[string]map[*Card]interface{}
	Mana                 int32
	LastManaAdjustPlayer string
}

type Move struct {
	ApplyMove func(gs *GameState) *GameState // Returns a copy
  IdOne int32
  IdTwo int32
  Description string
}

type DecisionTreeNode struct {
	Gs                 *GameState
	Moves              []Move
	SuccessProbability float32
}

func WalkDecisionTree(workChan chan DecisionTreeNode, successChan chan DecisionTreeNode, timeoutChan chan bool) {
	for {
		select {
		case <-timeoutChan:
			// Shut down after draining the workChan.
			for {
				select {
				case <-workChan:
				default:
					return
				}
			}
		case node := <-workChan:
			go func() {
				nextMoves := getNextMoves()
				for _, move := range nextMoves {
					move = move // Long story.
					go func() {
						// Deep copy node.gs
						// Apply move to gs to make new node.
						// If we win, put node into successChan.
						// Else put node into workChan.
					}()
				}
			}()
		}
	}
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

func getLineGroups(line string, pattern *regexp.Regexp) map[string]string {
	match := pattern.FindStringSubmatch(line)
	if match == nil {
		return nil
	}
	result := make(map[string]string)
	for i, name := range pattern.SubexpNames() {
		// Hack around the fact that SubexpNames returns '' first.
		if i == 0 {
			continue
		}
		result[name] = match[i]
	}
	return result
}

func getMatchingParser(line string, parsers []LineParser) (*LineParser, map[string]string) {
	for _, parser := range parsers {
		if match := getLineGroups(line, parser.pattern); match != nil {
			return &parser, match
		}
	}
	return nil, nil
}

type JsonCardData struct {
	Id        string   `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Text      string   `json:"text"`
	Cost      int32    `json:"cost"`
	Attack    int32    `json:"attack"`
	Health    int32    `json:"health"`
	Mechanics []string `json:"mechanics"`
}

type Card struct {
	InstanceId int32
	JsonCardId string
	Name       string
	Cost       int32
	Attack     int32
	Health     int32
	Damage     int32
	Exhausted  bool
	Frozen     bool
	Taunt      bool
	Silenced   bool
	Zone       string
}

func newCardFromJson(jsonCardId string, instanceId int32) Card {
	if jsonCard, ok := GlobalCardJsonData[jsonCardId]; ok {
		return Card{
			InstanceId: instanceId,
			JsonCardId: jsonCardId,
			Name:       jsonCard.Name,
			Cost:       jsonCard.Cost,
			Attack:     jsonCard.Attack,
			Health:     jsonCard.Health,
			Exhausted:  true,
		}
	}
	fmt.Printf("ERROR: Unknown jsonCardId: %v\n", jsonCardId)
	return Card{
		InstanceId: instanceId,
		JsonCardId: jsonCardId,
		Exhausted:  true}
}

var GlobalCardJsonData map[string]JsonCardData

func loadCardJson() {
	GlobalCardJsonData = make(map[string]JsonCardData)
	var jsonFileData struct {
		Basic     []JsonCardData `json:"Basic"`
		Classic   []JsonCardData `json:"Classic"`
		Nax       []JsonCardData `json:"Curse of Naxxramas"`
		Brm       []JsonCardData `json:"Blackrock Mountain"`
		Promotion []JsonCardData `json:"Promotion"`
		Reward    []JsonCardData `json:"Reward"`
		Gvg       []JsonCardData `json:"Goblins vs Gnomes"`
		Tb        []JsonCardData `json:"Tavern Brawl"`
	}
	cardFile, _ := os.Open("AllSets.json")
	jsonParser := json.NewDecoder(cardFile)
	if err := jsonParser.Decode(&jsonFileData); err != nil {
		fmt.Println("ERROR: Cannot parse json card data: ", err.Error())
	}
	for _, card := range jsonFileData.Basic {
		GlobalCardJsonData[card.Id] = card
	}
	for _, card := range jsonFileData.Classic {
		GlobalCardJsonData[card.Id] = card
	}
	for _, card := range jsonFileData.Nax {
		GlobalCardJsonData[card.Id] = card
	}
	for _, card := range jsonFileData.Brm {
		GlobalCardJsonData[card.Id] = card
	}
	for _, card := range jsonFileData.Promotion {
		GlobalCardJsonData[card.Id] = card
	}
	for _, card := range jsonFileData.Reward {
		GlobalCardJsonData[card.Id] = card
	}
	for _, card := range jsonFileData.Gvg {
		GlobalCardJsonData[card.Id] = card
	}
	for _, card := range jsonFileData.Tb {
		GlobalCardJsonData[card.Id] = card
	}
}

func main() {
	loadCardJson()

	hsLogFile := flag.String("log", "no-log-file-specified", "The file path to the Hearthstone log file.")

	flag.Parse()

	log, _ := tail.TailFile(*hsLogFile, tail.Config{Follow: true})
	startOfTurnString := `[Power] GameState.DebugPrintPower() - TAG_CHANGE Entity=GameEntity tag=STEP value=MAIN_ACTION`
	lineParsers := []LineParser{
		LineParser{applyManaUpdate, regexp.MustCompile(`\[Power\] GameState.DebugPrintPower\(\) -\s+` +
			`TAG_CHANGE Entity=(?P<name>.*) tag=RESOURCES value=(?P<mana>\d+)`)},
		LineParser{applyNewGame, regexp.MustCompile(`\[Power\] GameState.DebugPrintPower\(\) -\s+` +
			`CREATE_GAME`)},
		LineParser{applyDebugWriteLine, regexp.MustCompile(`\[Power\] GameState.DebugPrintPower\(\) -\s+` +
			`TAG_CHANGE Entity=(?P<player_name>.*) tag=PLAYER_ID value=(?P<player_id>.*)`)},
		LineParser{applyDebugWriteLine, regexp.MustCompile(`\[Power\] GameState.DebugPrintPower\(\) -\s+` +
			`TAG_CHANGE Entity=(?P<player_name>.*) tag=PLAYER_ID value=(?P<player_id>.*)`)},

		LineParser{applyTagChange, regexp.MustCompile(`\[Power\] GameState.DebugPrintPower\(\) -\s+` +
			`TAG_CHANGE .*id=(?P<instance_id>\d+).* tag=(?P<tag_name>ATK|COST|DAMAGE|EXHAUSTED|FROZEN|HEALTH|TAUNT|SILENCED) value=(?P<tag_value>.*)`)},
		//LineParser{applyDebugWriteLine, regexp.MustCompile(`\[Zone\] ZoneChangeList.ProcessChanges\(\) -\s+` +
		//	`id=.* local=.* \[name=(?P<name>.*) id=(?P<instanceId>.*) zone=.* zonePos=.* cardId=(?P<class_id>.*) player=(?P<player_id>.*)\] zone from (?P<zome_from>.*) -> (?P<zome_to>.*)`)},
		LineParser{applyZoneChange, regexp.MustCompile(`\[Zone\] ZoneChangeList.ProcessChanges\(\) -\s+` +
			`TRANSITIONING card \[name=(?P<name>.*) id=(?P<instance_id>.*) zone=.* zonePos=.* cardId=(?P<class_id>.*) player=(?P<player_id>.*)\] to (?P<zone_to>.*)$`)},
		//LineParser{regexp.MustCompile(`\[Power\] .*`), applyDebugWriteLine},
	}

	gs := GameState{}
	gs.resetGameState()
	for line := range log.Lines {
		if line.Text == startOfTurnString {
			// Call WalkDecisionTree
			fmt.Println("It is the start of turn for:", gs.LastManaAdjustPlayer)
		} else {
			// Need to abort ongoing decision tree search.
			parser, match := getMatchingParser(line.Text, lineParsers)
			if parser != nil {
				if parser.applyFunc == nil {
					fmt.Println("ERROR: LineParser has no applyFunc!")
				} else {
					parser.applyFunc(&LineParserApplyArgs{
						&gs,
						line.Text,
						match,
					})
				}
			}
		}
	}
}
