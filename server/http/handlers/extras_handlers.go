package handlers

import (
	"encoding/json"

	// "fmt"
	"net/http"
	// "net/url"

	"github.com/aremxyplug-be/db/models"
	// "github.com/aremxyplug-be/lib/referral"
	"github.com/aremxyplug-be/lib/responseFormat"
)

func (handler *HttpHandler) Referral(w http.ResponseWriter, r *http.Request) {

	// TODO: first create the referral upon signup.
	// this function should be the endpoint where the user retrieves referral information
	/*
		user, err := handler.GetUserDetails(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		referral, err := handler.referral.GetReferral(user.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		requestURL := r.URL.String()

		parsedURL, err := url.Parse(requestURL)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}
		schema := parsedURL.Scheme
		host := parsedURL.Host

		referralString := fmt.Sprintf("%s://%s/%s/%s?%s=%s", schema, host, "app", "register", "referral", referral)


		json.NewEncoder(w).Encode(referralString)
	*/
	referralString := "https://www.aremxyplug.com/app/register/referral/username"
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"referral_link": referralString}}
	json.NewEncoder(w).Encode(response)

}

func (handler *HttpHandler) Points(w http.ResponseWriter, r *http.Request) {
	// TODO: implememt logic for point balance retrival for GET requests
	/*
		user, err := handler.GetUserDetails(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}
	*/
	if r.Method == "GET" {
		/*
			points, err := handler.point.GetPoints(user.ID)
			if err != nil {

			}

			json.NewEncoder(w).Encode(points)
		*/

		dummy_points := 30
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"available_points": dummy_points}}
		json.NewEncoder(w).Encode(response)

	}
	// TODO: implememt logic for point balance usage for POST requests
	/*
		if r.Method == "POST" {
			// TODO: first check if the user can redeem point. If user can redeem point then return true and allow user to carry out transaction
			var pointsToRedeem int
			canRedeem := handler.point.RedeemPoints(user.ID, pointsToRedeem)
			if !canRedeem {
				w.WriteHeader(http.StatusBadRequest)
			}

			w.WriteHeader(http.StatusOK)

		}
	*/
}

// should write a function for redeem point...

func (handler *HttpHandler) addPoints(w http.ResponseWriter, r *http.Request) {
	// TODO: implement the point based on the required module
	// TODO: call the addPoints method after the neccessary conditions has been met
	user, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	var points int
	if err := handler.point.UpdatePoints(user.ID, points); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
}

func (handler *HttpHandler) Pin(w http.ResponseWriter, r *http.Request) {

	user, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method == "POST" {

		type newPinInput struct {
			Pin string `json:"pin"`
		}

		newPin := newPinInput{}
		if err := json.NewDecoder(r.Body).Decode(&newPin); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		pin := models.UserPin{
			UserID: user.ID,
			Pin:    newPin.Pin,
		}

		if err := handler.pin.SavePin(pin); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusCreated)
		response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{"msg": "user pin created successfully"}}
		json.NewEncoder(w).Encode(response)

	}

	if r.Method == "PATCH" {

		type userPin struct {
			Pin string `json:"pin"`
		}

		updatePin := userPin{}

		if err := json.NewDecoder(r.Body).Decode(&updatePin); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.pin.UpdatePin(user.ID, updatePin.Pin); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"msg": "user pin updated successfully"}}
		json.NewEncoder(w).Encode(response)

	}

}

func (handler *HttpHandler) VerifyPIN(w http.ResponseWriter, r *http.Request) {
	user, err := handler.GetUserDetails(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	type userPin struct {
		Pin string `json:"pin"`
	}

	pin := userPin{}

	if err := json.NewDecoder(r.Body).Decode(&pin); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	valid, err := handler.pin.VerifyPin(user.ID, pin.Pin)
	if err != nil {
		if !valid {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	if !valid {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "incorrect pin"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": "pin OK"}}
	json.NewEncoder(w).Encode(response)

}

// When payment to be used is point, the point endpoint should be called
// The point balance should be displayed at all times

// what response should be used?
