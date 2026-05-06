package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/models/telcom"
	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/aremxyplug-be/lib/telcom/airtime"
	"github.com/aremxyplug-be/lib/telcom/data"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Airtime is use to carry out buying of airtime(POST) and returning all the transactions made by the user(GET)
func (handler *HttpHandler) Airtime(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Failed to get user details", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": fmt.Sprintf("failed to get user details: %s", err.Error())},
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	id := userDetails.ID
	fullName := userDetails.FullName

	if r.Method == "POST" {
		data := airtime.AirtimeInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Failed to decode airtime request", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Invalid request format"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if len(data.Phone_no) != 11 {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Invalid phone number length", zap.Int("length", len(data.Phone_no)))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Phone number must be 11 digits"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		amt, err := strconv.ParseFloat(data.Amount, 64)
		if err != nil {
			handler.logger.Error("Failed to convert amount to integer", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Invalid amount format"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		amt_decimal := decimal.NewFromFloatWithExponent(amt, -2)

		network := ""

		switch data.Network {
		case "1":
			network = "MTN"
		case "2":
			network = "AIRTEL"
		case "3":
			network = "GLO"
		case "4":
			network = "9MOBILE"
		}
		if !handler.ensureWalletUnlocked(ctx, w, id) {
			return
		}
		if !handler.ensureProductCategoryUnlocked(ctx, w, models.AirtimePurchasesEnabled, "Airtime products") {
			return
		}
		// Get product details from database
		airtimeProduct, err := handler.productClient.GetAirtimeProduct(ctx, network)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				handler.logger.Warn("No active airtime product available", zap.String("network", network))
				writeItemUnavailableResponse(w, "The selected airtime product is currently unavailable.")
				return
			}
			handler.logger.Error("Failed to get airtime product details", zap.String("network", network), zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Failed to retrieve airtime product details"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Get discount percentages from product
		customer_discount := airtimeProduct.Customer_Discount
		provider_discount := airtimeProduct.Provider_Discount

		// Calculate customer discount amount and final price customer pays
		discount_amount := amt_decimal.Mul(decimal.NewFromFloat(customer_discount / 100.0))
		discounted_amount := amt_decimal.Sub(discount_amount)

		// Store discount info for transaction
		data.Discount_percent = fmt.Sprintf("%.2f%%", customer_discount)
		data.Discount_amount = discounted_amount.StringFixed(2)

		data.Provider_Discount = fmt.Sprintf("%.2f%%", provider_discount)
		data.ProviderName = airtimeProduct.ProviderName

		// Calculate profit margin: difference between what provider gives us and what we give customer
		// Example: Airtime 100, provider offers at 97 (3% discount), we offer customer 2%, profit = 1%
		profit_margin := airtimeProduct.Profit_Margin
		profit := amt_decimal.Mul(decimal.NewFromFloat(profit_margin / 100.0)).StringFixed(2)

		_, balance, _, err := handler.getBalance(ctx, id)
		handler.logger.Info("fallback to DB for user balance", zap.String("userID", id), zap.Error(err))
		if err != nil {
			handler.logger.Error("Failed to retrieve user balance from DB", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Failed to retrieve user balance"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		handler.logger.Info("User balance", zap.String("userID", id), zap.String("balance", balance.String()))

		// Check payment validity
		newBal, valid, err := handler.checkPayment(balance, discounted_amount)
		if !valid || err != nil {
			handler.logger.Error("Payment validation failed", zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "insufficient funds"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		data.FullName = fullName
		data.UserID = id

		txnID, err := handler.placeRedisHoldAndMeta(ctx, w, id, discounted_amount, balance)
		if err != nil {
			return
		}

		data.TXN = txnID
		data.Profit_Margin = profit

		res, err := handler.vtuClient.BuyAirtime(ctx, data)
		if err != nil {
			// Release hold on error
			handler.redisClient.ReleaseHold(ctx, id, txnID)
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to purchase airtime", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "An internal error occurred while purchasing airtime, please try again."},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		switch res.Status {
		case "success":
			if ok, err := handler.redisClient.ConfirmHold(ctx, id, txnID); err != nil || !ok {
				handler.logger.Warn("Failed to confirm hold in Redis", zap.Error(err))
			}
			if err := handler.updateBalance(ctx, id, newBal); err != nil {
				if err == ErrorRedisBalanceUpdate {
					// Log and continue
					handler.logger.Warn("Airtime balance update failed in Redis", zap.Error(err))
				}
				w.WriteHeader(http.StatusInternalServerError)
				handler.logger.Error("Airtime balance update failed", zap.Error(err))
				response := responseFormat.CustomResponse{
					Status:  http.StatusInternalServerError,
					Message: "error",
					Data:    map[string]interface{}{"data": "Airtime active but balance update failed. Contact support."},
				}
				json.NewEncoder(w).Encode(response)
				return
			}
			pointsEarned := 2
			if err := handler.addPoints(ctx, w, id, pointsEarned, res.TransactionProduct, res.TransactionID, "transaction"); err != nil {
				handler.logger.Warn("failed to add points and update transaction time", zap.Error(err))
			}

			// Emit utility.payment event
			if handler.processor != nil {
				handler.addevent(ctx, id, data.Amount, res.TransactionID, "airtime.purchase")
			}

			w.WriteHeader(http.StatusOK)
			handler.logger.Info("Airtime processed successfully", zap.Any("response", res))
			response := responseFormat.CustomResponse{
				Status:  http.StatusOK,
				Message: "success",
				Data:    map[string]interface{}{"data": res},
			}
			json.NewEncoder(w).Encode(response)
			return

		case "pending":
			w.WriteHeader(http.StatusOK)
			response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": res}}
			json.NewEncoder(w).Encode(response)
			return

		case "failed":
			if ok, err := handler.redisClient.ReleaseHold(ctx, id, txnID); err != nil {
				handler.logger.Warn("Failed to release hold in Redis", zap.Error(err))
			} else if !ok {
				handler.logger.Warn("Failed to release hold in Redis: not found", zap.String("txnID", txnID))
			}
			response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": res}}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return

		default:
			w.WriteHeader(http.StatusOK)
			response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": res}}
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	if r.Method == "GET" {
		res, err := handler.vtuClient.GetUserTransaction(ctx, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to get user's airtime transactions", zap.String("user_id", id), zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Error occurred while retrieving user's records"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"transactions": res},
		}
		json.NewEncoder(w).Encode(response)
	}
}

// GetAirtimeTransactions return all the airtime transactions in the database, to be used by admin.
func (handler *HttpHandler) GetAirtimeTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp, err := handler.vtuClient.GetAllTransactions(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Failed to get all airtime transactions", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error occurred while retrieving transactions"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"transactions": resp},
	}
	json.NewEncoder(w).Encode(response)
}

// GetAirtimeInfo returns the details of an airtime transaction.
func (handler *HttpHandler) GetAirtimeInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	res, err := handler.dataClient.GetTransactionDetail(ctx, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Failed to get transaction detail", zap.String("transactionID", id), zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error getting transaction detail"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"transaction_details": res},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) TelcomRecipient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("failed to get user details: %s", err.Error())}}
		json.NewEncoder(w).Encode(response)
		return
	}
	userID := userDetails.ID

	if r.Method == "POST" {
		data := telcom.Recipient{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Failed to decode recipient request", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Invalid request format"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.vtuClient.SaveRecipient(ctx, userID, data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("failed while saving recipient", zap.Error(err))
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": "telcom recipient saved successfully"}}

		json.NewEncoder(w).Encode(response)

	}

	if r.Method == "PUT" {
		data := telcom.Recipient{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("error decoding json payload", zap.Error(err))
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.vtuClient.UpdateRecipient(ctx, userID, data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("failed while updating recipient", zap.Error(err))
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": "updated recipient successfully"}}

		json.NewEncoder(w).Encode(response)
	}

	if r.Method == "GET" {
		recipients, err := handler.vtuClient.GetRecipients(ctx, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("failed to get recipients", zap.Error(err))
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": "failed to retrieve recipients"}}
			json.NewEncoder(w).Encode(response)
			return
		}

		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"recipients": recipients}}

		json.NewEncoder(w).Encode(response)
	}

	if r.Method == "DELETE" {

		var recipient struct {
			ID int `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&recipient); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("error decoding json payload", zap.Error(err))
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.vtuClient.DeleteRecipient(ctx, recipient.ID, userID); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("failed to delete recipient", zap.Error(err))
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": "successfully deleted telcom recipient"}}

		json.NewEncoder(w).Encode(response)
	}

}

// Data send a call to the API to buy data(POST) or return users transaction history(GET)
func (handler *HttpHandler) Data(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Failed to get user details", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": fmt.Sprintf("failed to get user details: %s", err.Error())},
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	id := userDetails.ID
	fullName := userDetails.FullName

	if r.Method == "POST" {
		data := data.DataInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Failed to decode DATA request", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Invalid request format"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		if !handler.ensureWalletUnlocked(ctx, w, userDetails.ID) {
			return
		}
		if !handler.ensureProductCategoryUnlocked(ctx, w, models.DataPurchasesEnabled, "Data products") {
			return
		}

		_, userBalance, _, err := handler.getBalance(ctx, userDetails.ID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Failed to retrieve user balance", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": err.Error()},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		plan, err := handler.productClient.GetPlanByID(ctx, data.Plan)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeItemUnavailableResponse(w, "The selected data plan is currently unavailable.")
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to get plan amount", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Failed to retrieve plan amount"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		newBal, valid, err := handler.checkPayment(userBalance, decimal.NewFromFloat(plan.Amount))
		if !valid || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Payment validation failed", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": err.Error()},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		data.FullName = fullName
		data.UserID = id
		data.ProviderID = plan.ProviderID
		data.PlanID = int(plan.PlanID.Int64)
		data.Amount = fmt.Sprintf("%.f", plan.Amount)
		data.Plan_Name = plan.PlanType
		data.PlanSize = plan.Size
		data.Validity = plan.Validity
		data.Profit_Margin = decimal.NewFromFloat(plan.ProfitMargin).StringFixed(2)

		txnID, err := handler.placeRedisHoldAndMeta(ctx, w, userDetails.ID, data.Amount, userBalance)
		if err != nil {
			return
		}

		res, err := handler.dataClient.BuyData(ctx, data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to purchase data", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "An internal error occurred while purchasing data, please try again."},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		switch res.Status {
		case "success":
			// confirm the hold in Redis (finalize funds)
			if ok, err := handler.redisClient.ConfirmHold(ctx, userDetails.ID, txnID); err != nil || !ok {
				handler.logger.Warn("Failed to confirm hold in Redis", zap.Error(err))

			}

			if err := handler.updateBalance(ctx, id, newBal); err != nil {
				if err == ErrorRedisBalanceUpdate {
					// Log and continue
					handler.logger.Warn("Data balance update failed in Redis", zap.Error(err))
				}
				w.WriteHeader(http.StatusInternalServerError)
				handler.logger.Error("Failed to update user balance", zap.Error(err))
				response := responseFormat.CustomResponse{
					Status:  http.StatusInternalServerError,
					Message: "error",
					Data:    map[string]interface{}{"data": "Payment successful but server failed to modify balance"},
				}
				json.NewEncoder(w).Encode(response)
				return
			}

			pointsEarned := 2

			if err := handler.addPoints(ctx, w, id, pointsEarned, res.TransactionProduct, res.TransactionID, "transaction"); err != nil {
				handler.logger.Warn("failed to add points and update transaction time", zap.Error(err))
			}

			// Emit utility.payment event
			if handler.processor != nil {
				handler.addevent(ctx, id, data.Amount, res.TransactionID, "data.purchase")
			}

			w.WriteHeader(http.StatusOK)
			response := responseFormat.CustomResponse{
				Status:  http.StatusOK,
				Message: "success",
				Data:    map[string]interface{}{"data": res},
			}
			json.NewEncoder(w).Encode(response)
			return

		case "pending":

			w.WriteHeader(http.StatusOK)
			response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": res}}
			json.NewEncoder(w).Encode(response)
			return

		case "failed":

			// release the hold in Redis
			if ok, err := handler.redisClient.ReleaseHold(ctx, userDetails.ID, txnID); err != nil || !ok {
				handler.logger.Warn("Failed to release hold in Redis", zap.Error(err))
			}
			response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": res}}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return

		default:

			w.WriteHeader(http.StatusOK)
			response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": res}}
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	if r.Method == "GET" {
		res, err := handler.dataClient.GetUserTransactions(ctx, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to get user's data transactions", zap.String("user_id", id), zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Error occurred while retrieving user's records"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"transactions": res},
		}
		json.NewEncoder(w).Encode(response)
	}

}

// GetDataInfo checks and returns the details of a given transaction.
func (handler *HttpHandler) GetDataInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := chi.URLParam(r, "id")

	res, err := handler.dataClient.GetTransactionDetail(ctx, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Failed to get transaction detail", zap.String("transactionID", id), zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error getting transaction detail"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"transaction_details": res},
	}
	json.NewEncoder(w).Encode(response)
}

// GetTransactions returns the list of transaction carried out in the server. It is for admins to view all transactions.
func (handler *HttpHandler) GetDataTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp, err := handler.dataClient.GetAllTransactions(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Failed to get all transactions", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error occurred while retrieving transactions"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"transactions": resp},
	}
	json.NewEncoder(w).Encode(response)

}

func (handler *HttpHandler) SpectranetData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Failed to get user details", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": fmt.Sprintf("failed to get user details: %s", err.Error())},
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	id := userDetails.ID
	username := userDetails.Username

	if r.Method == "POST" {
		data := telcom.SpectranetInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Failed to decode Spectranet data request", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Invalid request format"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		if !handler.ensureWalletUnlocked(ctx, w, userDetails.ID) {
			return
		}
		if !handler.ensureProductCategoryUnlocked(ctx, w, models.DataPurchasesEnabled, "Data products") {
			return
		}

		_, userBalance, _, err := handler.getBalance(ctx, userDetails.ID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Failed to retrieve user balance", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": err.Error()},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		newBal, valid, err := handler.checkPayment(userBalance, decimal.NewFromFloatWithExponent(float64(data.Amount), -2))
		if !valid || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Payment validation failed", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": err.Error()},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		data.UserID = id
		res, err := handler.dataClient.BuySpecData(ctx, data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to purchase Spectranet data", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "An internal error occurred while purchasing data, please try again."},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.updateBalance(ctx, id, newBal); err != nil {
			if err == ErrorRedisBalanceUpdate {
				// Log and continue
				handler.logger.Warn("Balance update failed in Redis", zap.Error(err))
			}
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to update user balance", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Payment successful but server failed to modify balance"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		pointsEarned := 2

		if err := handler.addPoints(ctx, w, id, pointsEarned, res.TransactionProduct, res.TransactionID, "transaction"); err != nil {
			handler.logger.Warn("failed to add points and update transaction time", zap.Error(err))
		}

		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": res},
		}
		json.NewEncoder(w).Encode(response)
	}

	if r.Method == "GET" {
		res, err := handler.dataClient.GetSpecUserTransactions(ctx, username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to get user's Spectranet transactions", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Error occurred while retrieving user's records"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"transactions": res},
		}
		json.NewEncoder(w).Encode(response)
	}

}

func (handler *HttpHandler) GetSpecDataDetails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := chi.URLParam(r, "id")

	res, err := handler.dataClient.GetSpecTransDetails(ctx, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Failed to get Spectranet transaction details", zap.String("transactionID", id), zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error getting transaction detail"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"transaction_details": res},
	}
	json.NewEncoder(w).Encode(response)
}

// To be used by admin
func (handler *HttpHandler) GetSpectranetTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp, err := handler.dataClient.GetAllSpecTransactions(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Failed to get Spectranet transactions", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error occurred while retrieving transactions"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"transactions": resp},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) SmileData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Failed to get user details", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": fmt.Sprintf("failed to get user details: %s", err.Error())},
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	id := userDetails.ID
	username := userDetails.Username

	if r.Method == "POST" {
		data := telcom.SmileInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Failed to decode Smile data request", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Invalid request format"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		if !handler.ensureWalletUnlocked(ctx, w, userDetails.ID) {
			return
		}
		if !handler.ensureProductCategoryUnlocked(ctx, w, models.DataPurchasesEnabled, "Data products") {
			return
		}

		// bal, err := handler.getBalance(id)
		// if err != nil {
		// 	w.WriteHeader(http.StatusBadRequest)
		// 	handler.logger.Error("Failed to retrieve user balance", zap.Error(err))
		// 	response := responseFormat.CustomResponse{
		// 		Status:  http.StatusBadRequest,
		// 		Message: "error",
		// 		Data:    map[string]interface{}{"data": err.Error()},
		// 	}
		// 	json.NewEncoder(w).Encode(response)
		// 	return
		// }

		// newBal, valid, err := handler.checkPayment(bal, decimal.NewFromFloat(float64(amount)))
		// if !valid || err != nil {
		// 	w.WriteHeader(http.StatusBadRequest)
		// 	handler.logger.Error("Payment validation failed", zap.Error(err))
		// 	response := responseFormat.CustomResponse{
		// 		Status:  http.StatusBadRequest,
		// 		Message: "error",
		// 		Data:    map[string]interface{}{"data": err.Error()},
		// 	}
		// 	json.NewEncoder(w).Encode(response)
		// 	return
		// }

		data.UserID = userDetails.ID
		res, err := handler.dataClient.BuySmileData(ctx, data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to purchase Smile data", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "An internal error occurred while purchasing data, please try again."},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// if err := handler.updateBalance(id, newBal); err != nil {
		// 	w.WriteHeader(http.StatusNotModified)
		// 	handler.logger.Error("Failed to update user balance", zap.Error(err))
		// 	response := responseFormat.CustomResponse{
		// 		Status:  http.StatusNotModified,
		// 		Message: "error",
		// 		Data:    map[string]interface{}{"data": "Payment successful but server failed to modify balance"},
		// 	}
		// 	json.NewEncoder(w).Encode(response)
		// 	return
		// }

		pointsEarned := 2

		if err := handler.addPoints(ctx, w, id, pointsEarned, res.TransactionProduct, res.TransactionID, "transaction"); err != nil {
			handler.logger.Warn("failed to add points and update transaction time", zap.Error(err))
		}

		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": res},
		}
		json.NewEncoder(w).Encode(response)
	}

	if r.Method == "GET" {
		res, err := handler.dataClient.GetSmileUserTransactions(ctx, username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to get user's Smile transactions", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Error occurred while retrieving user's records"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"transactions": res},
		}
		json.NewEncoder(w).Encode(response)
	}

}

func (handler *HttpHandler) GetSmileDataDetails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	//id := r.URL.Query().Get("id")
	id := chi.URLParam(r, "id")

	res, err := handler.dataClient.GetSmileTransDetails(ctx, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintln(w, "Error getting transaction detail.")
		return
	}

	json.NewEncoder(w).Encode(res)
}

// To be used by admin
func (handler *HttpHandler) GetSmileTransactions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp, err := handler.dataClient.GetAllSmileTransactions(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Error geeting user's transaction", zap.Error(err))
		fmt.Fprintln(w, "Error occurred while getting transactions")
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (handler *HttpHandler) TelcomProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !handler.ensureProductCategoryUnlocked(ctx, w, models.DataPurchasesEnabled, "Data products") {
		return
	}

	id := chi.URLParam(r, "networkID")

	networkID, _ := strconv.Atoi(id)

	products, err := handler.productClient.GetProducts(ctx, networkID)
	if err != nil {
		handler.logger.Error("Failed to retrieve products", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Failed to retrieve products"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"products": products},
	}

	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) TelecomPlans(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method == "POST" {

		newPlan := models.Plan{}

		if err := json.NewDecoder(r.Body).Decode(&newPlan); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Invalid JSON payload", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Invalid JSON payload"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		createdPlan, err := handler.productClient.CreatePlan(ctx, newPlan)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to create plan", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Failed to create plan"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "Plan created successfully",
			Data: map[string]interface{}{
				"new_plan": newPlan,
				"plan_ID":  createdPlan,
			},
		}
		json.NewEncoder(w).Encode(response)
	}

	if r.Method == "GET" {
		if !handler.ensureProductCategoryUnlocked(ctx, w, models.DataPurchasesEnabled, "Data products") {
			return
		}
		id := chi.URLParam(r, "productID")

		productID, _ := strconv.Atoi(id)

		plans, err := handler.productClient.GetPlans(ctx, productID)
		if err != nil {
			handler.logger.Error("Failed to retrieve plans", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Failed to retrieve plans"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Return the plans as a response
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"plans": plans},
		}
		json.NewEncoder(w).Encode(response)
	}

	if r.Method == "PATCH" {

		id := chi.URLParam(r, "planID")
		planID, _ := strconv.Atoi(id)

		updatedPlan := models.PlanUpdate{}

		if err := json.NewDecoder(r.Body).Decode(&updatedPlan); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Invalid JSON payload", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Invalid JSON payload"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.productClient.UpdatePlan(ctx, planID, updatedPlan); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to update plan", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Failed to update plan"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data: map[string]interface{}{
				"updated_plan": updatedPlan,
				"message":      fmt.Sprintf("Plan with ID %d updated successfully", planID),
			},
		}
		json.NewEncoder(w).Encode(response)

	}

	if r.Method == "DELETE" {

		id := chi.URLParam(r, "planID")

		planID, err := strconv.Atoi(id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Invalid plan ID", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Invalid plan ID"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.productClient.DeletePlan(ctx, planID); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to delete plan", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Failed to delete plan"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": fmt.Sprintf("Plan with ID %d deleted successfully", planID)},
		}
		json.NewEncoder(w).Encode(response)
	}
}

func (handler *HttpHandler) AirtimeDiscount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !handler.ensureProductCategoryUnlocked(ctx, w, models.AirtimePurchasesEnabled, "Airtime products") {
		return
	}

	network := chi.URLParam(r, "network")

	network = strings.ToUpper(network)

	prod, err := handler.productClient.GetAirtimeProduct(ctx, network)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			handler.logger.Warn("No active airtime product available", zap.String("network", network))
			response := responseFormat.CustomResponse{
				Status:  http.StatusNotFound,
				Message: "error",
				Data:    map[string]interface{}{"data": "No active airtime product available for this network"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Failed to retrieve airtime product", zap.String("network", network), zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Failed to retrieve airtime product"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data: map[string]interface{}{
			"discount_percent": prod.Customer_Discount,
			"id":               prod.ID,
			"provider_id":      prod.ProviderID,
		},
	}
	json.NewEncoder(w).Encode(response)
}
