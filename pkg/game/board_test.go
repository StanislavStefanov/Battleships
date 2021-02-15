package game

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestShip_GetPositions(t *testing.T) {
	// given
	errorMsg := "ship goes out of bounds"

	testCases := []struct {
		Name               string
		Ship               Ship
		ExpectedPositions  []Position
		ExpectedErrMessage string
	}{
		{
			Name: "Success for direction up",
			Ship: Ship{
				x:         6,
				y:         6,
				direction: "up",
				length:    4,
			},
			ExpectedPositions:  []Position{{6, 6}, {5, 6}, {4, 6}, {3, 6}},
			ExpectedErrMessage: "",
		},
		{
			Name: "Success for direction down",
			Ship: Ship{
				x:         3,
				y:         6,
				direction: "down",
				length:    4,
			},
			ExpectedPositions: []Position{{3, 6}, {4, 6}, {5, 6}, {6, 6}},

			ExpectedErrMessage: "",
		},
		{
			Name: "Success for direction left",
			Ship: Ship{
				x:         3,
				y:         6,
				direction: "left",
				length:    4,
			},
			ExpectedPositions:  []Position{{3, 6}, {3, 5}, {3, 4}, {3, 3}},
			ExpectedErrMessage: "",
		},
		{
			Name: "Success for direction right",
			Ship: Ship{
				x:         3,
				y:         3,
				direction: "right",
				length:    4,
			},
			ExpectedPositions:  []Position{{3, 3}, {3, 4}, {3, 5}, {3, 6}},
			ExpectedErrMessage: "",
		},
		{
			Name: "Fail out of bounds for direction up",
			Ship: Ship{
				x:         3,
				y:         2,
				direction: "up",
				length:    4,
			},
			ExpectedPositions:  nil,
			ExpectedErrMessage: errorMsg,
		},
		{
			Name: "Fail out of bounds for direction down",
			Ship: Ship{
				x:         8,
				y:         8,
				direction: "down",
				length:    4,
			},
			ExpectedPositions:  nil,
			ExpectedErrMessage: errorMsg,
		},
		{
			Name: "Fail out of bounds for direction left",
			Ship: Ship{
				x:         3,
				y:         3,
				direction: "left",
				length:    4,
			},
			ExpectedPositions:  nil,
			ExpectedErrMessage: errorMsg,
		},
		{
			Name: "Fail out of bounds for direction right",
			Ship: Ship{
				x:         8,
				y:         8,
				direction: "right",
				length:    4,
			},
			ExpectedPositions:  nil,
			ExpectedErrMessage: errorMsg,
		},
		{
			Name: "Fail unknown direction",
			Ship: Ship{
				x:         8,
				y:         8,
				direction: "up-left-diagonal",
				length:    4,
			},
			ExpectedPositions:  nil,
			ExpectedErrMessage: "unknown positioning direction",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// when
			positions, err := testCase.Ship.GetPositions()

			// then
			if testCase.ExpectedErrMessage == "" {
				require.NoError(t, err)
				assert.Equal(t, testCase.ExpectedPositions, positions)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.ExpectedErrMessage)
			}
		})
	}
}

func TestBoard_PlaceShip(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// when
		ship := Ship{
			x:         6,
			y:         3,
			direction: "up",
			length:    4,
		}

		board := InitBoard()
		ExpectedBoard := [][]rune{{Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty},
			{Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty},
			{Empty, Empty, Empty, ShipArea, Empty, Empty, Empty, Empty, Empty, Empty},
			{Empty, Empty, ShipArea, Taken, ShipArea, Empty, Empty, Empty, Empty, Empty},
			{Empty, Empty, ShipArea, Taken, ShipArea, Empty, Empty, Empty, Empty, Empty},
			{Empty, Empty, ShipArea, Taken, ShipArea, Empty, Empty, Empty, Empty, Empty},
			{Empty, Empty, ShipArea, Taken, ShipArea, Empty, Empty, Empty, Empty, Empty},
			{Empty, Empty, Empty, ShipArea, Empty, Empty, Empty, Empty, Empty, Empty},
			{Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty},
			{Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty, Empty}}

		// then
		err := board.PlaceShip(ship)
		assert.NoError(t, err)
		assert.Equal(t, ExpectedBoard, board.ownFields)
	})

	t.Run("failure ship goes out of bounds", func(t *testing.T) {
		// when
		ship := Ship{
			x:         3,
			y:         6,
			direction: "up",
			length:    4,
		}

		board := InitBoard()
		expectedBoard := InitBoard().ownFields

		// then
		err := board.PlaceShip(ship)
		assert.Equal(t, "ship goes out of bounds", err.Error())
		assert.Equal(t, expectedBoard, board.ownFields)
	})

	t.Run("failure ships overlap", func(t *testing.T) {
		// when
		firstShip := Ship{
			x:         6,
			y:         6,
			direction: "up",
			length:    4,
		}

		secondShip := Ship{
			x:         8,
			y:         6,
			direction: "up",
			length:    4,
		}

		board := InitBoard()
		err := board.PlaceShip(firstShip)
		assert.NoError(t, err)

		expectedBoard := board.ownFields

		// then
		err = board.PlaceShip(secondShip)
		assert.Equal(t, "some of the fields are already taken", err.Error())
		assert.Equal(t, expectedBoard, board.ownFields)
	})
}

