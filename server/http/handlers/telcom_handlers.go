package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/models/telcom"
	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Airtime is use to carry out buying of airtime(POST) and returning all the transactions made by the user(GET)
func (handler *HttpHandler) Airtime(w http.ResponseWriter, r *http.Request) {

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("failed to get user details: %s", err.Error())}}
		json.NewEncoder(w).Encode(response)
		return
	}
	id := userDetails.ID
	username := userDetails.Username

	if r.Method == "POST" {
		data := telcom.AirtimeInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Decoding JSON response", zap.Error(err))
			fmt.Fprintf(w, "%v", err)
			return

		}
		if len(data.Phone_no) != 11 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Phone number must be %d digits, got %d. Check the phone number and try again.", 11, len(data.Phone_no))
			return
		}

		userBalance, err := handler.getUserBalance(id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		amount, err := strconv.Atoi(data.Amount)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		balance_string := userBalance.Balance.String()
		bal, _ := strconv.ParseFloat(balance_string, 64)

		newBal, valid, err := handler.checkPayment(bal, float64(amount))
		if !valid || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		data.Username = username
		res, err := handler.vtuClient.BuyAirtime(data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintf(w, "An internal error occurred while purchasing data, please try again...\n %s\n", err)
			return
		}

		if err := handler.updateBalance(id, newBal); err != nil {
			w.WriteHeader(http.StatusNotModified)
			response := responseFormat.CustomResponse{Status: http.StatusNotModified, Message: "error", Data: map[string]interface{}{"data": "payment successful but server failed to modify balance"}}
			json.NewEncoder(w).Encode(response)
			return
		}

		json.NewEncoder(w).Encode(res)
	}

	if r.Method == "GET" {
		res, err := handler.vtuClient.GetUserTransaction(username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintln(w, "Errror occurred while getting user's records")
			return
		}

		json.NewEncoder(w).Encode(res)
	}
}

// GetAirtimeTransactions return all the airtime transactions in the database, to be used by admin.
func (handler *HttpHandler) GetAirtimeTransactions(w http.ResponseWriter, r *http.Request) {

	resp, err := handler.vtuClient.GetAllTransactions()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Error geeting user's transaction", zap.Error(err))
		fmt.Fprintln(w, "Error occurred while getting transactions")
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// GetAirtimeInfo returns the details of an airtime transaction.
func (handler *HttpHandler) GetAirtimeInfo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	res, err := handler.dataClient.GetTransactionDetail(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintln(w, "Error getting transaction detail.")
		return
	}

	json.NewEncoder(w).Encode(res)
}

func (handler *HttpHandler) TelcomRecipient(w http.ResponseWriter, r *http.Request) {

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
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("error decoding json payload", zap.Error(err))
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.vtuClient.SaveRecipient(userID, data); err != nil {
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
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("error decoding json payload", zap.Error(err))
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.vtuClient.UpdateRecipient(userID, data); err != nil {
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
		recipients, err := handler.vtuClient.GetRecipients(userID)
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
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("error decoding json payload", zap.Error(err))
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.vtuClient.DeleteRecipient(recipient.ID, userID); err != nil {
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

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("failed to get user details: %s", err.Error())}}
		json.NewEncoder(w).Encode(response)
		return
	}
	// id := userDetails.ID
	username := userDetails.Username

	if r.Method == "POST" {
		data := telcom.DataInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%v", err)
			handler.logger.Error("Decoding JSON response", zap.Error(err))
			return

		}
		/*
			bal, err := handler.getBalance(id)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
				json.NewEncoder(w).Encode(response)
				return
			}

			amount, err := strconv.Atoi(data.Amount)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
			}

			newBal, valid, err := handler.checkTransfer(bal, float64(amount))
			if !valid || err != nil {
				w.WriteHeader(http.StatusBadRequest)
				response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
				json.NewEncoder(w).Encode(response)
				return
			}
		*/
		data.Username = username
		res, err := handler.dataClient.BuyData(data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintf(w, "An internal error occurred while purchasing data, please try again...")
			return
		}
		/*
			if err := handler.updateBalance(id, newBal); err != nil {
				w.WriteHeader(http.StatusNotModified)
				response := responseFormat.CustomResponse{Status: http.StatusNotModified, Message: "error", Data: map[string]interface{}{"data": "payment successful but server failed to modify balance"}}
				json.NewEncoder(w).Encode(response)
				return
			}
		*/
		json.NewEncoder(w).Encode(res)
	}

	if r.Method == "GET" {
		res, err := handler.dataClient.GetUserTransactions(username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintln(w, "Errror occurred while getting user's records")
			return
		}

		json.NewEncoder(w).Encode(res)
	}

}

// GetDataInfo checks and returns the details of a given transaction.
func (handler *HttpHandler) GetDataInfo(w http.ResponseWriter, r *http.Request) {

	//id := r.URL.Query().Get("id")
	id := chi.URLParam(r, "id")

	res, err := handler.dataClient.GetTransactionDetail(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintln(w, "Error getting transaction detail.")
		return
	}

	json.NewEncoder(w).Encode(res)
}

// GetTransactions returns the list of transaction carried out in the server. It is for admins to view all transactions.
func (handler *HttpHandler) GetDataTransactions(w http.ResponseWriter, r *http.Request) {

	resp, err := handler.dataClient.GetAllTransactions()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintf(w, "Error getting users transactions records: %v", err)
		return
	}

	json.NewEncoder(w).Encode(resp)

}

