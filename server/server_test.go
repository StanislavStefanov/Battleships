package main

import (
	"encoding/json"
	"errors"
	"github.com/StanislavStefanov/Battleships/pkg"
	"github.com/StanislavStefanov/Battleships/pkg/web"
	"github.com/StanislavStefanov/Battleships/server/automock"
	"github.com/StanislavStefanov/Battleships/server/player"
	connection "github.com/StanislavStefanov/Battleships/server/player/automock"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

func TestServer_Delete(t *testing.T) {
	t.Run("delete client", func(t *testing.T) {
		// when
		con := &websocket.Conn{}
		responseSender :=
			func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", mock.Anything, con).Return(nil).Once()
				return sender
			}()
		s := Server{
			clients: map[string]*player.Player{},
			sender:  responseSender,
			UUID:    uuid.UUID{},
		}
		client := s.RegisterClient(con)
		// then
		s.deletePlayer(client.Id)
		_, ok := s.clients[client.Id]
		assert.False(t, ok)
	})
}

func TestServer_DeleteRoom(t *testing.T) {
	t.Run("delete room", func(t *testing.T) {
		// when
		id := "room"
		r := &Room{
			Id: id,
		}
		s := Server{
			clients:     map[string]*player.Player{},
			rooms:       map[string]*Room{id: r},
			connectRoom: map[string]chan *player.Player{id: nil},
		}

		// then
		s.deleteRoom(id)
		_, ok := s.rooms[id]
		assert.False(t, ok)
		_, ok = s.connectRoom[id]
		assert.False(t, ok)
	})
}

func TestServer_ListRooms(t *testing.T) {
	t.Run("list rooms", func(t *testing.T) {
		// when
		id1 := "room1"
		r1 := &Room{
			Id: id1,
		}

		id2 := "room2"
		r2 := &Room{
			Id:      id2,
			Current: &player.Player{},
		}

		id3 := "room3"
		r3 := &Room{
			Id:      id3,
			Current: &player.Player{},
			Next:    &player.Player{},
		}

		s := Server{
			clients: map[string]*player.Player{},
			rooms:   map[string]*Room{id1: r1, id2: r2, id3: r3},
		}

		// then
		rooms := s.ListRooms()
		plCount, ok := rooms[id1]
		assert.True(t, ok)
		assert.Equal(t, 0, plCount)

		plCount, ok = rooms[id2]
		assert.True(t, ok)
		assert.Equal(t, 1, plCount)

		plCount, ok = rooms[id3]
		assert.True(t, ok)
		assert.Equal(t, 2, plCount)
	})
}

func TestServer_CreateRoom(t *testing.T) {
	t.Run("create room", func(t *testing.T) {
		// when
		pl := &player.Player{
			Conn:  nil,
			Board: nil,
			Id:    "player",
		}

		s := Server{
			clients:     map[string]*player.Player{"player": pl},
			rooms:       map[string]*Room{},
			connectRoom: map[string]chan *player.Player{},
		}

		// then
		room := s.CreateRoom("player")
		_, ok := s.rooms[room.Id]
		assert.True(t, ok)
		_, ok = s.connectRoom[room.Id]
		assert.True(t, ok)
		assert.Equal(t, "player", room.Current.Id)
		assert.Nil(t, room.Next)
	})
}

