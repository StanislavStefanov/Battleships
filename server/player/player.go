package player

import (
	"github.com/StanislavStefanov/Battleships/pkg/game"
)

//go:generate mockery -name=Connection -output=automock -outpkg=automock -case=underscore
type Connection interface {
	WriteMessage(int, []byte) error
	Close() error
	ReadMessage() (int, []byte, error)
}

type Player struct {
	Conn  Connection
	Board *game.Board
	Id    string
}

func (p *Player) PlaceShip(ship game.Ship) error {
	return p.Board.PlaceShip(ship)
}
