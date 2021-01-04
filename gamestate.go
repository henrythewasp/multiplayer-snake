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
	food []Pos
}

func NewGameState() *GameState {
	log.Println("Building new GameState")
	return &GameState{
		snakes: make(map[string]Snake),
		food: []Pos{NewRandomPos(false)},
	}
}
func (gs *GameState) NewSnake() Snake {
	log.Println("Building new Snake")
	s := make([]Pos, 0, 10)
	s = append(s, gs.NewRandomFreePos(true))
	return s
}
// will also need to check that this is a feee point on the grid and isn't
// already taken up by a snake or food.
func (gs *GameState) NewRandomFreePos(awayFromEdge bool) Pos {
	var p Pos
	for {
		p = NewRandomPos(awayFromEdge)
		if !gs.IsPosOccupied(p) {
			break
		}
		log.Println("Pos is occupied. Try another.")
	}
	return p
}
func NewRandomPos(awayFromEdge bool) Pos {
	if awayFromEdge {
		// Return a position that is not too near the edge
		return Pos{rand.Intn(12)+4, rand.Intn(12)+4}
	} else {
		return Pos{rand.Intn(20), rand.Intn(20)}
	}
}

type JSONGameStateMsg struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func (gs *GameState) AddSnake(msg ClientMessage) string {
	gs.mutex.Lock()
	id := strconv.Itoa(len(gs.snakes)+1)
	gs.snakes[id] = gs.NewSnake()
	gs.mutex.Unlock()
	return id
}
func (gs *GameState) UpdateSnake(msg ClientMessage) {
	isDead := false

	s := msg.S
	h := s[0]

	gs.mutex.Lock()

	// Check if any bounds are exceeded by head
	if h.X < 0 || h.X > 20 || h.Y < 0 || h.Y > 20 {
		// Snake is DEAD  XXX TODO XXX
		isDead = true
	}

	// Check if head is touching anything other than food
	for _, v := range gs.snakes {
		if IsPosInSlice(h, v) {
			// Sname is DEAD XXX TODO XXX
			isDead = true
			break
		}
	}

	if !gs.IsPosFood(h) {
		// Remove the last element (moving, not growing)
		s = s[1:]
	}

	if !isDead {
		gs.snakes[msg.Id] = s
	}
	gs.mutex.Unlock()
}

// Need func to test if pos is food, if pos is occupied by another snake
// (except THIS sname's head)
func (gs *GameState) IsPosOccupied(p Pos) bool {
	if gs.IsPosFood(p) {
		return true
	}

	for _, v := range gs.snakes {
		if IsPosInSlice(p, v) {
			return true
		}
	}
	return false
}
func (gs *GameState) IsPosFood(p Pos) bool {
	return IsPosInSlice(p, gs.food)
}

func IsPosInSlice(p Pos, s []Pos) bool {
	for _, pv := range s {
		if IsEqualPos(p, pv) {
			return true
		}
	}
	return false
}
func IsEqualPos(p1 Pos, p2 Pos) bool {
	return (p1.X == p2.X && p1.Y == p2.Y)
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
