package main

import (
	"encoding/json"
	"flag"
	"fmt"
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
		_, bytes, _ := conn.ReadMessage()
		var msg map[string]interface{}
		json.Unmarshal(bytes, &msg)
		action := msg["action"].(string)
		id := msg["id"].(string)
		fmt.Println(action, id)
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

	readLoop(done, c)
}
