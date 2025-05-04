package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/balance"
	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
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
		info := models.TransferInfo{}
		if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		userBalance, err := handler.getUserBalance(userDetails.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("could not get user balance: %s", err.Error())}}
			json.NewEncoder(w).Encode(response)
			return
		}

		bal, err := userBalance.Decimal()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("could not get user balance decimal: %s", err.Error())}}
			json.NewEncoder(w).Encode(response)
			return
		}

		newBal, valid, err := handler.checkTransfer(bal, info.Amount)
		if !valid || err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("could not complete transfer: %s", err.Error())}}
			json.NewEncoder(w).Encode(response)
			return
		}

		resp, err := handler.bankTrf.TransferToBank(info)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return

		}

		if err := handler.updateBalance(userDetails.ID, newBal); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
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

	err := handler.bankTrf.ListBanks()

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
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("failed to get user's details: %s", err.Error())}}
		json.NewEncoder(w).Encode(response)
		return
	}

	id := userDetails.ID

	if err := handler.refreshBalance(id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("could not refresh balance: %s", err.Error())}}
		json.NewEncoder(w).Encode(response)
		return
	}

	bal, err := handler.getUserBalance(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	userBal, err := bal.Decimal()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": fmt.Sprintf("failed to get user's balance: %s", err.Error())}}
		json.NewEncoder(w).Encode(response)
		return
	}

	userBalance := struct {
		Balance string `json:"balance"`
		UserID  string `json:"user_id"`
	}{
		Balance: userBal.String(),
		UserID:  bal.UserID,
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": userBalance}}
	json.NewEncoder(w).Encode(response)
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

func (handler *HttpHandler) updateBalance(userID string, newBalance float64) error {

	if err := handler.bankTranc.UpdateBalance(userID, newBalance); err != nil {
		return err
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

func (handler *HttpHandler) refreshBalance(userID string) error {
	virtualNubanID, err := handler.getVirtualNuban(userID)
	if err != nil {
		handler.logger.Error(err.Error())
		return err
	}

	if err := handler.bankDep.Deposit(virtualNubanID, userID); err != nil {
		handler.logger.Error(err.Error())
		return err
	}

	return nil
}

func (handler *HttpHandler) getBalance(userID string) (balance decimal.Decimal, err error) {

	bal, err := handler.bankTranc.GetBalance(userID)
	if err != nil {
		return decimal.Decimal{}, err
	}

	return bal, nil
}

func (handler *HttpHandler) getUserBalance(userID string) (models.Balance, error) {
	bal, err := handler.store.GetBalanceDetails(userID)
	if err != nil {
		return models.Balance{}, err
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
