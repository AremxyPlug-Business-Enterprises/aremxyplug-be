package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/bills/tvsub"
	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// EduPins is use to carry out buying of education pins(POST) and returning all the transactions made by the user(GET)
func (handler *HttpHandler) EduPins(w http.ResponseWriter, r *http.Request) {

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	id := userDetails.ID

	if r.Method == "POST" {
		data := models.EduInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusBadRequest) // Changed from 500 to 400
			handler.logger.Error("Failed to decode EduPins request", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Invalid request format"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if data.Quantity >= 5 && data.Quantity < 10 {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Invalid pin quantity", zap.Int("quantity", data.Quantity))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Quantity between 5-10 not allowed"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		bal, err := handler.getBalance(id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError) // Assume DB error
			handler.logger.Error("Failed to get balance", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Could not retrieve balance"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		amount, err := strconv.Atoi(data.Amount)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Invalid amount format", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Invalid amount value"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		newBal, valid, err := handler.checkPayment(bal, decimal.NewFromFloatWithExponent(float64(amount), -2))
		if !valid || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Payment validation failed", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Insufficient balance or invalid payment"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		data.UserID = id
		data.Name = userDetails.FullName
		// res, err := handler.eduClient.BuyEduPin(data)
		// if err != nil {
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	handler.logger.Error("Failed to buy EduPin", zap.Error(err))
		// 	response := responseFormat.CustomResponse{
		// 		Status:  http.StatusInternalServerError,
		// 		Message: "error",
		// 		Data:    map[string]interface{}{"data": "Failed to purchase education pin"},
		// 	}
		// 	json.NewEncoder(w).Encode(response)
		// 	return
		// }

		receiptAmount, _ := strconv.ParseFloat(data.Amount, 64)
		reference := "ID93717490080"
		transactionID := "AP-EDU-UMTIQ"
		status := "success"
		txnDesc := ""
		orderID := 7888346081
		pinGenerated := []string{
			"430402339547<=>NRCP10455329",
		}

		result := &models.EduResponse{
			UserID:                 data.UserID,
			Amount:                 receiptAmount,
			Exam_Type:              data.Exam_Type,
			Quantity:               data.Quantity,
			PhoneNumber:            data.Phone_Number,
			ReferenceNumber:        reference,
			FullName:               data.Name,
			Email:                  data.Email,
			TransactionProduct:     data.Exam_Type,
			Status:                 status,
			TransactionDescription: txnDesc,
			OrderID:                orderID,
			Pin_Generated:          pinGenerated,
			CreatedAt:              time.Now().UTC(),
			TransactionID:          transactionID,
		}

		if err := handler.updateBalance(id, newBal); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Balance update failed after purchase", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Purchase succeeded but balance update failed. Contact support."},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		pointsEarned := 2

		if err := handler.addPoints(w, id, pointsEarned, result.TransactionProduct, result.TransactionID, "transaction"); err != nil {
			handler.logger.Warn("failed to add points and update transaction time", zap.Error(err))
		}

		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data: map[string]interface{}{
				"data": result,
			},
		}
		json.NewEncoder(w).Encode(response)
	}

	if r.Method == "GET" {
		res, err := handler.eduClient.QueryTransaction("id")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Error occurred while getting user's records"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": res},
		}
		json.NewEncoder(w).Encode(response)
	}

}

// GetEduInfo returns the details of an airtime transaction.
func (handler *HttpHandler) GetEduInfo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	res, err := handler.dataClient.GetTransactionDetail(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Failed to get education transaction detail", zap.String("id", id), zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error occurred while retrieving transaction details"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": res},
	}
	json.NewEncoder(w).Encode(response)
}

