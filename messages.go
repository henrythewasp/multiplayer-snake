package main

import (
	"golang.org/x/net/websocket"
)

type ClientMessage struct {
	Type	string  `json:"type"`
	Id		string  `json:"id"`
	S		Snake   `json:"s"`
	ws 		*websocket.Conn
}

type GameStateMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
