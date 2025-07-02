package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/responseFormat"
	"go.uber.org/zap"
)

func (handler *HttpHandler) ReferralCode(w http.ResponseWriter, r *http.Request) {

	// TODO: first create the referral upon signup.
	// this function should be the endpoint where the user retrieves referral information

	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	referralString := fmt.Sprintf("%s/%s", "https://www.aremxyplug.com/app/register/referral", user.Username)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data: map[string]interface{}{
			"referral_code": user.Username,
			"referral_link": referralString,
		},
	}
	json.NewEncoder(w).Encode(response)
	handler.logger.Info("Referral link generated successfully", zap.String("referral_link", referralString))
}

func (handler *HttpHandler) Referral(w http.ResponseWriter, r *http.Request) {

	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	referrals, err := handler.store.GetReferredUsers(user.Username)
	if err != nil {
		handler.logger.Error("Failed to get referred users", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data: map[string]interface{}{
			"referrals": referrals,
		},
	}
	json.NewEncoder(w).Encode(response)

}

func (handler *HttpHandler) Points(w http.ResponseWriter, r *http.Request) {

	// TODO: implement logic for point balance retrieval for GET requests

	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method == "GET" {

		points, err := handler.point.GetPoints(user.ID)
		if err != nil {
			handler.logger.Error("Failed to get points", zap.Error(err))
		}

		json.NewEncoder(w).Encode(points)

		dummy_points := 30
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"available_points": dummy_points}}
		json.NewEncoder(w).Encode(response)
		handler.logger.Info("Points retrieved successfully", zap.Int("available_points", dummy_points))
	}
	// TODO: implement logic for point balance usage for POST requests

	if r.Method == "POST" {

		pointsToRedeem := struct {
			Point int `json:"points"`
		}{}

		if err := json.NewDecoder(r.Body).Decode(&pointsToRedeem); err != nil {

		}

		receipt, err := handler.point.RedeemPoints(user.ID, pointsToRedeem.Point)
		if err != nil {
			handler.logger.Error("Failed to redeem points", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		handler.logger.Info("Points redeemed successfully", zap.Any("receipt", receipt))
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": receipt}}
		json.NewEncoder(w).Encode(response)
	}

}

func (handler *HttpHandler) addPoints(w http.ResponseWriter, userID string, points int) error {

	if err := handler.store.UpdatePointAndTransactionTime(userID, points); err != nil {
		handler.logger.Error("Failed to update points", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return err
	}

	handler.logger.Info("User points and transaction time updated successfully", zap.Int("points", points))
	return nil
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

		if user.HasPin {
			handler.logger.Warn("User already has a pin", zap.String("user_id", user.ID))
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "user already has a pin"}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if err := handler.pin.SavePin(pin); err != nil {
			handler.logger.Error("Failed to save pin", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusCreated)
		handler.logger.Info("User pin created successfully", zap.String("user_id", user.ID))
		response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{
			"msg": "user pin created successfully",
		}}
		json.NewEncoder(w).Encode(response)

	}

	if r.Method == "PATCH" {
		type userPin struct {
			NewPin string `json:"new_pin"`
			OldPin string `json:"old_pin"`
		}

		updatePin := userPin{}

		if err := json.NewDecoder(r.Body).Decode(&updatePin); err != nil {
			handler.logger.Error("Failed to decode request body", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		valid, err := handler.pin.VerifyPin(user.ID, updatePin.OldPin)
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

		if err := handler.pin.UpdatePin(user.ID, updatePin.NewPin); err != nil {
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

func (handler *HttpHandler) ResetPin(w http.ResponseWriter, r *http.Request) {

	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	type resetPinInput struct {
		Pin string `json:"pin"`
	}

	resetPin := resetPinInput{}

	if err := json.NewDecoder(r.Body).Decode(&resetPin); err != nil {
		handler.logger.Error("Failed to decode request body", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := handler.pin.UpdatePin(user.ID, resetPin.Pin); err != nil {
		handler.logger.Error("Failed to update pin", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": "database error"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	handler.logger.Info("Pin reset successful", zap.String("user_id", user.ID))
	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"msg": "pin reset successfully"}}
	json.NewEncoder(w).Encode(response)

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

	hasBVN := req.BVN != ""
	hasNIN := req.NIN != ""

	switch {
	case !hasBVN && !hasNIN:
		handler.logger.Warn("No identity provided")
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": "No BVN or NIN provided"},
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return

	case user.HasBVN && hasBVN:
		handler.logger.Warn("BVN already verified")
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": "BVN is already verified"},
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return

	case user.HasNIN && hasNIN:
		handler.logger.Warn("NIN already verified")
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": "NIN is already verified"},
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return

	case hasBVN && len(req.BVN) != 11:
		handler.logger.Warn("Invalid BVN length")
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": "BVN must be 11 digits"},
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return

	case hasNIN && len(req.NIN) != 11:
		handler.logger.Warn("Invalid NIN length")
		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": "NIN must be 11 digits"},
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if hasBVN {
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

	if hasNIN {
		result, err := handler.verifyClient.VerifyNIN(req.NIN, *user)
		if err != nil {
			handler.logger.Error("Failed to verify NIN", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if !result.NameMatched {
			handler.logger.Warn("NIN name mismatch", zap.String("nin", req.NIN))
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "NIN name mismatch"}}
			json.NewEncoder(w).Encode(response)
			return
		}

		if !result.Success {
			handler.logger.Warn("NIN verification failed", zap.String("nin", req.NIN))
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

}

func (handler *HttpHandler) Chart(w http.ResponseWriter, r *http.Request) {

	// userDetails, err := handler.GetUserDetails(r)
	// if err != nil {
	// 	handler.logger.Error("Failed to get user details", zap.Error(err))
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
	// 	json.NewEncoder(w).Encode(response)
	// 	return
	// }

	rangeType := r.URL.Query().Get("range")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	var fromTime, toTime time.Time
	now := time.Now()
	switch rangeType {
	case "today":
		fromTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		toTime = fromTime.Add(24 * time.Hour)
	case "7d":
		fromTime = now.AddDate(0, 0, -7)
		toTime = now
	case "30d":
		fromTime = now.AddDate(0, 0, -30)
		toTime = now
	case "custom":
		var err error
		fromTime, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			http.Error(w, "Invalid 'from' date", http.StatusBadRequest)
			return
		}
		toTime, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			http.Error(w, "Invalid 'to' date", http.StatusBadRequest)
			return
		}
	default:
		fromTime = time.Time{} // all time
		toTime = now.Add(24 * time.Hour)
	}

	chartData, err := handler.store.GetChart("", rangeType, fromTime, toTime)
	if err != nil {
		handler.logger.Error("Failed to get chart data", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"data": chartData},
	}
	handler.logger.Info("Chart data retrieved successfully", zap.String("user_id", ""), zap.String("range_type", rangeType), zap.Time("from_time", fromTime), zap.Time("to_time", toTime))

	json.NewEncoder(w).Encode(response)
}