func TestServer_JoinRoom(t *testing.T) {
	t.Run("fail when room does not exist", func(t *testing.T) {
		// when
		con := &websocket.Conn{}
		resp := web.Response{
			Action:  pkg.Retry,
			Message: "room with id nonexisting doesnt exist",
			Args:    nil,
		}
		responseSender :=
			func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", resp, con).Return(nil).Once()
				return sender
			}()
		s := Server{
			clients: map[string]*player.Player{},
			sender:  responseSender,
			UUID:    uuid.UUID{},
		}
		pl := &player.Player{
			Conn: con,
		}
		// then
		result := s.JoinRoom("nonexisting", pl)
		assert.False(t, result)
	})
	t.Run("fail when room is already full", func(t *testing.T) {
		// when
		con := &websocket.Conn{}
		resp := web.Response{
			Action:  pkg.Retry,
			Message: "room room is already full",
			Args:    nil,
		}
		responseSender :=
			func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", resp, con).Return(nil).Once()
				return sender
			}()

		r := &Room{
			Current: &player.Player{},
			Next:    &player.Player{},
			Id:      "room",
		}
		s := Server{
			clients: map[string]*player.Player{},
			rooms:   map[string]*Room{"room": r},
			sender:  responseSender,
			UUID:    uuid.UUID{},
		}
		pl := &player.Player{
			Conn: con,
		}
		// then
		result := s.JoinRoom("room", pl)
		assert.False(t, result)
	})
	t.Run("success", func(t *testing.T) {
		// when
		con := &websocket.Conn{}

		r := &Room{
			Current: &player.Player{},
			Id:      "room",
		}

		pl := &player.Player{
			Conn: con,
			Id:   "player",
		}

		connect := make(chan *player.Player, 1)
		s := Server{
			clients:     map[string]*player.Player{"player": pl},
			rooms:       map[string]*Room{"room": r},
			connectRoom: map[string]chan *player.Player{"room": connect},
			UUID:        uuid.UUID{},
		}

		// then
		result := s.JoinRoom("room", pl)
		assert.True(t, result)
		joined := <-connect
		assert.Equal(t, pl, joined)
	})
}

func TestServer_JoinRandomRoom(t *testing.T) {
	t.Run("fail when there are no rooms", func(t *testing.T) {
		// when
		con := &websocket.Conn{}
		resp := web.Response{
			Action:  pkg.Retry,
			Message: "there are no free rooms at the moment",
			Args:    nil,
		}
		responseSender :=
			func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", resp, con).Return(nil).Once()
				return sender
			}()
		s := Server{
			rooms:  map[string]*Room{},
			sender: responseSender,
			UUID:   uuid.UUID{},
		}
		pl := &player.Player{
			Conn: con,
		}
		// then
		result := s.JoinRandomRoom(pl)
		assert.False(t, result)
	})
	t.Run("fail when there are no free places in the rooms", func(t *testing.T) {
		// when
		con := &websocket.Conn{}
		resp := web.Response{
			Action:  pkg.Retry,
			Message: "there are no free rooms at the moment",
			Args:    nil,
		}
		responseSender :=
			func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", resp, con).Return(nil).Once()
				return sender
			}()
		s := Server{
			rooms: map[string]*Room{"room": &Room{
				Current: &player.Player{},
				Next:    &player.Player{},
				Id:      "room",
			}},
			sender: responseSender,
			UUID:   uuid.UUID{},
		}
		pl := &player.Player{
			Conn: con,
		}
		// then
		result := s.JoinRandomRoom(pl)
		assert.False(t, result)
	})
	t.Run("success", func(t *testing.T) {
		// when
		con := &websocket.Conn{}

		r := &Room{
			Current: &player.Player{},
			Id:      "room",
		}

		pl := &player.Player{
			Conn: con,
			Id:   "player",
		}

		connect := make(chan *player.Player, 1)
		s := Server{
			clients:     map[string]*player.Player{"player": pl},
			rooms:       map[string]*Room{"room": r},
			connectRoom: map[string]chan *player.Player{"room": connect},
			UUID:        uuid.UUID{},
		}

		// then
		result := s.JoinRandomRoom(pl)
		assert.True(t, result)
		joined := <-connect
		assert.Equal(t, pl, joined)
	})
}

