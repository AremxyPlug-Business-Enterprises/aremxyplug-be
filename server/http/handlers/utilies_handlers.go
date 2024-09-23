package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// EduPins is use to carry out buying of education pins(POST) and returning all the transactions made by the user(GET)
func (handler *HttpHandler) EduPins(w http.ResponseWriter, r *http.Request) {
	/*
		userDetails, err := handler.GetUserDetails(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}
		id := userDetails.ID
	*/
	if r.Method == "POST" {
		data := models.EduInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Decoding JSON response", zap.Error(err))
			fmt.Fprintf(w, "%v", err)
			return
		}
		if data.Quantity >= 5 && data.Quantity < 10 {
			w.WriteHeader(http.StatusBadRequest)
			handler.logger.Error("invalid number of buy pins")
			fmt.Fprintf(w, "Invalid number of pins!! Pins between %d and %d are not allowed. Try again...", 5, 10)
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
		res, err := handler.eduClient.BuyEduPin(data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintf(w, "An internal error occurred while purchasing %s pin, please try again...", data.Exam_Type)
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
		res, err := handler.eduClient.QueryTransaction("id")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintln(w, "Errror occurred while getting user's records")
			return
		}

		json.NewEncoder(w).Encode(res)
	}

}

// GetEduInfo returns the details of an airtime transaction.
func (handler *HttpHandler) GetEduInfo(w http.ResponseWriter, r *http.Request) {

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

// To be used by admins to view transactions in the databases
func (handler *HttpHandler) GetEduTransactions(w http.ResponseWriter, r *http.Request) {

	resp, err := handler.eduClient.GetAllTransaction("user")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Error geeting user's transaction", zap.Error(err))
		fmt.Fprintln(w, "Error occurred while getting transactions")
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (handler *HttpHandler) TVSubscriptions(w http.ResponseWriter, r *http.Request) {
	/*
		userDetails, err := handler.GetUserDetails(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}
		id := userDetails.ID
	*/
	if r.Method == "POST" {
		data := models.TvInfo{}
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

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
			}

			newBal, valid, err := handler.checkTransfer(bal, float64(data.Amount))
			if !valid || err != nil {
				w.WriteHeader(http.StatusBadRequest)
				response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
				json.NewEncoder(w).Encode(response)
				return
			}
		*/
		res, err := handler.tvClient.BuySub(data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			// change error message
			fmt.Fprintf(w, "An internal error occurred while purchasing tv subscription, please try again...")
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
		res, err := handler.tvClient.GetUserTransactions("user")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintln(w, "Errror occurred while getting user's records")
			return
		}

		json.NewEncoder(w).Encode(res)
	}
}

func (handler *HttpHandler) GetTvSubscriptions(w http.ResponseWriter, r *http.Request) {
	resp, err := handler.tvClient.GetAllTransactions()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Error geeting user's transaction", zap.Error(err))
		fmt.Fprintln(w, "Error occurred while getting transactions")
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (handler *HttpHandler) GetTvSubDetails(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	res, err := handler.tvClient.GetTransactionDetails(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintln(w, "Error getting transaction details...")
		return
	}

	json.NewEncoder(w).Encode(res)
}

func (handler *HttpHandler) ElectricBill(w http.ResponseWriter, r *http.Request) {
	/*
		userDetails, err := handler.GetUserDetails(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}
		id := userDetails.ID
	*/
	if r.Method == "POST" {
		data := models.ElectricInfo{}
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

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
			}

			newBal, valid, err := handler.checkTransfer(bal, float64(data.Amount))
			if !valid || err != nil {
				w.WriteHeader(http.StatusBadRequest)
				response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
				json.NewEncoder(w).Encode(response)
				return
			}
		*/
		if data.Amount < 1000 {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": "amount is less than 1000"}}
			json.NewEncoder(w).Encode(response)
			return
		}
		res, err := handler.electClient.PayBill(data)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			// change error message
			fmt.Fprintf(w, "An internal error occurred while paying electricity bill, please try again...")
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
		res, err := handler.electClient.GetUserTransactions("user")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintln(w, "Errror occurred while getting user's records")
			return
		}

		json.NewEncoder(w).Encode(res)
	}
}

func (handler *HttpHandler) GetElectricBills(w http.ResponseWriter, r *http.Request) {
	resp, err := handler.electClient.GetAllTransactions()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Error geeting user's transaction", zap.Error(err))
		fmt.Fprintln(w, "Error occurred while getting transactions")
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (handler *HttpHandler) GetElectricBillDetails(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	res, err := handler.electClient.GetTransactionDetails(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintln(w, "Error getting transaction detail.")
		return
	}

	json.NewEncoder(w).Encode(res)
}
