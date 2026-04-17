package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// placeRedisHoldAndMeta places a hold in Redis and sets minimal meta for a transaction. Returns txnID or writes error response and returns error.
// amount can be string, float64, or int
func (handler *HttpHandler) placeRedisHoldAndMeta(ctx context.Context, w http.ResponseWriter, userID string, amountRaw interface{}, bal decimal.Decimal) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	txnID := uuid.New().String()
	var amount float64
	switch v := amountRaw.(type) {
	case string:
		var err error
		amount, err = strconv.ParseFloat(v, 64)
		if err != nil {
			handler.logger.Error("Failed to parse amount string for redis hold", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "invalid amount format"},
			}
			json.NewEncoder(w).Encode(response)
			return "", err
		}
	case float64:
		amount = v
	case int:
		amount = float64(v)
	case int64:
		amount = float64(v)
	case float32:
		amount = float64(v)
	case decimal.Decimal:
		amount, _ = v.Float64()
	default:
		handler.logger.Error("Unsupported amount type for redis hold", zap.Any("type", v))
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": "invalid amount type"},
		}
		json.NewEncoder(w).Encode(response)
		return "", nil
	}
	holdTTL := 48 * time.Hour
	amountDec := decimal.NewFromFloat(amount)
	if err := handler.redisClient.HoldFunds(ctx, userID, txnID, amountDec, holdTTL); err != nil {
		handler.logger.Error("Failed to place hold in redis", zap.Error(err), zap.String("user", userID))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "failed to reserve funds, try again"},
		}
		json.NewEncoder(w).Encode(response)
		return "", err
	}

	return txnID, nil
}
