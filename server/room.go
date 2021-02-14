package main

import (
	"errors"
	"fmt"
	"github.com/StanislavStefanov/Battleships/pkg"
	"github.com/StanislavStefanov/Battleships/pkg/board"
	"github.com/StanislavStefanov/Battleships/pkg/web"
	"github.com/StanislavStefanov/Battleships/server/player"
	"strconv"
)

type Room struct {
	Current         *player.Player
	Next            *player.Player
	First           chan web.Request
	Second          chan web.Request
	FirstExit       chan struct{}
	SecondExit      chan struct{}
	Done            chan struct{}
	Phase           string
	ShipSizeToCount map[int]int
	NextShipSize    int
	Id              string
	ResponseSender  ResponseSender
}

const (
	destroyer  = 5
	battleship = 4
	ship       = 3
	boat       = 2
)

const (
	destroyerCount  = 1
	battleshipCount = 4
	shipCount       = 6
	boatCount       = 8
)

func CreateRoom(id string, player *player.Player, done chan struct{}) Room {
	r := Room{
		Current:         player,
		Next:            nil,
		First:           make(chan web.Request),
		Second:          make(chan web.Request),
		FirstExit:       make(chan struct{}),
		SecondExit:      make(chan struct{}),
		Done:            done,
		Phase:           "wait",
		ShipSizeToCount: map[int]int{destroyer: destroyerCount, battleship: battleshipCount, ship: shipCount, boat: boatCount},
		NextShipSize:    destroyer,
		Id:              id,
		ResponseSender:  &Sender{},
	}
	fmt.Println(r)
	return r
}

func (r *Room) Join(player *player.Player) error {
	if r.Next != nil {
		return errors.New("room is already full")
	}
	r.Next = player
	return nil
}

func (r *Room) GetRoomInfo() (string, int) {
	playersCount := 0
	if r.Current != nil {
		playersCount++
	}
	if r.Next != nil {
		playersCount++
	}
	return r.Id, playersCount
}

func (r *Room) ProcessCommand(request web.Request) {

	id := request.GetId()
	if id != r.Current.Id {
		var resp web.Response
		if request.GetAction() == pkg.Exit {
			resp = web.BuildResponse(pkg.Win, "Your opponent exited the game. Congratulations, you win!", nil)
			r.ResponseSender.SendResponse(resp, r.Current.Conn)
			r.Done <- struct{}{}
		} else {
			resp = web.BuildResponse(pkg.Wait, "Wait for enemy to make his turn.", nil)
			r.ResponseSender.SendResponse(resp, r.Next.Conn)
		}
		return
	}

	action := request.GetAction()
	if action != r.Phase && action != pkg.Exit {
		resp := web.BuildResponse(pkg.Retry,
			fmt.Sprintf("Invalid action during Phase: %s.", r.Phase),
			nil)
		r.ResponseSender.SendResponse(resp, r.Current.Conn)
		return
	}

	switch action {
	case pkg.PlaceShip:
		r.processShipPlacement(request)
	case pkg.Shoot:
		r.processShoot(request)
	case pkg.Exit:
		if r.Next != nil {
			resp := web.BuildResponse(pkg.Win, "Your opponent exited the game. Congratulations, you win!", nil)
			r.ResponseSender.SendResponse(resp, r.Next.Conn)
		}
		r.Done <- struct{}{}
	}
}

func (r *Room) processShipPlacement(request web.Request) {
	ship, err := r.placeShip(request)
	if err != nil {
		resp := web.BuildResponse(pkg.Retry, err.Error(), nil)
		r.ResponseSender.SendResponse(resp, r.Current.Conn)
		return
	}

	resp := web.BuildResponse(pkg.Placed,
		"Ship placed successfully. Wait for opponent to make his turn.",
		map[string]interface{}{"x": ship.GetX(), "y": ship.GetY(), "direction": ship.GetDirection(), "length": ship.GetLength()})
	r.ResponseSender.SendResponse(resp, r.Current.Conn)

	length, err := r.getNextShipSize()
	if err != nil {
		r.Phase = pkg.Shoot
		response := web.BuildResponse(pkg.Shoot, "Select filed to attack.", nil)
		r.ResponseSender.SendResponse(response, r.Next.Conn)
		r.switchPlayers()
		return
	}

	resp = web.BuildResponse(pkg.PlaceShip,
		fmt.Sprintf("Select where to place ship with length %d", length),
		nil)
	r.ResponseSender.SendResponse(resp, r.Next.Conn)

	r.switchPlayers()
}

