package main

import "testing"

func TestDeepCopy(t *testing.T) {
  gs := GameState{}
  gs.resetGameState()
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