package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/StanislavStefanov/Battleships/utils"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"os"
	"os/signal"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func readLoop(done chan<- struct{}, conn *websocket.Conn) {
	defer func() {
		done <- struct{}{}
	}()
	for {
		_, bytes, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}
		var msg utils.Response
		json.Unmarshal(bytes, &msg)
		fmt.Println(msg)

		//req := utils.BuildRequest("az", "ls-rooms", nil)
		//marshal, _ := json.Marshal(req)
		//conn.WriteMessage(websocket.BinaryMessage, marshal)
	}
}

func writeLoop(done chan<- struct{}, conn *websocket.Conn) {
	defer func() {
		done <- struct{}{}
	}()
	for {
		fmt.Println("enter command")
		buf := bufio.NewReader(os.Stdin)
		fmt.Print("> ")
		sentence, _ := buf.ReadBytes('\n')
		s := string(sentence)
		fmt.Println(s)
		var msg = utils.BuildRequest("az", s, nil)
		marshal, _ := json.Marshal(msg)
		conn.WriteMessage(websocket.BinaryMessage, marshal)
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

	done := make(chan struct{})

	go readLoop(done, c)
	go writeLoop(done, c)
	<-done
}
