package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/responseFormat"
	"go.uber.org/zap"
)

func (handler *HttpHandler) Referral(w http.ResponseWriter, r *http.Request) {

	// TODO: first create the referral upon signup.
	// this function should be the endpoint where the user retrieves referral information
	/*
		user, err := handler.GetUserDetails(r)
		if err != nil {
			handler.logger.Error("Failed to get user details", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		referral, err := handler.referral.GetReferral(user.ID)
		if err != nil {
			handler.logger.Error("Failed to get referral", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		requestURL := r.URL.String()

		parsedURL, err := url.Parse(requestURL)
		if err != nil {
			handler.logger.Error("Failed to parse URL", zap.Error(err))
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
	handler.logger.Info("Referral link generated successfully", zap.String("referral_link", referralString))
}

func (handler *HttpHandler) Points(w http.ResponseWriter, r *http.Request) {

	// TODO: implement logic for point balance retrieval for GET requests
	/*
		user, err := handler.GetUserDetails(r)
		if err != nil {
			handler.logger.Error("Failed to get user details", zap.Error(err))
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
				handler.logger.Error("Failed to get points", zap.Error(err))
			}

			json.NewEncoder(w).Encode(points)
		*/

		dummy_points := 30
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"available_points": dummy_points}}
		json.NewEncoder(w).Encode(response)
		handler.logger.Info("Points retrieved successfully", zap.Int("available_points", dummy_points))
	}
	// TODO: implement logic for point balance usage for POST requests
	/*
		if r.Method == "POST" {
			// TODO: first check if the user can redeem point. If user can redeem point then return true and allow user to carry out transaction
			var pointsToRedeem int
			canRedeem := handler.point.RedeemPoints(user.ID, pointsToRedeem)
			if !canRedeem {
				handler.logger.Warn("User cannot redeem points", zap.Int("points_to_redeem", pointsToRedeem))
				w.WriteHeader(http.StatusBadRequest)
			}

			w.WriteHeader(http.StatusOK)
		}
	*/
}

// should write a function for redeem point...

func (handler *HttpHandler) addPoints(w http.ResponseWriter, r *http.Request) {

	// TODO: implement the point based on the required module
	// TODO: call the addPoints method after the necessary conditions have been met
	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	var points int
	if err := handler.point.UpdatePoints(user.ID, points); err != nil {
		handler.logger.Error("Failed to update points", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	handler.logger.Info("Points updated successfully", zap.Int("points", points))
}

func (handler *HttpHandler) Pin(w http.ResponseWriter, r *http.Request) {

	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
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
			handler.logger.Error("Failed to decode request body", zap.Error(err))
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
			handler.logger.Error("Failed to save pin", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusCreated)
		response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{"msg": "user pin created successfully"}}
		json.NewEncoder(w).Encode(response)
		handler.logger.Info("User pin created successfully", zap.String("user_id", user.ID))
	}

	if r.Method == "PATCH" {
		type userPin struct {
			Pin string `json:"pin"`
		}

		updatePin := userPin{}

		if err := json.NewDecoder(r.Body).Decode(&updatePin); err != nil {
			handler.logger.Error("Failed to decode request body", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.pin.UpdatePin(user.ID, updatePin.Pin); err != nil {
			handler.logger.Error("Failed to update pin", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"msg": "user pin updated successfully"}}
		json.NewEncoder(w).Encode(response)
		handler.logger.Info("User pin updated successfully", zap.String("user_id", user.ID))
	}
}

func (handler *HttpHandler) VerifyPIN(w http.ResponseWriter, r *http.Request) {

	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
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
		handler.logger.Error("Failed to decode request body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	valid, err := handler.pin.VerifyPin(user.ID, pin.Pin)
	if err != nil {
		handler.logger.Error("Failed to verify pin", zap.Error(err))
		if !valid {
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	if !valid {
		handler.logger.Warn("Incorrect pin", zap.String("user_id", user.ID))
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "incorrect pin"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": "pin OK"}}
	json.NewEncoder(w).Encode(response)
	handler.logger.Info("Pin verified successfully", zap.String("user_id", user.ID))
}

func (handler *HttpHandler) VerifyIdentity(w http.ResponseWriter, r *http.Request) {

	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	type identityRequest struct {
		BVN string `json:"bvn,omitempty"`
		NIN string `json:"nin,omitempty"`
	}

	var req identityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.logger.Error("Failed to decode request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "invalid request body"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.BVN != "" {
		result, err := handler.verifyClient.VerifyBVN(req.BVN, *user)
		if err != nil {
			handler.logger.Error("Failed to verify BVN", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if !result.NameMatched {
			handler.logger.Warn("BVN name mismatch", zap.String("bvn", req.BVN))
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "BVN name mismatch"}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if !result.Success {
			handler.logger.Warn("BVN verification failed", zap.String("bvn", req.BVN))
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "BVN verification failed"}}
			json.NewEncoder(w).Encode(response)
			return
		}

		user.BVN = req.BVN
		if err := handler.store.UpdateBVNField(*user); err != nil {
			handler.logger.Error("Failed to update BVN field", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": result}}
		json.NewEncoder(w).Encode(response)
		handler.logger.Info("BVN verified successfully", zap.String("bvn", req.BVN))
		return
	}

	if req.NIN != "" {
		result, err := handler.verifyClient.VerifyNIN(req.NIN, *user)
		if err != nil {
			handler.logger.Error("Failed to verify NIN", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if !result.NameMatched {
			handler.logger.Warn("NIN name mismatch", zap.String("bvn", req.NIN))
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "NIN name mismatch"}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if !result.Success {
			handler.logger.Warn("NIN verification failed", zap.String("bvn", req.NIN))
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "NIN verification failed"}}
			json.NewEncoder(w).Encode(response)
			return
		}

		user.NIN = req.NIN
		if err := handler.store.UpdateNINField(*user); err != nil {
			handler.logger.Error("Failed to update NIN field", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Before we send back the response, we have to update the status of the user in the database
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": result}}
		json.NewEncoder(w).Encode(response)
		handler.logger.Info("NIN verified successfully", zap.String("nin", req.NIN))
		return
	}

	w.WriteHeader(http.StatusBadRequest)
	response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "no valid identity provided"}}
	json.NewEncoder(w).Encode(response)
	handler.logger.Warn("No valid identity provided")
}
