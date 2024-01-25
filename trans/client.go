package trans

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type      string `json:"type"`
	SDP       string `json:"sdp"`
	Candidate string `json:"candidate"`
}

type TransEx struct {
	conn *websocket.Conn
	mess chan Message
	send chan Message
}

func NewTrans(host string) *TransEx {
	conn, _, err := websocket.DefaultDialer.Dial(host, nil)
	if err != nil {
		log.Fatal("dial error:", err)
	}
	var trans = &TransEx{
		conn: conn,
		mess: make(chan Message, 1),
		send: make(chan Message),
	}
	go func() {
		// Wait for an SDP offer from the server
		for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				log.Fatal("read error:", err)
				close(trans.mess)
				close(trans.send)
				return
			}

			var message Message
			err = json.Unmarshal(p, &message)
			if err != nil {
				log.Fatal("unmarshal error:", err)
			}
			trans.mess <- message
		}
	}()

	go func() {
		for m := range trans.send {
			err := conn.WriteJSON(m)
			if err != nil {
				log.Fatal("Send error:", err)
				return
			}
		}
	}()
	return trans
}

func (t *TransEx) GetMess() chan Message {
	return t.mess
}

func (t *TransEx) GetSend() chan Message {
	return t.send
}
