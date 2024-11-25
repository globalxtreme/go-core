package xtremews

import (
	"github.com/gorilla/websocket"
	"sync"
)

type hub struct {
	Groups     map[string]map[string]bool
	Rooms      map[string]*websocket.Conn
	Broadcast  chan Message
	Register   chan Subscription
	Unregister chan Subscription
	Mutex      sync.Mutex
}

type Subscription struct {
	Conn    *websocket.Conn
	GroupId string
	RoomId  string
}

type Message struct {
	GroupId     string
	RoomId      string
	Content     []byte
	MessageType int
}

var (
	Hub *hub
)

func Init() {
	Hub = &hub{
		Groups:     make(map[string]map[string]bool),
		Rooms:      make(map[string]*websocket.Conn),
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
			if sub.GroupId != "" {
				if _, ok := Hub.Groups[sub.GroupId]; !ok {
					Hub.Groups[sub.GroupId] = make(map[string]bool)
				}

				Hub.Groups[sub.GroupId][sub.RoomId] = true
			}

			Hub.Rooms[sub.RoomId] = sub.Conn
			Hub.Mutex.Unlock()

		case sub := <-Hub.Unregister:
			Hub.Mutex.Lock()

			if _, ok := Hub.Rooms[sub.RoomId]; ok {
				delete(Hub.Rooms, sub.RoomId)
				sub.Conn.Close()
			}

			if sub.GroupId != "" {
				if _, ok := Hub.Groups[sub.GroupId][sub.RoomId]; ok {
					delete(Hub.Groups[sub.GroupId], sub.RoomId)
				}
			}

			Hub.Mutex.Unlock()

		case msg := <-Hub.Broadcast:
			Hub.Mutex.Lock()

			if msg.GroupId != "" {
				if rooms, ok := Hub.Groups[msg.GroupId]; ok && rooms != nil && len(rooms) > 0 {
					for room, _ := range rooms {
						if conn, ok := Hub.Rooms[room]; ok {
							err := conn.WriteMessage(msg.MessageType, msg.Content)
							if err != nil {
								delete(Hub.Rooms, room)
								delete(Hub.Groups[msg.GroupId], room)

								conn.Close()
							}
						}
					}
				}
			} else if conn, ok := Hub.Rooms[msg.RoomId]; ok {
				err := conn.WriteMessage(msg.MessageType, msg.Content)
				if err != nil {
					delete(Hub.Rooms, msg.RoomId)
					conn.Close()
				}
			}

			Hub.Mutex.Unlock()
		}
	}
}
