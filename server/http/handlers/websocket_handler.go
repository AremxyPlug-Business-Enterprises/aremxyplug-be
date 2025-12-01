package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") != ""
	},
}

// UserEventsWS upgrades to websocket, subscribes to Redis channel transfer:events:{userID},
// and forwards messages. Cancel/close is handled on client disconnect.
func (handler *HttpHandler) UserEventsWS(w http.ResponseWriter, r *http.Request) {
	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userDetails.ID

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// subscribe to user's redis channel
	msgCh, err := handler.redisClient.SubscribeUserEvents(ctx, userID)
	if err != nil {
		return
	}

	// reader goroutine: detect client disconnects (will cancel context)
	go func() {
		conn.SetReadLimit(512)
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	// writer loop: forward redis messages to websocket
	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-msgCh:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(m)); err != nil {
				return
			}
		}
	}
}
