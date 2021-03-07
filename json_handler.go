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
	update chan ClientMessage
}

type JSONWsMsg struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func NewJSONHandler(update chan ClientMessage) *jsonHandler {
	return &jsonHandler{
		update: update,
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

		msg.ws = ws   // Save websocket for this msg in case we need to reply

		// log.Println(msg)

		h.update <- msg
	}
}

func (h *jsonHandler) echo(ws *websocket.Conn, payload interface{}) error {
	return websocket.JSON.Send(ws, &JSONWsMsg{Type: "echo", Payload: payload})
}

func (h *jsonHandler) broadcast(payload interface{}) error {
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

	return err
}

func (h *jsonHandler) cleanupAll() {
	log.Println("cleanup all WS connections")
	for c := range h.conns {
		h.cleanup(c)
	}
}
func (h *jsonHandler) cleanup(ws *websocket.Conn) {
	ws.Close()
	h.mutex.Lock()

	delete(h.conns, ws)

	h.mutex.Unlock()
}
