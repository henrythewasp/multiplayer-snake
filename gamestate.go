package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
)

type Pos struct {
	X int	`json:"x"`
	Y int	`json:"y"`
}

type Snake []Pos

type JSONSnakeData struct {
	Id string	`json:"id"`
	S Snake		`json:"s"`
}
type GameState struct {
	mutex sync.RWMutex
	snakes map[string]Snake
}

func NewGameState() *GameState {
	return &GameState{
		snakes: make(map[string]Snake),
	}
}
func NewSnake() Snake {
	s := make([]Pos, 0, 10)
	s = append(s, Pos{rand.Intn(20), rand.Intn(20)})
	return s
}

type JSONGameStateMsg struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func (gs *GameState) AddSnake(msg ClientMessage) string {
	gs.mutex.Lock()
	id := strconv.Itoa(len(gs.snakes)+1)
	gs.snakes[id] = NewSnake()
	gs.mutex.Unlock()
	return id
}
func (gs *GameState) UpdateSnake(msg ClientMessage) {
	gs.mutex.Lock()
	gs.snakes[msg.Id] = msg.S
	gs.mutex.Unlock()
}

// This function needs fixing. It's not marshalling the payload properly.
func (gs *GameState) GetGameStateJSON() (string, error) {
	msg, err := json.Marshal(&JSONGameStateMsg{Type: "gamestate", Payload: gs.snakes})
	if err != nil {
		log.Println("gamestate json marshal err:", err)
	}

	fmt.Printf("GetGameStateJSON: %+v\n", string(msg))
	fmt.Printf("GetGameStateJSON: %+v\n", gs.snakes)

	return string(msg), err
}
