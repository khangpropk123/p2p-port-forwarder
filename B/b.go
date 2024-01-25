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

	// Wait for an SDP offer from the server
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

		if message.Type == "offer" {
			fmt.Println("Received SDP offer from client 1:", message.SDP)

			// Send an SDP answer to the server
			answer := "SDP answer from Client 2"
			message = Message{Type: "answer", SDP: answer}
			err = conn.WriteJSON(message)
			if err != nil {
				log.Fatal("write error:", err)
			}

			// Wait for an ICE candidate from client 1
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

				if message.Type == "candidate" {
					fmt.Println("Received ICE candidate from client 1:", message.Candidate)
					break
				}
			}
			ca := "Candidate from Client 2"
			message = Message{Type: "candidate", SDP: ca}
			err = conn.WriteJSON(message)
			if err != nil {
				log.Fatal("write error:", err)
			}

			break
		}
	}
}
