package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/mongo"
	"github.com/aremxyplug-be/lib/balance"
	"github.com/aremxyplug-be/lib/bank/transfer"
	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func (handler *HttpHandler) Transfer(w http.ResponseWriter, r *http.Request) {

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("failed to get user's details: %s", err.Error())}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method == "POST" {

		// first decode the request body
		info := transfer.TransferInfo{}
		if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		_, userBalance, _, err := handler.getBalance(userDetails.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("could not get user balance: %s", err.Error())}}
			json.NewEncoder(w).Encode(response)
			return
		}

		// bal, err := userBalance.Decimal()
		// if err != nil {
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("could not get user balance decimal: %s", err.Error())}}
		// 	json.NewEncoder(w).Encode(response)
		// 	return
		// }

		newBal, valid, err := handler.checkTransfer(userBalance, info.Amount)
		if !valid || err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("could not complete transfer: %s", err.Error())}}
			json.NewEncoder(w).Encode(response)
			return
		}

		// --- Create a txID (used for Redis hold) ---
		txID, err := handler.placeRedisHoldAndMeta(w, userDetails.ID, info.Amount, userBalance)
		if err != nil {
			return
		}

		amount := float64(info.Amount)

		// Save minimal meta so webhook can find user/amount without extra DB reads
		meta := map[string]interface{}{
			"user_id":    userDetails.ID,
			"amount":     amount,
			"created_at": time.Now().UTC().Format(time.RFC3339),
		}
		// non-fatal if this fails; just log
		if err := handler.redisClient.SetMeta(txID, meta, 7*24*time.Hour); err != nil {
			handler.logger.Warn("failed to set transfer meta in redis", zap.Error(err), zap.String("txID", txID))
		}

		// should create a redis job to hold balance

		info.UserID = userDetails.ID
		info.Source = "direct"
		info.TXN = txID

		resp, err := handler.bankTrf.TransferToBank(info)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		// ✅ map external provider reference to our internal transaction ID
		apiRef := resp.Reference // provider's unique reference
		holdTTL := 48 * time.Hour
		if err := handler.redisClient.SetExternalMapping(apiRef, txID, holdTTL); err != nil {
			handler.logger.Error("Failed to set external mapping in Redis", zap.Error(err))
		}

		switch resp.Status {
		case "success":
			// confirm the hold in Redis (finalize funds)
			if ok, err := handler.redisClient.ConfirmHold(userDetails.ID, txID); err != nil || !ok {
				handler.logger.Warn("Failed to confirm hold in Redis", zap.Error(err))

			}
			// persist DB balance (idempotent) — this preserves your original behaviour
			if err := handler.updateBalance(userDetails.ID, newBal); err != nil {
				if err == ErrorRedisBalanceUpdate {
					// Log and continue
					handler.logger.Warn("Balance update failed in Redis", zap.Error(err))
				}
				handler.logger.Error("Failed to update user balance", zap.Error(err))
				// keep behavior: return NotModified if persistence fails (as your original did)
				w.WriteHeader(http.StatusInternalServerError)
				response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
				json.NewEncoder(w).Encode(response)
				return
			}

			// return success (same response structure you had)
			w.WriteHeader(http.StatusOK)
			response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": resp}}
			json.NewEncoder(w).Encode(response)
			return
		case "pending":

			w.WriteHeader(http.StatusOK)
			response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": resp}}
			json.NewEncoder(w).Encode(response)
			return

		case "failed":

			// release the hold in Redis
			if ok, err := handler.redisClient.ReleaseHold(userDetails.ID, txID); err != nil || !ok {
				handler.logger.Warn("Failed to release hold in Redis", zap.Error(err))
			}
			response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": resp}}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return

		default:

			w.WriteHeader(http.StatusOK)
			response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": resp}}
			json.NewEncoder(w).Encode(response)
			return
		}

	}

	if r.Method == "GET" {
		resp, err := handler.bankTranc.GetTransferHistory(userDetails.Username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
		}

		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": resp}}
		json.NewEncoder(w).Encode(response)
	}

}

