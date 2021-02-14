package player

import (
	"github.com/StanislavStefanov/Battleships/pkg/board"
)

//go:generate mockery -name=Connection -output=automock -outpkg=automock -case=underscore
type Connection interface {
	WriteMessage(int, []byte) error
	Close() error
	ReadMessage() (int, []byte, error)
}

type Player struct {
	Conn  Connection
	Board *board.Board
	Id    string
}

func (p *Player) PlaceShip(ship board.Ship) error {
	return p.Board.PlaceShip(ship)
}
