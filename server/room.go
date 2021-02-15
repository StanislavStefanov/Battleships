package main

import (
	"errors"
	"fmt"
	"github.com/StanislavStefanov/Battleships/pkg"
	"github.com/StanislavStefanov/Battleships/pkg/game"
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
	Sender          ResponseSender
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

//CreateRoom creates and returns new room with the provided player as First to play. The second player
// is nil until it is set through the Join method.
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
		Sender:          &Sender{},
	}
	fmt.Println(r)
	return r
}

//Join adds second player to the room if there is free place. If the room is already full
//an error is returned.
func (r *Room) Join(player *player.Player) error {
	if r.Next != nil {
		return errors.New("room is already full")
	}
	r.Next = player
	return nil
}

//GetRoomInfo returns the room id and the count of players currently in the room.
//The valid counts of players and their meanings are: 1 - the second player hasn't
//joined yet and the game hasn't started, 2 - all players are on their seats and
//the game has already began. Any other player counts are considered invalid.
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

//ProcessCommand checks some preconditions before processing the request. If the request
//is not from the player whose turn it is it will be rejected and Response with status Wait
//will be sent back. Exception is if the Request action is Exit. Then the player will leave
//the room and its opponent will be notified. If the request is made from the player on turn
//but the request action doesn't match the room's Phase the request will be rejected and
//Response with status Retry will be sent back. Exception is if the Request action is Exit.
//If the preconditions are met then the request is processed according to it's action. The
//allowed actions are: place, shoot, exit. If the request action is Exit message is passed
//through the room's done channel as notification about the event.
func (r *Room) ProcessCommand(request web.Request) {

	id := request.GetId()
	if id != r.Current.Id {
		var resp web.Response
		if request.GetAction() == pkg.Exit {
			resp = web.BuildResponse(pkg.Win, "Your opponent exited the game. Congratulations, you win!", nil)
			r.Sender.SendResponse(resp, r.Current.Conn)
			r.Done <- struct{}{}
		} else {
			resp = web.BuildResponse(pkg.Wait, "Wait for enemy to make his turn.", nil)
			r.Sender.SendResponse(resp, r.Next.Conn)
		}
		return
	}

	action := request.GetAction()
	if action != r.Phase && action != pkg.Exit {
		resp := web.BuildResponse(pkg.Retry,
			fmt.Sprintf("Invalid action during Phase: %s.", r.Phase),
			nil)
		r.Sender.SendResponse(resp, r.Current.Conn)
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
			r.Sender.SendResponse(resp, r.Next.Conn)
		}
		r.Done <- struct{}{}
	}
}

//processShipPlacement processes requests with action "place". Response with status "placed"
//and args containing info about the placed ship(keys: x, y, direction, length) is returned
//to the player who sent the request and response with action "place" is sent to the next player.
//If the next player has already placed all of his ships his response has action "shoot".
//If the method fails to retrieve the ship from the request response wit status "retry" is sent
//to the player who sent the request.
func (r *Room) processShipPlacement(request web.Request) {
	ship, err := r.placeShip(request)
	if err != nil {
		resp := web.BuildResponse(pkg.Retry, err.Error(), nil)
		r.Sender.SendResponse(resp, r.Current.Conn)
		return
	}

	resp := web.BuildResponse(pkg.Placed,
		"Ship placed successfully. Wait for opponent to make his turn.",
		map[string]interface{}{"x": ship.GetX(), "y": ship.GetY(), "direction": ship.GetDirection(), "length": ship.GetLength()})
	r.Sender.SendResponse(resp, r.Current.Conn)

	length, err := r.getNextShipSize()
	if err != nil {
		r.Phase = pkg.Shoot
		response := web.BuildResponse(pkg.Shoot, "Select filed to attack.", nil)
		r.Sender.SendResponse(response, r.Next.Conn)
		r.switchPlayers()
		return
	}

	resp = web.BuildResponse(pkg.PlaceShip,
		fmt.Sprintf("Select where to place ship with length %d", length),
		nil)
	r.Sender.SendResponse(resp, r.Next.Conn)

	r.switchPlayers()
}

func (r *Room) placeShip(req web.Request) (*game.Ship, error) {
	ship, err := getShip(req)
	if err != nil {
		return nil, err
	}

	ship.SetLength(r.NextShipSize)

	return ship, r.Current.PlaceShip(*ship)
}

func getShip(req web.Request) (*game.Ship, error) {
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

	ship := game.CreateShip(x, y, direction, 0)
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

//processShoot processes requests with action "shoot". Response with status "shoot outcome"
//is returned to the player who sent the request and response with action "shoot" is sent to
//the next player. Both responses have args containing info about the shot ship(keys: hit -
//true if enemy ship is hit and false if not, sunk - true if all fields of the hit enemy ship
//are destroyed and false if not, x, y - coordinates that were targeted by the shoot).
//If the method fails to retrieve the coordinates from the request or an error occurs while
//shooting response with status "retry" is sent to the player who sent the request. If all
//of the enemy fields are already hit requestwith status "win" is returned to the player who
//sent the request and response with action "shoot" is sent to the next player.
func (r *Room) processShoot(request web.Request) {
	position, err := getPosition(request)
	if err != nil {
		resp := web.BuildResponse(pkg.Retry, err.Error(), nil)
		r.Sender.SendResponse(resp, r.Current.Conn)
		return
	}

	success, sunk, err := r.shootAtField(*position)
	if err != nil {
		resp := web.BuildResponse(pkg.Retry, err.Error(), nil)
		r.Sender.SendResponse(resp, r.Current.Conn)
		return
	}

	if success && r.Next.Board.IsBeaten() {
		resp := web.BuildResponse(pkg.Win, "Congratulations, you win!", nil)
		r.Sender.SendResponse(resp, r.Current.Conn)

		resp = web.BuildResponse(pkg.Lose, "Defeat!", nil)
		r.Sender.SendResponse(resp, r.Next.Conn)

		r.Done <- struct{}{}
		return
	}

	args := make(map[string]interface{})
	args["hit"] = success
	args["sunk"] = sunk
	args["x"] = position.X
	args["y"] = position.Y

	resp := web.BuildResponse(pkg.ShootOutcome, "", args)
	r.Sender.SendResponse(resp, r.Current.Conn)

	resp = web.BuildResponse(pkg.Shoot, "Select filed to attack.", args)
	r.Sender.SendResponse(resp, r.Next.Conn)

	r.switchPlayers()

}

func (r *Room) shootAtField(position game.Position) (bool, bool, error) {
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

func getPosition(req web.Request) (*game.Position, error) {
	args := req.GetArgs()

	x, err := extractIntFromArgs("x", args)
	if err != nil {
		return nil, err
	}

	y, err := extractIntFromArgs("y", args)
	if err != nil {
		return nil, err
	}

	return &game.Position{
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
