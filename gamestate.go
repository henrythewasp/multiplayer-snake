package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
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

type GameState struct {
	mutex sync.RWMutex
	config *GameConfig
	IsRunning bool 				`json:"isrunning"`
	Snakes map[string]Snake 	`json:"snakes"`
	Food []Pos					`json:"food"`
}

func NewGameState(cfg *GameConfig) *GameState {
	log.Println("Building new GameState")

	// Seed the rand package
	tn := time.Now().UnixNano()
	log.Println("UnixNano: ", tn)

	rand.Seed(tn)

	return &GameState{
		config: cfg,
		IsRunning: false,
		Snakes: make(map[string]Snake),
	}
}
func (gs *GameState) CleanUp() {
	log.Println("Game finished. Cleaning up")
	gs.IsRunning = false
	gs.Snakes = make(map[string]Snake)
	gs.Food = gs.Food[:0]
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

	// Create a 2-part snake initially. Head + 1 body.
	b = append(b, p)
	b = append(b, Pos{p.X, p.Y+1})

	return Snake{
		Body: b,
		State: Pending,
	}
}
func (gs *GameState) NewRandomFreePos(awayFromEdge bool) Pos {
	var p Pos
	for {
		p = NewRandomPos(awayFromEdge, gs.config.Game.BoardSize)
		if 	!gs.IsPosOccupied(p) &&
			!gs.IsPosOccupied(Pos{p.X, p.Y+1}) {
			break
		}
		log.Println("Pos is occupied. Try another.")
	}
	return p
}
func NewRandomPos(awayFromEdge bool, boardSize int) Pos {
	if awayFromEdge {
		// Return a position that is not too near the edge
		area := boardSize / 2
		border := boardSize / 4
		return Pos{rand.Intn(area)+border, rand.Intn(area)+border}
	} else {
		return Pos{rand.Intn(boardSize), rand.Intn(boardSize)}
	}
}

func (gs *GameState) GetGameStateJSON() (string, error) {
	msg, err := json.Marshal(&GameStateMessage{Type: "gamestate", Payload: gs})
	if err != nil {
		log.Println("gamestate json marshal err:", err)
	}

	return string(msg), err
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

	b := gs.config.Game.BoardSize

	isDead := (s.State == Dead)
	if isDead {
		log.Println("ERROR! Snake is already dead 8X")
		return
	}

	gs.mutex.Lock()

	if !isDead {
		// Check if any bounds are exceeded by head
		if h.X < 0 || h.X > b || h.Y < 0 || h.Y > b {
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
				break
			}
		}
	}

	if !isDead {
		if !gs.IsPosFood(h) {
			// Remove the LAST element (moving, not growing)
			s.Body = s.Body[:len(s.Body)-1]
		} else {
			// Remove the food and place a new one somewhere on the board
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
		// If there are no snakes left alive, end the game.
		for _, v := range gs.Snakes {
			if v.State != Dead {
				stopGame = false
				break
			}
		}

		if stopGame {
			log.Println("Game finished.")
			gs.IsRunning = false
		}
	}
}

// Need func to test if pos is food, if pos is occupied by another snake
// (except THIS snake's head)
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

