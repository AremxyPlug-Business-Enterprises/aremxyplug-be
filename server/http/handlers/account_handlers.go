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
		type requestPayload struct {
			Bvn string `json:"bvn"`
		}

		data := requestPayload{}
		user := *userDetails

		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": "invalid JSON request"}}
			json.NewEncoder(w).Encode(response)
			return
		}
		// get the user;s other infomation at this point and then associate the BVN field to this point

		user.BVN = data.Bvn
		_, err := handler.virtualAcc.VirtualAccount(user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.store.UpdateBVNField(user); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		response := responseFormat.CustomResponse{
			Status:  http.StatusCreated,
			Message: "success",
			Data:    map[string]interface{}{"data": "successfully created virtual account"},
		}

		// update with the appropriate method for creating a virtual number

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

func (handler *HttpHandler) getVirtualAccDetails(id string) (models.AccountDetails, error) {
	acc_details, err := handler.store.GetVirtualNuban(id)
	if err != nil {
		handler.logger.Error(err.Error())
		return models.AccountDetails{}, err
	}

	return acc_details, nil
}
