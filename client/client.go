package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/StanislavStefanov/Battleships/server/board"
	"github.com/StanislavStefanov/Battleships/utils"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"os"
	"os/signal"
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
	Create       = "create"
	Created      = "created"
	Join         = "join-room"
	JoinRandom   = "join-random"
)

type Client struct {
	id    string
	conn  *websocket.Conn
	board *board.Board
}

func printMessage(resp utils.Response) {
	fmt.Println("---------------------")
	fmt.Println("Status: ", resp.GetAction())
	fmt.Println("Message: ", resp.GetMessage())
	if len(resp.GetArgs()) != 0 {
		fmt.Println("Additional info: ", resp.GetArgs())
	}
}

func (c *Client) processResponse(resp utils.Response) {
	printMessage(resp)

	switch resp.GetAction() {
	case Register:
		c.id = resp.Args["id"].(string)
	case PlaceShip:
		c.board.Print()
	case Placed:
		x := int(resp.Args["x"].(float64))
		y := int(resp.Args["y"].(float64))
		direction := resp.Args["direction"].(string)
		length := int(resp.Args["length"].(float64))

		ship := board.CreateShip(x, y, direction, length)
		c.board.PlaceShip(ship)
		c.board.Print()
	case ShootOutcome:
		success := resp.Args["hit"].(bool)
		x := resp.Args["x"].(int)
		y := resp.Args["y"].(int)
		c.board.Attack(board.Position{
			X: x,
			Y: y,
		}, success)
		c.board.Print()
	case Shoot:
		if len(resp.GetArgs()) != 0 {
			x := resp.Args["x"].(int)
			y := resp.Args["y"].(int)
			c.board.ReceiveAttack(board.Position{
				X: x,
				Y: y,
			})
		}
		c.board.Print()
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
			fmt.Println("read error", err)
			return
		}
		var resp utils.Response
		json.Unmarshal(bytes, &resp)
		client.processResponse(resp)
	}
}

func writeLoop(done chan<- struct{}, client *Client) {
	defer func() {
		done <- struct{}{}
	}()
	for {
		fmt.Println("enter action")
		buf := bufio.NewReader(os.Stdin)

		b, _ := buf.ReadBytes('\n')
		action := string(b)
		action = strings.TrimSuffix(action, "\n")

		request := utils.BuildRequest(client.id, action, nil)

		switch action {
		case Create:
			sendRequest(request, client)
		case List:
			sendRequest(request, client)
		case Join:
			fmt.Println("enter room ID")
			buf := bufio.NewReader(os.Stdin)

			b, _ := buf.ReadBytes('\n')
			id := string(b)

			id = strings.TrimSuffix(id, "\n")
			args := map[string]interface{}{"roomId": id}
			request.Args = args

			sendRequest(request, client)
		case JoinRandom:
			sendRequest(request, client)
		case PlaceShip:

			buf := bufio.NewReader(os.Stdin)

			fmt.Println("enter x coordinate")
			b, _ = buf.ReadBytes('\n')
			x := string(b)
			x = strings.TrimSuffix(x, "\n")

			fmt.Println("enter y coordinate")
			b, _ = buf.ReadBytes('\n')
			y := string(b)
			y = strings.TrimSuffix(y, "\n")

			//b, _ := buf.ReadBytes('\n')
			//x, _ := strconv.Atoi(strings.TrimSuffix(string(b), "\n"))
			//
			//fmt.Println("enter y coordinate")
			//b, _ = buf.ReadBytes('\n')
			//y, _ := strconv.Atoi(strings.TrimSuffix(string(b), "\n"))

			fmt.Println("enter placement direction")
			b, _ = buf.ReadBytes('\n')
			direction := string(b)
			direction = strings.TrimSuffix(direction, "\n")

			args := map[string]interface{}{"x": x, "y": y, "direction": direction}
			request.Args = args
			sendRequest(request, client)
		case Shoot:
			buf := bufio.NewReader(os.Stdin)

			fmt.Println("enter x coordinate")
			b, _ = buf.ReadBytes('\n')
			x := string(b)
			x = strings.TrimSuffix(x, "\n")

			fmt.Println("enter y coordinate")
			b, _ = buf.ReadBytes('\n')
			y := string(b)
			y = strings.TrimSuffix(y, "\n")

			args := map[string]interface{}{"x": x, "y": y}
			request.Args = args
			sendRequest(request, client)

		case Exit:
			sendRequest(request, client)
			return

		}

		marshal, _ := json.Marshal(request)
		err := client.conn.WriteMessage(websocket.BinaryMessage, marshal)
		if err != nil {
			fmt.Println(">>", err)
		}
	}
}

func sendRequest(request utils.Request, client *Client) {
	marshal, _ := json.Marshal(request)
	err := client.conn.WriteMessage(websocket.BinaryMessage, marshal)
	if err != nil {
		fmt.Println(">>", err)
	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

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
		board: board.InitBoard(),
	}
	done := make(chan struct{})

	go readLoop(done, client)
	go writeLoop(done, client)
	<-done
}