func TestBoard_Attack(t *testing.T) {
	t.Run("attack hit", func(t *testing.T) {
		// when
		board := InitBoard()

		// then
		err := board.Attack(Position{
			X: 5,
			Y: 6,
		}, true)
		assert.NoError(t, err)
		assert.Equal(t, Hit, board.enemyFields[5][6])
	})

	t.Run("attack miss", func(t *testing.T) {
		// when
		board := InitBoard()

		// then
		err := board.Attack(Position{
			X: 5,
			Y: 6,
		}, false)
		assert.NoError(t, err)
		assert.Equal(t, Miss, board.enemyFields[5][6])
	})

	t.Run("fail index out of bounds", func(t *testing.T) {
		// when
		board := InitBoard()
		// then
		err := board.Attack(Position{
			X: -2,
			Y: 12,
		}, false)
		assert.Equal(t, "position out of bounds", err.Error())
	})
}

func TestBoard_ReceiveAttack(t *testing.T) {
	t.Run("attack hit", func(t *testing.T) {
		// when
		ship := Ship{
			x:         6,
			y:         3,
			direction: "up",
			length:    2,
		}

		board := InitBoard()
		err := board.PlaceShip(ship)
		assert.NoError(t, err)

		// then
		hit, sunk, err := board.ReceiveAttack(Position{
			X: 5,
			Y: 3,
		})
		assert.NoError(t, err)
		assert.True(t, hit)
		assert.False(t, sunk)
		assert.Equal(t, Hit, board.ownFields[5][3])
		assert.Equal(t, Taken, board.ownFields[6][3])
	})

	t.Run("attack miss", func(t *testing.T) {
		// when
		ship := Ship{
			x:         6,
			y:         3,
			direction: "up",
			length:    2,
		}

		board := InitBoard()
		err := board.PlaceShip(ship)
		assert.NoError(t, err)

		// then
		hit, sunk, err := board.ReceiveAttack(Position{
			X: 5,
			Y: 5,
		})
		assert.NoError(t, err)
		assert.False(t, hit)
		assert.False(t, sunk)
		assert.Equal(t, Taken, board.ownFields[5][3])
		assert.Equal(t, Taken, board.ownFields[6][3])
		assert.Equal(t, Miss, board.ownFields[5][5])
	})

	t.Run("fail index out of bounds", func(t *testing.T) {
		// when
		board := InitBoard()
		// then
		_, _, err := board.ReceiveAttack(Position{
			X: -2,
			Y: 12,
		})
		assert.Equal(t, "position out of bounds", err.Error())
	})
}

func TestBoard_IsBeaten(t *testing.T) {
	t.Run("beaten", func(t *testing.T) {
		// when
		ship := Ship{
			x:         6,
			y:         3,
			direction: "up",
			length:    2,
		}

		board := InitBoard()
		err := board.PlaceShip(ship)
		assert.NoError(t, err)

		// then
		board.ReceiveAttack(Position{
			X: 5,
			Y: 3,
		})
		board.ReceiveAttack(Position{
			X: 6,
			Y: 3,
		})

		beaten := board.IsBeaten()
		assert.True(t, beaten)
	})

	t.Run("alive", func(t *testing.T) {
		// when
		ship := Ship{
			x:         6,
			y:         3,
			direction: "up",
			length:    2,
		}

		board := InitBoard()
		err := board.PlaceShip(ship)
		assert.NoError(t, err)

		// then
		beaten := board.IsBeaten()
		assert.False(t, beaten)

		board.ReceiveAttack(Position{
			X: 5,
			Y: 3,
		})
		beaten = board.IsBeaten()
		assert.False(t, beaten)
	})
}

func TestBoard_IsSunk(t *testing.T) {
	t.Run("untouched part of ship under attacking position", func(t *testing.T) {
		// when
		ship := Ship{
			x:         6,
			y:         3,
			direction: "up",
			length:    3,
		}

		board := InitBoard()
		err := board.PlaceShip(ship)
		assert.NoError(t, err)

		p := Position{
			X: 4,
			Y: 3,
		}

		// then
		board.ownFields[4][3] = 'x'
		board.ownFields[5][3] = 'x'

		beaten := board.ShipIsSunk(p)
		assert.False(t, beaten)
	})
	t.Run("untouched part of ship above attacking position", func(t *testing.T) {
		// when
		ship := Ship{
			x:         6,
			y:         3,
			direction: "up",
			length:    3,
		}

		board := InitBoard()
		err := board.PlaceShip(ship)
		assert.NoError(t, err)

		p := Position{
			X: 6,
			Y: 3,
		}

		// then
		board.ownFields[6][3] = 'x'
		board.ownFields[5][3] = 'x'

		beaten := board.ShipIsSunk(p)
		assert.False(t, beaten)
	})
	t.Run("untouched part of ship left from attacking position", func(t *testing.T) {
		// when
		ship := Ship{
			x:         6,
			y:         3,
			direction: "left",
			length:    3,
		}

		board := InitBoard()
		err := board.PlaceShip(ship)
		assert.NoError(t, err)

		p := Position{
			X: 6,
			Y: 3,
		}

		// then
		board.ownFields[6][3] = 'x'
		board.ownFields[6][2] = 'x'

		beaten := board.ShipIsSunk(p)
		assert.False(t, beaten)
	})
	t.Run("untouched part of ship right from attacking position", func(t *testing.T) {
		// when
		ship := Ship{
			x:         6,
			y:         3,
			direction: "left",
			length:    3,
		}

		board := InitBoard()
		err := board.PlaceShip(ship)
		assert.NoError(t, err)

		p := Position{
			X: 6,
			Y: 1,
		}

		// then
		board.ownFields[6][1] = 'x'
		board.ownFields[6][2] = 'x'

		beaten := board.ShipIsSunk(p)
		assert.False(t, beaten)
	})
}
