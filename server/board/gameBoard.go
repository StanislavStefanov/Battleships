package board

import (
	"errors"
)

const (
	Up    = "up"
	Down  = "down"
	Left  = "left"
	Right = "right"
)

const (
	Hit   = 'x'
	Miss  = 'o'
	Taken = 's'
	Empty = '-'
)

type Position struct {
	X int
	Y int
}

type Ship struct {
	x         int
	y         int
	direction string
	length    int
}

func CreateShip(x, y int, direction string, length int) Ship {
	return Ship{
		x:         x,
		y:         y,
		direction: direction,
		length:    length,
	}
}

func (s Ship) SetLength(length int){
	s.length = length
}

func (s Ship) GetPositions() ([]Position, error) {
	start := Position{
		X: s.x,
		Y: s.y,
	}

	switch s.direction {
	case Up:
		return fill(start,
			s.length,
			func(p Position, offset int) bool { return p.X-offset < 0 },
			func(p Position, i int) Position {
				return Position{
					X: p.X - i,
					Y: p.Y,
				}
			})
	case Down:
		return fill(start,
			s.length,
			func(p Position, offset int) bool { return p.X+offset >= 10 },
			func(p Position, i int) Position {
				return Position{
					X: p.X + i,
					Y: p.Y,
				}
			})
	case Left:
		return fill(start,
			s.length, func(p Position, offset int) bool { return p.Y-offset < 0 },
			func(p Position, i int) Position {
				return Position{
					X: p.X,
					Y: p.Y - i,
				}
			})
	case Right:
		return fill(start,
			s.length,
			func(p Position, offset int) bool { return p.Y+offset >= 10 },
			func(p Position, i int) Position {
				return Position{
					X: p.X,
					Y: p.Y + i,
				}
			})
	default:
		return nil, errors.New("unknown positioning direction")
	}
}

func fill(start Position, size int, outOfBounds func(Position, int) bool, next func(Position, int) Position) ([]Position, error) {
	if outOfBounds(start, size) {
		return nil, errors.New("ship goes out of bounds")
	}
	var positions []Position
	for i := 0; i < size; i++ {
		positions = append(positions, next(start, i))
	}
	return positions, nil
}

type Board struct {
	ownFields   [][]rune
	enemyFields [][]rune
}

func initFields() [][]rune {
	fields := make([][]rune, 10)
	for i := 0; i < 10; i++ {
		fields[i] = make([]rune, 10)
		for j := 0; j < 10; j++ {
			fields[i][j] = Empty
		}
	}
	return fields
}

func InitBoard() *Board {
	return &Board{
		ownFields:   initFields(),
		enemyFields: initFields(),
	}
}

func (b *Board) PlaceShip(ship Ship) error {
	positions, err := ship.GetPositions()
	if err != nil {
		return err
	}

	for _, position := range positions {
		if b.ownFields[position.X][position.Y] == Taken {
			return errors.New("some of the fields are already taken")
		}
	}

	for _, position := range positions {
		b.ownFields[position.X][position.Y] = Taken
	}
	return nil
}

func (b *Board) Attack(p Position, success bool) error {
	if p.X < 0 || p.X >= 10 || p.Y < 0 || p.Y >= 10 {
		return errors.New("position out of bounds")
	}

	if success == true {
		b.enemyFields[p.X][p.Y] = Hit
	} else {
		b.enemyFields[p.X][p.Y] = Miss
	}
	return nil
}

func (b *Board) ReceiveAttack(p Position) (bool, error) {
	if p.X < 0 || p.X >= 10 || p.Y < 0 || p.Y >= 10 {
		return false, errors.New("position out of bounds")
	}

	if b.ownFields[p.X][p.Y] == Taken {
		b.ownFields[p.X][p.Y] = Hit
		return true, nil
	} else {
		b.ownFields[p.X][p.Y] = Miss
		return false, nil
	}
}

func (b *Board) IsBeaten() bool {
	for _, r := range b.ownFields {
		for _, v := range r {
			if v == Taken {
				return false
			}
		}
	}

	return true
}