func (handler *HttpHandler) TransferRecipient(w http.ResponseWriter, r *http.Request) {
	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("failed to get user's details: %s", err.Error())}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method == "GET" {

		handler.logger.Info("Fetching user transfer recipient details", zap.String("userID", userDetails.ID))
		recipients, err := handler.store.GetTransferRecipients(userDetails.ID)
		if err != nil {
			if err == mongo.ErrNoRecipientFound {
				handler.logger.Info("No transfer recipients found for user", zap.String("userID", userDetails.ID))
				w.WriteHeader(http.StatusOK)
				response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": "No transfer recipients found"}}
				json.NewEncoder(w).Encode(response)
				return
			}
			handler.logger.Error("Failed to fetch transfer recipients", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}
		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": recipients}}
		json.NewEncoder(w).Encode(response)
	}

	if r.Method == "POST" {

		req := struct {
			Username string `json:"username,omitempty"`
			Email    string `json:"email,omitempty"`
			Phone    string `json:"phone"`
			FullName string `json:"fullname,omitempty"`
		}{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.store.SaveTransferRecipient(userDetails.ID, req.Username, req.Email, req.Phone, req.FullName); err != nil {
			handler.logger.Error("Failed to save transfer recipient", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": "failed to save transfer recipient"}}
			json.NewEncoder(w).Encode(response)
		}

		handler.logger.Info("Transfer recipient saved successfully")
		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": "Transfer recipient saved successfully"}}
		json.NewEncoder(w).Encode(response)
	}

	if r.Method == "DELETE" {

		req := struct {
			Email string `json:"email"`
		}{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.store.DeleteTransferRecipient(userDetails.ID, req.Email); err != nil {
			if err == mongo.ErrNoRecipientFound {
				handler.logger.Info("No transfer recipient found for deletion", zap.String("email", req.Email))
				w.WriteHeader(http.StatusNotFound)
				response := responseFormat.CustomResponse{Status: http.StatusNotFound, Message: "error", Data: map[string]interface{}{"data": "No transfer recipient found"}}
				json.NewEncoder(w).Encode(response)
				return
			}
			handler.logger.Error("Failed to delete transfer recipient", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}
		handler.logger.Info("Transfer recipient deleted successfully", zap.String("email", req.Email))
		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": "Transfer recipient deleted successfully"}}
		json.NewEncoder(w).Encode(response)

	}
}

func (handler *HttpHandler) TransferToAremxyPlug(w http.ResponseWriter, r *http.Request) {
	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("failed to get user's details: %s", err.Error())}}
		json.NewEncoder(w).Encode(response)
		return
	}

	info := transfer.AremxyPlugTransfer{}
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if info.Amount < 99 {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data: map[string]interface{}{
				"data": "invalid amount, amount should be greater than 100",
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	_, userBalance, _, err := handler.getBalance(userDetails.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("could not get user balance: %s", err.Error())}}
		json.NewEncoder(w).Encode(response)
		return
	}

	newBal, valid, err := handler.checkTransfer(userBalance, info.Amount)
	if !valid || err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("could not complete transfer: %s", err.Error())}}
		json.NewEncoder(w).Encode(response)
		return
	}

	txnID, err := handler.placeRedisHoldAndMeta(w, userDetails.ID, info.Amount, userBalance)
	if err != nil {
		return
	}

	holdTTL := 48 * time.Hour // tune to your needs (how long to keep a pending hold)

	info.UserID = userDetails.ID
	info.FullName = userDetails.FullName
	info.TXN = txnID

	resp, err := handler.bankTrf.TransferToAremxyPlug(info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	// ✅ map external provider reference to our internal transaction ID
	apiRef := resp.Reference // provider's unique reference
	if err := handler.redisClient.SetExternalMapping(apiRef, txnID, holdTTL); err != nil {
		handler.logger.Error("Failed to set external mapping in Redis", zap.Error(err))
	}

	switch resp.Status {
	case "success":
		// confirm the hold in Redis (finalize funds)
		if ok, err := handler.redisClient.ConfirmHold(userDetails.ID, txnID); err != nil || !ok {
			handler.logger.Warn("Failed to confirm hold in Redis", zap.Error(err))

		}
		// persist DB balance (idempotent) — this preserves your original behaviour
		if err := handler.updateBalance(userDetails.ID, newBal); err != nil {
			if err == ErrorRedisBalanceUpdate {
				// Log and continue
				handler.logger.Warn("Balance update failed in Redis", zap.Error(err))
			}
			handler.logger.Error("Failed to update user balance", zap.Error(err))
			// keep behavior: return NotModified if persistence fails (as your original did)
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		handler.logger.Info("Transfer to AremxyPlug account successful", zap.String("userID", userDetails.ID), zap.String("response", resp.Status))
		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": resp}}
		json.NewEncoder(w).Encode(response)
		return
	case "pending":
		handler.logger.Info("Transfer to AremxyPlug account pending", zap.String("userID", userDetails.ID), zap.String("response", resp.Status))
		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": resp}}
		json.NewEncoder(w).Encode(response)
		return

	case "failed":
		handler.logger.Info("Transfer to AremxyPlug account failed", zap.String("userID", userDetails.ID), zap.String("response", resp.Status))
		// release the hold in Redis
		if ok, err := handler.redisClient.ReleaseHold(userDetails.ID, txnID); err != nil || !ok {
			handler.logger.Warn("Failed to release hold in Redis", zap.Error(err))
		}
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": resp}}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return

	default:

		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": resp}}
		json.NewEncoder(w).Encode(response)
		return
	}

}

func (handler *HttpHandler) GetTransferDetails(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	resp, err := handler.bankTranc.GetTransferDetails(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	// should be call the function to get the transfer history.
	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"transfer": resp}}
	json.NewEncoder(w).Encode(response)
	// return succesfull and the transfer history

}

