package main

import (
	"github.com/StanislavStefanov/Battleships/pkg"
	"github.com/StanislavStefanov/Battleships/pkg/game"
	"github.com/StanislavStefanov/Battleships/pkg/web"
	"github.com/StanislavStefanov/Battleships/server/automock"
	"github.com/StanislavStefanov/Battleships/server/player"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRoom_GetNextShipSize(t *testing.T) {
	t.Run("get all ship sizes", func(t *testing.T) {
		// when
		room := CreateRoom("", nil, nil)
		expectedSizes := []int{5, 4, 4, 4, 4, 3, 3, 3, 3, 3, 3, 2, 2, 2, 2, 2, 2, 2, 2}
		// then
		for i := 0; i < 19; i++ {
			size, err := room.getNextShipSize()
			assert.NoError(t, err)
			assert.Equal(t, expectedSizes[i], size)
		}
	})

	t.Run("fail, all ships already placed", func(t *testing.T) {
		// when
		room := CreateRoom("", nil, nil)
		room.ShipSizeToCount = map[int]int{2: 0}
		room.NextShipSize = 2
		// then
		_, err := room.getNextShipSize()
		assert.Equal(t, "all ships already placed", err.Error())
	})
}

func TestRoom_Join(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// when
		first := player.Player{}
		room := CreateRoom("", &first, nil)

		// then
		second := player.Player{}
		err := room.Join(&second)
		assert.NoError(t, err)
		assert.Equal(t, &second, room.Next)
	})

	t.Run("fail", func(t *testing.T) {
		// when
		first := player.Player{}
		room := CreateRoom("", &first, nil)
		second := player.Player{}
		err := room.Join(&second)
		assert.NoError(t, err)
		// then
		third := player.Player{}
		err = room.Join(&third)
		assert.Equal(t, "room is already full", err.Error())
	})
}

func TestRoom_GetRoomInfo(t *testing.T) {
	t.Run("One place taken", func(t *testing.T) {
		// when
		first := player.Player{}
		room := CreateRoom("", &first, nil)

		// then
		_, count := room.GetRoomInfo()
		assert.Equal(t, 1, count)
	})

	t.Run("Room is full", func(t *testing.T) {
		// when
		first := player.Player{}
		room := CreateRoom("", &first, nil)
		second := player.Player{}
		err := room.Join(&second)
		assert.NoError(t, err)
		// then
		_, count := room.GetRoomInfo()
		assert.Equal(t, 2, count)
	})

	t.Run("One place taken", func(t *testing.T) {
		// when
		first := player.Player{}
		room := CreateRoom("room-id", &first, nil)

		// then
		id, _ := room.GetRoomInfo()
		assert.Equal(t, "room-id", id)
	})
}

var firstConn = &websocket.Conn{}
var secondConn = &websocket.Conn{}

func getBoard() *game.Board {
	b := game.InitBoard()
	ship := game.CreateShip(3, 3, "down", 4)
	b.PlaceShip(ship)
	return b
}

func getBoardWithOneTakenField(x, y int) *game.Board {
	b := game.InitBoard()
	ship := game.CreateShip(x, y, "down", 1)
	b.PlaceShip(ship)
	return b
}

