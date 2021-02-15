package game

import (
	"errors"
	"fmt"
)

const (
	Up    = "up"
	Down  = "down"
	Left  = "left"
	Right = "right"
)

const (
	Hit      = 'x'
	Miss     = 'o'
	Taken    = 's'
	ShipArea = 'b'
	Empty    = '-'
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

func (s *Ship) SetLength(length int) {
	s.length = length
}

func (s *Ship) GetX() int {
	return s.x
}

func (s *Ship) GetY() int {
	return s.y
}

func (s *Ship) GetDirection() string {
	return s.direction
}

func (s *Ship) GetLength() int {
	return s.length
}

//GetPositions returns the Positions on which the ship will be placed. The slice of positions
//is generated based on the ship length, starting point(x,y) and direction. A ship can be
//placed horizontally or vertically. All valid directions are up, down, left and right. If
//invalid direction is provided an error will be returned.
func (s *Ship) GetPositions() ([]Position, error) {
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

//InitBoard returns board with all own and enemy fields set to empty(_).
func InitBoard() *Board {
	return &Board{
		ownFields:   initFields(),
		enemyFields: initFields(),
	}
}

func printBoard(b [][]rune) {
	fmt.Print("  ")
	for i := 0; i < 10; i++ {
		fmt.Print(i, " ")
	}
	fmt.Println("")
	for i, r := range b {
		fmt.Print(i, " ")
		for _, c := range r {
			fmt.Print(string(c), " ")
		}
		fmt.Println("")
	}
}

func (b *Board) Print() {
	fmt.Println("Enemy fields:")
	printBoard(b.enemyFields)

	fmt.Println("Own fields:")
	printBoard(b.ownFields)
}

//PlaceShip marks the fields of the ship on own fields as taken(s) and the neighboring fields
//as ship area(b). If any of the ship's fields is out of bounds or is not empty(_) an error
//is returned and the ship is not placed on the board.
func (b *Board) PlaceShip(ship Ship) error {
	positions, err := ship.GetPositions()
	if err != nil {
		return err
	}

	for _, position := range positions {
		if b.ownFields[position.X][position.Y] != Empty {
			return errors.New("some of the fields are already taken")
		}
	}

	for _, position := range positions {
		b.ownFields[position.X][position.Y] = Taken
		for _, p := range getNeighbours(position) {
			if !isOutOfBounds(p) && b.ownFields[p.X][p.Y] != Taken {
				b.ownFields[p.X][p.Y] = ShipArea
			}
		}
	}
	return nil
}

//Attack marks the targeted enemy field as hit(x) if success is true or marks the targeted
//field enemy field as miss(o) if success is false. Returns error if p is out of bounds.
func (b *Board) Attack(p Position, success bool) error {
	if isOutOfBounds(p) {
		return errors.New("position out of bounds")
	}

	if success == true {
		b.enemyFields[p.X][p.Y] = Hit
	} else {
		b.enemyFields[p.X][p.Y] = Miss
	}
	return nil
}

//ReceiveAttack returns hit status, sunk status and error. If the targeted field on own fields
//is taken(s) the hit status is true, otherwise false. If the targeted field is taken(s) and
//every other field which is part of the ship that is hit has already been hit the sunk status
//is true, otherwise false. If the targeted field is out of bounds the method returns false,
//false and non nil error.
func (b *Board) ReceiveAttack(p Position) (bool, bool, error) {
	if isOutOfBounds(p) {
		return false, false, errors.New("position out of bounds")
	}

	if b.ownFields[p.X][p.Y] == Taken {
		b.ownFields[p.X][p.Y] = Hit
		return true, b.ShipIsSunk(p), nil
	} else {
		b.ownFields[p.X][p.Y] = Miss
		return false, false, nil
	}
}

//ShipIsSunk returns true if all fields taken by the hit ship are already hit, false otherwise.
func (b *Board) ShipIsSunk(p Position) bool {
	pos := Position{
		X: p.X - 1,
		Y: p.Y,
	}
	for !isOutOfBounds(pos) {
		if b.ownFields[pos.X][pos.Y] == Taken {
			return false
		} else if b.ownFields[pos.X][pos.Y] == ShipArea {
			break
		}
		pos.X--
	}

	pos = Position{
		X: p.X + 1,
		Y: p.Y,
	}
	for !isOutOfBounds(pos) {
		if b.ownFields[pos.X][pos.Y] == Taken {
			return false
		} else if b.ownFields[pos.X][pos.Y] == ShipArea {
			break
		}
		pos.X++
	}

	pos = Position{
		X: p.X,
		Y: p.Y - 1,
	}
	for !isOutOfBounds(pos) {
		if b.ownFields[pos.X][pos.Y] == Taken {
			return false
		} else if b.ownFields[pos.X][pos.Y] == ShipArea {
			break
		}
		pos.Y--
	}

	pos = Position{
		X: p.X,
		Y: p.Y + 1,
	}
	for !isOutOfBounds(pos) {
		if b.ownFields[pos.X][pos.Y] == Taken {
			return false
		} else if b.ownFields[pos.X][pos.Y] == ShipArea {
			break
		}
		pos.Y++
	}

	return true
}

//IsBeaten returns true if all ships are sunk, false otherwise.
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

func isOutOfBounds(p Position) bool {
	return p.X < 0 || p.X >= 10 || p.Y < 0 || p.Y >= 10
}

func getNeighbours(p Position) []Position {
	return []Position{
		{
			X: p.X - 1,
			Y: p.Y,
		},
		{
			X: p.X + 1,
			Y: p.Y,
		},
		{
			X: p.X,
			Y: p.Y - 1,
		},
		{
			X: p.X,
			Y: p.Y + 1,
		},
	}
}