func TestServer_ReadLoop(t *testing.T) {
	t.Run("exit", func(t *testing.T) {
		// when
		req := web.BuildRequest("id", pkg.Exit, nil)
		bytes, _ := json.Marshal(req)
		con := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("ReadMessage").Return(0, bytes, nil).Once()
			con.On("Close").Return(nil).Once()
			return con
		}()

		pl := &player.Player{
			Id:   "player",
			Conn: con,
		}

		s := &Server{
			clients: map[string]*player.Player{"player": pl},
			UUID:    uuid.UUID{},
		}

		// then
		ReadLoop(pl, s)

		_, ok := s.clients["player"]
		assert.False(t, ok)
		con.AssertExpectations(t)
	})
	t.Run("list rooms", func(t *testing.T) {
		// when

		req := web.BuildRequest("id", pkg.ListRooms, nil)
		listRooms, _ := json.Marshal(req)

		roomsInfo := map[string]interface{}{"room1": 0, "room2": 1, "room3": 2}

		resp := web.BuildResponse(pkg.Info, "Rooms: ", roomsInfo)
		rooms, _ := json.Marshal(resp)

		req = web.BuildRequest("id", pkg.Exit, nil)
		exit, _ := json.Marshal(req)

		con := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("ReadMessage").Return(0, listRooms, nil).Once()
			con.On("WriteMessage", websocket.BinaryMessage, rooms).Return(nil).Once()
			con.On("ReadMessage").Return(0, exit, nil).Once()

			con.On("Close").Return(nil).Once()
			return con
		}()

		pl := &player.Player{
			Id:   "player",
			Conn: con,
		}

		id1 := "room1"
		r1 := &Room{
			Id: id1,
		}

		id2 := "room2"
		r2 := &Room{
			Id:      id2,
			Current: &player.Player{},
		}

		id3 := "room3"
		r3 := &Room{
			Id:      id3,
			Current: &player.Player{},
			Next:    &player.Player{},
		}

		s := &Server{
			clients: map[string]*player.Player{"player": pl},
			sender:  &Sender{},
			UUID:    uuid.UUID{},
			rooms:   map[string]*Room{id1: r1, id2: r2, id3: r3},
		}

		// then
		ReadLoop(pl, s)

		_, ok := s.clients["player"]
		assert.False(t, ok)
		con.AssertExpectations(t)
	})
	t.Run("create room", func(t *testing.T) {
		// when
		req := web.BuildRequest("player", pkg.CreateRoom, nil)
		create, _ := json.Marshal(req)

		req = web.BuildRequest("player", pkg.Exit, nil)
		exit, _ := json.Marshal(req)
		con := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("ReadMessage").Return(0, create, nil).Once()
			con.On("WriteMessage", websocket.BinaryMessage, mock.Anything).Return(nil).Once()
			con.On("ReadMessage").Return(0, exit, nil).Once()

			con.On("Close").Return(nil).Once()
			return con
		}()

		pl := &player.Player{
			Id:   "player",
			Conn: con,
		}

		s := &Server{
			clients:     map[string]*player.Player{"player": pl},
			rooms:       map[string]*Room{},
			connectRoom: map[string]chan *player.Player{},
			sender:      &Sender{},
			UUID:        uuid.UUID{},
		}

		// then
		ReadLoop(pl, s)

		assert.Equal(t, 1, len(s.rooms))

		//for _, r := range s.rooms {
		//	r.Done <- struct{}{}
		//}

		assert.Equal(t, 1, len(s.connectRoom))
		time.Sleep(2 * time.Second)
		con.AssertExpectations(t)

	})
	t.Run("success join room", func(t *testing.T) {
		// when

		req := web.BuildRequest("id", pkg.JoinRoom, map[string]interface{}{"roomId": "room"})
		joinRoom, _ := json.Marshal(req)

		con := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("ReadMessage").Return(0, joinRoom, nil).Once()

			return con
		}()

		pl := &player.Player{
			Id:   "player",
			Conn: con,
		}

		id := "room"
		r := &Room{
			Id: id,
			Current: &player.Player{Conn: func() *connection.Connection {
				con := &connection.Connection{}
				con.On("ReadMessage").Return(0, joinRoom, nil).Once()

				return con
			}()},
		}

		s := &Server{
			clients:     map[string]*player.Player{"player": pl},
			sender:      &Sender{},
			UUID:        uuid.UUID{},
			rooms:       map[string]*Room{id: r},
			connectRoom: map[string]chan *player.Player{"room": make(chan *player.Player, 1)},
		}

		// then
		ReadLoop(pl, s)

		_, ok := s.clients["player"]
		assert.False(t, ok)

		assert.Equal(t, 1, len(s.rooms))
		assert.Equal(t, 0, len(s.connectRoom))
		assert.Equal(t, 0, len(s.clients))

		con.AssertExpectations(t)
	})
	t.Run("fail to join room when room id format is invalid", func(t *testing.T) {
		// when

		req := web.BuildRequest("id", pkg.JoinRoom, map[string]interface{}{"roomId": 2})
		joinRoom, _ := json.Marshal(req)

		resp := web.BuildResponse(pkg.Retry, "Invalid room ID", nil)
		errorResp, _ := json.Marshal(resp)

		req = web.BuildRequest("id", pkg.Exit, nil)
		exit, _ := json.Marshal(req)

		con := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("ReadMessage").Return(0, joinRoom, nil).Once()
			con.On("WriteMessage", websocket.BinaryMessage, errorResp).Return(nil).Once()
			con.On("ReadMessage").Return(0, exit, nil).Once()

			con.On("Close").Return(nil).Once()
			return con
		}()

		pl := &player.Player{
			Id:   "player",
			Conn: con,
		}

		s := &Server{
			clients: map[string]*player.Player{"player": pl},
			sender:  &Sender{},
			UUID:    uuid.UUID{},
		}

		// then
		ReadLoop(pl, s)

		con.AssertExpectations(t)
	})
	t.Run("success join random room", func(t *testing.T) {
		// when
		req := web.BuildRequest("id", pkg.JoinRandom, nil)
		joinRandom, _ := json.Marshal(req)

		con := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("ReadMessage").Return(0, joinRandom, nil).Once()

			return con
		}()

		pl := &player.Player{
			Id:   "player",
			Conn: con,
		}

		id := "room"
		r := &Room{
			Id: id,
			Current: &player.Player{Conn: func() *connection.Connection {
				con := &connection.Connection{}
				con.On("ReadMessage").Return(0, joinRandom, nil).Once()

				return con
			}()},
		}

		s := &Server{
			clients:     map[string]*player.Player{"player": pl},
			sender:      &Sender{},
			UUID:        uuid.UUID{},
			rooms:       map[string]*Room{id: r},
			connectRoom: map[string]chan *player.Player{"room": make(chan *player.Player, 1)},
		}

		// then
		ReadLoop(pl, s)

		_, ok := s.clients["player"]
		assert.False(t, ok)

		assert.Equal(t, 1, len(s.rooms))
		assert.Equal(t, 0, len(s.connectRoom))
		assert.Equal(t, 0, len(s.clients))

		con.AssertExpectations(t)
	})
	t.Run("fail to join room when there are no available rooms", func(t *testing.T) {
		// when

		req := web.BuildRequest("id", pkg.JoinRandom, nil)
		joinRandom, _ := json.Marshal(req)

		resp := web.BuildResponse(pkg.Retry, "there are no free rooms at the moment", nil)
		errorResp, _ := json.Marshal(resp)

		req = web.BuildRequest("id", pkg.Exit, nil)
		exit, _ := json.Marshal(req)

		con := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("ReadMessage").Return(0, joinRandom, nil).Once()
			con.On("WriteMessage", websocket.BinaryMessage, errorResp).Return(nil).Once()
			con.On("ReadMessage").Return(0, exit, nil).Once()

			con.On("Close").Return(nil).Once()
			return con
		}()

		pl := &player.Player{
			Id:   "player",
			Conn: con,
		}

		s := &Server{
			clients: map[string]*player.Player{"player": pl},
			sender:  &Sender{},
			UUID:    uuid.UUID{},
		}

		// then
		ReadLoop(pl, s)

		con.AssertExpectations(t)
	})
	t.Run("unknown action", func(t *testing.T) {
		// when
		req := web.BuildRequest("id", "invalid action", nil)
		invalidAction, _ := json.Marshal(req)

		resp := web.BuildResponse(pkg.Retry, "unknown", nil)
		unknown, _ := json.Marshal(resp)

		req = web.BuildRequest("id", pkg.Exit, nil)
		exit, _ := json.Marshal(req)

		con := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("ReadMessage").Return(0, invalidAction, nil).Once()
			con.On("WriteMessage", websocket.BinaryMessage, unknown).Return(nil).Once()
			con.On("ReadMessage").Return(0, exit, nil).Once()

			con.On("Close").Return(nil).Once()
			return con
		}()

		pl := &player.Player{
			Id:   "player",
			Conn: con,
		}

		s := &Server{
			clients: map[string]*player.Player{"player": pl},
			sender:  &Sender{},
			UUID:    uuid.UUID{},
		}

		// then
		ReadLoop(pl, s)

		_, ok := s.clients["player"]
		assert.False(t, ok)
		con.AssertExpectations(t)
	})
	t.Run("fail when error occurs while reading from connection", func(t *testing.T) {
		// when
		con := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("ReadMessage").Return(0, nil, errors.New("read failure")).Once()
			return con
		}()

		pl := &player.Player{
			Id:   "player",
			Conn: con,
		}

		s := &Server{
			clients: map[string]*player.Player{"player": pl},
			sender:  &Sender{},
			UUID:    uuid.UUID{},
		}

		// then
		ReadLoop(pl, s)

		_, ok := s.clients["player"]
		assert.False(t, ok)
		con.AssertExpectations(t)
	})
}

