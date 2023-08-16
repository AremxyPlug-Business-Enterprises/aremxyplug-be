package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/lib/emailclient"
	"github.com/aremxyplug-be/lib/errorvalues"
	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/aremxyplug-be/types/dto"
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"time"

	"github.com/aremxyplug-be/config"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/encryptor"
	"github.com/aremxyplug-be/lib/idgenerator"
	"github.com/aremxyplug-be/lib/timehelper"
	tokengenerator "github.com/aremxyplug-be/lib/tokekngenerator"
	uuidgenerator "github.com/aremxyplug-be/lib/uuidgeneraor"
	"github.com/go-playground/validator/v10"
	mongodb "go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const (
	// email templates
	PasswordResetAlias = "password-reset"
)

var validate = validator.New()

type HttpHandler struct {
	logger               *zap.Logger
	idGenerator          idgenerator.IdGenerator
	timeHelper           timehelper.TimeHelper
	store                db.DataStore
	secrets              *config.Secrets
	encrypt              encryptor.Encryptor
	jwt                  tokengenerator.TokenGenerator
	refreshTokenDuration time.Duration
	authTokenDuration    time.Duration
	uuidGenerator        uuidgenerator.UUIDGenerator
	emailClient          emailclient.EmailClient
}

type HandlerOptions struct {
	Logger      *zap.Logger
	Store       db.DataStore
	Secrets     *config.Secrets
	EmailClient emailclient.EmailClient
}

func NewHttpHandler(opt *HandlerOptions) *HttpHandler {
	refreshTokenDuration := calculateDefaultDuration(
		tokengenerator.RefreshTokenDuration,
		time.Duration(opt.Secrets.RefreshTokenDuration),
	)
	authTokenDuration := calculateDefaultDuration(
		tokengenerator.AuthTokenDuration,
		time.Duration(opt.Secrets.AuthTokenDuration),
	)

	tokenGeneratorPublicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(opt.Secrets.JWTPublicKey))
	if err != nil {
		opt.Logger.Error(
			"error parsing public key for token encryption",
			zap.Error(err),
		)
	}

	tokenGeneratorPrivateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(opt.Secrets.JWTPrivateKey))
	if err != nil {
		opt.Logger.Error(
			"error parsing private key for token encryption",
			zap.Error(err),
		)
	}

	return &HttpHandler{
		logger:      opt.Logger,
		idGenerator: idgenerator.New(),
		timeHelper:  timehelper.New(),
		store:       opt.Store,
		secrets:     opt.Secrets,
		encrypt:     encryptor.NewEncryptor(),
		jwt: tokengenerator.New(
			tokenGeneratorPublicKey,
			tokenGeneratorPrivateKey,
		),
		refreshTokenDuration: refreshTokenDuration,
		authTokenDuration:    authTokenDuration,
		uuidGenerator:        uuidgenerator.NewGoogleUUIDGenerator(),
		emailClient:          opt.EmailClient,
	}
}

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
		w.Header().Set("Content-Type", "application/json")
		//response := responseFormat.RespondWithError(w, http.StatusBadRequest, err.Error())
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if !handler.isValidNewUser(user.Email) {
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "email exist", Data: map[string]interface{}{"data": "user already exist"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	// use the validator library to validate required fields
	if validationErr := validate.Struct(&user); validationErr != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
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

	newUser := models.User{
		ID:             userId,
		FullName:       user.FullName,
		Email:          user.Email,
		Username:       user.Username,
		Password:       string(hashedPassword),
		PhoneNumber:    user.PhoneNumber,
		Country:        user.Country,
		InvitationCode: user.InvitationCode,
		CreatedAt:      timestamp,
		UpdatedAt:      timestamp,
	}

	err = handler.store.SaveUser(newUser)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{"data": "user created"}}
	json.NewEncoder(w).Encode(response)
}

