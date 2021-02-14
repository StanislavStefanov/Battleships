package main

import (
	"encoding/json"
	"fmt"
	"github.com/StanislavStefanov/Battleships/pkg/web"
	"github.com/StanislavStefanov/Battleships/server/player"
	"github.com/gorilla/websocket"
)

//go:generate mockery -name=ResponseSender -output=automock -outpkg=automock -case=underscore
type ResponseSender interface {
	SendResponse(response web.Response, conn player.Connection)
}

type Sender struct {
}

func (s *Sender) SendResponse(response web.Response, conn player.Connection) {
	resp, err := json.Marshal(response)
	if err != nil {
		fmt.Println("Send response: marshal error: ", err)
		return
	}
	err = conn.WriteMessage(websocket.BinaryMessage, resp)
	if err != nil {
		fmt.Println("Send response: send message error: ", err)
		return
	}
}