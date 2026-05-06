package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/responseFormat"
	"go.uber.org/zap"
)

const (
	accountBlockedCode = "ACCOUNT_BLOCKED"
	walletLockedCode   = "WALLET_LOCKED"
	productLockedCode  = "PRODUCT_NOT_PURCHASABLE"
)

var errAccountBlocked = errors.New("account is blocked")

func writeStructuredBlockedResponse(w http.ResponseWriter, status int, code, reason, popupMessage, message string) {
	payload := map[string]interface{}{
		"code":          code,
		"reason":        reason,
		"popup_message": popupMessage,
		"data":          message,
	}

	w.WriteHeader(status)
	response := responseFormat.CustomResponse{
		Status:  status,
		Message: "error",
		Data:    payload,
	}
	json.NewEncoder(w).Encode(response)
}

func writeAccountBlockedResponse(w http.ResponseWriter) {
	writeStructuredBlockedResponse(
		w,
		http.StatusForbidden,
		accountBlockedCode,
		"USER_BLOCKED",
		"Your account has been blocked. Please contact support.",
		"account blocked",
	)
}

func writeWalletLockedResponse(w http.ResponseWriter) {
	writeStructuredBlockedResponse(
		w,
		http.StatusConflict,
		walletLockedCode,
		"WALLET_STATUS_LOCKED",
		"Your wallet is locked and cannot be used for transactions right now.",
		"wallet locked",
	)
}

func writeProductLockedResponse(w http.ResponseWriter, reason, popupMessage string) {
	writeStructuredBlockedResponse(
		w,
		http.StatusConflict,
		productLockedCode,
		reason,
		popupMessage,
		"product not purchasable",
	)
}

func ensureUserNotBlocked(user *models.User) error {
	if user == nil {
		return nil
	}
	if user.IsBlocked {
		return errAccountBlocked
	}
	return nil
}

func writeBlockedErrorIfNeeded(w http.ResponseWriter, err error) bool {
	if errors.Is(err, errAccountBlocked) {
		writeAccountBlockedResponse(w)
		return true
	}
	return false
}

func (handler *HttpHandler) resolvePendingLoginUser(w http.ResponseWriter, ctx context.Context, token string) (*models.User, *pendingLoginState, bool) {
	user, state, err := handler.getPendingLoginUser(ctx, token)
	if err == nil {
		return user, state, true
	}
	if writeBlockedErrorIfNeeded(w, err) {
		return nil, nil, false
	}
	respondWithError(w, http.StatusUnauthorized, "login expired", err)
	return nil, nil, false
}

func (handler *HttpHandler) ensureWalletUnlocked(ctx context.Context, w http.ResponseWriter, userID string) bool {
	account, err := handler.store.GetVirtualNuban(ctx, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "failed to retrieve wallet access details"},
		}
		json.NewEncoder(w).Encode(response)
		return false
	}

	if strings.EqualFold(strings.TrimSpace(account.Status), "locked") {
		writeWalletLockedResponse(w)
		return false
	}

	return true
}

func (handler *HttpHandler) ensureProductCategoryUnlocked(ctx context.Context, w http.ResponseWriter, categoryKey, categoryLabel string) bool {
	globalLock, err := handler.productClient.GetProductLock(ctx, models.AllProductsPurchasesEnabled)
	if err != nil {
		handler.logger.Error("failed to retrieve global product lock", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "failed to retrieve product lock"},
		}
		json.NewEncoder(w).Encode(response)
		return false
	}
	if !globalLock.Enabled {
		writeProductLockedResponse(w, models.ProductLockReasonGlobalLock, "All products are currently unavailable.")
		return false
	}

	categoryLock, err := handler.productClient.GetProductLock(ctx, categoryKey)
	if err != nil {
		handler.logger.Error("failed to retrieve category product lock", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		response := responseFormat.CustomResponse{
			Status:  http.StatusInternalServerError,
			Message: "error",
			Data:    map[string]interface{}{"data": "failed to retrieve product lock"},
		}
		json.NewEncoder(w).Encode(response)
		return false
	}
	if !categoryLock.Enabled {
		writeProductLockedResponse(w, models.ProductLockReasonProductLock, fmt.Sprintf("%s are currently unavailable.", categoryLabel))
		return false
	}

	return true
}

func writeItemUnavailableResponse(w http.ResponseWriter, popupMessage string) {
	writeProductLockedResponse(w, models.ProductLockReasonItemUnavailable, popupMessage)
}
