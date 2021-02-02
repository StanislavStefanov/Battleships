package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/StanislavStefanov/Battleships/server/board"
	"github.com/StanislavStefanov/Battleships/server/player"
	"github.com/StanislavStefanov/Battleships/utils"
	"github.com/gorilla/websocket"
	"log"
	"sync"
)

type Room struct {
	Current         *player.Player
	Next            *player.Player
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
	destroyerCount  = 2
	battleshipCount = 4
	shipCount       = 6
	boatCount       = 8
)

//TODO new fields
func CreateRoom(id string, player *player.Player, done chan struct{}) Room {
	return Room{
		Current:         player,
		Next:            nil,
		Done:            done,
		Phase:           "wait",
		ShipSizeToCount: map[int]int{destroyer: destroyerCount, battleship: battleshipCount, ship: shipCount, boat: boatCount},
		NextShipSize:    destroyer,
		Id:              id,
	}
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

const (
	Exit      = "exit"
	Shoot     = "shoot"
	PlaceShip = "place"
	Wait      = "wait"
	Retry     = "retry"
	Win       = "win"
	Lose      = "lose"
)

func (r *Room) ProcessCommand(request utils.Request) {
	id := request.GetId()
	if id != r.Current.Id {
		resp := utils.BuildResponse(Wait, "Wait for enemy to make his turn.", nil)
		r.ResponseSender.SendResponse(resp, r.Next.Conn)
		return
	}

	action := request.GetAction()
	if action != r.Phase && action != Exit {
		resp := utils.BuildResponse(Retry,
			fmt.Sprintf("Invalid action during Phase: %s.", r.Phase),
			nil)
		r.ResponseSender.SendResponse(resp, r.Current.Conn)
		return
	}

	switch action {
	case PlaceShip:
		r.processShipPlacement(request)
	case Shoot:
		r.processShoot(request)
	case Exit:
		resp := utils.BuildResponse(Win, "Your opponent exited the game. Congratulations, you win!", nil)
		r.ResponseSender.SendResponse(resp, r.Next.Conn)
		r.Done <- struct{}{}
	}
}

func (r *Room) processShipPlacement(request utils.Request) {
	err := r.placeShip(request)
	if err != nil {
		resp := utils.BuildResponse(Retry, err.Error(), nil)
		r.ResponseSender.SendResponse(resp, r.Current.Conn)
		return
	}

	resp := utils.BuildResponse(Wait,
		"Ship placed successfully. Wait for opponent to make his turn.",
		nil)
	r.ResponseSender.SendResponse(resp, r.Current.Conn)

	length, err := r.getNextShipSize()
	if err != nil {
		r.Phase = Shoot
		response := utils.BuildResponse(Shoot, "Select filed to attack.", nil)
		r.ResponseSender.SendResponse(response, r.Next.Conn)
		r.switchPlayers()
		return
	}

	resp = utils.BuildResponse(PlaceShip,
		fmt.Sprintf("Select where to place ship with length %d", length),
		nil)
	r.ResponseSender.SendResponse(resp, r.Next.Conn)

	r.switchPlayers()
}

func (r *Room) placeShip(req utils.Request) error {
	ship, err := getShip(req)
	if err != nil {
		return err
	}

	ship.SetLength(r.NextShipSize)

	return r.Current.PlaceShip(*ship)
}

func getShip(req utils.Request) (*board.Ship, error) {
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

	length, err := extractIntFromArgs("length", args)
	if err != nil {
		return nil, err
	}
	ship := board.CreateShip(x, y, direction, length)
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

func (r *Room) processShoot(request utils.Request) {
	position, err := getPosition(request)
	if err != nil {
		resp := utils.BuildResponse(Retry, err.Error(), nil)
		r.ResponseSender.SendResponse(resp, r.Current.Conn)
		return
	}

	success, err := r.shootAtField(*position)
	if err != nil {
		resp := utils.BuildResponse(Retry, err.Error(), nil)
		r.ResponseSender.SendResponse(resp, r.Current.Conn)
		return
	}

	if success && r.Next.Board.IsBeaten() {
		resp := utils.BuildResponse(Win, "Congratulations, you win!", nil)
		r.ResponseSender.SendResponse(resp, r.Current.Conn)

		resp = utils.BuildResponse(Lose, "Defeat!", nil)
		r.ResponseSender.SendResponse(resp, r.Next.Conn)

		r.Done <- struct{}{}
		return
	}

	args := make(map[string]interface{})
	args["hit"] = success
	args["x"] = position.X
	args["y"] = position.Y

	resp := utils.BuildResponse(Wait, "", args)
	r.ResponseSender.SendResponse(resp, r.Current.Conn)

	resp = utils.BuildResponse(Shoot, "Select filed to attack.", args)
	r.ResponseSender.SendResponse(resp, r.Next.Conn)

	r.switchPlayers()

}

func (r *Room) shootAtField(position board.Position) (bool, error) {
	success, err := r.Next.Board.ReceiveAttack(position)
	if err != nil {
		return false, err
	}

	r.Current.Board.Attack(position, success)
	return true, nil
}

func (r *Room) switchPlayers() {
	p := r.Next
	r.Next = r.Current
	r.Current = p
}

func (r *Room) closeRoom() {
	_ = r.Current.Conn.Close()
	_ = r.Next.Conn.Close()
}

func getPosition(req utils.Request) (*board.Position, error) {
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
	value, ok := v.(int)
	if !ok {
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

func (s *Server) runRoom(r *Room, done <-chan struct{}) {
	var wg *sync.WaitGroup

	first := make(chan utils.Request)
	wg.Add(1)
	go playerReadLoop(r.Current.Conn, first, wg)

	second := make(chan utils.Request)
	wg.Add(1)
	go playerReadLoop(r.Next.Conn, second, wg)

	for {
		select {
		case request := <-first:
			r.ProcessCommand(request)
		case request := <-second:
			r.ProcessCommand(request)
		case <-done:
			r.closeRoom()
			s.deleteRoom(r.Id)
			wg.Wait()
			return
		}
	}
}

func playerReadLoop(conn *websocket.Conn, play chan utils.Request, wg *sync.WaitGroup) {
	//	cancel, cancelFunc := context.WithCancel(context.Background())
	fmt.Println("start Current read loop")
	defer wg.Done()
	for {
		_, bytes, err := conn.ReadMessage()
		if err != nil {
			log.Fatal(err)
			return
		}

		var req utils.Request
		json.Unmarshal(bytes, &req)
		play <- req
	}
}
