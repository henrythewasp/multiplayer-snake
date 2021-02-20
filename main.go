package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/websocket"
	"gopkg.in/yaml.v2"
)

var Config struct {
	Server struct {
		Port int `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`
	Game struct {
		Name string
		Bot string
		BoardSize int `yaml:"size"`
		BoardColour string `yaml:"colour"`
		BGColour string `yaml:"bgcolour"`
		SnakeColour1 string `yaml:"snakecolour1"`
		SnakeColour2 string `yaml:"snakecolour2"`
		FoodColour string `yaml:"foodcolour"`
	} `yaml:"game"`
}

func readConfigFile() {
    f, err := os.Open("config.yml")
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	defer f.Close()

    decoder := yaml.NewDecoder(f)
    err = decoder.Decode(&Config)
    if err != nil {
		fmt.Println(err)
		os.Exit(2)
    }
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("ws1.htm")
	if err != nil {
		fmt.Println(err)
	}

	// Prepare template actions
	Config.Game.Name = "SlitherSquare"
	Config.Game.Bot = r.URL.Query().Get("bot")

	t.Execute(w, Config)
}

func main() {
	readConfigFile()
	fmt.Printf("%+v", Config)

	// Create ticker channel
	ticker := time.NewTicker(100 * time.Millisecond)
	update := make(chan ClientMessage)

	// HTTP Request Handler
	http.HandleFunc("/", httpHandler);

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

	listenAt := fmt.Sprintf("%s:%d", Config.Server.Host, Config.Server.Port)
	log.Printf("Starting to listen on: %s\n", listenAt)

	if err := http.ListenAndServe(listenAt, nil); err != nil {
		log.Fatalf("Could not start web server: %v\n", err)
	}
}
