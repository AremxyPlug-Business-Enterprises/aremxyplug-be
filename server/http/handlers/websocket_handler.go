package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") != ""
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// UserEventsWS upgrades to websocket, subscribes to Redis channel transfer:events:{userID},
// and forwards messages. Cancel/close is handled on client disconnect.
func (handler *HttpHandler) UserEventsWS(w http.ResponseWriter, r *http.Request) {
	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Warn("unauthorized websocket connection attempt", zap.Error(err))
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userDetails.ID

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		handler.logger.Error("failed to upgrade websocket", zap.String("user_id", userID), zap.Error(err))
		return
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// subscribe to user's redis channel
	msgCh, err := handler.redisClient.SubscribeUserEvents(ctx, userID)
	if err != nil {
		handler.logger.Error("failed to subscribe to user events", zap.String("user_id", userID), zap.Error(err))
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "subscription failed"))
		return
	}

	handler.logger.Info("websocket connected", zap.String("user_id", userID))

	// reader goroutine: detect client disconnects and handle ping/pong
	go func() {
		defer func() {
			if r := recover(); r != nil {
				handler.logger.Error("panic in websocket reader", zap.String("user_id", userID), zap.Any("panic", r))
			}
			cancel() // Ensure context is cancelled on exit
		}()

		conn.SetReadLimit(512)
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				handler.logger.Debug("websocket client disconnected", zap.String("user_id", userID), zap.Error(err))
				return
			}
		}
	}()

	// Send periodic pings
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// writer loop: forward redis messages to websocket
	for {
		select {
		case <-ctx.Done():
			handler.logger.Info("websocket context done", zap.String("user_id", userID))
			return
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				handler.logger.Debug("ping failed", zap.String("user_id", userID), zap.Error(err))
				return
			}
		case m, ok := <-msgCh:
			if !ok {
				handler.logger.Debug("websocket message channel closed", zap.String("user_id", userID))
				return
			}
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(m)); err != nil {
				handler.logger.Warn("failed to write websocket message", zap.String("user_id", userID), zap.Error(err))
				return
			}
			handler.logger.Debug("websocket message sent", zap.String("user_id", userID), zap.String("message", m))
		}
	}
}
