package main

type ClientMessage struct {
  Type  string  `json:"type"`
	Id    string  `json:"id"`
	S     Snake   `json:"s"`
}

type GameStateMessage struct {
  Type    string            `json:"type"`
	Snakes  map[string]Snake  `json:"snakes"`
  Food    []Pos             `json:"food"`
}

func NewGameStateMessage(snakes map[string]Snake, food []Pos) *GameStateMessage {
  return &GameStateMessage{
    Type: "gamestate",
    Snakes: snakes,
    Food: food,
  }
}
