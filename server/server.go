package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/StanislavStefanov/Battleships/server/board"
	"github.com/StanislavStefanov/Battleships/server/player"
	"github.com/StanislavStefanov/Battleships/utils"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

//go:generate mockery -name=ResponseSender -output=automock -outpkg=automock -case=underscore
type ResponseSender interface {
	SendResponse(response utils.Response, conn *websocket.Conn)
}

type Sender struct {
}

func (s *Sender) SendResponse(response utils.Response, conn *websocket.Conn) {
	resp, err := json.Marshal(response)
	if err != nil {
		//TODO
	}
	err = conn.WriteMessage(websocket.BinaryMessage, resp)
	log.Fatal(err)
}

type Server struct {
	clients  map[string]*player.Player
	rooms    map[string]*Room
	register chan *websocket.Conn
	done     chan struct{}
	sender   ResponseSender
	uuid.UUID
}

func (s *Server) run() {
	for {
		select {
		case conn := <-s.register:
			s.registerClient(conn)
		case <-s.done:
			//TODO close open connections
			fmt.Println("shutting down server")
			return
		}
	}
}

func (s *Server) registerClient(conn *websocket.Conn) {
	//go playerReadLoop(conn, nil, nil)
	playerId := uuid.New().String()
	fmt.Printf("register %s \n", playerId)

	player := &player.Player{
		Conn:  conn,
		Board: board.InitBoard(),
		Id:    playerId}
	s.clients[playerId] = player

	resp := utils.BuildResponse("register", playerId,nil)
	marshal, _ := json.Marshal(resp)
	conn.WriteMessage(websocket.BinaryMessage, marshal)
	go readLoop(player, s, nil, nil)
}

func (s *Server) deletePlayer(id string) {
	delete(s.clients, id)
}

func (s *Server) deleteRoom(id string) {
	room := s.rooms[id]
	s.deletePlayer(room.Current.Id)
	s.deletePlayer(room.Next.Id)
	delete(s.rooms, id)
}

func (s *Server) listRooms() map[string]int {
	roomsInfo := make(map[string]int)
	for _, r := range s.rooms {
		name, playersCount := r.GetRoomInfo()
		roomsInfo[name] = playersCount
	}
	return roomsInfo
}

func (s *Server) createRoom(clientId string) {
	roomId := uuid.New().String()
	room := Room{
		Current: s.clients[clientId],
		Next:    nil,
		Id:      roomId,
	}
	s.rooms[roomId] = &room
	//TODO run room goroutine
}

func (s *Server) joinRoom(roomID string, player *player.Player) bool {
	room := s.rooms[roomID]
	if room == nil {
		resp := utils.BuildResponse(Retry, fmt.Sprintf("room with id %s doesnt exist", roomID), nil)
		s.sender.SendResponse(resp, player.Conn)
		return false
	}

	_, playersCount := room.GetRoomInfo()
	if playersCount == 2 {
		resp := utils.BuildResponse(Retry, fmt.Sprintf("room %s is already full", roomID), nil)
		s.sender.SendResponse(resp, player.Conn)
		return false
	}

	room.Next = player
	s.deletePlayer(player.Id)

	resp := utils.BuildResponse(Wait,
		fmt.Sprintf("You have joined room %s. Wait for your opponent to make his turn", roomID),
		nil)
	s.sender.SendResponse(resp, player.Conn)

	//TODO add chan
	go s.runRoom(room, nil)
	return true
}

func (s *Server) joinRandomRoom(player *player.Player) bool {
	roomID := s.findRoom()
	if roomID == "" {
		resp := utils.BuildResponse(Retry, "there are no free rooms at the moment", nil)
		s.sender.SendResponse(resp, player.Conn)
		return false
	}

	return s.joinRoom(roomID, player)
}

func (s *Server) findRoom() string {
	return ""
}

func readLoop(player *player.Player, s *Server, done chan<- struct{}, stop chan<- struct{}) {
	for {
		_, bytes, _ := player.Conn.ReadMessage()
		var msg map[string]interface{}
		json.Unmarshal(bytes, &msg)
		action := msg["action"].(string)
		fmt.Println(action)
		switch action {
		case "exit":
			s.deletePlayer(player.Id)
			player.Conn.Close()
			return
		case "ls-rooms":
			rooms := s.listRooms()
			marshal, _ := json.Marshal(rooms)
			player.Conn.WriteMessage(websocket.BinaryMessage, marshal)
		case "createRoom":
			s.createRoom(player.Id)
			return
		case "join-room":
			roomId, ok := msg["roomId"].(string)
			if !ok {
				resp := utils.BuildResponse(Retry, "Invalid room ID format", nil)
				s.sender.SendResponse(resp, player.Conn)
				continue
			}
			if s.joinRoom(roomId, player) {
				return
			}

		case "join-random":
			if s.joinRandomRoom(player) {
				return
			}
		default:
			resp := utils.BuildResponse(Retry, "unknown", nil)
			s.sender.SendResponse(resp, player.Conn)
		}
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func serveWs(s *Server, w http.ResponseWriter, r *http.Request) {
	fmt.Println("connection has arrived")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	s.register <- conn
}

func main() {
	register := make(chan *websocket.Conn)
	message := make(chan struct{})

	hub := Server{
		clients:  make(map[string]*player.Player, 0),
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
