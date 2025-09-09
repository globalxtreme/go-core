package xtremews

import (
	"context"
	"encoding/json"
	"fmt"
	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"net/http"
	"strings"
	"time"
)

func WSHandleFunc(router *mux.Router, path string, cb func(r *http.Request, opt *WSHandlerOption) (interface{}, error), args ...WSOption) {
	router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		conn, subscription, cleanup := upgrade(w, r)
		if conn == nil {
			return
		}
		defer cleanup()

		ctx := context.WithValue(r.Context(), WS_REQUEST_SUBSCRIPTION, subscription)

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

		hdlOpt := WSHandlerOption{}
		handleCallback := func(event string, r *http.Request) []byte {
			result, err := cb(r, &hdlOpt)
			return SetContent(event, result, err)
		}

		ctx = context.WithValue(ctx, WS_REQUEST_MESSAGE, message)
		Hub.Broadcast <- Message{
			MessageType: websocket.TextMessage,
			RoomId:      subscription.RoomId,
			Content:     handleCallback(defaultEvent, r.WithContext(ctx)),
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
							Content:     handleCallback(WS_EVENT_ROUTINE, r.WithContext(ctx)),
						}
					case <-subscription.StopChan:
						xtremepkg.LogError(fmt.Sprintf("Stopping goroutine for RoomId: %s", subscription.RoomId), false)
						return
					}
				}
			}()
		}

		if option.Channel != "" && len(option.Channel) > 0 {
			WSCustomSubscriptionEvent(subscription, option.Channel, func(msg map[string]interface{}) interface{} {
				return msg
			})
		}

		for {
			_, message, err = conn.ReadMessage()
			if err != nil {
				xtremepkg.LogError(fmt.Sprintf("Error reading message: %v", err), false)
				return
			}

			ctx = context.WithValue(ctx, WS_REQUEST_MESSAGE, message)
			Hub.Broadcast <- Message{
				MessageType: websocket.TextMessage,
				GroupId:     subscription.GroupId,
				RoomId:      subscription.RoomId,
				Content:     handleCallback(defaultEvent, r.WithContext(ctx)),
			}
		}
	}).Methods("GET")
}

func WSCustomSubscriptionEvent(subscription *Subscription, channel string, cb func(msg map[string]interface{}) interface{}) {
	go func() {
		subsCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			err := Subscribe(subsCtx, channel, subscription.GroupId, func(message []byte) {
				var messageMap map[string]interface{}
				json.Unmarshal(message, &messageMap)

				if resultMap, resOk := messageMap["result"].(map[string]interface{}); resOk {
					validMessage := cb(resultMap)
					if validMessage != nil {
						select {
						case Hub.Broadcast <- Message{
							MessageType: websocket.TextMessage,
							RoomId:      subscription.RoomId,
							Content:     SetContent(WS_EVENT_MONITORING, validMessage, nil),
						}:
						case <-subsCtx.Done():
							xtremepkg.LogError(fmt.Sprintf("Unsubscribing from Redis for RoomId: %s", subscription.RoomId), false)
							return
						}
					}
				}
			})
			if err != nil {
				xtremepkg.LogError(fmt.Sprintf("Error subscribing to Redis: %v", err), true)
				return
			}
		}()

		select {
		case <-subscription.StopChan:
			cancel()
			xtremepkg.LogError(fmt.Sprintf("Stopping goroutine for RoomId on the subscribtion redis: %s", subscription.RoomId), false)
			return
		}
	}()
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

	offered := websocket.Subprotocols(r)
	var authToken string

	prefix := "auth.jwt."
	for _, p := range offered {
		if strings.HasPrefix(p, prefix) {
			authToken = p
			break
		}
	}

	respHeader := http.Header{}
	respHeader.Set("Sec-WebSocket-Protocol", authToken)

	conn, err := upgrader.Upgrade(w, r, respHeader)
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
