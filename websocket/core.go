package xtremews

import (
	"github.com/gorilla/websocket"
	"sync"
)

type hub struct {
	Rooms      map[string]map[*websocket.Conn]bool
	Broadcast  chan Message
	Register   chan Subscription
	Unregister chan Subscription
	Mutex      sync.Mutex
}

type Subscription struct {
	Conn   *websocket.Conn
	RoomId string
}

type Message struct {
	MessageType int
	RoomId      string
	Content     []byte
}

var (
	Hub *hub
)

func Init() {
	Hub = &hub{
		Rooms:      make(map[string]map[*websocket.Conn]bool),
		Broadcast:  make(chan Message),
		Register:   make(chan Subscription),
		Unregister: make(chan Subscription),
	}
}

func Run() {
	for {
		select {
		case sub := <-Hub.Register:
			Hub.Mutex.Lock()

			if _, ok := Hub.Rooms[sub.RoomId]; !ok {
				Hub.Rooms[sub.RoomId] = make(map[*websocket.Conn]bool)
			}

			Hub.Rooms[sub.RoomId][sub.Conn] = true
			Hub.Mutex.Unlock()

		case sub := <-Hub.Unregister:
			Hub.Mutex.Lock()

			if connections, ok := Hub.Rooms[sub.RoomId]; ok {
				if _, ok := connections[sub.Conn]; ok {
					delete(connections, sub.Conn)

					sub.Conn.Close()

					if len(connections) == 0 {
						delete(Hub.Rooms, sub.RoomId)
					}
				}
			}

			Hub.Mutex.Unlock()

		case msg := <-Hub.Broadcast:
			Hub.Mutex.Lock()

			if connections, ok := Hub.Rooms[msg.RoomId]; ok {
				for conn := range connections {
					err := conn.WriteMessage(msg.MessageType, msg.Content)
					if err != nil {
						delete(connections, conn)
						conn.Close()
					}
				}
			}

			Hub.Mutex.Unlock()
		}
	}
}
