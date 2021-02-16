package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/StanislavStefanov/Battleships/pkg/game"
	"github.com/StanislavStefanov/Battleships/pkg/web"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

const (
	Register     = "register"
	Exit         = "exit"
	Shoot        = "shoot"
	ShootOutcome = "shoot-outcome"
	PlaceShip    = "place"
	Placed       = "placed"
	Wait         = "wait"
	Retry        = "retry"
	Win          = "win"
	Lose         = "lose"
	Info         = "info"
	List         = "ls-rooms"
	Create       = "create-room"
	Created      = "created"
	Join         = "join-room"
	JoinRandom   = "join-random"
)

type Client struct {
	id    string
	conn  *websocket.Conn
	board *game.Board
}

func printMessage(resp web.Response) {
	fmt.Println("---------------------")
	fmt.Println("Status: ", resp.GetAction())
	if len(resp.GetMessage()) > 0 {
		fmt.Println("Message: ", resp.GetMessage())
	}
	if len(resp.GetArgs()) != 0 {
		fmt.Println("Additional info: ", resp.GetArgs())
	}
}

func (c *Client) processResponse(resp web.Response) {
	printMessage(resp)

	switch resp.GetAction() {
	case Register:
		c.id = resp.Args["id"].(string)
	case PlaceShip:
		c.board.Print()
	case Placed:
		c.placeShip(resp)
		c.board.Print()
	case ShootOutcome:
		c.processShootOutcome(resp)
		c.board.Print()
	case Shoot:
		c.receiveAttack(resp)
		c.board.Print()
	}
}

func extractCoordinates(resp web.Response) (int, int) {
	x := int(resp.Args["x"].(float64))
	y := int(resp.Args["y"].(float64))
	return x, y
}

func (c *Client) placeShip(resp web.Response) {
	x, y := extractCoordinates(resp)
	direction := resp.Args["direction"].(string)
	length := int(resp.Args["length"].(float64))

	ship := game.CreateShip(x, y, direction, length)
	err := c.board.PlaceShip(ship)
	if err != nil {
		fmt.Println(err)
	}
}

func (c *Client) processShootOutcome(resp web.Response) {
	success := resp.Args["hit"].(bool)
	x, y := extractCoordinates(resp)
	err := c.board.Attack(game.Position{
		X: x,
		Y: y,
	}, success)
	if err != nil {
		fmt.Println(err)
	}
}

func (c *Client) receiveAttack(resp web.Response) {
	if len(resp.GetArgs()) != 0 {
		x, y := extractCoordinates(resp)
		c.board.ReceiveAttack(game.Position{
			X: x,
			Y: y,
		})
	}
}

func readLoop(done chan<- struct{}, client *Client) {
	defer func() {
		done <- struct{}{}
	}()
	for {
		fmt.Println("read")
		_, bytes, err := client.conn.ReadMessage()

		if err != nil {
			return
		}
		var resp web.Response
		json.Unmarshal(bytes, &resp)
		client.processResponse(resp)
	}
}

func writeLoop(done chan<- struct{}, client *Client) {
	defer func() {
		done <- struct{}{}
	}()
	for {
		b, action, err := readAction()
		if err != nil {
			fmt.Println("An error has occurred. Please enter your command again")
			continue
		}

		request := web.BuildRequest(client.id, action, nil)

		switch action {
		case Create:
			sendRequest(request, client)
		case List:
			sendRequest(request, client)
		case Join:
			joinRoom(request, client)
		case JoinRandom:
			sendRequest(request, client)
		case PlaceShip:
			placeShipOnBoard(b, request, client)
		case Shoot:
			shootAtEnemy(b, request, client)
		case Exit:
			sendRequest(request, client)
			return
		}
	}
}

func readAction() ([]byte, string, error) {
	fmt.Println("enter action")
	buf := bufio.NewReader(os.Stdin)

	b, err := buf.ReadBytes('\n')
	if err != nil {
		return nil, "", err
	}

	action := string(b)
	action = strings.TrimSuffix(action, "\n")
	return b, action, nil
}

func joinRoom(request web.Request, client *Client) {
	fmt.Println("enter room ID")
	buf := bufio.NewReader(os.Stdin)

	b, _ := buf.ReadBytes('\n')
	id := string(b)

	id = strings.TrimSuffix(id, "\n")
	args := map[string]interface{}{"roomId": id}
	request.Args = args

	sendRequest(request, client)
}

func placeShipOnBoard(b []byte, request web.Request, client *Client) {
	buf := bufio.NewReader(os.Stdin)

	fmt.Println("enter x coordinate")
	rune, _, _ := buf.ReadRune()
	x := strconv.Itoa(int(rune - 'A'))
	buf.ReadBytes('\n')

	fmt.Println("enter y coordinate")
	b, _ = buf.ReadBytes('\n')
	y := string(b)
	y = strings.TrimSuffix(y, "\n")

	fmt.Println("enter placement direction")
	b, _ = buf.ReadBytes('\n')
	direction := string(b)
	direction = strings.TrimSuffix(direction, "\n")

	args := map[string]interface{}{"x": x, "y": y, "direction": direction}
	request.Args = args
	sendRequest(request, client)
}

func shootAtEnemy(b []byte, request web.Request, client *Client) {
	buf := bufio.NewReader(os.Stdin)

	fmt.Println("enter x coordinate")
	rune, _, _ := buf.ReadRune()
	x := strconv.Itoa(int(rune - 'A'))
	buf.ReadBytes('\n')

	fmt.Println("enter y coordinate")
	b, _ = buf.ReadBytes('\n')
	y := string(b)
	y = strings.TrimSuffix(y, "\n")

	args := map[string]interface{}{"x": x, "y": y}
	request.Args = args
	sendRequest(request, client)
}

func sendRequest(request web.Request, client *Client) {
	marshal, _ := json.Marshal(request)
	err := client.conn.WriteMessage(websocket.BinaryMessage, marshal)
	if err != nil {
		fmt.Println(">>", err)
	}
}

func main() {
	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	client := &Client{
		id:    "",
		conn:  c,
		board: game.InitBoard(),
	}
	done := make(chan struct{})

	go readLoop(done, client)
	go writeLoop(done, client)
	<-done
}
