package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/balance"
	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/go-chi/chi/v5"
)

func (handler *HttpHandler) Transfer(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {

		// first decode the request body
		info := models.TransferInfo{}
		if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}
		// verify all needed parameters

		// call the transferMoney function
		bal, valid, err := handler.checkPayment(info.Amount)
		if !valid || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}
		resp, err := handler.bankTrf.TransferToBank(info)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
			// log the error
			// return the error to the user

		}
		if err := handler.updateBalance("", bal); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}
		// if successfull return the Transfer receipt, otherwise return the error

		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": resp}}
		json.NewEncoder(w).Encode(response)
	}

	if r.Method == "GET" {
		resp, err := handler.bankTranc.GetTransferHistory("")
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
	resp, err := handler.bankTranc.GetTransferDetails(id)
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
	dept, err := handler.bankTranc.GetDepositHistory("")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	// if a parameter is provided it should return just that deposit with the id.
	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"deposits": dept}}
	json.NewEncoder(w).Encode(response)
	// return successful and deposit history, if an error is encountered, return the error
}

// There should be a function or method to call for payment.
func (handler *HttpHandler) checkPayment(amount string) (float64, bool, error) {
	// call the payment function to verify if the user has enough money to carry out payment
	bal, err := handler.bankTranc.GetBalance("")
	if err != nil {
		return 0, false, err
		// do something with the error
	}

	payValue, err := strconv.Atoi(amount)
	if err != nil {
		return 0, false, err
	}
	valid, err := balance.CanPay(bal, float64(payValue))
	if !valid || err != nil {
		return 0, false, err
	}

	newBalance := balance.NewBalancePayment(bal, float64(payValue))

	return newBalance, true, nil
	// if payment is allowed, should return true and then proceed with payment
	// if payment is not allowed, should return false and the error
}

func (handler *HttpHandler) updateBalance(name string, newBalance float64) error {
	// get the user's name at this point
	virtualNuban, err := handler.getVirtualNuban(name)
	if err != nil {
		return err
	}
	if err := handler.bankTranc.UpdateBalance(virtualNuban, newBalance); err != nil {
		return err
	}

	return nil

}

func (handler *HttpHandler) getVirtualNuban(name string) (string, error) {
	virtualNuban, err := handler.store.GetVirtualNuban(name)
	if err != nil {
		return "", err
	}

	return virtualNuban, nil
}

func (handler *HttpHandler) refreshBalance(name string) error {
	virtualNuban, err := handler.getVirtualNuban(name)
	if err != nil {
		return err
	}

	if err := handler.bankDep.Deposit(virtualNuban); err != nil {
		return err
	}

	return nil
}

/*
func (handler *HttpHandler) getUser() string {

}
*/
