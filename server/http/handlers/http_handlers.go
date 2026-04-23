package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	//"strconv"
	"time"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/errorvalues"
	"github.com/aremxyplug-be/lib/events"
	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/aremxyplug-be/lib/smsclient/termii"
	"github.com/aremxyplug-be/types/dto"
	"github.com/go-chi/render"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

func calculateDefaultDuration(defaultDuration, configDuration time.Duration) time.Duration {
	duration := defaultDuration
	if configDuration > 0 {
		duration = configDuration * time.Minute
	}
	return duration
}

func splitAllowlistEmails(input string) []string {
	return strings.FieldsFunc(input, func(r rune) bool {
		switch r {
		case ',', ';', '\n', '\r', '\t', ' ':
			return true
		default:
			return false
		}
	})
}

func (handler *HttpHandler) initLoginAllowlist() {
	rawList := strings.TrimSpace(handler.secrets.LoginAllowedEmails)
	rawFile := strings.TrimSpace(handler.secrets.LoginAllowedFile)

	if rawList == "" && rawFile == "" {
		return
	}

	allowlist := map[string]struct{}{}

	if rawList != "" {
		for _, email := range splitAllowlistEmails(rawList) {
			email = strings.ToLower(strings.TrimSpace(email))
			if email == "" {
				continue
			}
			allowlist[email] = struct{}{}
		}
	}

	if rawFile != "" {
		b, err := os.ReadFile(rawFile)
		if err != nil {
			handler.loginAllowlistErr = err
			return
		}
		for _, email := range splitAllowlistEmails(string(b)) {
			email = strings.ToLower(strings.TrimSpace(email))
			if email == "" {
				continue
			}
			allowlist[email] = struct{}{}
		}
	}

	if len(allowlist) == 0 {
		handler.loginAllowlistErr = errors.New("login allowlist is configured but empty")
		return
	}

	handler.loginAllowlist = allowlist
}

func (handler *HttpHandler) isLoginEmailAllowed(email string) (bool, error) {
	handler.loginAllowlistOnce.Do(handler.initLoginAllowlist)
	if handler.loginAllowlistErr != nil {
		return false, handler.loginAllowlistErr
	}
	if handler.loginAllowlist == nil {
		return true, nil
	}
	email = strings.ToLower(strings.TrimSpace(email))
	_, ok := handler.loginAllowlist[email]
	return ok, nil
}

