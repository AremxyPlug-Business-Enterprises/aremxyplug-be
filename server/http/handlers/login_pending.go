package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/responseFormat"
	"github.com/aremxyplug-be/types/dto"
)

const pendingLoginTTL = 15 * time.Minute

var errPendingLoginExpired = errors.New("login session expired, please login again")

type pendingLoginState struct {
	Token      string    `json:"token"`
	UserID     string    `json:"user_id"`
	Email      string    `json:"email"`
	Phone      string    `json:"phone"`
	IsVerified bool      `json:"is_verified"`
	HasPin     bool      `json:"has_pin"`
	CreatedAt  time.Time `json:"created_at"`
}

func pendingLoginKey(token string) string {
	return fmt.Sprintf("login:pending:%s", token)
}

func (handler *HttpHandler) buildUserResponse(user *models.User) dto.UserResponse {
	return dto.UserResponse{
		ID:       user.ID,
		FullName: user.FullName,
		Email:    user.Email,
		Username: user.Username,
		Phone:    user.PhoneNumber,
	}
}

func (handler *HttpHandler) createPendingLogin(ctx context.Context, user *models.User) (*pendingLoginState, error) {
	state := &pendingLoginState{
		Token:      handler.uuidGenerator.Generate(),
		UserID:     user.ID,
		Email:      strings.TrimSpace(strings.ToLower(user.Email)),
		Phone:      strings.TrimSpace(user.PhoneNumber),
		IsVerified: user.IsVerified,
		HasPin:     user.HasPin,
		CreatedAt:  handler.timeHelper.Now().UTC(),
	}

	if err := handler.redisClient.SetWithTTL(ctx, pendingLoginKey(state.Token), state, pendingLoginTTL); err != nil {
		return nil, err
	}

	return state, nil
}

func (handler *HttpHandler) getPendingLogin(ctx context.Context, token string) (*pendingLoginState, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errPendingLoginExpired
	}

	value, err := handler.redisClient.Get(ctx, pendingLoginKey(token))
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, errPendingLoginExpired
	}

	var raw []byte
	switch v := value.(type) {
	case string:
		raw = []byte(v)
	case []byte:
		raw = v
	default:
		raw, err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
	}

	state := &pendingLoginState{}
	if err := json.Unmarshal(raw, state); err != nil {
		return nil, err
	}
	if state.Token == "" {
		state.Token = token
	}

	return state, nil
}

func (handler *HttpHandler) deletePendingLogin(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}

	return handler.redisClient.Del(ctx, pendingLoginKey(token))
}

func (handler *HttpHandler) getPendingLoginUser(ctx context.Context, token string) (*models.User, *pendingLoginState, error) {
	state, err := handler.getPendingLogin(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	user, err := handler.store.GetUserByID(ctx, state.UserID)
	if err != nil {
		return nil, nil, err
	}

	if state.Email != "" && !strings.EqualFold(state.Email, user.Email) {
		return nil, nil, errors.New("login session no longer matches this user")
	}
	if state.Phone != "" && strings.TrimSpace(state.Phone) != strings.TrimSpace(user.PhoneNumber) {
		return nil, nil, errors.New("login session no longer matches this user")
	}

	return user, state, nil
}

func (handler *HttpHandler) writeLoginFlowResponse(w http.ResponseWriter, status int, user *models.User, token, nextStep, message string) {
	response := responseFormat.CustomResponse{
		Status:  status,
		Message: message,
		Data: map[string]interface{}{
			"customer":                        handler.buildUserResponse(user),
			"has_pin":                         user.HasPin,
			"is_verified":                     user.IsVerified,
			"pending_login_token":             token,
			"next_step":                       nextStep,
			"pending_login_expires_in_seconds": int(pendingLoginTTL.Seconds()),
		},
	}

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

func (handler *HttpHandler) issueLoginSession(w http.ResponseWriter, ctx context.Context, user *models.User) error {
	refreshTokenClaims := dto.Claims{
		PersonId: user.ID,
	}

	claims := dto.Claims{
		PersonId: user.ID,
	}

	jwtToken, err := handler.jwt.GenerateTokenWithExpiration(claims, handler.authTokenDuration)
	if err != nil {
		return err
	}

	refreshToken, err := handler.jwt.GenerateTokenWithExpiration(refreshTokenClaims, handler.refreshTokenDuration)
	if err != nil {
		return err
	}

	sessionKey := fmt.Sprintf("session:%s", user.ID)
	if err := handler.redisClient.SetWithTTL(ctx, sessionKey, refreshToken, handler.refreshTokenDuration); err != nil {
		return err
	}

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
	return nil
}

func (handler *HttpHandler) writeAuthSuccessResponse(w http.ResponseWriter, status int, user *models.User, message string) {
	response := responseFormat.CustomResponse{
		Status:  status,
		Message: message,
		Data: map[string]interface{}{
			"customer":    handler.buildUserResponse(user),
			"has_pin":     user.HasPin,
			"is_verified": user.IsVerified,
		},
	}

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}