// Login is the api used to login a single user
func (handler *HttpHandler) Login(w http.ResponseWriter, r *http.Request) {
	var userlogin dto.LoginInput

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&userlogin); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	user, err := handler.store.GetUserByEmail(userlogin.Email)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusNotFound, Message: "user not found", Data: map[string]interface{}{"data": "user not found"}}
		json.NewEncoder(w).Encode(response)
		return
	}
	hashedPassword := user.Password

	ok := handler.encrypt.ComparePasscode(userlogin.Password, hashedPassword)
	if !ok {
		handler.logger.Error("store validating password")
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusUnauthorized, Message: "error", Data: map[string]interface{}{"data": "password incorrect"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	refreshTokenClaims := dto.Claims{
		PersonId: user.ID,
		Email:    user.Email,
	}

	claims := dto.Claims{
		PersonId: user.ID,
		Email:    user.Email,
	}

	jwtToken, err := handler.jwt.GenerateTokenWithExpiration(claims, handler.authTokenDuration)
	if err != nil {
		handler.logger.Error("fail to generate token", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	userResponse := dto.UserResponse{
		FullName: user.FullName,
		Email:    user.Email,
		Phone:    user.PhoneNumber,
	}

	refreshToken, err := handler.jwt.GenerateTokenWithExpiration(refreshTokenClaims, handler.refreshTokenDuration)
	if err != nil {
		handler.logger.Error("fail to generate refresh token", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{"auth_token": jwtToken, "refresh_token": refreshToken, "customer": userResponse}}
	json.NewEncoder(w).Encode(response)

}

// PasswordReset
func (handler *HttpHandler) PasswordReset(w http.ResponseWriter, r *http.Request) {
	var userlogin dto.LoginInput

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&userlogin); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	user, err := handler.store.GetUserByEmail(userlogin.Email)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusNotFound, Message: "user not found", Data: map[string]interface{}{"data": "user not found"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Creating Message
	message := models.Message{
		ID:         handler.idGenerator.Generate(),
		CustomerID: user.ID,
		Target:     user.Email,
		Type:       "email",
		Title:      "Password Reset",
		Body:       "",
		TemplateID: PasswordResetAlias,
		DataMap:    map[string]string{},
		Ts:         handler.timeHelper.Now().Unix(),
	}
	message.DataMap["FullName"] = user.FullName
	message.DataMap["Email"] = user.Email

	// send message
	fmt.Println("about send email")
	err = handler.emailClient.Send(&message)
	fmt.Println("email sent")
	if err != nil {
		handler.logger.Error("error sending password reset email", zap.String("target", user.Email), zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	handler.logger.Info("password reset email sent", zap.String("target", user.Email))
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{"msg": "email sent successfully"}}
	json.NewEncoder(w).Encode(response)

}

func (handler *HttpHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var userlogin dto.LoginInput

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&userlogin); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	user, err := handler.store.GetUserByEmail(userlogin.Email)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusNotFound, Message: "user not found", Data: map[string]interface{}{"data": "user not found"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Creating Message
	message := models.Message{
		ID:         handler.idGenerator.Generate(),
		CustomerID: user.ID,
		Target:     user.Email,
		Type:       "email",
		Title:      "Password Reset",
		Body:       "",
		TemplateID: PasswordResetAlias,
		DataMap:    map[string]string{},
		Ts:         handler.timeHelper.Now().Unix(),
	}
	message.DataMap["FullName"] = user.FullName
	message.DataMap["Email"] = user.Email

	// send message
	fmt.Println("about send email")
	err = handler.emailClient.Send(&message)
	fmt.Println("email sent")
	if err != nil {
		handler.logger.Error("error sending password reset email", zap.String("target", user.Email), zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	handler.logger.Info("password reset email sent", zap.String("target", user.Email))
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{"msg": "email sent successfully"}}
	json.NewEncoder(w).Encode(response)

}

func (handler *HttpHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var userlogin dto.LoginInput

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&userlogin); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	user, err := handler.store.GetUserByEmail(userlogin.Email)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusNotFound, Message: "user not found", Data: map[string]interface{}{"data": "user not found"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Creating Message
	message := models.Message{
		ID:         handler.idGenerator.Generate(),
		CustomerID: user.ID,
		Target:     user.Email,
		Type:       "email",
		Title:      "Password Reset",
		Body:       "",
		TemplateID: PasswordResetAlias,
		DataMap:    map[string]string{},
		Ts:         handler.timeHelper.Now().Unix(),
	}
	message.DataMap["FullName"] = user.FullName
	message.DataMap["Email"] = user.Email

	// send message
	fmt.Println("about send email")
	err = handler.emailClient.Send(&message)
	fmt.Println("email sent")
	if err != nil {
		handler.logger.Error("error sending password reset email", zap.String("target", user.Email), zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	handler.logger.Info("password reset email sent", zap.String("target", user.Email))
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	response := responseFormat.CustomResponse{Status: http.StatusCreated, Message: "success", Data: map[string]interface{}{"msg": "email sent successfully"}}
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
		Email:    claims.Email,
	}
}

func (handler *HttpHandler) Testtoken(w http.ResponseWriter, r *http.Request) {
	var tokenIn dto.TokenInput

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&tokenIn); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	isValid, claims := handler.validateToken(tokenIn.Token)
	fmt.Println(isValid)
	fmt.Println(claims)
}

func (handler *HttpHandler) isValidNewUser(email string) bool {
	_, err := handler.store.GetUserByEmail(email)
	if err != nil {
		switch err {
		case mongodb.ErrNoDocuments:
			return true
		}
	}
	return false
}