// SignUp is the api used to create a single user
func (handler *HttpHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var user models.User
	//defer cancel()
	ctx := r.Context()

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		w.WriteHeader(http.StatusBadRequest)

		//response := responseFormat.RespondWithError(w, http.StatusBadRequest, err.Error())
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	// use the validator library to validate required fields
	if validationErr := validate.Struct(&user); validationErr != nil {
		w.WriteHeader(http.StatusBadRequest)

		response := responseFormat.CustomResponse{
			Status:  http.StatusBadRequest,
			Message: "error",
			Data:    map[string]interface{}{"data": validationErr.Error()},
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	timestamp := handler.timeHelper.Now().UTC()
	id, err := handler.otp.GenerateID(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	userId := strconv.Itoa(id)

	hashedPassword, err := handler.encrypt.GenerateFromPassword(user.Password)
	if err != nil {
		handler.logger.Error("fail to generate password", zap.Error(err))
		return
	}

	to := cases.Title(language.English)
	full_name := to.String(user.FullName)
	username := to.String(user.Username)
	email := strings.TrimSpace(cases.Lower(language.English).String(user.Email))
	rawInvitationCode := strings.TrimSpace(user.InvitationCode)
	normalizedUsername := strings.TrimSpace(username)
	isSelfReferral := rawInvitationCode != "" && strings.EqualFold(rawInvitationCode, normalizedUsername)
	inviteCode := ""
	if rawInvitationCode != "" && !isSelfReferral {
		inviteCode = to.String(rawInvitationCode)
	}

	validUser, field, err := handler.isValidNewUser(ctx, user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if !validUser {
		w.WriteHeader(http.StatusConflict)
		response := responseFormat.CustomResponse{Status: http.StatusConflict, Message: "sign-up failed", Data: map[string]interface{}{"data": field}}
		json.NewEncoder(w).Encode(response)
		return
	}

	newUser := models.User{
		ID:              userId,
		FullName:        full_name,
		Email:           email,
		Username:        username,
		Password:        string(hashedPassword),
		PhoneNumber:     user.PhoneNumber,
		Country:         user.Country,
		InvitationCode:  inviteCode,
		CreatedAt:       timestamp,
		UpdatedAt:       timestamp,
		HasBVN:          false,
		HasNIN:          false,
		HasVirtualNuban: false,
		IsVerified:      false,
		ReferralCount:   0,
		LastTransaction: timestamp,
		HasPin:          false,
		Beta:            false,
	}

	err = handler.store.SaveUser(ctx, newUser)
	if err != nil {
		handler.logger.Error("error saving user", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	userResponse := dto.UserResponse{
		ID:       newUser.ID,
		FullName: newUser.FullName,
		Email:    newUser.Email,
		Username: newUser.Username,
		Phone:    newUser.PhoneNumber,
	}

	w.WriteHeader(http.StatusCreated)
	response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{"data": userResponse}}
	json.NewEncoder(w).Encode(response)
}

// Login is the api used to login a single user
func (handler *HttpHandler) Login(w http.ResponseWriter, r *http.Request) {
	var userlogin dto.LoginInput
	ctx := r.Context()
	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&userlogin); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	to := cases.Title(language.English)
	username := to.String(userlogin.Username)
	email := cases.Lower(language.English).String(userlogin.Email)

	user, err := handler.store.GetUserByUsernameOrEmail(ctx, email, username)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		response := responseFormat.CustomResponse{Status: http.StatusNotFound, Message: "user not found", Data: map[string]interface{}{"data": "user not found"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	allowed, allowErr := handler.isLoginEmailAllowed(user.Email)
	if allowErr != nil {
		handler.logger.Error("login allowlist error", zap.Error(allowErr))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "login temporarily unavailable"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	if !allowed {
		w.WriteHeader(http.StatusForbidden)
		response := responseFormat.CustomResponse{
			Status:  http.StatusForbidden,
			Message: "login failed",
			Data:    map[string]interface{}{"data": "email not allowed"},
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	hashedPassword := user.Password

	attemptsKey := fmt.Sprintf("password_attempts:%s", user.ID)
	blockedKey := fmt.Sprintf("password_blocked:%s", user.ID)

	maxAttempts := 5
	blockDuration := 1 * time.Hour
	attemptsTTL := 10 * time.Minute

	// Check if user is already blocked
	if blockedUntil, err := handler.redisClient.Get(ctx, blockedKey); err == nil && blockedUntil != nil {
		msg := fmt.Sprintf("PIN blocked until %s", blockedUntil.(string))
		handler.logger.Warn(msg, zap.String("user_id", user.ID))
		writeError(w, http.StatusForbidden, msg)
		return
	}

	ok := handler.encrypt.ComparePasscode(userlogin.Password, hashedPassword)
	if !ok {
		// Increment failed login attempts in Redis with a TTL of 15 minutes
		attempts, incrErr := handler.redisClient.IncrWithTTL(ctx, attemptsKey, attemptsTTL)
		if incrErr != nil {
			handler.logger.Error("Failed to increment attempts", zap.Error(incrErr))
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		if attempts >= int64(maxAttempts) {
			unblockTime := time.Now().Add(blockDuration)
			if err := handler.redisClient.SetWithTTL(ctx, blockedKey, unblockTime.Format(time.RFC3339), blockDuration); err != nil {
				handler.logger.Error("Failed to set block key", zap.Error(err))
			}
			handler.redisClient.Del(ctx, attemptsKey)

			handler.logger.Warn("user temporarily blocked due to too many failed login attempts", zap.String("userID", user.ID))
			handler.logger.Warn("Too many incorrect paswsword attempts - account blocked",
				zap.String("user_id", user.ID),
				zap.Time("unblock_time", unblockTime),
			)
			writeError(w, http.StatusForbidden, "account blocked due to too many incorrect paasword attempts")
			return
		}

		handler.logger.Warn("Incorrect PIN attempt",
			zap.String("user_id", user.ID),
			zap.Int64("attempt", attempts),
			zap.Int("max_attempts", maxAttempts),
		)

		handler.logger.Error("store validating password")
		w.WriteHeader(http.StatusUnauthorized)
		response := responseFormat.CustomResponse{Status: http.StatusUnauthorized, Message: "error", Data: map[string]interface{}{"data": "password incorrect"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	// On successful login, reset the login attempts counter
	if err := handler.redisClient.Del(ctx, attemptsKey); err != nil {
		handler.logger.Warn("failed to reset login attempts after successful login", zap.Error(err))
	}

	if !ensureSignupVerified(w, user) {
		handler.logger.Warn("blocked login for unverified user", zap.String("user_id", user.ID))

		return
	}

	refreshTokenClaims := dto.Claims{
		PersonId: user.ID,
	}

	claims := dto.Claims{
		PersonId: user.ID,
	}

	jwtToken, err := handler.jwt.GenerateTokenWithExpiration(claims, handler.authTokenDuration)
	if err != nil {
		handler.logger.Error("fail to generate token", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	userResponse := dto.UserResponse{
		ID:       user.ID,
		FullName: user.FullName,
		Email:    user.Email,
		Username: user.Username,
		Phone:    user.PhoneNumber,
	}

	refreshToken, err := handler.jwt.GenerateTokenWithExpiration(refreshTokenClaims, handler.refreshTokenDuration)
	if err != nil {
		handler.logger.Error("fail to generate refresh token", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	hasPin := user.HasPin

	// Store refresh token session in Redis (session model)
	sessionKey := fmt.Sprintf("session:%s", user.ID)
	handler.redisClient.SetWithTTL(ctx, sessionKey, refreshToken, handler.refreshTokenDuration)

	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    jwtToken,
		MaxAge:   900,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteNoneMode,
		Domain:   "aremxyplug.com",
	}

	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		MaxAge:   1800,
		HttpOnly: true,
		Secure:   true,
		Path:     "/api/v1/refresh-token",
		SameSite: http.SameSiteNoneMode,
		Domain:   "aremxyplug.com",
	}

	http.SetCookie(w, accessCookie)
	http.SetCookie(w, refreshCookie)

	if !hasPin {
		handler.logger.Warn("pin not yet set", zap.Any("userID", user.ID))
		w.WriteHeader(http.StatusAccepted)
		response := responseFormat.CustomResponse{Status: http.StatusAccepted, Message: "success", Data: map[string]interface{}{
			"msg":      "user's pin not set",
			"customer": userResponse,
		}}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"customer": userResponse}}
	json.NewEncoder(w).Encode(response)

}

// ForgotPassword
func (handler *HttpHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var userlogin dto.LoginInput
	ctx := r.Context()

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&userlogin); err != nil {
		respondWithError(w, http.StatusBadRequest, "error", err)
		return
	}

	// Checking if the user exists (replace with your actual user lookup logic)
	user, err := handler.store.GetUserByEmail(ctx, userlogin.Email)
	if err != nil || user == nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Sorry, this user does not exist"})
		return
	}

	claims := dto.Claims{
		PersonId: user.ID,
	}

	token, err := handler.jwt.GenerateToken(claims)
	if err != nil {
		handler.logger.Error("fail to generate token", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "error", err)
		return
	}

	storeKey := fmt.Sprintf("pwdreset:%s", token)
	ttl := 15 * time.Minute
	if err := handler.redisClient.SetWithTTL(ctx, storeKey, "arm", ttl); err != nil {
		handler.logger.Error("failed to persist reset token", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "error", err)
		return
	}

	//var uri string
	uri := "/api/v1/reset-password?token="
	Scheme := "https"
	host := "test.aremxyplug.com"
	link := fmt.Sprintf("%s://%s%s%s", Scheme, host, uri, token)
	fmt.Println(link)

	// Creating Message
	message := models.Message{
		ID:         handler.idGenerator.Generate(),
		CustomerID: user.ID,
		Target:     user.Email,
		Type:       "email",
		Title:      "Password Reset",
		Body:       link,
		TemplateID: PasswordResetAlias,
		DataMap:    map[string]string{},
		Ts:         handler.timeHelper.Now().Unix(),
	}
	message.DataMap["FullName"] = user.FullName
	message.DataMap["Email"] = user.Email
	message.DataMap["Link"] = link

	// send message
	fmt.Println("about send email")
	err = handler.emailClient.Send(&message)
	if err != nil {
		handler.logger.Error("error sending password reset email", zap.String("target", user.Email), zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "error", err)
		return
	}
	handler.logger.Info("password reset email sent", zap.String("target", user.Email))
	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"msg": "email sent successfully"}}
	json.NewEncoder(w).Encode(response)

}

func (handler *HttpHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {

	token := r.URL.Query().Get("token")

	storeKey := fmt.Sprintf("pwdreset:%s", token)
	ctx := r.Context()

	// lookup token in redis
	val, err := handler.redisClient.Get(ctx, storeKey)
	if err != nil || val == nil {
		respondWithError(w, http.StatusBadRequest, "error", errors.New("link either invalid or expired, request for a new link"))
		return
	}

	userID, ok := val.(string)
	if !ok || userID != "arm" {
		// unexpected value type; delete key and error
		_ = handler.redisClient.Del(ctx, storeKey)
		respondWithError(w, http.StatusBadRequest, "error", errors.New("invalid reset token"))
		return
	}

	// delete key to enforce one-time use (best effort)
	if err := handler.redisClient.Del(ctx, storeKey); err != nil {
		// log but continue
		handler.logger.Warn("failed to delete reset token from redis", zap.Error(err), zap.String("key", storeKey))
	}

	claims, err := handler.jwt.ValidateToken(token)
	if err != nil {
		handler.logger.Error("failed to validate token", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": "link either invalid or expired, request for a new link"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	type NewPassword struct {
		Password string `json:"password"`
	}

	newPassword := NewPassword{}
	json.NewDecoder(r.Body).Decode(&newPassword)

	hashedPassword, err := handler.encrypt.GenerateFromPassword(newPassword.Password)
	if err != nil {
		handler.logger.Error("error hashing password", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": "something unexpected occured, please try again"}}
		json.NewEncoder(w).Encode(response)
		return
	}
	newPassword.Password = string(hashedPassword)

	err = handler.store.UpdateUserPasswordByID(ctx, claims.ID, newPassword.Password)
	if err != nil {
		handler.logger.Error("failed to update password", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusCreated)
	response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{"data": "Password updated successfully"}}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {

	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	ctx := r.Context()

	payload := struct {
		NewEmail string `json:"new_email"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		handler.logger.Error("error decoding request body", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "error", errors.New("error decoding request payload"))
		return
	}

	if payload.NewEmail == user.Email {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "new email cannot be the same as the current email"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	user.Email = payload.NewEmail
	if err := handler.sendOTP(ctx, user, "Email Change", changeEmail); err != nil {
		handler.logger.Error("failed to send OTP", zap.Error(err))
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
			"message": "an email with otp has been sent, input to update your email",
		},
	}
	json.NewEncoder(w).Encode(response)

}

func (handler *HttpHandler) UpdateEmail(w http.ResponseWriter, r *http.Request) {

	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	ctx := r.Context()

	payload := struct {
		New_Email string `json:"new_email"`
		OTP       string `json:"otp"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		handler.logger.Error("error decoding request body", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "error", errors.New("error decoding request payload"))
		return
	}

	valid, err := handler.otp.ValidateOTP(ctx, payload.OTP, otpChannelEmail, payload.New_Email)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if !valid {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("otp verification failed at validation")
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "otp verification failed"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := handler.store.UpdateEmail(ctx, user.ID, payload.New_Email); err != nil {
		handler.logger.Error("failed to update phone number", zap.Error(err))
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
			"message": "email successfully updated",
			"email":   payload.New_Email,
		},
	}
	json.NewEncoder(w).Encode(response)

}

func (handler *HttpHandler) ChangePhoneNumber(w http.ResponseWriter, r *http.Request) {

	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	ctx := r.Context()

	payload := struct {
		New_Phone string `json:"new_phone"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		handler.logger.Error("error decoding request body", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "error", errors.New("error decoding request payload"))
		return
	}

	if payload.New_Phone == user.PhoneNumber {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "new phone number cannot be the same as the current phone number"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := handler.smsClient.SendSMS(ctx, payload.New_Phone); err != nil {
		if err == termii.ErrSMSFailed {
			handler.logger.Error("SMS sending failed", zap.String("phone", payload.New_Phone), zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "failed to send OTP, please try again"}}
			json.NewEncoder(w).Encode(response)
			return
		}
		handler.logger.Error("Failed to send OTP", zap.String("phone", payload.New_Phone), zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "failed to send otp", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data: map[string]interface{}{
			"message": "phone change otp sent",
		},
	}
	json.NewEncoder(w).Encode(response)

}

func (handler *HttpHandler) UpdatePhoneNumber(w http.ResponseWriter, r *http.Request) {

	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	ctx := r.Context()

	payload := struct {
		New_Phone string `json:"new_phone"`
		OTP       string `json:"otp"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		handler.logger.Error("error decoding request body", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "error", errors.New("error decoding request payload"))
		return
	}

	err = handler.smsClient.VerifyToken(ctx, payload.OTP, payload.New_Phone)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error", err)
		return
	}

	if err := handler.store.UpdatePhone(ctx, user.ID, payload.New_Phone); err != nil {
		handler.logger.Error("failed to update phone number", zap.Error(err))
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
			"message": "phone number successfully updated",
			"phone":   payload.New_Phone,
		},
	}
	json.NewEncoder(w).Encode(response)

}

func (handler *HttpHandler) ChangePhoneNumberWhatsApp(w http.ResponseWriter, r *http.Request) {
	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	ctx := r.Context()

	payload := struct {
		NewPhone string `json:"new_phone"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		handler.logger.Error("error decoding request body", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "error", errors.New("error decoding request payload"))
		return
	}

	if payload.NewPhone == user.PhoneNumber {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "new phone number cannot be the same as the current phone number"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := handler.sendWhatsAppOTP(ctx, payload.NewPhone); err != nil {
		handler.logger.Error("Failed to send WhatsApp OTP", zap.String("phone", payload.NewPhone), zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "failed to send whatsapp otp", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data: map[string]interface{}{
			"message": "phone change whatsapp otp sent",
		},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) UpdatePhoneNumberWhatsApp(w http.ResponseWriter, r *http.Request) {
	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	ctx := r.Context()

	payload := struct {
		NewPhone string `json:"new_phone"`
		OTP      string `json:"otp"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		handler.logger.Error("error decoding request body", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "error", errors.New("error decoding request payload"))
		return
	}

	if err := handler.validateWhatsAppOTP(ctx, payload.OTP, payload.NewPhone); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error", err)
		return
	}

	if err := handler.store.UpdatePhone(ctx, user.ID, payload.NewPhone); err != nil {
		handler.logger.Error("failed to update phone number", zap.Error(err))
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
			"message": "phone number successfully updated",
			"phone":   payload.NewPhone,
		},
	}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {

	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	ctx := r.Context()

	payload := struct {
		Old_password string `json:"old_password"`
		New_password string `json:"new_password"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "error", err)
		return
	}
	hashedPassword := user.Password

	ok := handler.encrypt.ComparePasscode(payload.Old_password, hashedPassword)
	if !ok {
		handler.logger.Error("incorrect password")
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "incorrect password"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	newHashedPassword, err := handler.encrypt.GenerateFromPassword(payload.New_password)
	if err != nil {
		handler.logger.Error("fail to generate password", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "error generating hash", err)
		return
	}

	if err := handler.store.UpdateUserPassword(ctx, user.Email, string(newHashedPassword)); err != nil {
		handler.logger.Error("error updating the user's password")
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"data": "password change successful"}}
	json.NewEncoder(w).Encode(response)

}

func (handler *HttpHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var userLogin dto.LoginInput

	// Decode and validate the request body
	if err := json.NewDecoder(r.Body).Decode(&userLogin); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	ctx := r.Context()

	// Retrieve user by email
	user, err := handler.store.GetUserByEmail(ctx, userLogin.Email)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found", nil)
		return
	}

	// Determine the action based on the URL path
	action := getLastPathSegment(r.URL.Path)
	switch action {
	case "signup":
		if err := handler.sendOTP(ctx, user, "Sign-Up Verification", verifyEmailAlias); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error sending verification OTP", err)
			return
		}
		respondWithSuccess(w, http.StatusOK, "success", "Verification email sent successfully")

	case "signin":
		if err := handler.sendOTP(ctx, user, "Sign-in Verification", signInVerification); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error sending sign-in OTP", err)
			return
		}
		respondWithSuccess(w, http.StatusOK, "success", "Sign-in email sent successfully")

	case "resetpassword":
		if err := handler.sendOTP(ctx, user, "Password OTP", PasswordOTPAlias); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error sending password reset OTP", err)
			return
		}
		respondWithSuccess(w, http.StatusCreated, "success", "Password reset email sent successfully")

	case "resetpin":
		if err := handler.sendOTP(ctx, user, "PIN Reset", resetPinAlias); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error sending PIN reset OTP", err)
			return
		}
		respondWithSuccess(w, http.StatusOK, "success", "PIN reset OTP sent successfully")

	default:
		http.NotFound(w, r)
	}
}

/*
func (handler *HttpHandler) SendOTPWIthTermii(w http.ResponseWriter, r *http.Request) {

	var userLogin dto.LoginInput

	// Decode and validate the request body
	if err := json.NewDecoder(r.Body).Decode(&userLogin); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Retrieve user by email
	user, err := handler.store.GetUserByEmail(ctx, userLogin.Email)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found", nil)
		return
	}

	otp, err := handler.otp.GenerateOTP(user.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error generating otp", err)
		return
	}

	message := &models.Message{
		Body:   otp,
		Target: user.Email,
	}

	if err := handler.emailClient.SendWithTermii(message); err != nil {
		handler.logger.Error("error sending OTP with Termii", zap.String("target", user.Email), zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "error sending OTP with Termii", err)
		return
	}

	respondWithSuccess(w, http.StatusOK, "success", "OTP sent successfully")

}
*/

func (handler *HttpHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	type otp struct {
		OTP string `json:"otp"`
	}
	Otp := otp{}
	ctx := r.Context()

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&Otp); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	email := r.URL.Query().Get("email")
	valid, err := handler.otp.ValidateOTP(ctx, Otp.OTP, otpChannelEmail, email)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if !valid {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("otp verification failed at validation")
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "otp verification failed"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	action := getLastPathSegment(r.URL.Path)
	switch action {
	case "signin":
		data := map[string]interface{}{"data": email}
		respondWithSuccess(w, http.StatusOK, "otp verification successful", data)
	case "signup":
		user, err := handler.store.VerifyUser(ctx, email)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "error", err)
			return
		}

		ev := &events.Event{
			UserID:    user.ID,
			Type:      "signup.completed",
			TS:        time.Now().UTC(),
			Published: false,
		}

		if err := handler.processor.ProcessEvent(ctx, ev); err != nil {
			handler.logger.Error("error processing signup completed event", zap.String("user_id", user.ID), zap.Error(err))
		}

		err = handler.sendOTP(ctx, user, "verify-email", welcomeMessage)
		if err != nil {
			handler.logger.Error("error sending email verification otp", zap.String("target", user.Email), zap.Error(err))
			respondWithError(w, http.StatusInternalServerError, "error", err)
			return
		}

		data := map[string]interface{}{"email": email}
		respondWithSuccess(w, http.StatusOK, "otp verification successful", data)
	case "resetpassword":
		user, err := handler.store.GetUserByEmail(ctx, email)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "error", err)
			return
		}

		claims := dto.Claims{
			PersonId: user.ID,
		}

		jwtToken, err := handler.jwt.GenerateTokenWithExpiration(claims, handler.authTokenDuration)
		if err != nil {
			handler.logger.Error("fail to generate token", zap.Error(err))
			respondWithError(w, http.StatusInternalServerError, "error", err)
			return
		}

		w.Header().Set("Authorization", jwtToken)
		data := map[string]interface{}{"data": "otp verification successful"}
		respondWithSuccess(w, http.StatusOK, "success", data)

	case "resetpin":
		data := map[string]interface{}{"data": email}
		respondWithSuccess(w, http.StatusOK, "otp verification successful", data)

	default:
		http.NotFound(w, r)
	}
}

func (handler *HttpHandler) SendSMSOTP(w http.ResponseWriter, r *http.Request) {

	type input struct {
		Phone string `json:"phone_number"`
	}
	ctx := r.Context()

	data := input{}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		handler.logger.Error("Invalid request body", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	_, err := handler.store.GetUserByPhone(ctx, data.Phone)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			handler.logger.Warn("No user found with phone number", zap.String("phone", data.Phone))
			respondWithError(w, http.StatusNotFound, "no user found with phone number", err)
			return
		}
		handler.logger.Error("Database error", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "database error", err)
		return
	}

	err = handler.smsClient.SendSMS(ctx, data.Phone)
	if err != nil {
		if err == termii.ErrSMSFailed {
			handler.logger.Error("SMS sending failed", zap.String("phone", data.Phone), zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "failed to send OTP, please try again"}}
			json.NewEncoder(w).Encode(response)
			return
		}
		handler.logger.Error("Failed to send OTP", zap.String("phone", data.Phone), zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "failed to send otp", err)
		return
	}

	handler.logger.Info("OTP sent successfully", zap.String("phone", data.Phone))
	respondWithSuccess(w, http.StatusOK, "success", "OTP sent successfully")
}

func (handler *HttpHandler) VerifySMSOTP(w http.ResponseWriter, r *http.Request) {

	type input struct {
		OTP string `json:"otp"`
	}
	data := input{}
	ctx := r.Context()

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	phone := r.URL.Query().Get("phone")

	err := handler.smsClient.VerifyToken(ctx, data.OTP, phone)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error", err)
		return
	}

	action := getLastPathSegment(r.URL.Path)
	switch action {
	case "signin":
		data := map[string]interface{}{"phone": phone}
		respondWithSuccess(w, http.StatusOK, "otp verification successful", data)
	case "signup":
		_, err := handler.store.VerifyUser(ctx, phone)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "error", err)
			return
		}

		data := map[string]interface{}{"phone": phone}
		respondWithSuccess(w, http.StatusOK, "otp verification successful", data)
	case "resetpassword":
		user, err := handler.store.GetUserByPhone(ctx, phone)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "error", err)
			return
		}

		claims := dto.Claims{
			PersonId: user.ID,
		}

		jwtToken, err := handler.jwt.GenerateTokenWithExpiration(claims, handler.authTokenDuration)
		if err != nil {
			handler.logger.Error("fail to generate token", zap.Error(err))
			respondWithError(w, http.StatusInternalServerError, "error", err)
			return
		}

		w.Header().Set("Authorization", jwtToken)
		data := map[string]interface{}{"message": "otp verification successful"}
		respondWithSuccess(w, http.StatusOK, "success", data)
	default:
		http.NotFound(w, r)
	}

}

func (handler *HttpHandler) SendWhatsAppOTP(w http.ResponseWriter, r *http.Request) {
	type input struct {
		Phone string `json:"phone_number"`
	}
	ctx := r.Context()

	data := input{}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		handler.logger.Error("Invalid request body", zap.Error(err))
		respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	_, err := handler.store.GetUserByPhone(ctx, data.Phone)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			handler.logger.Warn("No user found with phone number", zap.String("phone", data.Phone))
			respondWithError(w, http.StatusNotFound, "no user found with phone number", err)
			return
		}
		handler.logger.Error("Database error", zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "database error", err)
		return
	}

	if err := handler.sendWhatsAppOTP(ctx, data.Phone); err != nil {
		handler.logger.Error("Failed to send WhatsApp OTP", zap.String("phone", data.Phone), zap.Error(err))
		respondWithError(w, http.StatusInternalServerError, "failed to send whatsapp otp", err)
		return
	}

	handler.logger.Info("WhatsApp OTP sent successfully", zap.String("phone", data.Phone))
	respondWithSuccess(w, http.StatusOK, "success", "WhatsApp OTP sent successfully")
}

func (handler *HttpHandler) VerifyWhatsAppOTP(w http.ResponseWriter, r *http.Request) {
	type input struct {
		OTP string `json:"otp"`
	}
	data := input{}
	ctx := r.Context()

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	phone := r.URL.Query().Get("phone")

	if err := handler.validateWhatsAppOTP(ctx, data.OTP, phone); err != nil {
		respondWithError(w, http.StatusInternalServerError, "error", err)
		return
	}

	action := getLastPathSegment(r.URL.Path)
	switch action {
	case "signin":
		data := map[string]interface{}{"phone": phone}
		respondWithSuccess(w, http.StatusOK, "otp verification successful", data)
	case "signup":
		_, err := handler.store.VerifyUser(ctx, phone)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "error", err)
			return
		}

		data := map[string]interface{}{"phone": phone}
		respondWithSuccess(w, http.StatusOK, "otp verification successful", data)
	case "resetpassword":
		user, err := handler.store.GetUserByPhone(ctx, phone)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "error", err)
			return
		}

		claims := dto.Claims{
			PersonId: user.ID,
		}

		jwtToken, err := handler.jwt.GenerateTokenWithExpiration(claims, handler.authTokenDuration)
		if err != nil {
			handler.logger.Error("fail to generate token", zap.Error(err))
			respondWithError(w, http.StatusInternalServerError, "error", err)
			return
		}

		w.Header().Set("Authorization", jwtToken)
		data := map[string]interface{}{"message": "otp verification successful"}
		respondWithSuccess(w, http.StatusOK, "success", data)
	default:
		http.NotFound(w, r)
	}
}

func (handler *HttpHandler) GetUserInfo(w http.ResponseWriter, r *http.Request) {

	handler.logger.Info("getting user info")
	email := r.URL.Query().Get("email")
	u_name := r.URL.Query().Get("username")

	username := cases.Title(language.English).String(u_name)
	if email == "" && username == "" {
		handler.logger.Error("email and username cannot be empty")
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "email and username cannot be empty"}}
		json.NewEncoder(w).Encode(response)
		return
	}
	ctx := r.Context()

	user, err := handler.store.GetUserByUsernameOrEmail(ctx, email, username)
	if err != nil {
		handler.logger.Error("error retrieving user info", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if !user.HasVirtualNuban {
		handler.logger.Warn("user has no virtual nuban account", zap.String("userID", user.ID))
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "user hasn't generated a virtual nuban account yet"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	userResponse := struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		FullName string `json:"full_name"`
	}{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Phone:    user.PhoneNumber,
		FullName: user.FullName,
	}

	handler.logger.Info("user info retrieved successfully", zap.String("email", user.ID))
	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "success", Data: map[string]interface{}{"userDetails": userResponse}}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) validateToken(token string) (isValid bool, response *dto.Claims) {
	claims, err := handler.jwt.ValidateToken(token)
	if err != nil {
		err := errorvalues.Format(errorvalues.InvalidTokenErr, err)
		handler.logger.Error("validating token", zap.Error(err))
		return
	}

	// Response
	return true, &dto.Claims{
		PersonId: claims.ID,
	}
}

func (handler *HttpHandler) sendOTP(ctx context.Context, user *models.User, title string, templateID string) error {
	otp, err := handler.otp.GenerateOTP(ctx, otpChannelEmail, user.Email)
	if err != nil {
		return err
	}

	// Creating Message
	message := models.Message{
		ID:         handler.idGenerator.Generate(),
		CustomerID: user.ID,
		Target:     user.Email,
		Type:       "email",
		Title:      title,
		Body:       otp,
		TemplateID: templateID,
		DataMap:    map[string]string{},
		Ts:         handler.timeHelper.Now().Unix(),
	}
	message.DataMap["FullName"] = user.FullName
	message.DataMap["Username"] = user.Username
	message.DataMap["Email"] = user.Email
	message.DataMap["OTP"] = otp

	// send message
	fmt.Println("about send email")
	if err := handler.emailClient.Send(&message); err != nil {
		return err
	}
	fmt.Println("email sent")
	return nil
}

func (handler *HttpHandler) sendWhatsAppOTP(ctx context.Context, phone string) error {
	otp, err := handler.otp.GenerateOTP(ctx, otpChannelWhatsApp, phone)
	if err != nil {
		return err
	}

	return handler.whatsAppClient.SendWhatsAppToken(ctx, otp, phone)
}

func (handler *HttpHandler) validateWhatsAppOTP(ctx context.Context, otp, phone string) error {
	valid, err := handler.otp.ValidateOTP(ctx, otp, otpChannelWhatsApp, phone)
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("otp verification failed")
	}

	return nil
}

func (handler *HttpHandler) Testtoken(w http.ResponseWriter, r *http.Request) {
	var tokenIn dto.TokenInput

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&tokenIn); err != nil {
		w.WriteHeader(http.StatusBadRequest)

		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	isValid, claims := handler.validateToken(tokenIn.Token)
	fmt.Println(isValid)
	fmt.Println(claims)
}

func (handler *HttpHandler) isValidNewUser(ctx context.Context, user models.User) (bool, string, error) {
	userDetails, err := handler.store.GetUserByUsernameOrEmailOrPhone(ctx, user.Username, user.Email, user.PhoneNumber)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return true, "", nil
		}
		return false, "", err
	}

	if user.Email == userDetails.Email {
		return false, "email already exists", nil
	}
	if user.PhoneNumber == userDetails.PhoneNumber {
		return false, "phone number already exists", nil
	}
	if user.Username == userDetails.Username {
		return false, "username already exists", nil
	}

	return true, "", nil
}

// PingUser pings the api with client credentials. It not used.
func (handler *HttpHandler) PingUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	res, err := handler.dataClient.PingUser(ctx, w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	results := make(map[string]interface{})

	json.NewDecoder(res.Body).Decode(&results)
	json.NewEncoder(w).Encode(results)
}

// Helper function to extract the last part of the URL path
func getLastPathSegment(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

// Helper function to respond with error
func respondWithError(w http.ResponseWriter, statusCode int, message string, err error) {
	w.WriteHeader(statusCode)
	data := map[string]interface{}{"message": message}
	if err != nil {
		data["error"] = err.Error()
	}
	response := responseFormat.CustomResponse{
		Status:  statusCode,
		Message: "error",
		Data:    data,
	}
	json.NewEncoder(w).Encode(response)
}

// Helper function to respond with success
func respondWithSuccess(w http.ResponseWriter, statusCode int, message string, datamsg interface{}) {
	w.WriteHeader(statusCode)

	data := map[string]interface{}{"data": datamsg}

	// Create a response structure
	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: message,
		Data:    data,
	}

	// Encode the response as JSON and send it to the client
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {

	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "refresh token missing", http.StatusUnauthorized)
		return
	}
	ctx := r.Context()

	refresh := cookie.Value
	handler.logger.Debug("RefreshToken: cookie value", zap.String("refresh_cookie", refresh))

	claims, err := handler.jwt.ValidateToken(refresh)
	if err != nil {
		handler.logger.Warn("RefreshToken: token validation failed", zap.Error(err))
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}
	handler.logger.Debug("RefreshToken: token validated", zap.Any("claims", claims))

	userID := claims.ID
	sessionKey := fmt.Sprintf("session:%s", userID)

	storedRefresh, err := handler.redisClient.Get(ctx, sessionKey)
	if err != nil || storedRefresh == nil {
		http.Error(w, "session expired", http.StatusUnauthorized)
		return
	}

	// Compare stored token (rotation protection)
	if storedRefresh.(string) != refresh {
		http.Error(w, "refresh token rotated or invalid", http.StatusUnauthorized)
		return
	}

	// ROTATE refresh token
	newClaims := dto.Claims{PersonId: userID}
	newRefresh, err := handler.jwt.GenerateTokenWithExpiration(newClaims, handler.refreshTokenDuration)
	if err != nil {
		http.Error(w, "cannot refresh token", http.StatusInternalServerError)
		return
	}

	// Update session in Redis
	handler.redisClient.SetWithTTL(ctx, sessionKey, newRefresh, handler.refreshTokenDuration)

	// New access token
	newAccess, err := handler.jwt.GenerateTokenWithExpiration(newClaims, handler.authTokenDuration)
	if err != nil {
		http.Error(w, "cannot refresh token", http.StatusInternalServerError)
		return
	}

	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    newAccess,
		MaxAge:   900,
		HttpOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: http.SameSiteNoneMode,
		Domain:   "aremxyplug.com",
	}

	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefresh,
		MaxAge:   1800,
		HttpOnly: true,
		Secure:   true,
		Path:     "/api/v1/refresh-token",
		SameSite: http.SameSiteNoneMode,
		Domain:   "aremxyplug.com",
	}

	http.SetCookie(w, accessCookie)
	http.SetCookie(w, refreshCookie)

	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"message": "session refreshed"},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) Logout(w http.ResponseWriter, r *http.Request) {

	user, err := handler.GetUserDetails(r)
	if err != nil {
		handler.logger.Error("Failed to get user details", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	ctx := r.Context()

	access := &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}

	refresh := &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/refresh-token",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}

	http.SetCookie(w, access)
	http.SetCookie(w, refresh)

	// delete redis session
	userID := user.ID
	sessionKey := fmt.Sprintf("session:%s", userID)
	if err := handler.redisClient.Del(ctx, sessionKey); err != nil {
		handler.logger.Error("error deleting user session from redis", zap.String("userID", userID), zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": "error logging out, please try again"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	response := responseFormat.CustomResponse{
		Status:  http.StatusOK,
		Message: "success",
		Data:    map[string]interface{}{"message": "logged out successfully"},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
