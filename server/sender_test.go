package main

import (
	"encoding/json"
	"errors"
	"github.com/StanislavStefanov/Battleships/pkg/web"
	connection "github.com/StanislavStefanov/Battleships/server/player/automock"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestSender_SendResponse(t *testing.T) {
	t.Run("fail when error occurs while writing response into connection", func(t *testing.T) {
		// when
		resp := web.Response{
			Action:  "action",
			Message: "msg",
			Args:    nil,
		}

		rr, _ := json.Marshal(resp)

		con := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("WriteMessage", websocket.BinaryMessage, rr).Return(errors.New("error")).Once()
			return con
		}()

		rescueStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		s := Sender{}
		s.SendResponse(resp, con)

		w.Close()
		out, _ := ioutil.ReadAll(r)
		os.Stdout = rescueStdout

		assert.Equal(t, "Send response: send message error:  error\n", string(out))
		con.AssertExpectations(t)
	})
	t.Run("success", func(t *testing.T) {
		// when
		resp := web.Response{
			Action:  "action",
			Message: "msg",
			Args:    nil,
		}

		rr, _ := json.Marshal(resp)

		con := func() *connection.Connection {
			con := &connection.Connection{}
			con.On("WriteMessage", websocket.BinaryMessage, rr).Return(nil).Once()
			return con
		}()

		s := Sender{}
		s.SendResponse(resp, con)

		con.AssertExpectations(t)
	})
}