func TestServer_RunRoom(t *testing.T) {
	t.Run("success when receive actions from first Player", func(t *testing.T) {
		// when
		createdRoom := web.BuildResponse(pkg.Wait, "You have created room room. Wait for an opponent to join the room.", map[string]interface{}{"id": "room"})
		createdRoomMarshal, _ := json.Marshal(createdRoom)

		resp := web.BuildResponse(pkg.Retry, "Invalid action during Phase: phase.", nil)
		marshal, _ := json.Marshal(resp)

		exit := web.BuildRequest("first", pkg.Exit, nil)
		exitMarshal, _ := json.Marshal(exit)
		firstConn := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("WriteMessage", websocket.BinaryMessage, createdRoomMarshal).Return(nil).Once()
			con.On("WriteMessage", websocket.BinaryMessage, marshal).Return(nil).Once()
			con.On("ReadMessage").Return(0, exitMarshal, nil).Once()
			con.On("Close").Return(nil).Once()
			return con
		}()

		first := &player.Player{
			Conn:  firstConn,
			Board: nil,
			Id:    "first",
		}

		win := web.BuildResponse(pkg.Win, "Your opponent exited the game. Congratulations, you win!", nil)
		winMarshal, _ := json.Marshal(win)
		secondConn := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("WriteMessage", websocket.BinaryMessage, winMarshal).Return(nil).Once()
			con.On("Close").Return(nil).Once()
			return con
		}()

		second := &player.Player{
			Conn:  secondConn,
			Board: nil,
			Id:    "second",
		}

		room := &Room{
			Current:         first,
			Next:            second,
			First:           make(chan web.Request, 2),
			Second:          make(chan web.Request),
			Done:            make(chan struct{}, 1),
			Phase:           "phase",
			ShipSizeToCount: nil,
			NextShipSize:    0,
			Id:              "room",
			Sender:          &Sender{},
		}
		create := web.BuildRequest("first", "test", nil)
		room.First <- create

		s := &Server{
			rooms:  map[string]*Room{"room": room},
			sender: &Sender{},
			UUID:   uuid.UUID{},
		}

		// then
		s.RunRoom(room, nil)

		assert.Equal(t, 0, len(s.rooms))
		assert.Equal(t, 0, len(s.connectRoom))

		firstConn.AssertExpectations(t)
		secondConn.AssertExpectations(t)
	})
	t.Run("success join second user", func(t *testing.T) {
		// when
		createdRoom := web.BuildResponse(pkg.Wait, "You have created room room. Wait for an opponent to join the room.", map[string]interface{}{"id": "room"})
		createdRoomMarshal, _ := json.Marshal(createdRoom)

		resp := web.BuildResponse(pkg.PlaceShip, "Select where to place ship with length 5", nil)
		place, _ := json.Marshal(resp)

		firstConn := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("WriteMessage", websocket.BinaryMessage, createdRoomMarshal).Return(nil).Once()
			con.On("WriteMessage", websocket.BinaryMessage, place).Return(nil).Once()
			con.On("Close").Return(nil).Once()
			return con
		}()

		first := &player.Player{
			Conn:  firstConn,
			Board: nil,
			Id:    "first",
		}

		resp = web.BuildResponse(pkg.Wait, "You have joined room room. Wait for your opponent to make his turn.", nil)
		joined, _ := json.Marshal(resp)

		secondConn := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("WriteMessage", websocket.BinaryMessage, joined).Return(nil).Once()
			con.On("Close").Return(nil).Once()
			return con
		}()

		second := &player.Player{
			Conn:  secondConn,
			Board: nil,
			Id:    "second",
		}

		firstExit := make(chan struct{}, 2)
		firstExit <- struct{}{}
		secondExit := make(chan struct{}, 2)
		secondExit <- struct{}{}

		done := make(chan struct{}, 1)
		go func() {
			time.Sleep(1 * time.Second)
			done <- struct{}{}
		}()

		room := &Room{
			Current:    first,
			Sender:     &Sender{},
			FirstExit:  firstExit,
			SecondExit: secondExit,
			Id:         "room",
			Done:       done,
			NextShipSize: destroyer,
		}

		s := &Server{
			sender: &Sender{},
			UUID:   uuid.UUID{},
		}

		join := make(chan *player.Player, 1)
		join <- second
		// then
		s.RunRoom(room, join)

		time.Sleep(1 * time.Second)
		firstConn.AssertExpectations(t)
		secondConn.AssertExpectations(t)
	})
	t.Run("success receive action from second user", func(t *testing.T) {
		// when
		createdRoom := web.BuildResponse(pkg.Wait, "You have created room room. Wait for an opponent to join the room.", map[string]interface{}{"id": "room"})
		createdRoomMarshal, _ := json.Marshal(createdRoom)

		win := web.BuildResponse(pkg.Win, "Your opponent exited the game. Congratulations, you win!", nil)
		winMarshal, _ := json.Marshal(win)
		firstConn := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("WriteMessage", websocket.BinaryMessage, createdRoomMarshal).Return(nil).Once()
			con.On("WriteMessage", websocket.BinaryMessage, winMarshal).Return(nil).Once()
			con.On("Close").Return(nil).Once()
			return con
		}()

		first := &player.Player{
			Conn:  firstConn,
			Board: nil,
			Id:    "first",
		}

		secondConn := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("Close").Return(nil).Once()
			return con
		}()

		second := &player.Player{
			Conn:  secondConn,
			Board: nil,
			Id:    "second",
		}

		firstExit := make(chan struct{}, 2)
		firstExit <- struct{}{}
		secondExit := make(chan struct{}, 2)
		secondExit <- struct{}{}

		input := make(chan web.Request, 1)
		exit := web.BuildRequest("second", pkg.Exit, nil)
		input <- exit

		room := &Room{
			Current:    first,
			Next:       second,
			Second:     input,
			Sender:     &Sender{},
			FirstExit:  firstExit,
			SecondExit: secondExit,
			Id:         "room",
			Done:       make(chan struct{}, 1),
		}

		s := &Server{
			sender: &Sender{},
			UUID:   uuid.UUID{},
		}

		// then
		s.RunRoom(room, nil)

		firstConn.AssertExpectations(t)
		secondConn.AssertExpectations(t)
	})
}

