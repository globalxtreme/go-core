package xtremews

import (
	"encoding/json"
	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
	"net/http"
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
	Conn     *websocket.Conn
	GroupId  string
	RoomId   string
	StopChan chan struct{}
}

type Message struct {
	GroupId     string
	RoomId      string
	Content     []byte
	MessageType int
}

type WSOption struct {
	Interval     int
	Channel      string
	DefaultEvent string
}

// ** --- EVENT --- */

const WS_EVENT_RESPONSE = "response"
const WS_EVENT_ROUTINE = "routine"
const WS_EVENT_CONVERSATION = "conversation"
const WS_EVENT_ERROR = "error"
const WS_EVENT_ACTION_CREATE = "action-create"
const WS_EVENT_ACTION_UPDATE = "action-update"
const WS_EVENT_ACTION_DELETE = "action-delete"

// ** --- REQUEST --- */

const WS_REQUEST_MESSAGE = "ws-request-message"

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
				close(sub.StopChan)
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

func Publish(channel string, action string, message interface{}) error {
	conn := xtremepkg.RedisPool.Get()
	defer conn.Close()

	_, err := conn.Do("PUBLISH", channel, SetContent(action, message))
	if err != nil {
		return err
	}
	return nil
}

func Subscribe(channel string, handleMessage func(message []byte)) error {
	conn := xtremepkg.RedisPool.Get()
	defer conn.Close()

	psc := redis.PubSubConn{Conn: conn}
	if err := psc.Subscribe(channel); err != nil {
		return err
	}

	for {
		switch v := psc.Receive().(type) {
		case redis.Message:
			handleMessage(v.Data)
		case error:
			return v
		}
	}
}

func GetMessage(r *http.Request) []byte {
	return r.Context().Value(WS_REQUEST_MESSAGE).([]byte)
}

func SetContent(event string, content interface{}) []byte {
	data := map[string]interface{}{
		"event":  event,
		"result": content,
	}

	result, _ := json.Marshal(data)
	return result
}
