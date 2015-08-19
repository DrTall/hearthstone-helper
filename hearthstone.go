package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ActiveState/tail"
	"os"
	"regexp"
	"strconv"
	"time"
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
	//applyDebugWriteLine(args)
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
	card := args.gs.getOrCreateCard(args.match["class_id"], int32(instance_id))
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
	default:
		fmt.Println("ERROR: Unknown tag_name:", args.match["tag_name"])
	}
	prettyPrint(*card)
}

func applyTagChangeNoJsonId(args *LineParserApplyArgs) {
	instance_id, _ := strconv.ParseInt(args.match["instance_id"], 10, 32)
	if _, ok := args.gs.CardsById[int32(instance_id)]; ok {
		// This card already exists so we will succeed in finding it without
		// the json id.
		applyTagChange(args)
	} else {
		fmt.Println("ERROR: Got a tag change for instance_id before "+
			"it was given a json class:", instance_id)
	}
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
	ApplyMove   func(gs *GameState) *GameState // Returns a copy
	IdOne       int32
	IdTwo       int32
	Description string
}

type DecisionTreeNode struct {
	Gs                 *GameState
	Moves              []Move
	SuccessProbability float32
}

func getNextMoves(node DecisionTreeNode) []Move {
	return nil
}

func WalkDecisionTree(successChan <-chan *DecisionTreeNode, abortChan <-chan time.Time) {
	fmt.Println("DEBUG: Beginning decision tree walk.")
	workChan := make(chan DecisionTreeNode)
	timeoutChan := time.After(time.Second * 70)
	for {
		select {
		case <-abortChan:
			fmt.Println("DEBUG: Decision tree walk aborting...")
			return
		case <-timeoutChan:
			fmt.Println("DEBUG: Decision tree walk timing out...")
			return
		case node := <-workChan:
			go func() {
				nextMoves := getNextMoves(node) // Does not modify node.
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
	Armor      int32
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
	startTurnPattern := regexp.MustCompile(`Entity=GameEntity tag=STEP value=MAIN_ACTION`)
	lineParsers := []LineParser{
		LineParser{applyManaUpdate, regexp.MustCompile(`\[Power\] GameState.DebugPrintPower\(\) -\s+` +
			`TAG_CHANGE Entity=(?P<name>.*) tag=RESOURCES value=(?P<mana>\d+)`)},
		LineParser{applyNewGame, regexp.MustCompile(`\[Power\] GameState.DebugPrintPower\(\) -\s+` +
			`CREATE_GAME`)},
		LineParser{applyTagChange, regexp.MustCompile(`\[Power\] GameState.DebugPrintPower\(\) -\s+` +
			`TAG_CHANGE .*id=(?P<instance_id>\d+).*cardId=(?P<class_id>\S+).*tag=(?P<tag_name>ATK|ARMOR|COST|DAMAGE|FROZEN|HEALTH|TAUNT|SILENCED) value=(?P<tag_value>.*)`)},
		LineParser{applyTagChangeNoJsonId, regexp.MustCompile(`\[Power\] GameState.DebugPrintPower\(\) -\s+` +
			`TAG_CHANGE .*id=(?P<instance_id>\d+).*tag=(?P<tag_name>ATK|ARMOR|COST|DAMAGE|FROZEN|HEALTH|TAUNT|SILENCED) value=(?P<tag_value>.*)`)},
		//LineParser{applyDebugWriteLine, regexp.MustCompile(`\[Zone\] ZoneChangeList.ProcessChanges\(\) -\s+` +
		//	`id=.* local=.* \[name=(?P<name>.*) id=(?P<instanceId>.*) zone=.* zonePos=.* cardId=(?P<class_id>.*) player=(?P<player_id>.*)\] zone from (?P<zome_from>.*) -> (?P<zome_to>.*)`)},
		LineParser{applyZoneChange, regexp.MustCompile(`\[Zone\] ZoneChangeList.ProcessChanges\(\) -\s+` +
			`TRANSITIONING card \[name=(?P<name>.*) id=(?P<instance_id>.*) zone=.* zonePos=.* cardId=(?P<class_id>.*) player=(?P<player_id>.*)\] to (?P<zone_to>.*)$`)},
		//LineParser{regexp.MustCompile(`\[Power\] .*`), applyDebugWriteLine},
	}

	gs := GameState{}
	gs.resetGameState()
	var abortChan *chan time.Time
	successChan := make(chan *DecisionTreeNode)
	for {
		select {
		case line := <-log.Lines:
			if match := startTurnPattern.FindStringSubmatch(line.Text); len(match) > 0 && abortChan == nil {
				fmt.Println("It is the start of turn for:", gs.LastManaAdjustPlayer)
				newAbortChan := make(chan time.Time, 1)
				abortChan = &newAbortChan
				go WalkDecisionTree(successChan, newAbortChan)
			} else {
				parser, match := getMatchingParser(line.Text, lineParsers)
				if parser != nil {
					if parser.applyFunc == nil {
						fmt.Println("ERROR: LineParser has no applyFunc!")
					} else {
						if abortChan != nil {
							*abortChan <- time.Now()
							abortChan = nil
						}
						parser.applyFunc(&LineParserApplyArgs{
							&gs,
							line.Text,
							match,
						})
					}
				}
			}
		case solution := <-successChan:
			fmt.Println("DEBUG: Found a solution!")
			prettyPrint(*solution)
		}
	}
}
