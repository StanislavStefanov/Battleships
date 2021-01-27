package main

import "errors"

const (
	Up    = "up"
	Down  = "down"
	Left  = "left"
	Right = "right"
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

func (s Ship) getPositions() ([]Position, error) {
	start := Position{
		X: s.x,
		Y: s.y,
	}

	switch s.direction {
	case Up:
		return fillUp(start, s.length)
	case Down:
		return fillDown(start, s.length)
	case Left:
		return fillLeft(start, s.length)
	case Right:
		return fillRight(start, s.length)
	default:
		return nil, errors.New("unknown positioning direction")
	}
}

func fillUp(start Position, size int) ([]Position, error) {
	if start.Y-size < 0 {
		return nil, errors.New("ship goes out of bounds")
	}
	var positions []Position
	for i := 0; i < size; i++ {
		positions = append(positions, Position{
			X: start.X,
			Y: start.Y - i,
		})
	}
	return positions, nil
}

func fillDown(start Position, size int) ([]Position, error) {
	if start.Y+size >= 10 {
		return nil, errors.New("ship goes out of bounds")
	}
	var positions []Position
	for i := 0; i < size; i++ {
		positions = append(positions, Position{
			X: start.X,
			Y: start.Y + i,
		})
	}
	return positions, nil
}

func fillLeft(start Position, size int) ([]Position, error) {
	if start.X-size < 0 {
		return nil, errors.New("ship goes out of bounds")
	}
	var positions []Position
	for i := 0; i < size; i++ {
		positions = append(positions, Position{
			X: start.X - i,
			Y: start.Y,
		})
	}
	return positions, nil
}

func fillRight(start Position, size int) ([]Position, error) {
	if start.X+size >= 10 {
		return nil, errors.New("ship goes out of bounds")
	}
	var positions []Position
	for i := 0; i < size; i++ {
		positions = append(positions, Position{
			X: start.X + i,
			Y: start.Y,
		})
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
			fields[i][j] = '_'
		}
	}
	return fields
}

func initBoard() *Board {
	return &Board{
		ownFields:   initFields(),
		enemyFields: initFields(),
	}
}

func (b *Board) placeShip(ship Ship) error {
	positions, err := ship.getPositions()
	if err != nil {
		return err
	}

	for _, position := range positions {
		b.ownFields[position.X][position.Y] = 's'
	}
	return nil
}

func (b *Board) attack(p Position, success bool) error {
	if p.X < 0 || p.X >= 10 || p.Y < 0 || p.Y >= 10 {
		return errors.New("position out of bounds")
	}

	if success == true {
		b.enemyFields[p.X][p.Y] = 'x'
	} else {
		b.enemyFields[p.X][p.Y] = 'o'
	}
	return nil
}

func (b *Board) receiveAttack(p Position) (bool, error) {
	if p.X < 0 || p.X >= 10 || p.Y < 0 || p.Y >= 10 {
		return false, errors.New("position out of bounds")
	}

	if b.ownFields[p.X][p.Y] == 's' {
		b.ownFields[p.X][p.Y] = 'x'
		return true, nil
	} else {
		b.ownFields[p.X][p.Y] = 'o'
		return false, nil
	}
}

func (b *Board) isBeaten() bool {
	for _, r := range b.ownFields {
		for _, v := range r {
			if v == 's' {
				return false
			}
		}
	}

	return true
}
