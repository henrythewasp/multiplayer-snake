package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/websocket"
)

var Options struct {
	address string
	port    int
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("ws1.htm")
	if err != nil {
		fmt.Println(err)
	}
	items := struct {
		Name string
		Address string
		Port int
		Bot string
	}{
		Name: "SlitherSquare",
		Address: Options.address,
		Port: Options.port,
		Bot: "",
	}
	t.Execute(w, items)
}

func httpBotHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("ws1.htm")
	if err != nil {
		fmt.Println(err)
	}
	items := struct {
		Name string
		Address string
		Port int
		Bot string
	}{
		Name: "SlitherSquare",
		Address: Options.address,
		Port: Options.port,
		Bot: "Y",
	}
	t.Execute(w, items)
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage:  %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&Options.address, "address", "localhost", "address to listen on")
	flag.IntVar(&Options.port, "port", 3000, "port to listen on")
	flag.Parse()

	// Read config file for settings XXX TODO XXX (JSON / TOML / YAML)

	// Create ticker channel
	ticker := time.NewTicker(100 * time.Millisecond)
	update := make(chan ClientMessage)

	// HTTP Request Handler
	http.HandleFunc("/", httpHandler);

	// HTTP Request Handler
	http.HandleFunc("/bot", httpBotHandler);

	// WebSockets JSON Handler
	wsJSONHandler := NewJSONHandler(update)
	wsJSONServer := websocket.Server{Handler: wsJSONHandler.Accept}
	http.Handle("/json", wsJSONServer)


	// Create gameloop goroutine
	go func(h *jsonHandler) {
		gs := NewGameState()

		for {
			select {
			case msg := <-update:
				// Handle client message
				switch msg.Type {
				case "startgame":
					// Game must not already be running.
					if !gs.IsRunning {
						log.Println("Starting the game! >>>>>>>>>>>>>>>>>>> ")
						gs.StartGame()
					}

				case "addsnake":
					// Game must not already be running.
					if !gs.IsRunning {
						log.Println("addSnake MSG")
						id := gs.AddSnake(msg)

						// Send this ID back to caller
						log.Println("New snake: ", id)
						if err := h.echo(msg.ws, id); err != nil {
							log.Println("echo err:", err)
						}

						// Send out new gamestate to all users
						log.Println("Updating state to all clients")
						if data, err := gs.GetGameStateJSON(); err != nil {
							log.Println("GetGameStateJSON err:", err)
						} else if err = h.broadcast(data); err != nil {
							log.Println("broadcast err:", err)
						}
					}

				case "updatesnake":
					// Game must be running.
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


	listenAt := fmt.Sprintf("%s:%d", Options.address, Options.port)
	log.Printf("Starting to listen on: %s\n", listenAt)

	if err := http.ListenAndServe(listenAt, nil); err != nil {
		log.Fatalf("Could not start web server: %v\n", err)
	}
}
