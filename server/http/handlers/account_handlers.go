package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/responseFormat"
)

func (handler *HttpHandler) VirtualAccount(w http.ResponseWriter, r *http.Request) {

	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method == "POST" {

		user := *userDetails

		hasAcc, err := handler.hasVirtualAccount(user.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if !hasAcc {
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "existing account", Data: map[string]interface{}{"data": "virtual account already created"}}
			json.NewEncoder(w).Encode(response)
			return
		}

		account, err := handler.virtualAcc.VirtualAccount(user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"error": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		response := responseFormat.CustomResponse{
			Status:  http.StatusCreated,
			Message: "success",
			Data:    map[string]interface{}{"message": "successfully created virtual account", "data": account},
		}

		json.NewEncoder(w).Encode(response)
	}

	if r.Method == "GET" {

		userID := userDetails.ID
		virtualNuban, err := handler.getVirtualAccDetails(userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}
		response := responseFormat.CustomResponse{
			Status:  200,
			Message: "success",
			Data:    map[string]interface{}{"acc_details": virtualNuban},
		}

		json.NewEncoder(w).Encode(response)
	}

}

func (handler *HttpHandler) CheckVerification(w http.ResponseWriter, r *http.Request) {
	userDetails, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"error": err.Error()},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	user := *userDetails

	if !user.HasBVN && !user.HasNIN {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "unverified",
			Data:    map[string]interface{}{"message": "User not yet verified - need BVN or NIN"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	if !user.HasVirtualNuban {

		data := map[string]interface{}{}

		data["message"] = "please generate a virtual account"

		if user.HasBVN {
			data["bvn"] = user.BVN
		}

		if user.HasNIN {
			data["nin"] = user.NIN
		}

		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "action_required",
			Data:    data,
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	data := map[string]interface{}{}

	data["message"] = "user fully verified, with virtual Nuban account"

	if user.HasBVN {
		data["bvn"] = user.BVN
	}

	if user.HasNIN {
		data["nin"] = user.NIN
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    data,
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) getVirtualAccDetails(id string) (models.AccountDetails, error) {
	acc_details, err := handler.store.GetVirtualNuban(id)
	if err != nil {
		handler.logger.Error(err.Error())
		return models.AccountDetails{}, err
	}

	return acc_details, nil
}

func (handler *HttpHandler) hasVirtualAccount(id string) (bool, error) {
	detail, err := handler.store.GetVirtualNuban(id)
	if err != nil {
		return false, err
	}

	if detail == (models.AccountDetails{}) {
		return true, nil
	}

	return false, nil
}
