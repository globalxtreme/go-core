package xtremews

import (
	"fmt"
	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

func ConversationHandler(router *mux.Router, path string, cb func(r *http.Request, message []byte) []byte) {
	router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		conn, subscription, cleanup := upgrade(w, r)
		if conn == nil {
			return
		}
		defer cleanup()

		conn.SetPingHandler(nil)

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				xtremepkg.LogError(fmt.Sprintf("Error reading message: %v", err), false)
				return
			}

			Hub.Broadcast <- Message{MessageType: websocket.TextMessage, RoomId: subscription.RoomId, Content: cb(r, msg)}
		}
	}).Methods("GET")
}

func MonitoringHandler(router *mux.Router, path string, period int, cb func(r *http.Request, message []byte) []byte) {
	router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		conn, subscription, cleanup := upgrade(w, r)
		if conn == nil {
			return
		}
		defer cleanup()

		conn.SetPingHandler(nil)

		var message []byte

		go func() {
			tinker := time.NewTicker(time.Duration(period) * time.Second)
			defer tinker.Stop()

			for range tinker.C {
				Hub.Broadcast <- Message{MessageType: websocket.TextMessage, RoomId: subscription.RoomId, Content: cb(r, message)}
			}
		}()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				xtremepkg.LogError(fmt.Sprintf("Error reading message: %v", err), false)
				return
			}

			message = msg
			Hub.Broadcast <- Message{MessageType: websocket.TextMessage, RoomId: subscription.RoomId, Content: cb(r, message)}
		}
	}).Methods("GET")
}

func upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, *Subscription, func()) {
	var roomId string

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
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

	subscription := Subscription{Conn: conn, RoomId: roomId}
	Hub.Register <- subscription

	cleanup := func() {
		defer conn.Close()
		Hub.Unregister <- subscription
	}

	return conn, &subscription, cleanup
}
