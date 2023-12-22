package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/render"

	"github.com/aremxyplug-be/db"
	elect "github.com/aremxyplug-be/lib/bills/electricity"
	"github.com/aremxyplug-be/lib/bills/tvsub"
	"github.com/aremxyplug-be/lib/emailclient"
	"github.com/aremxyplug-be/lib/errorvalues"
	otpgen "github.com/aremxyplug-be/lib/otp_gen"
	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/aremxyplug-be/lib/telcom/airtime"
	"github.com/aremxyplug-be/lib/telcom/data"
	"github.com/aremxyplug-be/lib/telcom/edu"
	"github.com/aremxyplug-be/types/dto"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi/v5"

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
	PasswordOTPAlias   = "password-otp"
	verifyEmailAlias   = "verify-email"
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
	dataClient           *data.DataConn
	eduClient            *edu.EduConn
	vtuClient            *airtime.AirtimeConn
	tvClient             *tvsub.TvConn
	electClient          *elect.ElectricConn
	otp                  *otpgen.OTPConn
}

type HandlerOptions struct {
	Logger      *zap.Logger
	Store       db.DataStore
	Data        *data.DataConn
	Edu         *edu.EduConn
	VTU         *airtime.AirtimeConn
	TvSub       *tvsub.TvConn
	ElectSub    *elect.ElectricConn
	Secrets     *config.Secrets
	EmailClient emailclient.EmailClient
	Otp         *otpgen.OTPConn
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
		dataClient:           opt.Data,
		vtuClient:            opt.VTU,
		tvClient:             opt.TvSub,
		electClient:          opt.ElectSub,
		otp:                  opt.Otp,
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

	if !handler.isValidNewUser(user.Username) {
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "username exist", Data: map[string]interface{}{"data": "user already exist"}}
		json.NewEncoder(w).Encode(response)
		return
	}

	if !handler.isValidNewUser(user.PhoneNumber) {
		response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "PhoneNumber exist", Data: map[string]interface{}{"data": "user already exist"}}
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
	user, err := handler.store.GetUserByUsernameOrEmail(userlogin.Email, userlogin.Username)
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
		Username: user.Username,
	}

	claims := dto.Claims{
		PersonId: user.ID,
		Email:    user.Email,
		Username: user.Username,
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
		Username: user.Username,
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

// ForgotPassword
func (handler *HttpHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var userlogin dto.LoginInput

	// validate the request body
	if err := json.NewDecoder(r.Body).Decode(&userlogin); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		response := responseFormat.CustomResponse{Status: http.StatusBadRequest, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}

	email := userlogin.Email
	// Checking if the user exists (replace with your actual user lookup logic)
	user, err := handler.store.GetUserByEmail(userlogin.Email)
	if err != nil || user == nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, map[string]string{"error": "Sorry, this user does not exist"})
		return
	}

	claims := dto.Claims{
		PersonId: user.ID,
		Email:    email,
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

func (handler *HttpHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	log.Print(token)

	//validate the token

	user, err := handler.jwt.ValidateToken(token)
	if err != nil {
		handler.logger.Error("fail to validate token", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{Status: http.StatusInternalServerError, Message: "error", Data: map[string]interface{}{"data": err.Error()}}
		json.NewEncoder(w).Encode(response)
		return
	}
	w.WriteHeader(http.StatusOK)
	response := responseFormat.CustomResponse{Status: http.StatusOK, Message: "email", Data: map[string]interface{}{"data": user.Email}}
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

	if err := handler.sendOTP(user, "Password OTP", PasswordOTPAlias); err != nil {
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

// Data send a call to the API to buy data(POST) or return users transaction history(GET)
func (handler *HttpHandler) Data(w http.ResponseWriter, r *http.Request) {

	// Get username from request or token

	if r.Method == "POST" {
		data := models.DataInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			handler.logger.Error("Decoding JSON response", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return

		}
		res, err := handler.dataClient.BuyData(data)
		if err != nil {
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintf(w, "An internal error occurred while purchasing data, please try again...")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(res)
	}

	if r.Method == "GET" {
		res, err := handler.dataClient.GetUserTransactions("user")
		if err != nil {
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintln(w, "Errror occurred while getting user's records")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(res)
	}

}

// GetDataInfo checks and returns the details of a given transaction.
func (handler *HttpHandler) GetDataInfo(w http.ResponseWriter, r *http.Request) {

	//id := r.URL.Query().Get("id")
	id := chi.URLParam(r, "id")

	res, err := handler.dataClient.GetTransactionDetail(id)
	if err != nil {
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintln(w, "Error getting transaction detail.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(res)
}

// GetTransactions returns the list of transaction carried out in the server. It is for admins to view all transactions.
func (handler *HttpHandler) GetDataTransactions(w http.ResponseWriter, r *http.Request) {

	resp, err := handler.dataClient.GetAllTransactions()
	if err != nil {
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintf(w, "Error getting users transactions records: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)

}

// EduPins is use to carry out buying of education pins(POST) and returning all the transactions made by the user(GET)
func (handler *HttpHandler) EduPins(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		data := models.EduInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			handler.logger.Error("Decoding JSON response", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if data.Quantity >= 5 && data.Quantity < 10 {
			handler.logger.Error("invalid number of buy pins")
			fmt.Fprintf(w, "Invalid number of pins!! Pins between %d and %d are not allowed. Try again...", 5, 10)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err := handler.eduClient.BuyEduPin(data)
		if err != nil {
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintf(w, "An internal error occurred while purchasing %s pin, please try again...", data.Exam_Type)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if r.Method == "GET" {
		res, err := handler.eduClient.QueryTransaction("id")
		if err != nil {
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintln(w, "Errror occurred while getting user's records")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(res)
	}

}

// GetEduInfo returns the details of an airtime transaction.
func (handler *HttpHandler) GetEduInfo(w http.ResponseWriter, r *http.Request) {

	//id := r.URL.Query().Get("id")
	id := chi.URLParam(r, "id")

	res, err := handler.dataClient.GetTransactionDetail(id)
	if err != nil {
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintln(w, "Error getting transaction detail.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(res)
}

// To be used by admins to view transactions in the databases
func (handler *HttpHandler) GetEduTransactions(w http.ResponseWriter, r *http.Request) {

	resp, err := handler.eduClient.GetAllTransaction("user")
	if err != nil {
		handler.logger.Error("Error geeting user's transaction", zap.Error(err))
		fmt.Fprintln(w, "Error occurred while getting transactions")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// Airtime is use to carry out buying of airtime(POST) and returning all the transactions made by the user(GET)
func (handler *HttpHandler) Airtime(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		data := models.AirtimeInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			handler.logger.Error("Decoding JSON response", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return

		}
		if len(data.Phone_no) != 11 {
			fmt.Fprintf(w, "Phone number must be %d digits, got %d. Check the phone number and try again.", 11, len(data.Phone_no))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		res, err := handler.vtuClient.BuyAirtime(data)
		if err != nil {
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintf(w, "An internal error occurred while purchasing data, please try again...")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(res)
	}

	if r.Method == "GET" {
		res, err := handler.vtuClient.GetUserTransaction("user")
		if err != nil {
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintln(w, "Errror occurred while getting user's records")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(res)
	}
}

// GetAirtimeTransactions return all the airtime transactions in the database, to be used by admin.
func (handler *HttpHandler) GetAirtimeTransactions(w http.ResponseWriter, r *http.Request) {

	resp, err := handler.vtuClient.GetAllTransactions()
	if err != nil {
		handler.logger.Error("Error geeting user's transaction", zap.Error(err))
		fmt.Fprintln(w, "Error occurred while getting transactions")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

// GetAirtimeInfo returns the details of an airtime transaction.
func (handler *HttpHandler) GetAirtimeInfo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	res, err := handler.dataClient.GetTransactionDetail(id)
	if err != nil {
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintln(w, "Error getting transaction detail.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(res)
}

func (handler *HttpHandler) TVSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		data := models.TvInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			handler.logger.Error("Decoding JSON response", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return

		}
		res, err := handler.tvClient.BuySub(data)
		if err != nil {
			handler.logger.Error("Api response error", zap.Error(err))
			// change error message
			fmt.Fprintf(w, "An internal error occurred while purchasing data, please try again...")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(res)
	}

	if r.Method == "GET" {
		res, err := handler.tvClient.GetUserTransactions("user")
		if err != nil {
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintln(w, "Errror occurred while getting user's records")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(res)
	}
}

func (handler *HttpHandler) GetTvSubscriptions(w http.ResponseWriter, r *http.Request) {
	resp, err := handler.tvClient.GetAllTransactions()
	if err != nil {
		handler.logger.Error("Error geeting user's transaction", zap.Error(err))
		fmt.Fprintln(w, "Error occurred while getting transactions")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (handler *HttpHandler) GetTvSubDetails(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	res, err := handler.tvClient.GetTransactionDetails(id)
	if err != nil {
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintln(w, "Error getting transaction detail.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(res)
}

func (handler *HttpHandler) ElectricBill(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		data := models.ElectricInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			handler.logger.Error("Decoding JSON response", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return

		}
		res, err := handler.electClient.PayBill(data)
		if err != nil {
			handler.logger.Error("Api response error", zap.Error(err))
			// change error message
			fmt.Fprintf(w, "An internal error occurred while purchasing data, please try again...")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(res)
	}

	if r.Method == "GET" {
		res, err := handler.electClient.GetUserTransactions("user")
		if err != nil {
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintln(w, "Errror occurred while getting user's records")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(res)
	}
}

func (handler *HttpHandler) GetElectricBills(w http.ResponseWriter, r *http.Request) {
	resp, err := handler.electClient.GetAllTransactions()
	if err != nil {
		handler.logger.Error("Error geeting user's transaction", zap.Error(err))
		fmt.Fprintln(w, "Error occurred while getting transactions")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (handler *HttpHandler) GetElectricBillDetails(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	res, err := handler.electClient.GetTransactionDetails(id)
	if err != nil {
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintln(w, "Error getting transaction detail.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(res)
}

func (handler *HttpHandler) SpectranetData(w http.ResponseWriter, r *http.Request) {

	// Get username from request or token

	if r.Method == "POST" {
		data := models.SpectranetInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			handler.logger.Error("Decoding JSON response", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return

		}
		res, err := handler.dataClient.BuySpecData(data)
		if err != nil {
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintf(w, "An internal error occurred while purchasing data, please try again...")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(res)
	}

	if r.Method == "GET" {
		res, err := handler.dataClient.GetSpecUserTransactions("user")
		if err != nil {
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintln(w, "Errror occurred while getting user's records")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(res)
	}

}

func (handler *HttpHandler) GetSpecDataDetails(w http.ResponseWriter, r *http.Request) {

	//id := r.URL.Query().Get("id")
	id := chi.URLParam(r, "id")

	res, err := handler.dataClient.GetSpecTransDetails(id)
	if err != nil {
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintln(w, "Error getting transaction detail.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(res)
}

// To be used by admin
func (handler *HttpHandler) GetSpectranetTransactions(w http.ResponseWriter, r *http.Request) {

	resp, err := handler.dataClient.GetAllSpecTransactions()
	if err != nil {
		handler.logger.Error("Error geeting user's transaction", zap.Error(err))
		fmt.Fprintln(w, "Error occurred while getting transactions")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (handler *HttpHandler) SmileData(w http.ResponseWriter, r *http.Request) {

	// Get username from request or token

	if r.Method == "POST" {
		data := models.SmileInfo{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			handler.logger.Error("Decoding JSON response", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return

		}
		res, err := handler.dataClient.BuySmileData(data)
		if err != nil {
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintf(w, "An internal error occurred while purchasing data, please try again...")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(res)
	}

	if r.Method == "GET" {
		res, err := handler.dataClient.GetSmileUserTransactions("user")
		if err != nil {
			handler.logger.Error("Api response error", zap.Error(err))
			fmt.Fprintln(w, "Errror occurred while getting user's records")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(res)
	}

}

func (handler *HttpHandler) GetSmileDataDetails(w http.ResponseWriter, r *http.Request) {

	//id := r.URL.Query().Get("id")
	id := chi.URLParam(r, "id")

	res, err := handler.dataClient.GetSmileTransDetails(id)
	if err != nil {
		handler.logger.Error("Api response error", zap.Error(err))
		fmt.Fprintln(w, "Error getting transaction detail.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(res)
}

// To be used by admin
func (handler *HttpHandler) GetSmileTransactions(w http.ResponseWriter, r *http.Request) {

	resp, err := handler.dataClient.GetAllSmileTransactions()
	if err != nil {
		handler.logger.Error("Error geeting user's transaction", zap.Error(err))
		fmt.Fprintln(w, "Error occurred while getting transactions")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resp)
}
