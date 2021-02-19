package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

type Pos struct {
	X int	`json:"x"`
	Y int	`json:"y"`
}

type SnakeState int

const (
	Pending SnakeState = iota
	Alive
	Dead
)
type Snake struct {
	Body []Pos 			`json:"body"`
	State SnakeState 	`json:"state"`
}

// Is ths struct required? XXX
type JSONSnakeData struct {
	Id string	`json:"id"`
	S Snake		`json:"s"`
}
type GameState struct {
	mutex sync.RWMutex
	IsRunning bool 				`json:"isrunning"`
	Snakes map[string]Snake 	`json:"snakes"`
	Food []Pos					`json:"food"`
}

func NewGameState() *GameState {
	log.Println("Building new GameState")

	// Seed the rand package
	tn := time.Now().UnixNano()
	log.Println("UnixName: ", tn)

	rand.Seed(tn)

	return &GameState{
		IsRunning: false,
		Snakes: make(map[string]Snake),
	}
}
func (gs *GameState) StartGame() {
	log.Println("Starting Game")

	// Add 1 random food for each snake
	gs.Food = make([]Pos, 0, len(gs.Snakes))
	for range gs.Snakes {
		gs.Food = append(gs.Food, gs.NewRandomFreePos(false))
	}

	// Start the game
	gs.IsRunning = true
}
func (gs *GameState) NewSnake() Snake {
	log.Println("Building new Snake")
	b := make([]Pos, 0, 10)
	p := gs.NewRandomFreePos(true)
	b = append(b, p)
	b = append(b, Pos{p.X, p.Y+1})
	return Snake{
		Body: b,
		State: Pending,
	}
}
// will also need to check that this is a feee point on the grid and isn't
// already taken up by a snake or food.
func (gs *GameState) NewRandomFreePos(awayFromEdge bool) Pos {
	var p Pos
	for {
		p = NewRandomPos(awayFromEdge)
		if 	!gs.IsPosOccupied(p) &&
			!gs.IsPosOccupied(Pos{p.X, p.Y+1}) {
			break
		}
		log.Println("Pos is occupied. Try another.")
	}
	return p
}
func NewRandomPos(awayFromEdge bool) Pos {
	if awayFromEdge {
		// Return a position that is not too near the edge
		return Pos{rand.Intn(32)+4, rand.Intn(32)+4}
	} else {
		return Pos{rand.Intn(40), rand.Intn(40)}
	}
}

type JSONGameStateMsg struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func (gs *GameState) AddSnake(msg ClientMessage) string {
	gs.mutex.Lock()
	id := strconv.Itoa(len(gs.Snakes)+1)
	gs.Snakes[id] = gs.NewSnake()
	gs.mutex.Unlock()
	return id
}
func (gs *GameState) UpdateSnake(msg ClientMessage) {
	s := msg.S
	h := s.Body[0]

	isDead := (s.State == Dead)
	if isDead {
		log.Println("ERROR! Snake is already dead 8X")
		return
	}

	gs.mutex.Lock()

	if !isDead {
		// Check if any bounds are exceeded by head
		if h.X < 0 || h.X > 40 || h.Y < 0 || h.Y > 40 {
			// Snake is DEAD
			fmt.Printf("(0) Snake is now dead: %+v\n", h)
			isDead = true
		}
	}

	if !isDead {
		// Check if head is touching anything other than food
		for i, v := range gs.Snakes {
			parts := v.Body
			if (msg.Id == i) {
				// Same snake. Exclude the head when checking.
				parts = parts[1:]
			}
			if IsPosInSlice(h, parts) {
				// Snake is DEAD.
				isDead = true
				fmt.Printf("(1) Snake is now dead: %+v\n", h)
				fmt.Printf("(1) gs %+v\n", gs)
				break
			}
		}
	}

	if !isDead {
		if !gs.IsPosFood(h) {
			// Remove the LAST element (moving, not growing)
			s.Body = s.Body[:len(s.Body)-1]
		} else {
			// Remove the food and place a new one
			gs.Food = RemovePosFromSlice(h, gs.Food)
			gs.Food = append(gs.Food, gs.NewRandomFreePos(false))
		}

	} else {
		// Update state for snake on gamestate so client knows that it's dead
		log.Println("Oh no! Snake is now dead 8X")
		s.State = Dead
	}

	gs.Snakes[msg.Id] = s
	gs.mutex.Unlock()

	if (isDead) {
		stopGame := true
		// If there are no snakes left alive, stop the game.
		for _, v := range gs.Snakes {
			if v.State != Dead {
				stopGame = false
				break
			}
		}

		if stopGame {
			// XXX TODO need to stop game rather than just exit
			os.Exit(1)
		}
	}
}

// Need func to test if pos is food, if pos is occupied by another snake
// (except THIS sname's head)
func (gs *GameState) IsPosOccupied(p Pos) bool {
	if gs.IsPosFood(p) {
		return true
	}

	for _, v := range gs.Snakes {
		if IsPosInSlice(p, v.Body) {
			return true
		}
	}
	return false
}
func (gs *GameState) IsPosFood(p Pos) bool {
	return IsPosInSlice(p, gs.Food)
}

// https://stackoverflow.com/questions/37334119/how-to-delete-an-element-from-a-slice-in-golang
func RemovePosFromSlice(p Pos, s []Pos) []Pos {
	for i, pv := range s {
		if IsEqualPos(p, pv) {
			s[len(s)-1], s[i] = s[i], s[len(s)-1]
			return s[:len(s)-1]
		}
	}
	return s
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
	msg, err := json.Marshal(&JSONGameStateMsg{Type: "gamestate", Payload: gs})
	if err != nil {
		log.Println("gamestate json marshal err:", err)
	}

	// fmt.Printf("GetGameStateJSON: %+v\n", string(msg))
	// fmt.Printf("GetGameStateJSON: %+v\n", gs)

	return string(msg), err
}
