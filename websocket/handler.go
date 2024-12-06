package xtremews

import (
	"context"
	"fmt"
	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
	"time"
)

func WSHandleFunc(router *mux.Router, path string, cb func(r *http.Request) interface{}, args ...WSOption) {
	router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		conn, subscription, cleanup := upgrade(w, r)
		if conn == nil {
			return
		}
		defer cleanup()

		conn.SetPingHandler(nil)

		var option WSOption
		if len(args) > 0 {
			option = args[0]
		}

		var err error
		var message []byte

		defaultEvent := WS_EVENT_RESPONSE
		if option.DefaultEvent != "" {
			defaultEvent = option.DefaultEvent
		}

		ctx := context.WithValue(r.Context(), WS_REQUEST_MESSAGE, message)
		Hub.Broadcast <- Message{
			MessageType: websocket.TextMessage,
			GroupId:     subscription.GroupId,
			RoomId:      subscription.RoomId,
			Content:     SetContent(defaultEvent, cb(r.WithContext(ctx))),
		}

		if option.Interval > 0 {
			go func() {
				tinker := time.NewTicker(time.Duration(option.Interval) * time.Second)
				defer tinker.Stop()

				for {
					select {
					case <-tinker.C:
						Hub.Broadcast <- Message{
							MessageType: websocket.TextMessage,
							RoomId:      subscription.RoomId,
							Content:     SetContent(WS_EVENT_ROUTINE, cb(r.WithContext(ctx))),
						}
					case <-subscription.StopChan:
						xtremepkg.LogError(fmt.Sprintf("Stopping goroutine for RoomId: %s", subscription.RoomId), false)
						return
					}
				}
			}()

		}

		if option.Channel != "" && len(option.Channel) > 0 {
			go func() {
				var once sync.Once

				stop := make(chan struct{})
				defer func() {
					once.Do(func() {
						close(stop)
					})
				}()

				go func() {
					err := Subscribe(option.Channel, func(message []byte) {
						select {
						case Hub.Broadcast <- Message{
							MessageType: websocket.TextMessage,
							RoomId:      subscription.RoomId,
							Content:     message,
						}:
						case <-stop:
							xtremepkg.LogError(fmt.Sprintf("Unsubscribing from Redis for RoomId: %s", subscription.RoomId), false)
							return
						}
					})
					if err != nil {
						xtremepkg.LogError(fmt.Sprintf("Error subscribing to Redis: %v", err), true)
						return
					}
				}()

				select {
				case <-subscription.StopChan:
					once.Do(func() {
						close(stop)
					})
					xtremepkg.LogError(fmt.Sprintf("Stopping goroutine for RoomId on the subscribtion redis: %s", subscription.RoomId), false)
					return
				}
			}()
		}

		for {
			_, message, err = conn.ReadMessage()
			if err != nil {
				xtremepkg.LogError(fmt.Sprintf("Error reading message: %v", err), false)
				return
			}

			ctx = context.WithValue(r.Context(), WS_REQUEST_MESSAGE, message)
			Hub.Broadcast <- Message{
				MessageType: websocket.TextMessage,
				GroupId:     subscription.GroupId,
				RoomId:      subscription.RoomId,
				Content:     SetContent(defaultEvent, cb(r.WithContext(ctx))),
			}
		}
	}).Methods("GET")
}

/** --- UNEXPORTED FUNCTIONS --- */

func upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, *Subscription, func()) {
	var groupId, roomId string

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			groupId = r.Header.Get("X-Group-ID")

			roomId = r.Header.Get("X-Room-ID")
			if roomId == "" {
				xtremepkg.LogError("Room ID is required", true)
				return false
			}

			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		xtremepkg.LogError(fmt.Sprintf("Error upgrading connection: %v", err), true)
		return nil, nil, nil
	}

	subscription := Subscription{
		Conn:     conn,
		GroupId:  groupId,
		RoomId:   roomId,
		StopChan: make(chan struct{}),
	}
	Hub.Register <- subscription

	cleanup := func() {
		Hub.Unregister <- subscription
	}

	return conn, &subscription, cleanup
}