// Admin handler function
func (handler *HttpHandler) GetTransferHistory(w http.ResponseWriter, r *http.Request) {
	trsf, err := handler.bankTranc.GetTransferHistory("")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	// if a parameter is provided it should return just that deposit with the id.
	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"transfers": trsf}}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) GetAllBankTransactions(w http.ResponseWriter, r *http.Request) {
	// should call the fuction for loading all the  bank transactions
	transactions, err := handler.bankTranc.GetAllTransactionHistory()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"transactions": transactions}}
	json.NewEncoder(w).Encode(response)

	//  return both transfer and deposit history
}

func (handler *HttpHandler) GetDepositDetail(w http.ResponseWriter, r *http.Request) {

	id := chi.URLParam(r, "id")
	resp, err := handler.bankTranc.GetDepositDetails(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	// should be call the function to get the transfer history.
	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"deposit": resp}}
	json.NewEncoder(w).Encode(response)
	// return succesfull and the transfer history
}

func (handler *HttpHandler) GetDepositHistory(w http.ResponseWriter, r *http.Request) {
	// should call the function for loading all the deposit history

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("failed to get user's details: %s", err.Error())}}
		json.NewEncoder(w).Encode(response)
		return
	}

	dept, err := handler.bankTranc.GetDepositHistory(userDetails.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	// if a parameter is provided it should return just that deposit with the id.
	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"deposits": dept}}
	json.NewEncoder(w).Encode(response)
	// return successful and deposit history, if an error is encountered, return the error
}