// To be used by admins to view transactions in the databases
func (handler *HttpHandler) GetEduTransactions(w http.ResponseWriter, r *http.Request) {

	resp, err := handler.eduClient.GetAllTransaction("user")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Error getting user's transactions", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error occurred while retrieving transactions"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": resp},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) TVSubscriptions(w http.ResponseWriter, r *http.Request) {

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	id := userDetails.ID

	if r.Method == "POST" {
		data := models.TvInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Failed to decode TV request", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Invalid request format"},
			}
			json.NewEncoder(w).Encode(response)
			return

		}
		data.Name = userDetails.Username

		if data.Amount == 0 {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Invalid TV subscription amount", zap.Int("amount", data.Amount))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Amount must be greater than zero"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		bal, err := handler.getBalance(id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError) // Assume DB error
			handler.logger.Error("Failed to get balance", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Could not retrieve balance"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		newBal, valid, err := handler.checkPayment(bal, decimal.NewFromFloat(float64(data.Amount)))
		if !valid || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("TV payment validation failed", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Insufficient balance or invalid payment"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		data.UserID = id
		res, err := handler.tvClient.BuySub(data)
		if err != nil {
			if err == tvsub.ErrInvalidCardNumber {
				handler.logger.Error("Invalid card number", zap.String("card_number", data.SmartCard_Number))
				w.WriteHeader(http.StatusBadRequest)
				response := responseFormat.CustomResponse{
					Status:  http.StatusBadRequest,
					Message: "error",
					Data:    map[string]interface{}{"data": "Invalid card number"},
				}
				json.NewEncoder(w).Encode(response)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to buy TV subscription", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Failed to process TV subscription"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.updateBalance(id, newBal); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("TV balance update failed", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Subscription active but balance update failed. Contact support."},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		pointsEarned := 2

		if err := handler.addPoints(w, id, pointsEarned, res.TransactionProduct, res.TransactionID, "transaction"); err != nil {
			handler.logger.Warn("failed to add points and update transaction time", zap.Error(err))
		}

		w.WriteHeader(http.StatusOK)
		handler.logger.Info("TV subscription processed successfully", zap.Any("response", res))
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": res},
		}
		json.NewEncoder(w).Encode(response)
	}

	if r.Method == "GET" {
		res, err := handler.tvClient.GetUserTransactions("user")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to get user's TV transactions", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Error occurred while retrieving user's TV transactions"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": res},
		}
		json.NewEncoder(w).Encode(response)
	}
}

func (handler *HttpHandler) GetTvSubscriptions(w http.ResponseWriter, r *http.Request) {
	resp, err := handler.tvClient.GetAllTransactions()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Error getting user's TV transactions", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error occurred while retrieving TV transactions"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": resp},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) GetTvSubDetails(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	res, err := handler.tvClient.GetTransactionDetails(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Failed to get TV subscription details", zap.String("id", id), zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error occurred while retrieving transaction details"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": res},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) ElectricBill(w http.ResponseWriter, r *http.Request) {

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	id := userDetails.ID

	if r.Method == "POST" {
		data := models.ElectricInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Failed to decode ElectricBill request", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Invalid request format"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}
		data.FullName = userDetails.Username

		bal, err := handler.getBalance(id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to get balance", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Could not retrieve balance"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		newBal, valid, err := handler.checkPayment(bal, decimal.NewFromFloat(float64(data.Amount)))
		if !valid || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Electricity payment validation failed", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Insufficient balance or invalid payment"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if data.Amount < 1000 {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Invalid electric bill amount", zap.Int("amount", data.Amount))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Amount must be at least 1000"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		data.UserID = userDetails.ID
		res, err := handler.electClient.PayBill(data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to buy Electricity subscription", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Failed to process Electricity subscription"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.updateBalance(id, newBal); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Electricity balance update failed", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"error": "Subscription active but balance update failed. Contact support."},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		pointsEarned := 2

		if err := handler.addPoints(w, id, pointsEarned, res.TransactionProduct, res.TransactionID, "transaction"); err != nil {
			handler.logger.Warn("failed to add points and update transaction time", zap.Error(err))
		}

		w.WriteHeader(http.StatusOK)
		handler.logger.Info("Electricity subscription processed successfully", zap.Any("response", res))
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": res},
		}
		json.NewEncoder(w).Encode(response)
	}

	if r.Method == "GET" {
		res, err := handler.electClient.GetUserTransactions(userDetails.Username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to get user's electric bill transactions", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Error occurred while retrieving user's electric bill transactions"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": res},
		}
		json.NewEncoder(w).Encode(response)
	}
}

func (handler *HttpHandler) GetElectricBills(w http.ResponseWriter, r *http.Request) {
	resp, err := handler.electClient.GetAllTransactions()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Error getting user's electric bill transactions", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error occurred while retrieving electric bill transactions"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": resp},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) GetElectricBillDetails(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	res, err := handler.electClient.GetTransactionDetails(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Failed to get electric bill transaction details", zap.String("id", id), zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "Error occurred while retrieving transaction details"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": res},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) TvSubHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the product type from the URL path
	product := chi.URLParam(r, "product")

	// Validate the product type
	validProducts := map[string]bool{
		"dstv":      true,
		"gotv":      true,
		"showmax":   true,
		"startimes": true,
	}

	if !validProducts[product] {
		w.WriteHeader(http.StatusBadRequest)
		handler.logger.Error("Invalid product type")
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": fmt.Sprintf("Invalid product type: %s", product)},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Handle different HTTP methods
	switch r.Method {
	case "POST":
		handler.handleCreateTVSub(w, r, product)
	case "GET":
		handler.handleGetTVSubs(w, product)
	case "PATCH":
		handler.handleUpdateTVSub(w, r, product)
	case "DELETE":
		handler.handleDeleteTVSub(w, r, product)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		response := responseFormat.CustomResponse{
			Status:  http.StatusMethodNotAllowed,
			Message: "error",
			Data:    map[string]interface{}{"data": "Method not allowed"},
		}
		json.NewEncoder(w).Encode(response)
	}
}

func (handler *HttpHandler) handleCreateTVSub(w http.ResponseWriter, r *http.Request, product string) {
	data := models.TVSub{}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handler.logger.Error("Decoding JSON response", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": fmt.Sprintf("Failed to decode request body: %s", err.Error())},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	if data.Package == "" || data.PackageName == "" {
		w.WriteHeader(http.StatusBadRequest)
		handler.logger.Error("Missing required fields")
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": "Package and package name are required"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	if data.Amount <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		handler.logger.Error("Invalid amount")
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": "Amount must be greater than zero"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	ID, err := handler.productClient.CreateTVSub(product, data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Error creating TV subscription", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": fmt.Sprintf("Failed to create TV subscription: %s", err.Error())},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusCreated)
	handler.logger.Info("TV subscription created successfully", zap.Int("ID", ID))
	response := responseFormat.CustomResponse{
		Status:  http.StatusCreated,
		Message: "success",
		Data:    map[string]interface{}{"data": fmt.Sprintf("TV subscription created successfully with ID %d", ID)},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) handleGetTVSubs(w http.ResponseWriter, product string) {
	res, err := handler.productClient.GetTVSubs(product)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Error fetching TV subscriptions", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": fmt.Sprintf("Failed to fetch TV subscriptions: %s", err.Error())},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": res},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) handleUpdateTVSub(w http.ResponseWriter, r *http.Request, product string) {
	id := chi.URLParam(r, "id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		handler.logger.Error("Missing ID parameter")
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": "ID parameter is required"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	subID, _ := strconv.Atoi(id)
	if subID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		handler.logger.Error("Invalid ID parameter")
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": "ID must be a positive integer"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	data := models.TVSubUpdate{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handler.logger.Error("Decoding JSON response", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": fmt.Sprintf("Failed to decode request body: %s", err.Error())},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	err := handler.productClient.UpdateTVSub(product, subID, data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Error updating TV subscription", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": fmt.Sprintf("Failed to update TV subscription: %s", err.Error())},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": fmt.Sprintf("TV subscription with ID %d updated successfully", subID)},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) handleDeleteTVSub(w http.ResponseWriter, r *http.Request, product string) {
	id := chi.URLParam(r, "id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		handler.logger.Error("Missing ID parameter")
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": "ID parameter is required"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	subID, _ := strconv.Atoi(id)
	if subID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		handler.logger.Error("Invalid ID parameter")
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": "ID must be a positive integer"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	err := handler.productClient.DeleteTVSub(product, subID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Error deleting TV subscription", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": fmt.Sprintf("Failed to delete TV subscription: %s", err.Error())},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": fmt.Sprintf("TV subscription with ID %d deleted successfully", subID)},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) VerifyBill(w http.ResponseWriter, r *http.Request) {
	data := struct {
		DiscoType        string `json:"disco_type"`
		Meter_No         string `json:"meter_no"`
		Meter_Type       string `json:"meter_type"`
		DecoderType      string `json:"decoder_type"`
		SmartCard_Number string `json:"iuc_number"`
	}{}

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		handler.logger.Error("Decoding JSON response", zap.Error(err))
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": fmt.Sprintf("Failed to decode request body: %s", err.Error())},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Ensure only one of disco_type or decoder_type is provided
	if data.DiscoType != "" && data.DecoderType != "" {
		handler.logger.Error("both disco_type and decoder_type provided")
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"error": "provide either 'disco_type' or 'decoder_type', not both"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Handle Electricity (DiscoType) case
	if data.DiscoType != "" {
		// Validate required fields
		if data.Meter_No == "" || data.Meter_Type == "" {
			handler.logger.Error("missing meter details for disco_type")
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"error": "meter_no and meter_type are required for disco_type"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Call your electricity verification function here
		result, err := handler.electClient.VerifyMeterNo(data.DiscoType, data.Meter_No, data.Meter_Type)
		if err != nil {
			handler.logger.Error("electricity bill verification failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"error": "electricity verification failed: " + err.Error()},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Success response
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": result},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Handle DecoderType case
	if data.DecoderType != "" {
		// Validate required fields
		if data.SmartCard_Number == "" {
			handler.logger.Error("missing smart card number for decoder_type")
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"error": "iuc_number is required for decoder_type"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Call your decoder verification function here
		result, err := handler.tvClient.VerifyCard(data.DecoderType, data.SmartCard_Number)
		if err != nil {
			handler.logger.Error("decoder verification failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"error": "decoder verification failed: " + err.Error()},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Success response
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": result},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// If neither type is provided
	handler.logger.Error("missing required type")
	w.WriteHeader(http.StatusBadRequest)
	response := responseFormat.CustomResponse{
		Status:  http.StatusBadRequest,
		Message: "error",
		Data:    map[string]interface{}{"error": "either 'disco_type' or 'decoder_type' must be provided"},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) EduProduct(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "PATCH":
		data := struct {
			ID     int    `json:"id"`
			Amount string `json:"amount"`
		}{}

		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Failed to decode EduProduct request", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Invalid request format"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if data.ID == 0 || data.Amount == "" {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Invalid EduProduct data", zap.Any("data", data))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "Name and price are required, and price must be greater than zero"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		record := models.EduRecord{
			Amount: data.Amount,
		}

		err := handler.productClient.UpdateRecord(data.ID, record)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to update EduProduct", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Failed to update education product"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": fmt.Sprintf("Education product updated successfully with ID %d", data.ID)},
		}
		json.NewEncoder(w).Encode(response)

	case "GET":
		id := chi.URLParam(r, "id")

		productID, err := strconv.Atoi(id)
		if err != nil || productID <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Invalid ID parameter", zap.String("id", id))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "ID must be a positive integer"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		res, err := handler.productClient.GetRecord(productID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to fetch EduProduct by ID", zap.Int("productID", productID), zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Failed to fetch education product by ID"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": res},
		}
		json.NewEncoder(w).Encode(response)
		return

	case "DELETE":
		id := chi.URLParam(r, "id")
		if id == "" {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Missing ID parameter")
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "ID parameter is required"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		productID, err := strconv.Atoi(id)
		if err != nil || productID <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("Invalid ID parameter", zap.String("id", id))
			response := responseFormat.CustomResponse{
				Status:  http.StatusBadRequest,
				Message: "error",
				Data:    map[string]interface{}{"data": "ID must be a positive integer"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		err = handler.productClient.DeleteRecord(productID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to delete EduProduct", zap.Int("productID", productID), zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Failed to delete education product"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{
			Status:  http.StatusOK,
			Message: "success",
			Data:    map[string]interface{}{"data": fmt.Sprintf("Education product with ID %d deleted successfully", productID)},
		}
		json.NewEncoder(w).Encode(response)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		response := responseFormat.CustomResponse{
			Status:  http.StatusMethodNotAllowed,
			Message: "error",
			Data:    map[string]interface{}{"data": "Method not allowed"},
		}
		json.NewEncoder(w).Encode(response)
	}
}
