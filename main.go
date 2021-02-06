package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/websocket"
)

func main() {
	var options struct {
		address string
		port    int
	}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage:  %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&options.address, "address", "localhost", "address to listen on")
	flag.IntVar(&options.port, "port", 3000, "port to listen on")
	flag.Parse()

	// Create ticker channel
	ticker := time.NewTicker(100 * time.Millisecond)
	update := make(chan ClientMessage)


	wsJSONHandler := NewJSONHandler(update)
	wsJSONServer := websocket.Server{Handler: wsJSONHandler.Accept}
	http.Handle("/json", wsJSONServer)

	listenAt := fmt.Sprintf("%s:%d", options.address, options.port)
	log.Printf("Starting to listen on: %s\n", listenAt)


	// Create gameloop goroutine
	go func(h *jsonHandler) {
		gs := NewGameState()

		for {
			select {
			case msg := <-update:
				// Handle client message
				switch msg.Type {
				case "startgame":
					log.Println("Starting the game! >>>>>>>>>>>>>>>>>>> ")
					gs.StartGame()

				case "addsnake":
					log.Println("addSnake MSG")
					id := gs.AddSnake(msg)
					// Send this ID back to caller
					log.Println("New snake: ", id)
					if err := h.echo(msg.ws, id); err != nil {
						log.Println("echo err:", err)
					}

				case "updatesnake":
					if gs.IsRunning {
						log.Println("updateSnake MSG")
						gs.UpdateSnake(msg)
					}

				default:
					log.Println("unknown msg.Type")
				}
			case t := <-ticker.C:
				if gs.IsRunning {
					// Send out gamestate to all clients
					fmt.Println("Tick at", t)
					if data, err := gs.GetGameStateJSON(); err != nil {
						log.Println("GetGameStateJSON err:", err)
					} else if err = h.broadcast(data); err != nil {
						log.Println("broadcast err:", err)
					}
				}
			}
		}
	}(wsJSONHandler)


	if err := http.ListenAndServe(listenAt, nil); err != nil {
		log.Fatalf("Could not start web server: %v\n", err)
	}
}
