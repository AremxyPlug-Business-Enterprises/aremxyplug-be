package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/mongo"
	"github.com/aremxyplug-be/lib/balance"
	"github.com/aremxyplug-be/lib/randomgen"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// Webhook handler: quick ack then background processing
func (handler *HttpHandler) WebhookHandler(w http.ResponseWriter, r *http.Request) {
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
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 60*time.Second)
		defer cancel()
		if err := handler.ProcessWebhook(ctx, b); err != nil {
			handler.logger.Error("Failed to process webhook", zap.Error(err))
		}
	}(body)
}

func (handler *HttpHandler) allowedIP(r *http.Request) bool {
	allowedIPs := map[string]bool{
		"18.133.55.102": true,
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

func (handler *HttpHandler) ProcessWebhookLegacy(body []byte) error {
	return handler.ProcessWebhook(context.Background(), body)
}

func (handler *HttpHandler) ProcessWebhook(ctx context.Context, body []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	// parse payload
	var p webhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		handler.logger.Error("failed parse webhook JSON", zap.Error(err))
		return err
	}

	switch p.Type {
	case "transfer.initiated", "transfer.success", "transfer.failed":
		// existing transfer processing
		return handler.processTransfer(ctx, p)

	case "payment.settled":
		// new deposit processing
		return handler.processDeposit(ctx, p)

	default:
		handler.logger.Warn("unhandled webhook type", zap.String("type", p.Type))
	}

	return nil
}

func (handler *HttpHandler) processTransfer(ctx context.Context, p webhookPayload) error {
	if ctx == nil {
		ctx = context.Background()
	}

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

	// extract sessionId and failureReason
	var sessionID, failureReason string
	if s, ok := p.Attributes["sessionId"].(string); ok {
		sessionID = s
	}
	if fr, ok := p.Attributes["failureReason"].(string); ok {
		failureReason = fr
	}

	// event type and reversed detection
	eventType := p.Type
	isReversed := strings.Contains(strings.ToLower(eventType), "transfer.reversed")
	failed := strings.Contains(strings.ToLower(eventType), "transfer.failed")
	success := strings.Contains(strings.ToLower(eventType), "transfer.successful")

	// Fast Redis lookup: transfer:ext:{apiRef} -> txID
	var txID string
	val, err := handler.redisClient.Get(ctx, fmt.Sprintf("transfer:ext:%s", apiRef))
	if err != nil {
		handler.logger.Warn("redis lookup error for ext mapping", zap.Error(err), zap.String("apiRef", apiRef))
	}
	if val != nil {
		switch v := val.(type) {
		case string:
			txID = v
		case []byte:
			txID = string(v)
		default:
			// attempt to marshal -> string
			if b, merr := json.Marshal(v); merr == nil {
				txID = string(b)
			}
		}
	}

	// fallback DB lookup by external ref
	if txID == "" {
		rec, derr := handler.store.GetReceiptByExternalRef(ctx, apiRef)
		if derr != nil {
			// unknown external ref — already acked upstream; log and stop
			handler.logger.Warn("unknown externalRef in webhook", zap.String("apiRef", apiRef))
			return nil
		}
		txID = rec.TXN
	}

	// load receipt by txID
	rec, err := handler.store.GetReceiptByTxID(ctx, txID)
	if err != nil {
		handler.logger.Error("store get receipt failed", zap.Error(err), zap.String("txID", txID))
		return err
	}

	// idempotency: if already final, skip
	statusLower := strings.ToLower(rec.Status)
	dbStatus := statusLower

	shouldProcess := false
	if dbStatus == "pending" {
		shouldProcess = true
	} else if dbStatus == "failed" && (success || isReversed) {
		shouldProcess = true
	} else if dbStatus == "success" && (isReversed || failed) {
		shouldProcess = true
	}

	if !shouldProcess {
		handler.logger.Info("receipt already final, skipping",
			zap.String("txID", txID),
			zap.String("dbStatus", dbStatus),
			zap.String("incoming", eventType),
		)
		return nil
	}

	// load metadata: try redis transfer:meta:{txID}
	var meta struct {
		UserID string `json:"user_id"`
		Amount int64  `json:"amount"`
	}
	found, err := handler.redisClient.GetMeta(ctx, txID, &meta) // pass pointer
	if err != nil {
		// Redis error: log and fall back to receipt values
		handler.logger.Warn("failed to read transfer meta from redis, falling back to receipt", zap.String("txID", txID), zap.Error(err))
		meta.UserID = rec.UserID
		if rec.Amount != "" {
			if amt, perr := strconv.ParseInt(rec.Amount, 10, 64); perr == nil {
				meta.Amount = amt
			} else {
				handler.logger.Warn("failed to parse receipt amount, using 0", zap.String("amount", rec.Amount), zap.Error(perr))
				meta.Amount = 0
			}
		}
	} else if !found {
		// Not found: fall back to receipt values
		handler.logger.Warn("transfer meta not found in redis, falling back to receipt", zap.String("txID", txID))
		meta.UserID = rec.UserID
		if rec.Amount != "" {
			if amt, perr := strconv.ParseInt(rec.Amount, 10, 64); perr == nil {
				meta.Amount = amt
			} else {
				handler.logger.Warn("failed to parse receipt amount, using 0", zap.String("amount", rec.Amount), zap.Error(perr))
				meta.Amount = 0
			}
		}
	} else {
		// meta loaded from redis successfully; keep it. If some fields missing, fall back to receipt minimally.
		if meta.UserID == "" {
			meta.UserID = rec.UserID
		}
		if meta.Amount == 0 && rec.Amount != "" {
			if amt, perr := strconv.ParseInt(rec.Amount, 10, 64); perr == nil {
				meta.Amount = amt
			}
		}
	}

	// Branch: failure OR reversed -> release hold and mark failed/reversed
	if failureReason != "" || isReversed {
		if meta.UserID != "" {
			if released, err := handler.redisClient.ReleaseHold(ctx, meta.UserID, txID); err != nil {
				handler.logger.Error("release hold failed", zap.Error(err), zap.String("txID", txID))
			} else if released {
				handler.logger.Info("hold released", zap.String("txID", txID))
			} else {
				handler.logger.Info("no hold to release", zap.String("txID", txID))
			}
		}

		finalStatus := "failed"
		if isReversed {
			finalStatus = "reversed"
		}

		if err := handler.store.UpdateReceiptFinal(ctx, txID, finalStatus, sessionID); err != nil {
			handler.logger.Error("failed update receipt final", zap.Error(err), zap.String("txID", txID))
			return err
		}

		handler.logger.Info("processed failed/reversed webhook",
			zap.String("txID", txID),
			zap.String("final_status", finalStatus),
			zap.String("reason", failureReason),
			zap.Bool("reversed", isReversed),
		)
		return nil
	}

	// Success path: confirm hold, persist balance snapshot to DB, mark success
	if meta.UserID != "" {
		if confirmed, err := handler.redisClient.ConfirmHold(ctx, meta.UserID, txID); err != nil {
			handler.logger.Error("confirm hold failed", zap.Error(err), zap.String("txID", txID))
		} else if confirmed {
			handler.logger.Info("hold confirmed", zap.String("txID", txID))
		} else {
			handler.logger.Info("no hold present to confirm", zap.String("txID", txID))
		}

		currentBal, _, _, err := handler.getBalance(ctx, meta.UserID)
		if err != nil {
			handler.logger.Error("failed to get current balance for user", zap.Error(err), zap.String("userID", meta.UserID))
		}

		if err := handler.store.UpdateUserBalanceFromRedis(ctx, meta.UserID, currentBal); err != nil {
			handler.logger.Error("failed to persist user balance to DB", zap.Error(err), zap.String("userID", meta.UserID))
		}
	}

	// final update to receipt = success
	if err := handler.store.UpdateReceiptFinal(ctx, txID, "success", sessionID); err != nil {
		handler.logger.Error("failed to update receipt final", zap.Error(err), zap.String("txID", txID))
		return err
	}

	handler.logger.Info("processed success webhook", zap.String("txID", txID))
	return nil
}

func (handler *HttpHandler) processDeposit(ctx context.Context, p webhookPayload) error {
	if ctx == nil {
		ctx = context.Background()
	}

	var paymentID string
	var virtualNubanID string
	var narration string
	var amount float64
	var created_At string
	accountNumber := ""
	accountName := ""
	bankName := ""
	if rel, ok := p.Attributes["payment"]; ok {
		if m, ok := rel.(map[string]interface{}); ok {
			if id, ok := m["paymentId"].(string); ok {
				paymentID = id
			}
			if nar, ok := m["narration"].(string); ok {
				narration = nar
			}
			if amt, ok := m["amount"].(float64); ok {
				amount = amt
			}
			if crt_at, ok := m["createdAt"].(string); ok {
				created_At = crt_at
			}
			if vn, ok := m["virtualNuban"].(map[string]interface{}); ok {
				if id, ok := vn["accountId"].(string); ok {
					virtualNubanID = id
				} else if acctNo, ok := vn["accountNumber"].(string); ok {
					virtualNubanID = acctNo
				}
			}
			if cp, ok := m["counterParty"].(map[string]interface{}); ok {
				if id, ok := cp["accountNumber"].(string); ok {
					accountNumber = id
				}
				if id, ok := cp["accountName"].(string); ok {
					accountName = id
				}
				if bank, ok := cp["bank"].(map[string]interface{}); ok {
					if name, ok := bank["name"].(string); ok {
						bankName = name
					}
				}
			}
		}
	}

	// get the userID from the virtualNubanID
	userID, err := handler.store.GetUserFromVirtualNuban(ctx, virtualNubanID)
	if err != nil {
		handler.logger.Error("Deposit failed: unable to get user ID from virtual Nuban", zap.Error(err))
		return err
	}

	orderID, err := randomgen.GenerateOrderID()
	if err != nil {
		handler.logger.Error("Deposit failed: unable to generate order ID", zap.Error(err))
		return err
	}

	transactionID := randomgen.GenerateTransactionID("dep")
	deposit := struct {
		VirtualNuban string `json:"virtualNuban" bson:"virtualNuban"`
		ID           string `json:"id" bson:"ID"`
	}{
		VirtualNuban: virtualNubanID,
		ID:           paymentID,
	}

	if err := handler.store.SaveDepositID(ctx, deposit); err != nil {
		if err == mongo.ErrDepositIDExist {
			handler.logger.Info("Deposit ID already exists, skipping", zap.String("depositID", paymentID))
			return nil
		}
		handler.logger.Error("Deposit failed: unable to save deposit ID", zap.Error(err))
		return err
	}

	bal, err := handler.store.GetBalance(ctx, userID)
	if err != nil {
		handler.logger.Error("Deposit failed: unable to fetch balance", zap.Error(err))
		return err
	}
	handler.logger.Debug("Fetched Balance", zap.Any("balance", bal))

	if handler.bankDep == nil {
		return fmt.Errorf("deposit configuration is not initialized")
	}
	breakdown, err := handler.bankDep.CalculateDepositBreakdown(ctx, amount)
	if err != nil {
		handler.logger.Error("Deposit failed: unable to calculate deposit breakdown", zap.Error(err))
		return err
	}

	newBalance, depositAmount := balance.NewBalanceDeposit(bal, breakdown.NetAmountCredited)
	parsedBalance, _ := primitive.ParseDecimal128(newBalance.String())
	handler.logger.Debug("New Balance Calculated", zap.String("newBalance", newBalance.String()))

	userBalance := models.Balance{
		VirtualNuban: virtualNubanID,
		Balance:      parsedBalance,
		UserID:       userID,
		UpdatedAt:    time.Now().UTC(),
	}
	if err := handler.store.SaveBalance(ctx, userID, userBalance); err != nil {
		handler.logger.Error("Deposit failed: unable to save user balance", zap.Error(err))
		return err
	}

	createdAt, err := time.Parse(time.RFC3339, created_At)
	if err != nil {
		handler.logger.Warn("Deposit: unable to parse createdAt, using current time", zap.String("createdAt", created_At), zap.Error(err))
		createdAt = time.Now().UTC()
	}

	result := models.DepositResponse{
		UserID:                 userID,
		Status:                 "success",
		Amount:                 depositAmount.StringFixed(2),
		WalletType:             "Nigerian NGN Wallet",
		Bank_Name:              bankName,
		Account_Name:           accountName,
		Account_No:             accountNumber,
		TransactionProduct:     "Virtual Account",
		TransactionDescription: "NGN Wallet Top Up",
		Message:                narration,
		Order_ID:               orderID,
		Transaction_ID:         transactionID,
		CreatedAt:              createdAt,
		Reference:              paymentID,
		APICharge:              breakdown.APICharge.StringFixed(2),
		ServiceCharge:          breakdown.ServiceCharge.StringFixed(2),
		GrossAmount:            breakdown.GrossAmount.StringFixed(2),
		ServiceChargeCap:       breakdown.ServiceChargeCap.StringFixed(2),
		ServiceChargeApplied:   breakdown.ServiceChargeApplied.StringFixed(2),
		NetAmountCredited:      breakdown.NetAmountCredited.StringFixed(2),
		ChargeWasCapped:        breakdown.ChargeWasCapped,
	}

	if err := handler.store.SaveDeposit(ctx, result); err != nil {
		handler.logger.Error("Deposit failed: unable to save transaction", zap.Error(err))
		return err
	}
	handler.logger.Info("Deposit transaction saved successfully", zap.Any("transaction", result))
	return nil
}
