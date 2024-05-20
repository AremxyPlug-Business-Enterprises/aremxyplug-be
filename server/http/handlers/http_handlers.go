package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	//"strconv"
	"time"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/errorvalues"
	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/aremxyplug-be/types/dto"
	"github.com/go-chi/render"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	mongodb "go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

func calculateDefaultDuration(defaultDuration, configDuration time.Duration) time.Duration {
	duration := defaultDuration
	if configDuration > 0 {
		duration = configDuration * time.Minute
	}
	return duration
}

// SignUp is the api used to create a single user
func (handler *HttpHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var user models.User
	//defer cancel()

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		w.WriteHeader(http.StatusBadRequest)

		//response := responseFormat.RespondWithError(w, http.StatusBadRequest, err.Error())
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if !handler.isValidNewUser(user) {
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "email exist", Data: map[string]interface{}{"data": "user already exist"}}
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
	timestamp := handler.timeHelper.Now().Unix()
	userId := handler.idGenerator.Generate()
	hashedPassword, err := handler.encrypt.GenerateFromPassword(user.Password)
	if err != nil {
		handler.logger.Error("fail to generate password", zap.Error(err))
		return
	}

	to := cases.Title(language.English)
	full_name := to.String(user.FullName)

	newUser := models.User{
		ID:             userId,
		FullName:       full_name,
		Email:          user.Email,
		Username:       user.Username,
		Password:       string(hashedPassword),
		PhoneNumber:    user.PhoneNumber,
		Country:        user.Country,
		InvitationCode: user.InvitationCode,
		CreatedAt:      timestamp,
		UpdatedAt:      timestamp,
		IsVerified:     false,
	}

	err = handler.sendOTP(&newUser, "verify-email", verifyEmailAlias)
	if err != nil {
		handler.logger.Error("error sending email verification otp", zap.String("target", user.Email), zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	err = handler.store.SaveUser(newUser)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusCreated)
	response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{"data": "user created"}}
	json.NewEncoder(w).Encode(response)
}

// Login is the api used to login a single user
func (handler *HttpHandler) Login(w http.ResponseWriter, r *http.Request) {
	var userlogin dto.LoginInput

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&userlogin); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	user, err := handler.store.GetUserByUsernameOrEmail(userlogin.Email, userlogin.Username)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		response := responseFormat.CustomResponse{Status: http.StatusNotFound, Message: "user not found", Data: map[string]interface{}{"data": "user not found"}}
		json.NewEncoder(w).Encode(response)
		return
	}
	hashedPassword := user.Password

	ok := handler.encrypt.ComparePasscode(userlogin.Password, hashedPassword)
	if !ok {
		handler.logger.Error("store validating password")
		w.WriteHeader(http.StatusUnauthorized)
		response := responseFormat.CustomResponse{Status: http.StatusUnauthorized, Message: "error", Data: map[string]interface{}{"data": "password incorrect"}}
		json.NewEncoder(w).Encode(response)
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

	/*
		if err := handler.refreshBalance(user.FullName); err != nil {
			if err == deposit.ErrEmptyVirtualNuban {
				w.WriteHeader(http.StatusBadRequest)
				response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": "virtualNuban is empty"}}
				json.NewEncoder(w).Encode(response)
				return
			}
			handler.logger.Error("failed to load user's balance", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
			json.NewEncoder(w).Encode(response)
			return
		}
	*/

	// should check if the user already has pin set otherwise return an status that should redirect the frontend to the pin endpoint

	w.Header().Set("Authorization", jwtToken)
	w.WriteHeader(http.StatusCreated)
	response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{"auth_token": jwtToken, "refresh_token": refreshToken, "customer": userResponse}}
	json.NewEncoder(w).Encode(response)

}

// ForgotPassword
func (handler *HttpHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var userlogin dto.LoginInput

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&userlogin); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Checking if the user exists (replace with your actual user lookup logic)
	user, err := handler.store.GetUserByEmail(userlogin.Email)
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
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	//var uri string
	uri := "/api/v1/verify-token?token="
	Scheme := "http"
	link := fmt.Sprintf("%s://%s%s%s", Scheme, r.Host, uri, token)
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
	fmt.Println("email sent")
	if err != nil {
		handler.logger.Error("error sending password reset email", zap.String("target", user.Email), zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	handler.logger.Info("password reset email sent", zap.String("target", user.Email))
	w.WriteHeader(http.StatusCreated)
	response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{"msg": "email sent successfully"}}
	json.NewEncoder(w).Encode(response)

}

func (handler *HttpHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	log.Print(token)

	//validate the token

	id, err := handler.jwt.ValidateToken(token)
	if err != nil {
		handler.logger.Error("fail to validate token", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "email", Data: map[string]interface{}{"data": id}}
	json.NewEncoder(w).Encode(response)
}

// ResetPassword
func (handler *HttpHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	//params := chi.URLParam(r, "token")
	//token := chi.URLParam(r, "token")
	//log.Print(token, params)

	//validate the token
	email := r.URL.Query().Get("email")

	//	hashing and updating user's password
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

	err = handler.store.UpdateUserPassword(email, newPassword.Password)
	if err != nil {
		handler.logger.Error("fail to update password", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	w.WriteHeader(http.StatusCreated)
	response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{"data": "Password updated successfully"}}
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var userlogin dto.LoginInput

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&userlogin); err != nil {
		w.WriteHeader(http.StatusBadRequest)

		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	user, err := handler.store.GetUserByEmail(userlogin.Email)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)

		response := responseFormat.CustomResponse{Status: http.StatusNotFound, Message: "user not found", Data: map[string]interface{}{"data": "user not found"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if err := handler.sendOTP(user, "Password OTP", PasswordOTPAlias); err != nil {
		handler.logger.Error("error sending password reset email", zap.String("target", user.Email), zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)

		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	handler.logger.Info("password reset email sent", zap.String("target", user.Email))
	w.WriteHeader(http.StatusCreated)
	response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{"msg": "email sent successfully"}}
	json.NewEncoder(w).Encode(response)

}

func (handler *HttpHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	type otp struct {
		OTP string `json:"otp"`
	}
	Otp := otp{}

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&Otp); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	email := r.URL.Query().Get("email")
	valid, err := handler.otp.ValidateOTP(Otp.OTP, email)
	if !valid || err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "otp verification successful", Data: map[string]interface{}{"data": email}}
	json.NewEncoder(w).Encode(response)
	// Creating Message
}

// Write the handler to save pin

// Write the handler to verify the pin

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

func (handler *HttpHandler) sendOTP(user *models.User, title string, templateID string) error {
	otp, err := handler.otp.GenerateOTP(user.Email)
	if err != nil {
		return err
	}
	log.Println(otp)

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

func (handler *HttpHandler) isValidNewUser(user models.User) bool {

	userDetails, err := handler.store.GetUserByEmail(user.Email)
	if err != nil {
		switch err {
		case mongodb.ErrNoDocuments:
			return true
		}
	}

	email := userDetails.Email
	phone := userDetails.PhoneNumber
	username := userDetails.Username

	if user.Email == email || user.PhoneNumber == phone || user.Username == username {
		return false
	}

	return false
}

// PingUser pings the api with client credentials. It not used.
func (handler *HttpHandler) PingUser(w http.ResponseWriter, r *http.Request) {

	res, err := handler.dataClient.PingUser(w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	results := make(map[string]interface{})

	json.NewDecoder(res.Body).Decode(&results)
	json.NewEncoder(w).Encode(results)
}
