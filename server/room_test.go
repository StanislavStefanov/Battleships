package main

import (
	"github.com/StanislavStefanov/Battleships/server/automock"
	"github.com/StanislavStefanov/Battleships/server/board"
	"github.com/StanislavStefanov/Battleships/server/player"
	"github.com/StanislavStefanov/Battleships/utils"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRoom_GetNextShipSize(t *testing.T) {
	t.Run("get all ship sizes", func(t *testing.T) {
		// when
		room := CreateRoom("", nil, nil)
		expectedSizes := []int{5, 5, 4, 4, 4, 4, 3, 3, 3, 3, 3, 3, 2, 2, 2, 2, 2, 2, 2, 2}
		// then
		for i := 0; i < 20; i++ {
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

func getBoard() *board.Board {
	b := board.InitBoard()
	ship := board.CreateShip(3, 3, "down", 4)
	b.PlaceShip(ship)
	return b
}

func getBoardWithOneTakenField(x, y int) *board.Board {
	b := board.InitBoard()
	ship := board.CreateShip(x, y, "down", 1)
	b.PlaceShip(ship)
	return b
}

func TestShip_ProcessCommand(t *testing.T) {
	var (
		waitResp = utils.Response{
			Action:  Wait,
			Message: "Wait for enemy to make his turn.",
			Args:    nil,
		}

		invalidActionResp = utils.Response{
			Action:  Retry,
			Message: "Invalid action during Phase: place.",
			Args:    nil,
		}

		missingXResp = utils.Response{
			Action:  Retry,
			Message: "missing value for x",
			Args:    nil,
		}

		incorrectValueTypeResp = utils.Response{
			Action:  Retry,
			Message: "invalid value for y",
			Args:    nil,
		}

		missingValueForDirectionResp = utils.Response{
			Action:  Retry,
			Message: "missing value for direction",
			Args:    nil,
		}

		incorrectValueTypeForDirectionResp = utils.Response{
			Action:  Retry,
			Message: "invalid value for direction",
			Args:    nil,
		}

		missingValueForLengthResp = utils.Response{
			Action:  Retry,
			Message: "missing value for length",
			Args:    nil,
		}

		incorrectValueTypeForLengthResp = utils.Response{
			Action:  Retry,
			Message: "invalid value for length",
			Args:    nil,
		}

		shipPlacedSuccessfullyResp = utils.Response{
			Action:  Wait,
			Message: "Ship placed successfully. Wait for opponent to make his turn.",
			Args:    nil,
		}

		placeShipResp = utils.Response{
			Action:  PlaceShip,
			Message: "Select where to place ship with length 5",
			Args:    nil,
		}

		shootResp = utils.Response{
			Action:  Shoot,
			Message: "Select filed to attack.",
			Args:    nil,
		}

		shootOutOfBoundsResp = utils.Response{
			Action:  Retry,
			Message: "position out of bounds",
			Args:    nil,
		}

		shootHitResp = utils.Response{
			Action:  Wait,
			Message: "",
			Args:    map[string]interface{}{"hit": true, "x": 3, "y": 3},
		}

		shootWithArgsResp = utils.Response{
			Action:  Shoot,
			Message: "Select filed to attack.",
			Args:    map[string]interface{}{"hit": true, "x": 3, "y": 3},
		}

		winResp = utils.Response{
			Action:  Win,
			Message: "Congratulations, you win!",
			Args:    nil,
		}

		defeatResp = utils.Response{
			Action:  Lose,
			Message: "Defeat!",
			Args:    nil,
		}
		exitResp = utils.Response{
			Action:  Win,
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
		Request        utils.Request
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
				Phase: PlaceShip,
			},
			Request: utils.Request{
				PlayerId: secondID,
			},
			Phase:     PlaceShip,
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
				Phase: PlaceShip,
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   Shoot,
			},
			Phase:     PlaceShip,
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
				Phase: PlaceShip,
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   PlaceShip,
			},
			Phase:     PlaceShip,
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
				Phase: PlaceShip,
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   PlaceShip,
				Args:     map[string]interface{}{"x": 2, "y": "y"},
			},
			Phase:     PlaceShip,
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
				Phase: PlaceShip,
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   PlaceShip,
				Args:     map[string]interface{}{"x": 2, "y": 2},
			},
			Phase:     PlaceShip,
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
				Phase: PlaceShip,
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   PlaceShip,
				Args:     map[string]interface{}{"x": 2, "y": 2, "direction": player.Player{}},
			},
			Phase:     PlaceShip,
			CurrentID: firstID,
			NextID:    secondID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", incorrectValueTypeForDirectionResp, firstConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "fail when length is missing",
			Room: &Room{
				Current: &player.Player{
					Id:   firstID,
					Conn: firstConn,
				},
				Next: &player.Player{
					Id: secondID,
				},
				Phase: PlaceShip,
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   PlaceShip,
				Args:     map[string]interface{}{"x": 2, "y": 2, "direction": "down"},
			},
			Phase:     PlaceShip,
			CurrentID: firstID,
			NextID:    secondID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", missingValueForLengthResp, firstConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "fail when length has incorrect type",
			Room: &Room{
				Current: &player.Player{
					Id:   firstID,
					Conn: firstConn,
				},
				Next: &player.Player{
					Id: secondID,
				},
				Phase: PlaceShip,
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   PlaceShip,
				Args:     map[string]interface{}{"x": 2, "y": 2, "direction": "down", "length": "invalid"},
			},
			Phase:     PlaceShip,
			CurrentID: firstID,
			NextID:    secondID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", incorrectValueTypeForLengthResp, firstConn).Return(nil).Once()
				return sender
			},
		},
		{
			Name: "success when ship placed, next player should place ship",
			Room: &Room{
				Current: &player.Player{
					Id:    firstID,
					Conn:  firstConn,
					Board: board.InitBoard(),
				},
				Next: &player.Player{
					Id:   secondID,
					Conn: secondConn,
				},
				Phase:           PlaceShip,
				ShipSizeToCount: map[int]int{destroyer: destroyerCount},
				NextShipSize:    destroyer,
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   PlaceShip,
				Args:     map[string]interface{}{"x": 2, "y": 2, "direction": "down", "length": 5},
			},
			Phase:     PlaceShip,
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
					Board: board.InitBoard(),
				},
				Next: &player.Player{
					Id:   secondID,
					Conn: secondConn,
				},
				Phase:           PlaceShip,
				ShipSizeToCount: map[int]int{boat: 0},
				NextShipSize:    boat,
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   PlaceShip,
				Args:     map[string]interface{}{"x": 2, "y": 2, "direction": "down", "length": 2},
			},
			Phase:     Shoot,
			CurrentID: secondID,
			NextID:    firstID,
			ResponseSender: func() *automock.ResponseSender {
				sender := &automock.ResponseSender{}
				sender.On("SendResponse", shipPlacedSuccessfullyResp, firstConn).Return(nil).Once()
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
				Phase: Shoot,
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   Shoot,
			},
			Phase:     Shoot,
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
				Phase: Shoot,
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   Shoot,
				Args:     map[string]interface{}{"x": 2, "y": "y"},
			},
			Phase:     Shoot,
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
				Phase: Shoot,
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   Shoot,
				Args:     map[string]interface{}{"x": 2, "y": 12},
			},
			Phase:     Shoot,
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
					Board: board.InitBoard(),
				},
				Next: &player.Player{
					Id:    secondID,
					Conn:  secondConn,
					Board: getBoard(),
				},
				Phase: Shoot,
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   Shoot,
				Args:     map[string]interface{}{"x": 3, "y": 3},
			},
			Phase:     Shoot,
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
					Board: board.InitBoard(),
				},
				Next: &player.Player{
					Id:    secondID,
					Conn:  secondConn,
					Board: getBoardWithOneTakenField(3, 3),
				},
				Phase: Shoot,
				Done:  make(chan struct{}, 1),
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   Shoot,
				Args:     map[string]interface{}{"x": 3, "y": 3},
			},
			Phase:     Shoot,
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
					Board: board.InitBoard(),
				},
				Next: &player.Player{
					Id:    secondID,
					Conn:  secondConn,
				},
				Phase: Shoot,
				Done:  make(chan struct{}, 1),
			},
			Request: utils.Request{
				PlayerId: firstID,
				Action:   Exit,
				Args:     nil,
			},
			Phase:     Shoot,
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
			room.ResponseSender = respSender
			room.ProcessCommand(testCase.Request)

			// then
			assert.Equal(t, testCase.Phase, room.Phase)
			assert.Equal(t, testCase.CurrentID, room.Current.Id)
			assert.Equal(t, testCase.NextID, room.Next.Id)
			respSender.AssertExpectations(t)
		})
	}
}