func TestShip_ProcessCommand(t *testing.T) {
	var (
		waitResp = web.Response{
			Action:  pkg.Wait,
			Message: "Wait for enemy to make his turn.",
			Args:    nil,
		}

		invalidActionResp = web.Response{
			Action:  pkg.Retry,
			Message: "Invalid action during Phase: place.",
			Args:    nil,
		}

		missingXResp = web.Response{
			Action:  pkg.Retry,
			Message: "missing value for x",
			Args:    nil,
		}

		incorrectValueTypeResp = web.Response{
			Action:  pkg.Retry,
			Message: "invalid value for y",
			Args:    nil,
		}

		missingValueForDirectionResp = web.Response{
			Action:  pkg.Retry,
			Message: "missing value for direction",
			Args:    nil,
		}

		incorrectValueTypeForDirectionResp = web.Response{
			Action:  pkg.Retry,
			Message: "invalid value for direction",
			Args:    nil,
		}

		shipPlacedSuccessfullyResp = web.Response{
			Action:  pkg.Placed,
			Message: "Ship placed successfully. Wait for opponent to make his turn.",
			Args:    map[string]interface{}{"direction": "down", "length": 5, "x": 2, "y": 2},
		}

		lastShipPlacedSuccessfullyResp = web.Response{
			Action:  pkg.Placed,
			Message: "Ship placed successfully. Wait for opponent to make his turn.",
			Args:    map[string]interface{}{"direction": "down", "length": 2, "x": 2, "y": 2},
		}

		placeShipResp = web.Response{
			Action:  pkg.PlaceShip,
			Message: "Select where to place ship with length 5",
			Args:    nil,
		}

		shootResp = web.Response{
			Action:  pkg.Shoot,
			Message: "Select filed to attack.",
			Args:    nil,
		}

		shootOutOfBoundsResp = web.Response{
			Action:  pkg.Retry,
			Message: "position out of bounds",
			Args:    nil,
		}

		shootHitResp = web.Response{
			Action:  pkg.ShootOutcome,
			Message: "",
			Args:    map[string]interface{}{"hit": true, "sunk": false, "x": 3, "y": 3},
		}

		shootWithArgsResp = web.Response{
			Action:  pkg.Shoot,
			Message: "Select filed to attack.",
			Args:    map[string]interface{}{"hit": true, "sunk": false, "x": 3, "y": 3},
		}

		winResp = web.Response{
			Action:  pkg.Win,
			Message: "Congratulations, you win!",
			Args:    nil,
		}

		defeatResp = web.Response{
			Action:  pkg.Lose,
			Message: "Defeat!",
			Args:    nil,
		}
		exitResp = web.Response{
			Action:  pkg.Win,
			Message: "Your opponent exited the game. Congratulations, you win!",
			Args:    nil,
		}
	)
	firstID := "first"
	secondID := "second"
	// given
	testCases := []struct {
		Name           string
		Room           *Room
		Request        web.Request
		Phase          string
		CurrentID      string
		NextID         string
		ResponseSender func() *automock.ResponseSender
	}{
		{
			Name: "fail when it is not your turn",
			Room: &Room{
				Current: &player.Player{
					Id: firstID,
				},
				Next: &player.Player{
					Id:   secondID,
					Conn: secondConn,
				},
				Phase: pkg.PlaceShip,
			},
			Request: web.Request{
				PlayerId: secondID,
			},
			Phase:     pkg.PlaceShip,
			CurrentID: firstID,
			NextID:    secondID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", waitResp, secondConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "fail when invalid action during phase",
			Room: &Room{
				Current: &player.Player{
					Id:   firstID,
					Conn: firstConn,
				},
				Next: &player.Player{
					Id: secondID,
				},
				Phase: pkg.PlaceShip,
			},
			Request: web.Request{
				PlayerId: firstID,
				Action:   pkg.Shoot,
			},
			Phase:     pkg.PlaceShip,
			CurrentID: firstID,
			NextID:    secondID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", invalidActionResp, firstConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "fail when x coordinate for ship is missing",
			Room: &Room{
				Current: &player.Player{
					Id:   firstID,
					Conn: firstConn,
				},
				Next: &player.Player{
					Id: secondID,
				},
				Phase: pkg.PlaceShip,
			},
			Request: web.Request{
				PlayerId: firstID,
				Action:   pkg.PlaceShip,
			},
			Phase:     pkg.PlaceShip,
			CurrentID: firstID,
			NextID:    secondID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", missingXResp, firstConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "fail when y coordinate for ship has wrong type",
			Room: &Room{
				Current: &player.Player{
					Id:   firstID,
					Conn: firstConn,
				},
				Next: &player.Player{
					Id: secondID,
				},
				Phase: pkg.PlaceShip,
			},
			Request: web.Request{
				PlayerId: firstID,
				Action:   pkg.PlaceShip,
				Args:     map[string]interface{}{"x": "2", "y": "y"},
			},
			Phase:     pkg.PlaceShip,
			CurrentID: firstID,
			NextID:    secondID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", incorrectValueTypeResp, firstConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "fail when direction is missing",
			Room: &Room{
				Current: &player.Player{
					Id:   firstID,
					Conn: firstConn,
				},
				Next: &player.Player{
					Id: secondID,
				},
				Phase: pkg.PlaceShip,
			},
			Request: web.Request{
				PlayerId: firstID,
				Action:   pkg.PlaceShip,
				Args:     map[string]interface{}{"x": "2", "y": "2"},
			},
			Phase:     pkg.PlaceShip,
			CurrentID: firstID,
			NextID:    secondID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", missingValueForDirectionResp, firstConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "fail when direction has incorrect type",
			Room: &Room{
				Current: &player.Player{
					Id:   firstID,
					Conn: firstConn,
				},
				Next: &player.Player{
					Id: secondID,
				},
				Phase: pkg.PlaceShip,
			},
			Request: web.Request{
				PlayerId: firstID,
				Action:   pkg.PlaceShip,
				Args:     map[string]interface{}{"x": "2", "y": "2", "direction": player.Player{}},
			},
			Phase:     pkg.PlaceShip,
			CurrentID: firstID,
			NextID:    secondID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", incorrectValueTypeForDirectionResp, firstConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "success when ship placed, next player should place ship",
			Room: &Room{
				Current: &player.Player{
					Id:    firstID,
					Conn:  firstConn,
					Board: game.InitBoard(),
				},
				Next: &player.Player{
					Id:   secondID,
					Conn: secondConn,
				},
				Phase:           pkg.PlaceShip,
				ShipSizeToCount: map[int]int{destroyer: destroyerCount},
				NextShipSize:    destroyer,
			},
			Request: web.Request{
				PlayerId: firstID,
				Action:   pkg.PlaceShip,
				Args:     map[string]interface{}{"x": "2", "y": "2", "direction": "down", "length": 5},
			},
			Phase:     pkg.PlaceShip,
			CurrentID: secondID,
			NextID:    firstID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", shipPlacedSuccessfullyResp, firstConn).Return(nil).Once()
				sender.On("SendResponse", placeShipResp, secondConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "success when ship placed, next player should shoot",
			Room: &Room{
				Current: &player.Player{
					Id:    firstID,
					Conn:  firstConn,
					Board: game.InitBoard(),
				},
				Next: &player.Player{
					Id:   secondID,
					Conn: secondConn,
				},
				Phase:           pkg.PlaceShip,
				ShipSizeToCount: map[int]int{boat: 0},
				NextShipSize:    boat,
			},
			Request: web.Request{
				PlayerId: firstID,
				Action:   pkg.PlaceShip,
				Args:     map[string]interface{}{"x": "2", "y": "2", "direction": "down", "length": 2},
			},
			Phase:     pkg.Shoot,
			CurrentID: secondID,
			NextID:    firstID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", lastShipPlacedSuccessfullyResp, firstConn).Return(nil).Once()
				sender.On("SendResponse", shootResp, secondConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "fail when shooting and x coordinate is missing",
			Room: &Room{
				Current: &player.Player{
					Id:   firstID,
					Conn: firstConn,
				},
				Next: &player.Player{
					Id: secondID,
				},
				Phase: pkg.Shoot,
			},
			Request: web.Request{
				PlayerId: firstID,
				Action:   pkg.Shoot,
			},
			Phase:     pkg.Shoot,
			CurrentID: firstID,
			NextID:    secondID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", missingXResp, firstConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "fail when shooting and y coordinate has wrong type",
			Room: &Room{
				Current: &player.Player{
					Id:   firstID,
					Conn: firstConn,
				},
				Next: &player.Player{
					Id: secondID,
				},
				Phase: pkg.Shoot,
			},
			Request: web.Request{
				PlayerId: firstID,
				Action:   pkg.Shoot,
				Args:     map[string]interface{}{"x": "2", "y": "y"},
			},
			Phase:     pkg.Shoot,
			CurrentID: firstID,
			NextID:    secondID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", incorrectValueTypeResp, firstConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "fail when shooting at field out of bounds",
			Room: &Room{
				Current: &player.Player{
					Id:   firstID,
					Conn: firstConn,
				},
				Next: &player.Player{
					Id: secondID,
				},
				Phase: pkg.Shoot,
			},
			Request: web.Request{
				PlayerId: firstID,
				Action:   pkg.Shoot,
				Args:     map[string]interface{}{"x": "2", "y": "12"},
			},
			Phase:     pkg.Shoot,
			CurrentID: firstID,
			NextID:    secondID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", shootOutOfBoundsResp, firstConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "success when shooting at field, next player should shoot",
			Room: &Room{
				Current: &player.Player{
					Id:    firstID,
					Conn:  firstConn,
					Board: game.InitBoard(),
				},
				Next: &player.Player{
					Id:    secondID,
					Conn:  secondConn,
					Board: getBoard(),
				},
				Phase: pkg.Shoot,
			},
			Request: web.Request{
				PlayerId: firstID,
				Action:   pkg.Shoot,
				Args:     map[string]interface{}{"x": "3", "y": "3"},
			},
			Phase:     pkg.Shoot,
			CurrentID: secondID,
			NextID:    firstID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", shootHitResp, firstConn).Return(nil).Once()
				sender.On("SendResponse", shootWithArgsResp, secondConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "enemy is defeated",
			Room: &Room{
				Current: &player.Player{
					Id:    firstID,
					Conn:  firstConn,
					Board: game.InitBoard(),
				},
				Next: &player.Player{
					Id:    secondID,
					Conn:  secondConn,
					Board: getBoardWithOneTakenField(3, 3),
				},
				Phase: pkg.Shoot,
				Done:  make(chan struct{}, 1),
			},
			Request: web.Request{
				PlayerId: firstID,
				Action:   pkg.Shoot,
				Args:     map[string]interface{}{"x": "3", "y": "3"},
			},
			Phase:     pkg.Shoot,
			CurrentID: firstID,
			NextID:    secondID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", winResp, firstConn).Return(nil).Once()
				sender.On("SendResponse", defeatResp, secondConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "opponent exited",
			Room: &Room{
				Current: &player.Player{
					Id:    firstID,
					Conn:  firstConn,
					Board: game.InitBoard(),
				},
				Next: &player.Player{
					Id:   secondID,
					Conn: secondConn,
				},
				Phase: pkg.Shoot,
				Done:  make(chan struct{}, 1),
			},
			Request: web.Request{
				PlayerId: firstID,
				Action:   pkg.Exit,
				Args:     nil,
			},
			Phase:     pkg.Shoot,
			CurrentID: firstID,
			NextID:    secondID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", exitResp, secondConn).Return(nil).Once()
				return sender
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// when
			respSender := testCase.ResponseSender()
			room := testCase.Room
			room.Sender = respSender
			room.ProcessCommand(testCase.Request)

			// then
			assert.Equal(t, testCase.Phase, room.Phase)
			assert.Equal(t, testCase.CurrentID, room.Current.Id)
			assert.Equal(t, testCase.NextID, room.Next.Id)
			respSender.AssertExpectations(t)
		})
	}
}
