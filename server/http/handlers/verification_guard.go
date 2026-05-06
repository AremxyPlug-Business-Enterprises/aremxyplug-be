package handlers

import (
	"net/http"

	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
)

const signupVerificationRequiredMessage = "complete signup verification before continuing"

func writeSignupVerificationRequired(w http.ResponseWriter) {
	writeError(w, http.StatusForbidden, signupVerificationRequiredMessage)
}

func ensureSignupVerified(w http.ResponseWriter, user *models.User) bool {
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return false
	}
	if err := ensureUserNotBlocked(user); err != nil {
		writeAccountBlockedResponse(w)
		return false
	}

	if user.IsVerified {
		return true
	}

	writeSignupVerificationRequired(w)
	return false
}

func (handler *HttpHandler) RequireVerifiedUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := handler.GetUserDetails(r)
		if err != nil {
			handler.logger.Warn("failed to resolve user details for verification guard", zap.Error(err))
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		if !ensureSignupVerified(w, user) {
			handler.logger.Warn("blocked unverified user from authenticated route", zap.String("user_id", user.ID))
			return
		}

		next.ServeHTTP(w, r)
	})
}
