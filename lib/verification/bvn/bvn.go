package bvn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
)

var (
	baseUrl = os.Getenv("PREMBLY_BASE_URL")
	apiKey  = os.Getenv("PREMBLY_LIVE_APIKEY")
	appID   = os.Getenv("PREMBLY_APP_ID")
)

type BvnConfig struct {
	logger *zap.Logger
}

func NewBvnConfig(logger *zap.Logger) *BvnConfig {
	return &BvnConfig{
		logger: logger,
	}
}

func (b *BvnConfig) VerifyBVN(bvn string, user models.User) (result *BVNVerificationResult, err error) {

	payload := bvnRequest{
		BVN_number: bvn,
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		b.logger.Error("Failed to marshal payload", zap.Error(err))
		return &BVNVerificationResult{}, err
	}

	url := fmt.Sprintf("%s/%s", baseUrl, "bvn_validation")
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return &BVNVerificationResult{}, err
	}
	req.Header.Set("app-id", appID)
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return &BVNVerificationResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b.logger.Error("API call returned non-OK status", zap.Int("statusCode", resp.StatusCode))
		return &BVNVerificationResult{
			Success:      false,
			NameMatched:  false,
			ResponseCode: resp.Status,
		}, nil
	}

	apiResponse := &bvnResponse{}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		b.logger.Error("Failed to read response body", zap.Error(err))
		return &BVNVerificationResult{}, err
	}

	if err := json.NewDecoder(bytes.NewBuffer(bodyBytes)).Decode(&apiResponse); err != nil {
		b.logger.Error("Failed to decode response", zap.Error(err))
		return &BVNVerificationResult{}, err
	}

	if !apiResponse.Status {
		b.logger.Error("NIN verification failed", zap.Any("response", apiResponse.Detail))
		return &BVNVerificationResult{
			Success:         false,
			NameMatched:     false,
			ResponseCode:    "400",
			ResponseMessage: "Verification failed",
		}, nil
	}

	b.logger.Info("API Response", zap.Any("response", apiResponse.Detail))

	apiFullName := fmt.Sprintf("%s %s %s", apiResponse.Data.FirstName, apiResponse.Data.MiddleName, apiResponse.Data.LastName)

	if !compareNames(apiFullName, user.FullName) {
		b.logger.Error("User's name does not match BVN data", zap.String("apiFullName", apiFullName), zap.String("userFullName", user.FullName))
		return &BVNVerificationResult{
			Success:         true,
			NameMatched:     false,
			ResponseCode:    "400",
			ResponseMessage: "Name does not match BVN name",
		}, nil
	}

	b.logger.Info("User name matches BVN data", zap.String("apiFullName", apiFullName), zap.String("FullName", user.FullName))

	dob := apiResponse.Data.DateOfBirth
	formattedDOB, err := formatDOB(dob)
	if err != nil {
		b.logger.Error("Failed to format date of birth", zap.Error(err))
		return &BVNVerificationResult{}, err
	}
	return &BVNVerificationResult{
		Success:         true,
		NameMatched:     true,
		DOB:             formattedDOB,
		ResponseCode:    "200",
		ResponseMessage: "BVN verification succesful",
	}, nil
}

func compareNames(apiFullName, userFullName string) bool {
	apiParts := toLowerSlice(strings.Fields(apiFullName))
	userParts := toLowerSlice(strings.Fields(userFullName))
	return isSubset(userParts, apiParts) || isSubset(apiParts, userParts)
}

func toLowerSlice(slice []string) []string {
	lowerSlice := make([]string, len(slice))
	for i, s := range slice {
		lowerSlice[i] = strings.ToLower(s)
	}
	return lowerSlice
}

func isSubset(sliceA, sliceB []string) bool {
	set := make(map[string]struct{})
	for _, s := range sliceB {
		set[s] = struct{}{}
	}

	for _, s := range sliceA {
		if _, exists := set[s]; !exists {
			return false
		}
	}
	return true
}

func formatDOB(d string) (string, error) {
	t, err := time.Parse("02-Jan-2006", d)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02"), nil
}
