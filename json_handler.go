package main

import (
	"encoding/json"
	"io"
	"log"
	"sync"

	"golang.org/x/net/websocket"
)

type jsonHandler struct {
	mutex sync.RWMutex
	conns map[*websocket.Conn]struct{}
	gameState *GameState
}

type JSONWsMsg struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type JSONBroadcastResult struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func NewJSONHandler(gs *GameState) *jsonHandler {
	return &jsonHandler{
		gameState: gs,
		conns: make(map[*websocket.Conn]struct{}),
	}
}

func (h *jsonHandler) Accept(ws *websocket.Conn) {
	defer h.cleanup(ws)

	h.mutex.Lock()
	h.conns[ws] = struct{}{}
	h.mutex.Unlock()

	for {
		var msg ClientMessage
		if err := websocket.JSON.Receive(ws, &msg); err == io.EOF {
			return
		} else if err != nil {
			log.Println("websocket.JSON.Receive err:", err)
			return
		}

		log.Println(msg)

		switch msg.Type {
		case "addsnake":
			id := h.gameState.AddSnake(msg)
			if err := h.echo(ws, id); err != nil {
				log.Println("echo err:", err)
			}
			if data, err := h.gameState.GetGameStateJSON(); err != nil {
				log.Println("GetGameStateJSON err:", err)
			} else if err = h.broadcast(ws, data); err != nil {
				log.Println("broadcast err:", err)
			}
		case "updatesnake":
			h.gameState.UpdateSnake(msg)
			if data, err := h.gameState.GetGameStateJSON(); err != nil {
				log.Println("GetGameStateJSON err:", err)
			} else if err = h.broadcast(ws, data); err != nil {
				log.Println("broadcast err:", err)
			}
		default:
			log.Println("unknown msg.Type")
			return
		}
	}
}

func (h *jsonHandler) echo(ws *websocket.Conn, payload interface{}) error {
	return websocket.JSON.Send(ws, &JSONWsMsg{Type: "echo", Payload: payload})
}

func (h *jsonHandler) broadcast(ws *websocket.Conn, payload interface{}) error {
	result := JSONBroadcastResult{Type: "broadcastResult", Payload: payload}

	msg, err := json.Marshal(&JSONWsMsg{Type: "broadcast", Payload: payload})
	if err != nil {
		log.Println("broadcast json marshal err:", err)
	} else {

		h.mutex.RLock()

		for c := range h.conns {
			if err := websocket.Message.Send(c, string(msg)); err != nil {
				h.mutex.RUnlock()
				return err
			}
		}
		h.mutex.RUnlock()
	}

	return websocket.JSON.Send(ws, &result)
}

func (h *jsonHandler) cleanup(ws *websocket.Conn) {
	ws.Close()
	h.mutex.Lock()

	delete(h.conns, ws)

	h.mutex.Unlock()
}
