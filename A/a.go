package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type      string `json:"type"`
	SDP       string `json:"sdp"`
	Candidate string `json:"candidate"`
}

func main() {
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatal("dial error:", err)
	}

	// Send an SDP offer to the server
	offer := "SDP offer from Client 1"
	message := Message{Type: "offer", SDP: offer}
	err = conn.WriteJSON(message)
	if err != nil {
		log.Fatal("write error:", err)
	}

	// Wait for an SDP answer from the server
	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Fatal("read error:", err)
		}

		var message Message
		err = json.Unmarshal(p, &message)
		if err != nil {
			log.Fatal("unmarshal error:", err)
		}

		if message.Type == "answer" {
			fmt.Println("Received SDP answer from server:", message.SDP)
			break
		}
	}

	// Send an ICE candidate to the server
	candidate := "ICE candidate from Client 1"
	message = Message{Type: "candidate", Candidate: candidate}
	err = conn.WriteJSON(message)
	if err != nil {
		log.Fatal("write error:", err)
	}
}
