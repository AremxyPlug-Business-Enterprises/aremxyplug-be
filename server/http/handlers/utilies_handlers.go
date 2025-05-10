package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aremxyplug-be/db/models"
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

		res, err := handler.eduClient.BuyEduPin(data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Failed to buy EduPin", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Failed to purchase education pin"},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.updateBalance(id, newBal); err != nil {
			w.WriteHeader(http.StatusInternalServerError) // Changed from 304 to 500
			handler.logger.Error("Balance update failed after purchase", zap.Error(err))
			response := responseFormat.CustomResponse{
				Status:  http.StatusInternalServerError,
				Message: "error",
				Data:    map[string]interface{}{"data": "Purchase succeeded but balance update failed. Contact support."},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		json.NewEncoder(w).Encode(res)
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

		res, err := handler.tvClient.BuySub(data)
		if err != nil {
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
		data.Name = userDetails.Username

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
