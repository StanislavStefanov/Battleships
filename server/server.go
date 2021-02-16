package main

import (
	"encoding/json"
	"fmt"
	"github.com/StanislavStefanov/Battleships/pkg"
	"github.com/StanislavStefanov/Battleships/pkg/game"
	"github.com/StanislavStefanov/Battleships/pkg/web"
	"github.com/StanislavStefanov/Battleships/server/player"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

type Server struct {
	clients     map[string]*player.Player
	rooms       map[string]*Room
	connectRoom map[string]chan *player.Player
	register    chan *websocket.Conn
	done        chan struct{}
	sender      ResponseSender
	uuid.UUID
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

//ServeWs upgrades the HTTP server connection to WebSocket protocol. Then registers client
//to whom the connection is attached and spawns new goroutine which will listen for messages
//on the connection.
func ServeWs(s *Server, w http.ResponseWriter, r *http.Request) {
	fmt.Println("connection has arrived")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	s.register <- conn
}

func (s *Server) run() {
	for {
		select {
		case conn := <-s.register:
			pl := s.RegisterClient(conn)
			if pl != nil {
				go ReadLoop(pl, s)
			}
		}
	}
}

//RegisterClient wraps the provided connection into Player and stores it into the server. The PLayer
//is assigned id(string). After the player is created the id is send back through the connection in
//args (key: "id") of a Response with action "register".
func (s *Server) RegisterClient(conn *websocket.Conn) *player.Player {
	playerId := uuid.New().String()
	fmt.Printf("register %s \n", playerId)

	pl := &player.Player{
		Conn:  conn,
		Board: game.InitBoard(),
		Id:    playerId}
	s.clients[playerId] = pl

	resp := web.BuildResponse("register", "Connected to server.", map[string]interface{}{"id": playerId})
	s.sender.SendResponse(resp, conn)

	return pl
}

//ReadLoop reads requests send by the player and calls server methods based of the action stated into the request.
//All valid actions are: exit, ls-rooms, create-room,join-room,join-random. If there is something wrong with the
//request or the command is not recognised by the server Response with status Retry is sent back through the connection.
func ReadLoop(player *player.Player, s *Server) {
	for {
		_, bytes, err := player.Conn.ReadMessage()
		if err != nil {
			fmt.Println("while read: ", err)
			s.deletePlayer(player.Id)
			return
		}

		var request web.Request
		err = json.Unmarshal(bytes, &request)
		if err != nil {
			fmt.Println("while unmarshal: ", err)
		}
		action := request.Action

		switch action {
		case pkg.Exit:
			s.deletePlayer(player.Id)
			_ = player.Conn.Close()
			return
		case pkg.ListRooms:
			rooms := s.ListRooms()
			resp := web.BuildResponse(pkg.Info, "Rooms: ", rooms)
			s.sender.SendResponse(resp, player.Conn)
		case pkg.CreateRoom:
			room := s.CreateRoom(player.Id)
			go s.RunRoom(room, s.connectRoom[room.Id])
			return
		case pkg.JoinRoom:
			roomId, ok := request.Args["roomId"].(string)
			if !ok {
				resp := web.BuildResponse(pkg.Retry, "Invalid room ID", nil)
				s.sender.SendResponse(resp, player.Conn)
				continue
			}
			if s.JoinRoom(roomId, player) {
				return
			}

		case pkg.JoinRandom:
			if s.JoinRandomRoom(player) {
				return
			}
		default:
			resp := web.BuildResponse(pkg.Retry, "unknown", nil)
			s.sender.SendResponse(resp, player.Conn)
		}
	}
}

func (s *Server) deletePlayer(id string) {
	delete(s.clients, id)
}

//ListRooms returns structured information about the rooms. The keys value pairs of the returned map contain:
//key = room id`s and values count of player in the room. The possible values are: 1 - there is only one player
//in the room and tha game hasn't started yet, 2 - the room is full and the game is in progress
func (s *Server) ListRooms() map[string]interface{} {
	roomsInfo := make(map[string]interface{})
	for _, r := range s.rooms {
		name, playersCount := r.GetRoomInfo()
		roomsInfo[name] = playersCount
	}
	return roomsInfo
}

//CreateRoom creates new room and sets the player corresponding to the provided id as First to play.
//The player is removed from the list of clients stored on the server as he is already room`s responsibility.
func (s *Server) CreateRoom(clientId string) *Room {
	roomID := uuid.New().String()
	p := s.clients[clientId]
	room := CreateRoom(roomID, p, make(chan struct{}, 1))
	s.rooms[roomID] = &room

	connect := make(chan *player.Player)
	s.connectRoom[roomID] = connect
	s.deletePlayer(clientId)
	return &room
}

//JoinRoom connects the player to the desired room. This will set him as Second to play and
//will notify the First player that he can make his turn. If the room doesn't exist or if it is
//already full the player will be notified with Response with status Retry and appropriate message.
func (s *Server) JoinRoom(roomID string, player *player.Player) bool {
	room := s.rooms[roomID]
	if room == nil {
		resp := web.BuildResponse(pkg.Retry, fmt.Sprintf("room with id %s doesnt exist", roomID), nil)
		s.sender.SendResponse(resp, player.Conn)
		return false
	}

	_, playersCount := room.GetRoomInfo()
	if playersCount == 2 {
		resp := web.BuildResponse(pkg.Retry, fmt.Sprintf("room %s is already full", roomID), nil)
		s.sender.SendResponse(resp, player.Conn)
		return false
	}

	connect := s.connectRoom[roomID]
	connect <- player
	s.deletePlayer(player.Id)
	delete(s.connectRoom, roomID)
	return true
}

//JoinRandomRoom searches for room with free place. If such room is found the player will join it.
//If there is no free room the player will receive Response with action Retry and appropriate message.
func (s *Server) JoinRandomRoom(player *player.Player) bool {
	roomID := s.findRoom()
	if roomID == "" {
		resp := web.BuildResponse(pkg.Retry, "there are no free rooms at the moment", nil)
		s.sender.SendResponse(resp, player.Conn)
		return false
	}

	return s.JoinRoom(roomID, player)
}

func (s *Server) findRoom() string {
	for id, r := range s.rooms {
		_, playerCount := r.GetRoomInfo()
		if playerCount == 1 {
			return id
		}
	}
	return ""
}

//RunRoom starts new room. Separate goroutines are spawned for the players. The room listens for commands on
//it's channels(one for each player) and on the provided join channel, where the second player should be received.
func (s *Server) RunRoom(r *Room, join chan *player.Player) {
	var wg = &sync.WaitGroup{}
	fmt.Println("Start room")

	wg.Add(1)
	go PlayerReadLoop(r.Current.Conn, r.First, wg, r.FirstExit)

	resp := web.BuildResponse(pkg.Wait,
		fmt.Sprintf("You have created room %s. Wait for an opponent to join the room.", r.Id),
		map[string]interface{}{"id": r.Id})
	s.sender.SendResponse(resp, r.Current.Conn)

	for {
		select {
		case request := <-r.First:
			fmt.Printf("Message from %s", r.Current.Id)
			r.ProcessCommand(request)
		case request := <-r.Second:
			r.ProcessCommand(request)
		case secondPlayer := <-join:
			s.joinRunningRoom(r, secondPlayer, wg, r.SecondExit)
		case <-r.Done:
			r.closeRoom()
			s.deleteRoom(r.Id)
			wg.Wait()
			return
		}
	}
}

func (s *Server) deleteRoom(id string) {
	delete(s.rooms, id)
	delete(s.connectRoom, id)
}

func (s *Server) joinRunningRoom(r *Room, secondPlayer *player.Player, wg *sync.WaitGroup, secondExit chan struct{}) {
	if r.Next == nil {
		r.Next = secondPlayer
		wg.Add(1)

		resp := web.BuildResponse(pkg.Wait,
			fmt.Sprintf("You have joined room %s. Wait for your opponent to make his turn.", r.Id),
			nil)
		s.sender.SendResponse(resp, secondPlayer.Conn)

		go PlayerReadLoop(secondPlayer.Conn, r.Second, wg, secondExit)

		r.Phase = pkg.PlaceShip

		resp = web.BuildResponse(pkg.PlaceShip,
			fmt.Sprintf("Select where to place ship with length %d", r.NextShipSize),
			nil)
		r.Sender.SendResponse(resp, r.Current.Conn)
	}
}

//PlayerReadLoop reads requests send by the player through it's connection and forwards the
//to the room through the play channel. The function will exit it's body if a message is sent
//through the exit channel.
func PlayerReadLoop(conn player.Connection, play chan web.Request, wg *sync.WaitGroup, exit chan struct{}) {
	fmt.Println("start Current read loop")
	defer wg.Done()

	for {
		select {
		case <-exit:
			return
		default:
			_, bytes, err := conn.ReadMessage()
			if err != nil {
				log.Println(err)
				return
			}

			var req web.Request
			_ = json.Unmarshal(bytes, &req)
			play <- req
			if req.Action == pkg.Exit {
				return
			}
		}
	}
}
