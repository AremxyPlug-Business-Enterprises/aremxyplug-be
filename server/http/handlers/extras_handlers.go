package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aremxyplug-be/db/models"
	auth_pin "github.com/aremxyplug-be/lib/auth/pin"
	"github.com/aremxyplug-be/lib/responseFormat"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

	referrals, err := handler.store.GetReferredUsers(user.ID)
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

		handler.logger.Info("Points retrieved successfully", zap.Int("points", points.TotalPoints))
		w.WriteHeader(http.StatusOK)
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"point": points}}
		json.NewEncoder(w).Encode(response)
	}

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

func (handler *HttpHandler) addPoints(w http.ResponseWriter, userID string, points int, transactiontype string, originalTxnID string, source string) error {

	if err := handler.store.UpdatePointAndTransactionTime(userID, points); err != nil {
		handler.logger.Error("Failed to update points", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return err
	}

	transaction := models.PointTransaction{
		UserID:              userID,
		TransactionType:     transactiontype,
		PointEarned:         points,
		OriginalTransaction: originalTxnID,
		Source:              source,
	}

	if err := handler.point.CreatePointTransaction(transaction); err != nil {
		handler.logger.Error("Failed to create point transaction", zap.Error(err))
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
	id := user.ID

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

		pointsEarned := 100

		if err := handler.addPoints(w, id, pointsEarned, "Sign-Up Points", "", "referrals"); err != nil {
			handler.logger.Warn("failed to add points and update transaction time", zap.Error(err))
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

		err := handler.pin.VerifyPin(user.ID, updatePin.OldPin)
		if err != nil {
			if err == auth_pin.ErrIncorrectPin {
				handler.logger.Warn("Incorrect pin", zap.String("user_id", user.ID))
				writeError(w, http.StatusBadRequest, "incorrect pin")
			}
			handler.logger.Error("Failed to verify pin", zap.Error(err))
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if err := handler.pin.UpdatePin(user.ID, updatePin.NewPin); err != nil {
			handler.logger.Error("Failed to update pin", zap.Error(err))
			writeError(w, http.StatusInternalServerError, err.Error())
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
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	attemptsKey := fmt.Sprintf("pin_attempts:%s", user.ID)
	blockKey := fmt.Sprintf("pin_blocked:%s", user.ID)

	maxAttempts := 5
	blockDuration := 30 * time.Minute
	attemptsTTL := 10 * time.Minute

	// Check if blocked - FIXED: Properly check and return
	if blockedUntil, err := handler.redisClient.Get(blockKey); err == nil && blockedUntil != nil {
		msg := fmt.Sprintf("PIN blocked until %s", blockedUntil.(string))
		handler.logger.Warn(msg, zap.String("user_id", user.ID))
		writeError(w, http.StatusForbidden, msg)
		return
	}

	err = handler.pin.VerifyPin(user.ID, pin.Pin)
	if err != nil {
		if err == auth_pin.ErrIncorrectPin {
			attempts, incrErr := handler.redisClient.IncrWithTTL(attemptsKey, attemptsTTL)
			if incrErr != nil {
				handler.logger.Error("Failed to increment attempts", zap.Error(incrErr))
				writeError(w, http.StatusInternalServerError, "internal server error")
				return
			}

			if attempts >= int64(maxAttempts) {
				unblockTime := time.Now().Add(blockDuration)
				if err := handler.redisClient.SetWithTTL(blockKey, unblockTime.Format(time.RFC3339), blockDuration); err != nil {
					handler.logger.Error("Failed to set block key", zap.Error(err))
				}
				handler.redisClient.Del(attemptsKey)

				handler.logger.Warn("Too many incorrect PIN attempts - account blocked",
					zap.String("user_id", user.ID),
					zap.Time("unblock_time", unblockTime),
				)
				writeError(w, http.StatusForbidden, "account blocked due to too many incorrect attempts")
				return
			}

			handler.logger.Warn("Incorrect PIN attempt",
				zap.String("user_id", user.ID),
				zap.Int64("attempt", attempts),
				zap.Int("max_attempts", maxAttempts),
			)
			writeError(w, http.StatusBadRequest, "incorrect pin")
			return
		}

		// Handle other errors
		handler.logger.Error("PIN verification failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "pin verification error")
		return
	}

	handler.redisClient.Del(attemptsKey)

	handler.logger.Info("PIN verified successfully", zap.String("user_id", user.ID))

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"message": "PIN verification successful"},
	}
	json.NewEncoder(w).Encode(response)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	response := responseFormat.CustomResponse{
		Status:  status,
		Message: "error",
		Data:    map[string]interface{}{"data": message},
	}
	json.NewEncoder(w).Encode(response)
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
		BVN        string `json:"bvn,omitempty"`
		NIN        string `json:"nin,omitempty"`
		Dob        string `json:"dob,omitempty"`
		Address    string `json:"address,omitempty"`
		Gender     string `json:"gender,omitempty"`
		PostalCode string `json:"postal_code,omitempty"`
	}

	var req identityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.logger.Error("Failed to decode request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "invalid request body"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	to := cases.Title(language.English)
	req.Gender = to.String(req.Gender)
	req.Address = to.String(req.Address)
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

		if err := handler.store.UpdatePointAfterVerify(user.ID); err != nil {
			handler.logger.Warn("Failed to update points after BVN verification", zap.Error(err))
		}

		if err := handler.store.UpdateUserAddress(user.ID, req.Gender, req.Dob, req.Address, req.PostalCode); err != nil {
			handler.logger.Warn("Failed to update user address", zap.Error(err))
		}

		data := map[string]interface{}{
			"BVN":     req.BVN,
			"Dob":     req.Dob,
			"Address": req.Address,
			"Gender":  req.Gender,
		}

		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{
			"data":    result,
			"details": data,
		}}

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

		if err := handler.store.UpdatePointAfterVerify(user.ID); err != nil {
			handler.logger.Warn("Failed to update points after NIN verification", zap.Error(err))
		}

		if err := handler.store.UpdateUserAddress(user.ID, req.Gender, req.Dob, req.Address, req.PostalCode); err != nil {
			handler.logger.Warn("Failed to update user address", zap.Error(err))
		}

		data := map[string]interface{}{
			"NIN":     req.NIN,
			"Dob":     req.Dob,
			"Address": req.Address,
			"Gender":  req.Gender,
		}

		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{
			"data":    result,
			"details": data,
		}}
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
