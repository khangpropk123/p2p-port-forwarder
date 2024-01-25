package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Message struct {
	id        string
	Type      string `json:"type"`
	SDP       string `json:"sdp"`
	Candidate string `json:"candidate"`
}

var (
	clients  = make(map[*websocket.Conn]string) // connected clients
	messages = make(chan Message)               // broadcast channel
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {

	id := uuid.New().String()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	clients[conn] = id

	for {
		var message Message
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, conn)
			break
		}
		fmt.Println(message)
		message.id = id

		messages <- message
	}
}

func broadcastMessages() {
	for {
		message := <-messages
	GOTO:
		if len(clients) < 2 {
			<-time.After(time.Millisecond * 200)
			goto GOTO
		}
		for client, id := range clients {
			if id != message.id {
				err := client.WriteJSON(message)
				if err != nil {
					log.Printf("error: %v", err)
					client.Close()
					delete(clients, client)
				}
			}
		}
	}
}

func main() {

	http.HandleFunc("/ws", handleWebSocket)
	go broadcastMessages()

	err := http.ListenAndServe(fmt.Sprintf(":%s", os.Args[1]), nil)
	if err != nil {
		log.Fatal("error starting server:", err)
	}
}
