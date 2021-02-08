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
	if conn == nil {
		fmt.Println("no connection")
	}
	resp, err := json.Marshal(response)
	if err != nil {
		fmt.Println("Send response: marshal error: ", err)
	}
	fmt.Println(response)
	err = conn.WriteMessage(websocket.BinaryMessage, resp)
	if err != nil {
		log.Fatal(err)
	}
}

type Server struct {
	clients     map[string]*player.Player
	rooms       map[string]*Room
	connectRoom map[string]chan *player.Player
	register    chan *websocket.Conn
	done        chan struct{}
	sender      ResponseSender
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

	pl := &player.Player{
		Conn:  conn,
		Board: board.InitBoard(),
		Id:    playerId}
	s.clients[playerId] = pl

	resp := utils.BuildResponse("register", "Connected to server.", map[string]interface{}{"id": playerId})
	marshal, _ := json.Marshal(resp)
	conn.WriteMessage(websocket.BinaryMessage, marshal)
	go readLoop(pl, s, nil, nil)
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

func (s *Server) listRooms() map[string]interface{} {
	roomsInfo := make(map[string]interface{})
	//fmt.Println(s.rooms)
	for _, r := range s.rooms {
		name, playersCount := r.GetRoomInfo()
		roomsInfo[name] = playersCount
	}
	return roomsInfo
}

func (s *Server) createRoom(clientId string) {
	roomID := uuid.New().String()
	p := s.clients[clientId]
	room := CreateRoom(roomID, p, nil)
	s.rooms[roomID] = &room

	connect := make(chan *player.Player)
	s.connectRoom[roomID] = connect
	go s.runRoom(&room, nil, connect)
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

	//room.Next = player
	s.deletePlayer(player.Id)

	connect := s.connectRoom[roomID]
	connect <- player
	delete(s.connectRoom, roomID)
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

//TODO implement
func (s *Server) findRoom() string {
	return ""
}

func readLoop(player *player.Player, s *Server, done chan<- struct{}, stop chan<- struct{}) {
	for {
		_, bytes, err := player.Conn.ReadMessage()
		if err != nil {
			fmt.Println("read error", err)
			return
		}
		var request utils.Request
		json.Unmarshal(bytes, &request)
		action := request.Action


		switch action {
		case "exit":
			s.deletePlayer(player.Id)
			player.Conn.Close()
			return
		case "ls-rooms":
			rooms := s.listRooms()
			resp := utils.BuildResponse(Info, "Rooms: ", rooms)
			s.sender.SendResponse(resp, player.Conn)
			//TODO
		case "createRoom":
			s.createRoom(player.Id)
			return
		case "join-room":
			roomId, ok := request.Args["roomId"].(string)
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
		clients:     make(map[string]*player.Player, 0),
		rooms:       make(map[string]*Room, 0),
		connectRoom: make(map[string]chan *player.Player, 0),
		register:    register,
		done:        message,
		sender:      &Sender{},
		UUID:        uuid.UUID{},
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
