package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

type DBInterface interface {

	// add any other methods you have already implemented
}

// Webhook handler: quick ack then background processing
func (handler HttpHandler) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	if !handler.allowedIP(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Respond immediately
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))

	// Process in background
	go func(b []byte) {
		if err := handler.ProcessWebhook(b); err != nil {
			handler.logger.Error("Failed to process webhook", zap.Error(err))
		}
	}(body)
}

func (handler HttpHandler) allowedIP(r *http.Request) bool {
	allowedIPs := map[string]bool{
		"123.45.67.89": true,
	}
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		parts := strings.Split(xf, ",")
		ip := strings.TrimSpace(parts[0])
		if allowedIPs[ip] {
			return true
		}
	}
	ip := strings.Split(r.RemoteAddr, ":")[0]
	return allowedIPs[ip]
}

// webhookPayload minimal struct
type webhookPayload struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Attributes    map[string]interface{} `json:"attributes"`
	Relationships map[string]interface{} `json:"relationships"`
}

func (handler HttpHandler) ProcessWebhook(body []byte) error {

	var p webhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		handler.logger.Error("failed parse webhook JSON", zap.Error(err))
		return err
	}

	// extract apiRef
	var apiRef string
	if rel, ok := p.Relationships["transfer"]; ok {
		if m, ok := rel.(map[string]interface{}); ok {
			if d, ok := m["data"].(map[string]interface{}); ok {
				if idv, ok := d["id"].(string); ok {
					apiRef = idv
				}
			}
		}
	}

	if apiRef == "" {
		apiRef = p.ID
	}

	// extract sessionId and failureReason
	var sessionID, failureReason string
	if s, ok := p.Attributes["sessionId"].(string); ok {
		sessionID = s
	}
	if fr, ok := p.Attributes["failureReason"].(string); ok {
		failureReason = fr
	}

	// Fast Redis lookup: transfer:ext:{apiRef} -> txID
	var txID string
	val, err := handler.redisClient.Get(fmt.Sprintf("transfer:ext:%s", apiRef))
	if err != nil {
		handler.logger.Warn("redis lookup error for ext mapping", zap.Error(err))
	}
	if val != nil {
		if s, ok := val.(string); ok {
			txID = s
		} else {
			// val may be []byte or numeric string; attempt conversion
			if bs, ok := val.([]byte); ok {
				txID = string(bs)
			}
		}
	}

	// fallback DB lookup
	if txID == "" {
		rec, derr := handler.store.GetReceiptByExternalRef(apiRef)
		if derr != nil {
			handler.logger.Warn("unknown externalRef in webhook", zap.String("apiRef", apiRef))
			return nil // ack done already
		}
		txID = rec.Transaction_ID
	}

	// load receipt and idempotency check
	rec, err := handler.store.GetReceiptByTxID(txID)
	if err != nil {
		handler.logger.Error("store get receipt failed", zap.Error(err), zap.String("txID", txID))
		return err
	}
	if rec.Status == "success" || rec.Status == "failed" {
		handler.logger.Info("receipt already final, skipping", zap.String("txID", txID), zap.String("status", rec.Status))
		return nil
	}

	// get meta: try redis Get on transfer:meta:{txID}
	var meta struct {
		UserID string `json:"user_id"`
		Amount int64  `json:"amount"`
	}
	metaVal, err := handler.redisClient.Get(fmt.Sprintf("transfer:meta:%s", txID))
	if err != nil {
		handler.logger.Warn("failed to read transfer meta from redis", zap.Error(err))
	}
	if metaVal != nil {
		switch v := metaVal.(type) {
		case string:
			_ = json.Unmarshal([]byte(v), &meta)
		case []byte:
			_ = json.Unmarshal(v, &meta)
		default:
			// attempt to marshal then unmarshal
			b, _ := json.Marshal(v)
			_ = json.Unmarshal(b, &meta)
		}
	} else {
		// fallback to receipt data
		meta.UserID = rec.UserID
		// Convert rec.Amount (string) to int64
		if amt, err := strconv.ParseInt(rec.Amount, 10, 64); err == nil {
			meta.Amount = amt
		} else {
			handler.logger.Warn("failed to parse receipt amount", zap.String("amount", rec.Amount), zap.Error(err))
			meta.Amount = 0
		}
	}

	// decide: failure -> release, else success -> confirm
	if failureReason != "" {
		if meta.UserID != "" {
			if released, err := handler.redisClient.ReleaseHold(meta.UserID, txID); err != nil {
				handler.logger.Error("release hold failed", zap.Error(err), zap.String("txID", txID))
			} else if released {
				handler.logger.Info("hold released", zap.String("txID", txID))
			} else {
				handler.logger.Info("no hold to release", zap.String("txID", txID))
			}
		}
		if err := handler.store.UpdateReceiptFinal(txID, "failed", sessionID); err != nil {
			handler.logger.Error("failed update receipt final", zap.Error(err))
			return err
		}
		handler.logger.Info("processed failed webhook", zap.String("txID", txID), zap.String("reason", failureReason))
		return nil
	}

	// success path
	if meta.UserID != "" {
		if confirmed, err := handler.redisClient.ConfirmHold(meta.UserID, txID); err != nil {
			handler.logger.Error("confirm hold failed", zap.Error(err))
		} else if confirmed {
			handler.logger.Info("hold confirmed", zap.String("txID", txID))
		} else {
			handler.logger.Info("no hold present to confirm", zap.String("txID", txID))
		}

		// persist current Redis balance snapshot to DB
		if currentBal, err := handler.redisClient.GetBalance(meta.UserID); err == nil {
			if err := handler.store.UpdateUserBalanceFromRedis(meta.UserID, currentBal); err != nil {
				handler.logger.Error("failed to persist user balance to DB", zap.Error(err))
			}
		} else {
			handler.logger.Warn("failed to read balance from redis for persist", zap.Error(err))
		}
	}

	if err := handler.store.UpdateReceiptFinal(txID, "success", sessionID); err != nil {
		handler.logger.Error("failed to update receipt final", zap.Error(err))
		return err
	}

	handler.logger.Info("processed success webhook", zap.String("txID", txID))
	return nil
}