func (handler *HttpHandler) SpectranetData(w http.ResponseWriter, r *http.Request) {

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("failed to get user details: %s", err.Error())}}
		json.NewEncoder(w).Encode(response)
		return
	}
	id := userDetails.ID
	username := userDetails.Username

	if r.Method == "POST" {
		data := telcom.SpectranetInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Decoding JSON response", zap.Error(err))
			fmt.Fprintf(w, "%v", err)
			return

		}

		userBalance, err := handler.getUserBalance(id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		balance_string := userBalance.Balance.String()
		bal, _ := strconv.ParseFloat(balance_string, 64)

		newBal, valid, err := handler.checkPayment(bal, float64(data.Amount))
		if !valid || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		res, err := handler.dataClient.BuySpecData(data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintf(w, "An internal error occurred while purchasing data, please try again...")
			return
		}

		if err := handler.updateBalance(id, newBal); err != nil {
			w.WriteHeader(http.StatusNotModified)
			response := responseFormat.CustomResponse{Status: http.StatusNotModified, Message: "error", Data: map[string]interface{}{"data": "payment successful but server failed to modify balance"}}
			json.NewEncoder(w).Encode(response)
			return
		}

		json.NewEncoder(w).Encode(res)
	}

	if r.Method == "GET" {
		res, err := handler.dataClient.GetSpecUserTransactions(username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintln(w, "Errror occurred while getting user's records")
			return
		}

		json.NewEncoder(w).Encode(res)
	}

}

func (handler *HttpHandler) GetSpecDataDetails(w http.ResponseWriter, r *http.Request) {

	//id := r.URL.Query().Get("id")
	id := chi.URLParam(r, "id")

	res, err := handler.dataClient.GetSpecTransDetails(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintln(w, "Error getting transaction detail.")
		return
	}

	json.NewEncoder(w).Encode(res)
}

// To be used by admin
func (handler *HttpHandler) GetSpectranetTransactions(w http.ResponseWriter, r *http.Request) {

	resp, err := handler.dataClient.GetAllSpecTransactions()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Error geeting user's transaction", zap.Error(err))
		fmt.Fprintln(w, "Error occurred while getting transactions")
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (handler *HttpHandler) SmileData(w http.ResponseWriter, r *http.Request) {

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("failed to get user details: %s", err.Error())}}
		json.NewEncoder(w).Encode(response)
		return
	}
	//id := userDetails.ID
	username := userDetails.Username

	if r.Method == "POST" {
		data := telcom.SmileInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Decoding JSON response", zap.Error(err))
			fmt.Fprintf(w, "%v", err)
			return

		}
		/*
			bal, err := handler.getBalance(id)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
				json.NewEncoder(w).Encode(response)
				return
			}

			amount, err := strconv.Atoi(data.Amount)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
			}

			newBal, valid, err := handler.checkTransfer(bal, float64(amount))
			if !valid || err != nil {
				w.WriteHeader(http.StatusBadRequest)
				response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
				json.NewEncoder(w).Encode(response)
				return
			}
		*/
		res, err := handler.dataClient.BuySmileData(data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintf(w, "An internal error occurred while purchasing data, please try again...")
			return
		}
		/*
			if err := handler.updateBalance(id, newBal); err != nil {
				w.WriteHeader(http.StatusNotModified)
				response := responseFormat.CustomResponse{Status: http.StatusNotModified, Message: "error", Data: map[string]interface{}{"data": "payment successful but server failed to modify balance"}}
				json.NewEncoder(w).Encode(response)
				return
			}
		*/
		json.NewEncoder(w).Encode(res)
	}

	if r.Method == "GET" {
		res, err := handler.dataClient.GetSmileUserTransactions(username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintln(w, "Errror occurred while getting user's records")
			return
		}

		json.NewEncoder(w).Encode(res)
	}

}

func (handler *HttpHandler) GetSmileDataDetails(w http.ResponseWriter, r *http.Request) {

	//id := r.URL.Query().Get("id")
	id := chi.URLParam(r, "id")

	res, err := handler.dataClient.GetSmileTransDetails(id)
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

	resp, err := handler.dataClient.GetAllSmileTransactions()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Error geeting user's transaction", zap.Error(err))
		fmt.Fprintln(w, "Error occurred while getting transactions")
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (handler *HttpHandler) TelcomProducts(w http.ResponseWriter, r *http.Request) {

	// Decode the JSON payload
	var payload struct {
		ProductID int `json:"product_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": "Invalid JSON payload"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	products, err := handler.dataClient.GetProductsByID(payload.ProductID)
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

		createdPlan, err := handler.dataClient.AddPlan(newPlan)
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
		id := r.URL.Query().Get("productID")

		productID, _ := strconv.Atoi(id)

		plans, err := handler.dataClient.GetPlans(productID)
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

	if r.Method == "PUT" {

		id := r.URL.Query().Get("planID")
		planID, _ := strconv.Atoi(id)

		updatedPlan := models.Plan{}

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

		if err := handler.dataClient.UpdatePlan(planID, updatedPlan); err != nil {
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
			Data:    map[string]interface{}{"updated_plan": updatedPlan},
		}
		json.NewEncoder(w).Encode(response)

	}

	if r.Method == "DELETE" {

		id := r.URL.Query().Get("planID")

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

		if err := handler.dataClient.DeletePlan(planID); err != nil {
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
