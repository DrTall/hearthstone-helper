// Parsing data from http://hearthstonejson.com/.

package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// The source of truth for what a card looks like in its default state.
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

func init() {
	loadCardJson()
}