func (r *Room) placeShip(req web.Request) (*board.Ship, error) {
	ship, err := getShip(req)
	if err != nil {
		return nil, err
	}

	ship.SetLength(r.NextShipSize)

	return ship, r.Current.PlaceShip(*ship)
}

func getShip(req web.Request) (*board.Ship, error) {
	args := req.GetArgs()

	x, err := extractIntFromArgs("x", args)
	if err != nil {
		return nil, err
	}

	y, err := extractIntFromArgs("y", args)
	if err != nil {
		return nil, err
	}

	direction, err := extractStringFromArgs("direction", args)
	if err != nil {
		return nil, err
	}

	ship := board.CreateShip(x, y, direction, 0)
	return &ship, err
}

func (r *Room) getNextShipSize() (int, error) {
	count := r.ShipSizeToCount[r.NextShipSize]
	if count > 0 {
		r.ShipSizeToCount[r.NextShipSize] = count - 1
		return r.NextShipSize, nil
	}

	r.NextShipSize--
	if r.NextShipSize < 2 {
		return 0, errors.New("all ships already placed")
	}
	count = r.ShipSizeToCount[r.NextShipSize]
	r.ShipSizeToCount[r.NextShipSize] = count - 1

	return r.NextShipSize, nil
}

func (r *Room) processShoot(request web.Request) {
	position, err := getPosition(request)
	if err != nil {
		resp := web.BuildResponse(pkg.Retry, err.Error(), nil)
		r.ResponseSender.SendResponse(resp, r.Current.Conn)
		return
	}

	success,sunk, err := r.shootAtField(*position)
	if err != nil {
		resp := web.BuildResponse(pkg.Retry, err.Error(), nil)
		r.ResponseSender.SendResponse(resp, r.Current.Conn)
		return
	}

	if success && r.Next.Board.IsBeaten() {
		resp := web.BuildResponse(pkg.Win, "Congratulations, you win!", nil)
		r.ResponseSender.SendResponse(resp, r.Current.Conn)

		resp = web.BuildResponse(pkg.Lose, "Defeat!", nil)
		r.ResponseSender.SendResponse(resp, r.Next.Conn)

		r.Done <- struct{}{}
		return
	}

	args := make(map[string]interface{})
	args["hit"] = success
	args["sunk"] = sunk
	args["x"] = position.X
	args["y"] = position.Y

	resp := web.BuildResponse(pkg.ShootOutcome, "", args)
	r.ResponseSender.SendResponse(resp, r.Current.Conn)

	resp = web.BuildResponse(pkg.Shoot, "Select filed to attack.", args)
	r.ResponseSender.SendResponse(resp, r.Next.Conn)

	r.switchPlayers()

}

func (r *Room) shootAtField(position board.Position) (bool, bool, error) {
	success, sunk, err := r.Next.Board.ReceiveAttack(position)
	if err != nil {
		return false, false, err
	}

	r.Current.Board.Attack(position, success)
	return success, sunk, nil
}

func (r *Room) switchPlayers() {
	p := r.Next
	r.Next = r.Current
	r.Current = p
}

func (r *Room) closeRoom() {
	_ = r.Current.Conn.Close()
	if r.Next != nil {
		_ = r.Next.Conn.Close()
	}
}

func getPosition(req web.Request) (*board.Position, error) {
	args := req.GetArgs()

	x, err := extractIntFromArgs("x", args)
	if err != nil {
		return nil, err
	}

	y, err := extractIntFromArgs("y", args)
	if err != nil {
		return nil, err
	}

	return &board.Position{
		X: x,
		Y: y,
	}, nil
}

func extractIntFromArgs(key string, args map[string]interface{}) (int, error) {
	v, ok := args[key]
	if !ok {
		return 0, errors.New(fmt.Sprintf("missing value for %s", key))
	}
	value, err := strconv.Atoi(v.(string))
	if err != nil {
		return 0, errors.New(fmt.Sprintf("invalid value for %s", key))
	}

	return value, nil
}

func extractStringFromArgs(key string, args map[string]interface{}) (string, error) {
	v, ok := args[key]
	if !ok {
		return "", errors.New(fmt.Sprintf("missing value for %s", key))
	}
	value, ok := v.(string)
	if !ok {
		return "", errors.New(fmt.Sprintf("invalid value for %s", key))
	}

	return value, nil
}
