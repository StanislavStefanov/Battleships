package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type Player struct {
	conn *websocket.Conn
	id   string
}

type Pair struct {
	c1 *Player
	c2 *Player
}

type myServer struct {
	clients  map[string]*Player
	register chan *websocket.Conn
	done     chan struct{}
	uuid.UUID
}

func (h *myServer) run() {
	for {
		select {
		case conn := <-h.register:
			playerId := uuid.New().String()
			fmt.Printf("register %s \n", playerId)

			player := &Player{conn: conn, id: playerId}
			h.clients[playerId] = player

			resp := make(map[string]string)
			resp["action"] = "register"
			resp["id"] = playerId
			marshal, _ := json.Marshal(resp)
			conn.WriteMessage(websocket.BinaryMessage, marshal)
			fmt.Println(h.clients)
		case <-h.done:

			fmt.Println("shutting down server")
			return
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func serveWs(hub *myServer, w http.ResponseWriter, r *http.Request) {
	fmt.Println("connection has arrived")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	hub.register <- conn
}

func main() {
	register := make(chan *websocket.Conn)
	message := make(chan struct{})

	hub := myServer{
		clients:  make(map[string]*Player, 0),
		register: register,
		done:     message,
	}
	go hub.run()
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(&hub, w, r)
	})

	var addr = flag.String("localhost", ":8080", "http service address")

	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
