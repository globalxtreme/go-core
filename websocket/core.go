package xtremews

import (
	"context"
	"encoding/json"
	"fmt"
	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
	"time"
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

type WSHandlerOption struct {
	HasSubscribedEvent bool
}

/** --- EVENT --- */

const WS_EVENT_RESPONSE = "response"
const WS_EVENT_ROUTINE = "routine"
const WS_EVENT_CONVERSATION = "conversation"
const WS_EVENT_MONITORING = "monitoring"
const WS_EVENT_ERROR = "error"
const WS_EVENT_ACTION_CREATE = "action-create"
const WS_EVENT_ACTION_UPDATE = "action-update"
const WS_EVENT_ACTION_DELETE = "action-delete"

/** --- REQUEST --- */

const WS_REQUEST_MESSAGE = "ws-request-message"

const WS_REQUEST_SUBSCRIPTION = "ws-request-subscription"

/** --- CHANNEL --- */

const WS_CHANNEL_MESSAGE_BROKER_ASYNC_WORKFLOW_MONITORING = "ws-channel.async-workflow.monitoring"

const WS_GROUP_ID_ASYNC_WORKFLOW_MONITORING_LIST = "asa.monitoring.list"

var (
	Hub *hub
)

func InitWebSocket() {
	Hub = &hub{
		Groups:     make(map[string]map[string]bool),
		Rooms:      make(map[string]*websocket.Conn),
		Broadcast:  make(chan Message),
		Register:   make(chan Subscription),
		Unregister: make(chan Subscription),
	}

	go Run()
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

func Publish(channel, groupId string, action string, message interface{}, connArg ...redis.Conn) error {
	var conn redis.Conn
	if len(connArg) > 0 {
		conn = connArg[0]
	} else {
		conn = xtremepkg.RedisAsyncWorkflowPool.Get()
		defer conn.Close()
	}

	channel += fmt.Sprintf(":%s", groupId)
	_, err := conn.Do("PUBLISH", channel, SetContent(action, message, nil))
	if err != nil {
		return err
	}
	return nil
}

func Subscribe(ctx context.Context, channel, groupId string, handleMessage func(message []byte)) error {
	retryDelay := time.Second * 3

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		conn := xtremepkg.RedisAsyncWorkflowPool.Get()
		psc := redis.PubSubConn{Conn: conn}

		fullChannel := fmt.Sprintf("%s:%s", channel, groupId)
		if err := psc.Subscribe(fullChannel); err != nil {
			conn.Close()

			xtremepkg.LogError(fmt.Sprintf("Failed to subscribe: %v", err), true)
			time.Sleep(retryDelay)

			continue
		}

		errChan := make(chan error, 1)
		go func() {
			defer close(errChan)
			for {
				switch v := psc.Receive().(type) {
				case redis.Message:
					handleMessage(v.Data)
				case redis.Subscription:
					// ignore
				case error:
					errChan <- v
					return
				}
			}
		}()

		select {
		case <-ctx.Done():
			psc.Unsubscribe()
			conn.Close()

			return nil
		case err := <-errChan:
			psc.Unsubscribe()
			conn.Close()

			xtremepkg.LogError(fmt.Sprintf("Redis subscription error: %v — reconnecting...", err), true)
			time.Sleep(retryDelay)

			continue // ⬅️ auto reconnect
		}
	}
}

func GetMessage(r *http.Request) []byte {
	return r.Context().Value(WS_REQUEST_MESSAGE).([]byte)
}

func SetContent(event string, content interface{}, processError error) []byte {
	var errMessage string
	if processError != nil {
		errMessage = processError.Error()
	}

	data := map[string]interface{}{
		"event":  event,
		"error":  errMessage,
		"result": content,
	}

	result, _ := json.Marshal(data)
	return result
}

func GetSubscription(r *http.Request) *Subscription {
	return r.Context().Value(WS_REQUEST_SUBSCRIPTION).(*Subscription)
}
