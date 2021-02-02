package player

import (
	"github.com/StanislavStefanov/Battleships/server/board"
	"github.com/gorilla/websocket"
)

type Player struct {
	Conn *websocket.Conn
	Board *board.Board
	Id   string
}

func (p *Player) PlaceShip(ship board.Ship) error{
	return p.Board.PlaceShip(ship)
}


