package main

import (
	"flag"
	"github.com/StanislavStefanov/Battleships/server/player"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

func main() {
	register := make(chan *websocket.Conn)
	message := make(chan struct{})

	server := Server{
		clients:     make(map[string]*player.Player, 0),
		rooms:       make(map[string]*Room, 0),
		connectRoom: make(map[string]chan *player.Player, 0),
		register:    register,
		done:        message,
		sender:      &Sender{},
		UUID:        uuid.UUID{},
	}
	go server.run()
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(&server, w, r)
	})

	var addr = flag.String("localhost", ":8080", "http service address")

	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
