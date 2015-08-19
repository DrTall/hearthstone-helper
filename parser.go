// Parsing the Hearthstone log file into a gamestate.

package main

import (
	"fmt"
	"regexp"
	"strconv"
)

var (
	startTurnPattern = regexp.MustCompile(`Entity=GameEntity tag=STEP value=MAIN_ACTION`)
	lineParsers      = []LineParser{
		LineParser{applyManaUpdate, regexp.MustCompile(`\[Power\] GameState.DebugPrintPower\(\) -\s+` +
			`TAG_CHANGE Entity=(?P<name>.*) tag=RESOURCES value=(?P<mana>\d+)`)},
		LineParser{applyNewGame, regexp.MustCompile(`\[Power\] GameState.DebugPrintPower\(\) -\s+` +
			`CREATE_GAME`)},
		LineParser{applyTagChange, regexp.MustCompile(`\[Power\] GameState.DebugPrintPower\(\) -\s+` +
			`TAG_CHANGE .*id=(?P<instance_id>\d+).*cardId=(?P<class_id>\S+).*tag=(?P<tag_name>ATK|ARMOR|COST|DAMAGE|FROZEN|HEALTH|TAUNT|SILENCED) value=(?P<tag_value>.*)`)},
		LineParser{applyTagChangeNoJsonId, regexp.MustCompile(`\[Power\] GameState.DebugPrintPower\(\) -\s+` +
			`TAG_CHANGE .*id=(?P<instance_id>\d+).*tag=(?P<tag_name>ATK|ARMOR|COST|DAMAGE|EXHAUSTED|FROZEN|HEALTH|TAUNT|SILENCED) value=(?P<tag_value>.*)`)},
		//LineParser{applyDebugWriteLine, regexp.MustCompile(`\[Zone\] ZoneChangeList.ProcessChanges\(\) -\s+` +
		//  `id=.* local=.* \[name=(?P<name>.*) id=(?P<instanceId>.*) zone=.* zonePos=.* cardId=(?P<class_id>.*) player=(?P<player_id>.*)\] zone from (?P<zome_from>.*) -> (?P<zome_to>.*)`)},
		LineParser{applyZoneChange, regexp.MustCompile(`\[Zone\] ZoneChangeList.ProcessChanges\(\) -\s+` +
			`TRANSITIONING card \[name=(?P<name>.*) id=(?P<instance_id>.*) zone=.* zonePos=.* cardId=(?P<class_id>.*) player=(?P<player_id>.*)\] to (?P<zone_to>.*)$`)},
		//LineParser{regexp.MustCompile(`\[Power\] .*`), applyDebugWriteLine},
	}
)

// Consumes a Hearthstone log line.
// turnStart -- Did this line indicate a player's turn just began?
// somethingHappened -- Did this line indicate anything relevant happened?
func ParseHearthstoneLogLine(line string, gs *GameState) (turnStart bool, somethingHappened bool) {
	if match := startTurnPattern.FindStringSubmatch(line); len(match) > 0 {
		turnStart = true
    return
	}
	parser, match := getMatchingParser(line, lineParsers)
	if parser != nil {
		if parser.applyFunc == nil {
			fmt.Println("ERROR: LineParser has no applyFunc!")
		} else {
      somethingHappened = true
			parser.applyFunc(&LineParserApplyArgs{
				gs,
				line,
				match,
			})
		}
	}
	return
}

type LineParser struct {
	applyFunc func(args *LineParserApplyArgs)
	pattern   *regexp.Regexp
}

type LineParserApplyArgs struct {
	gs    *GameState
	line  string
	match map[string]string
}

func applyDebugWriteLine(args *LineParserApplyArgs) {
	if len(args.match) > 0 {
		prettyPrint(args.match)
	} else {
		fmt.Println("DEBUG:", args.line)
	}
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
  case "FROZEN":
    card.Frozen = tag_value == 1
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
