package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/balance"
	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/go-chi/chi/v5"
)

func (handler *HttpHandler) Transfer(w http.ResponseWriter, r *http.Request) {

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method == "POST" {

		// first decode the request body
		info := models.TransferInfo{}
		if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		bal, err := handler.getBalance(userDetails.ID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		newBal, valid, err := handler.checkTransfer(bal, info.Amount)
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

		}

		if err := handler.updateBalance(userDetails.ID, newBal); err != nil {
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
		response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	dept, err := handler.bankTranc.GetDepositHistory(userDetails.Username)
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

func (handler *HttpHandler) GetAllDepositHistory(w http.ResponseWriter, r *http.Request) {
	dept, err := handler.bankTranc.GetDepositHistory("")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"deposits": dept}}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) GetBanks(w http.ResponseWriter, r *http.Request) {

	err := handler.bankTrf.ListBanks()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
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
		response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success"}
	json.NewEncoder(w).Encode(response)
}

// Call this fucction before payments.
func (handler *HttpHandler) checkPayment(bal, payValue float64) (newBal float64, canPay bool, err error) {
	paymentERROR := errors.New("insufficient funds to complete payment")

	valid, err := balance.CanPay(bal, payValue)
	if !valid || err != nil {
		return 0, false, paymentERROR
	}

	newBalance := balance.NewBalancePayment(bal, payValue)

	return newBalance, true, nil
}

func (handler *HttpHandler) checkTransfer(bal, amount float64) (newBal float64, canTrsf bool, err error) {

	transferERROR := errors.New("insufficient funds to complete transfer")

	valid, err := balance.CanTransfer(bal, amount)
	if !valid || err != nil {

		return 0, false, transferERROR
	}

	newBalance := balance.NewBalanceTransfer(bal, amount)

	return newBalance, true, nil

}

func (handler *HttpHandler) updateBalance(id string, newBalance float64) error {

	virtualNuban, err := handler.getVirtualNuban(id)
	if err != nil {
		return err
	}
	if err := handler.bankTranc.UpdateBalance(virtualNuban, newBalance); err != nil {
		return err
	}

	return nil

}

func (handler *HttpHandler) getVirtualNuban(id string) (string, error) {
	virtualNuban, err := handler.store.GetVirtualNuban(id)
	if err != nil {
		handler.logger.Error(err.Error())
		return "", err
	}

	return virtualNuban, nil
}

func (handler *HttpHandler) refreshBalance(name string) error {
	virtualNuban, err := handler.getVirtualNuban(name)
	if err != nil {
		handler.logger.Error(err.Error())
		return err
	}

	if err := handler.bankDep.Deposit(virtualNuban); err != nil {
		handler.logger.Error(err.Error())
		return err
	}

	return nil
}

func (handler *HttpHandler) getBalance(virtualNuban string) (balance float64, err error) {

	bal, err := handler.bankTranc.GetBalance(virtualNuban)
	if err != nil {
		return 0, err
	}

	return bal, nil
}

// with the username, or email, you should be able to get the full user's details
func (handler *HttpHandler) GetUserDetails(r *http.Request) (user *models.User, err error) {

	token := r.Header.Get("Authorization")

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

// update the user balance using the UserID
// get the user balance using the UserID

// create a delete user operation, delete the user and all associated virtualNuban. Save the transaction details.
