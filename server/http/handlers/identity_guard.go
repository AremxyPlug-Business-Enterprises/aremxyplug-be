package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/responseFormat"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const duplicateIdentityMessage = "existing account found, contact admin"

func normalizeNameParts(name string) map[string]struct{} {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(name)))
	normalized := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		normalized[part] = struct{}{}
	}
	return normalized
}

func sharedNamePartCount(left, right string) int {
	leftParts := normalizeNameParts(left)
	rightParts := normalizeNameParts(right)

	count := 0
	for part := range leftParts {
		if _, ok := rightParts[part]; ok {
			count++
		}
	}

	return count
}

func hasDOBNameConflict(currentUser, otherUser models.User) bool {
	if currentUser.ID == otherUser.ID {
		return false
	}
	if currentUser.DOB.IsZero() || otherUser.DOB.IsZero() {
		return false
	}
	if !currentUser.DOB.Equal(otherUser.DOB) {
		return false
	}

	return sharedNamePartCount(currentUser.FullName, otherUser.FullName) >= 2
}

func (handler *HttpHandler) findExactIdentityConflict(ctx context.Context, identityType, hash, currentUserID string) (*models.User, error) {
	var (
		user *models.User
		err  error
	)

	switch identityType {
	case "bvn":
		user, err = handler.store.GetUserByBVNHash(ctx, hash)
	case "nin":
		user, err = handler.store.GetUserByNINHash(ctx, hash)
	default:
		return nil, nil
	}

	switch {
	case err == mongo.ErrNoDocuments:
		return nil, nil
	case err != nil:
		return nil, err
	case user != nil && user.ID != "" && user.ID != currentUserID:
		return user, nil
	default:
		return nil, nil
	}
}

func (handler *HttpHandler) findDOBNameConflict(ctx context.Context, currentUser models.User, dob string) (*models.User, error) {
	parsedDOB, err := time.Parse("2006-01-02", dob)
	if err != nil {
		return nil, err
	}

	currentUser.DOB = parsedDOB

	users, err := handler.store.ListOtherVerifiedUsersByDOB(ctx, currentUser.ID, parsedDOB)
	if err != nil {
		return nil, err
	}

	for _, candidate := range users {
		if hasDOBNameConflict(currentUser, candidate) {
			candidateCopy := candidate
			return &candidateCopy, nil
		}
	}

	return nil, nil
}

func (handler *HttpHandler) respondDuplicateIdentityConflict(w http.ResponseWriter, currentUserID, verificationType, conflictType string, matchedUser *models.User) {
	fields := []zap.Field{
		zap.String("user_id", currentUserID),
		zap.String("verification_type", verificationType),
		zap.String("conflict_type", conflictType),
	}
	if matchedUser != nil && matchedUser.ID != "" {
		fields = append(fields, zap.String("matched_user_id", matchedUser.ID))
	}

	handler.logger.Warn("duplicate identity verification blocked", fields...)

	w.WriteHeader(http.StatusConflict)
	response := responseFormat.CustomResponse{
		Status:  http.StatusConflict,
		Message: "error",
		Data:    map[string]interface{}{"data": duplicateIdentityMessage},
	}
	json.NewEncoder(w).Encode(response)
}