func (handler *HttpHandler) GetAllDepositHistory(w http.ResponseWriter, r *http.Request) {
	dept, err := handler.bankTranc.GetDepositHistory("")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"deposits": dept}}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) GetBanks(w http.ResponseWriter, r *http.Request) {

	_, err := handler.store.GetAllBanks()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success"}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) DepositAccount(w http.ResponseWriter, r *http.Request) {
	err := handler.virtualAcc.CreateDepositAccount()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success"}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) GetBalance(w http.ResponseWriter, r *http.Request) {

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data: map[string]interface{}{
				"data": fmt.Sprintf("failed to get user's details: %s", err.Error()),
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	id := userDetails.ID

	// refresh balance from DB/external
	updatedBalance, err := handler.refreshBalance(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data: map[string]interface{}{
				"data": fmt.Sprintf("could not refresh balance: %s", err.Error()),
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	if updatedBalance {
		if err := handler.reconcileRedisWithMongo(id); err != nil {
			handler.logger.Warn("could not reconcile redis with mongo after balance refresh", zap.Error(err), zap.String("userID", id))
		}
	}

	// get actual balance
	bal, available, held, err := handler.getBalance(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": err.Error()},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	heldFloat, _ := held.Float64()

	// build response
	userBalance := struct {
		Balance          string  `json:"balance"`
		AvailableBalance string  `json:"available_balance"`
		HeldFunds        float64 `json:"held_funds"`
		UserID           string  `json:"user_id"`
	}{
		Balance:          bal.StringFixed(2),
		AvailableBalance: available.StringFixed(2),
		HeldFunds:        heldFloat,
		UserID:           id,
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": userBalance},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) reconcileRedisWithMongo(userID string) error {

	mongoBal, err := handler.store.GetBalance(userID)
	if err != nil {
		handler.logger.Error("failed to get mongo balance for reconciliation", zap.Error(err), zap.String("userID", userID))
		return err
	}

	redisBal, err := handler.redisClient.GetBalance(userID)
	if err != nil {
		handler.logger.Error("failed to get redis balance for reconciliation", zap.Error(err), zap.String("userID", userID))
		return err
	}

	if !mongoBal.Equal(redisBal) {
		// need to first add to balance if it exists in redis
		if err := handler.redisClient.AddToBalance(userID, mongoBal.Sub(redisBal)); err != nil {

		}
		// update redis to match mongo
		if err := handler.redisClient.InitializeBalance(userID, mongoBal); err != nil {
			handler.logger.Warn("failed to reconcile redis balance with mongo", zap.Error(err), zap.String("userID", userID))
			return err
		}
		handler.logger.Info("successfully reconciled redis balance with mongo", zap.String("userID", userID), zap.String("mongoBalance", mongoBal.String()), zap.String("redisBalance", redisBal.String()))
	}

	return nil

}

// Call this fucction before payments.
func (handler *HttpHandler) checkPayment(bal, payValue decimal.Decimal) (newBal float64, canPay bool, err error) {
	paymentERROR := errors.New("insufficient funds to complete payment")

	balanceValue, paymentValue := bal, payValue

	valid, err := balance.CanPay(balanceValue, paymentValue)
	if !valid || err != nil {
		return 0, false, paymentERROR
	}

	balanceAfter := balance.NewBalancePayment(balanceValue, paymentValue)

	bal_String := balanceAfter.String()
	newBalance, _ := strconv.ParseFloat(bal_String, 64)

	return newBalance, true, nil
}

func (handler *HttpHandler) checkTransfer(bal decimal.Decimal, amount float64) (newBal float64, canTrsf bool, err error) {

	transferERROR := errors.New("insufficient funds to complete transfer")

	transferAmount := decimal.NewFromFloatWithExponent(amount, -2)

	valid, err := balance.CanTransfer(bal, transferAmount)
	if !valid || err != nil {

		return 0, false, transferERROR
	}

	balanceAfter := balance.NewBalanceTransfer(bal, transferAmount)

	bal_String := balanceAfter.String()
	newBalance, _ := strconv.ParseFloat(bal_String, 64)

	return newBalance, true, nil

}

var ErrorRedisBalanceUpdate = errors.New("could not update redis balance")

func (handler *HttpHandler) updateBalance(userID string, newBalance float64) error {

	if err := handler.bankTranc.UpdateBalance(userID, newBalance); err != nil {
		return err
	}

	if err := handler.updateBalanceOnTransaction(userID, decimal.NewFromFloat(newBalance)); err != nil {
		return ErrorRedisBalanceUpdate
	}

	return nil

}

func (handler *HttpHandler) getVirtualNuban(id string) (string, error) {
	acc_details, err := handler.store.GetVirtualNuban(id)
	if err != nil {
		handler.logger.Error(err.Error())
		return "", err
	}

	return acc_details.VirtualAccountID, nil
}

func (handler *HttpHandler) refreshBalance(userID string) (bool, error) {
	virtualNubanID, err := handler.getVirtualNuban(userID)
	if err != nil {
		handler.logger.Error(err.Error())
		return false, err
	}

	updatedBalance, err := handler.bankDep.Deposit(virtualNubanID, userID)
	if err != nil {
		handler.logger.Error(err.Error())
		return false, err
	}

	return updatedBalance, nil
}

func (handler *HttpHandler) getBalance(userID string) (balance decimal.Decimal, available decimal.Decimal, held decimal.Decimal, err error) {
	// Try Redis first; only fallback on error (not on zero value).
	redisBal, err := handler.redisClient.GetBalance(userID)
	if err != nil {
		// Redis read failed — fallback to MongoDB
		mongoBal, dbErr := handler.store.GetBalance(userID)
		if dbErr != nil {
			return decimal.Decimal{}, decimal.Decimal{}, decimal.Decimal{}, dbErr
		}
		// Use decimal from Mongo value and attempt to seed Redis in background
		redisBal = mongoBal.Round(2)
		if setErr := handler.redisClient.InitializeBalance(userID, redisBal); setErr != nil {
			handler.logger.Warn("failed to seed redis initial balance", zap.Error(setErr), zap.String("userID", userID))
		}
	}

	// get held funds as decimal (best-effort)
	heldFloat, heldErr := handler.redisClient.GetHeldFundsLua(userID)
	if heldErr != nil {
		handler.logger.Warn("failed to get held funds from redis", zap.Error(heldErr), zap.String("userID", userID))
		held = decimal.Zero
	} else {
		held = decimal.NewFromFloat(heldFloat)
	}

	available = redisBal.Sub(held)

	return redisBal, available, held, nil
}

func (handler *HttpHandler) GetUserDetails(r *http.Request) (user *models.User, err error) {

	accessToken, err := r.Cookie("access_token")
	if err != nil {
		return nil, fmt.Errorf("could not get access token: %v", err)
	}

	token := accessToken.Value

	claim, err := handler.jwt.ValidateToken(token)
	if err != nil {
		return nil, fmt.Errorf("could not get user's details: %v", err)
	}

	userDetails, err := handler.store.GetUserByID(claim.ID)
	if err != nil {
		return nil, fmt.Errorf("could not get user's details: %v", err)
	}

	return userDetails, nil
}

// Call this after a successful deposit to update Redis balance
// ...existing code...
func (handler *HttpHandler) updateBalanceOnTransaction(userID string, amount decimal.Decimal) error {

	// Ensure Redis has an initialized balance for this user; if not, seed from Mongo.
	if _, err := handler.redisClient.GetBalance(userID); err != nil {
		// Redis missing or returned an error — fallback to Mongo to seed Redis
		mongoBal, dbErr := handler.store.GetBalance(userID)
		if dbErr != nil {
			handler.logger.Error("failed to read balance from mongo while seeding redis", zap.Error(dbErr), zap.String("userID", userID))
			return dbErr
		}

		seed := mongoBal.Round(2)
		if setErr := handler.redisClient.InitializeBalance(userID, seed); setErr != nil {
			// non-fatal: log and continue — AddToBalance will still create/increment the key
			handler.logger.Warn("failed to seed redis initial balance", zap.Error(setErr), zap.String("userID", userID))
		}
	}

	// Add the transaction amount to the Redis balance (best-effort)
	if err := handler.redisClient.AddToBalance(userID, amount); err != nil {
		handler.logger.Error("Failed to update Redis balance on transaction", zap.Error(err), zap.String("userID", userID))
		return err
	}

	return nil
}

// ...existing code...