func TestServer_joinRunningRoom(t *testing.T) {
	t.Run("join", func(t *testing.T) {
		// when
		resp := web.BuildResponse(pkg.PlaceShip, "Select where to place ship with length 5", nil)
		place, _ := json.Marshal(resp)
		firstConn := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("WriteMessage", websocket.BinaryMessage, place).Return(nil).Once()
			return con
		}()

		first := &player.Player{
			Conn:  firstConn,
			Board: nil,
			Id:    "first",
		}

		resp = web.BuildResponse(pkg.Wait, "You have joined room room. Wait for your opponent to make his turn.", nil)
		joined, _ := json.Marshal(resp)
		secondConn := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("WriteMessage", websocket.BinaryMessage, joined).Return(nil).Once()
			return con
		}()

		second := &player.Player{
			Conn:  secondConn,
			Board: nil,
			Id:    "second",
		}

		room := &Room{
			Current: first,
			Sender:  &Sender{},
			Id:      "room",
			NextShipSize: destroyer,
		}

		s := &Server{
			sender: &Sender{},
			UUID:   uuid.UUID{},
		}

		channel := make(chan struct{}, 1)
		channel <- struct{}{}
		// then
		s.joinRunningRoom(room, second, &sync.WaitGroup{}, channel)

		firstConn.AssertExpectations(t)
		secondConn.AssertExpectations(t)
	})
}
