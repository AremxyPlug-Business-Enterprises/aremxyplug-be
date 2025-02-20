package nin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
)

var (
	baseUrl = os.Getenv("PREMBLY_BASE_URL")
	apiKey  = os.Getenv("PREMBLY_LIVE_APIKEY")
	appID   = os.Getenv("PREMBLY_APP_ID")
)

type NINConfig struct {
	logger *zap.Logger
}

func NewNINConfig(logger *zap.Logger) *NINConfig {
	return &NINConfig{
		logger: logger,
	}
}

func (n *NINConfig) VerifyNIN(nin string, user models.User) (result *NINVerificationResult, err error) {

	payload := ninRequest{
		NIN_number: nin,
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		n.logger.Error("Failed to marshal payload", zap.Error(err))
		return &NINVerificationResult{}, err
	}

	url := fmt.Sprintf("%s/%s", baseUrl, "vnin")
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return &NINVerificationResult{}, err
	}
	req.Header.Set("app-id", appID)
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return &NINVerificationResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		n.logger.Error("API call returned non-OK status", zap.Int("statusCode", resp.StatusCode))
		return &NINVerificationResult{
			Success:      false,
			NameMatched:  false,
			ResponseCode: resp.Status,
		}, nil
	}

	apiResponse := &ninResponse{}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		n.logger.Error("Failed to read response body", zap.Error(err))
		return &NINVerificationResult{}, err
	}

	if err := json.NewDecoder(bytes.NewBuffer(bodyBytes)).Decode(&apiResponse); err != nil {
		n.logger.Error("Failed to decode response", zap.Error(err))
		return &NINVerificationResult{}, err
	}

	if !apiResponse.Status {
		n.logger.Error("NIN verification failed", zap.Any("response", apiResponse.Detail))
		return &NINVerificationResult{
			Success:         false,
			NameMatched:     false,
			ResponseCode:    "400",
			ResponseMessage: "Verification failed",
		}, nil
	}

	n.logger.Info("API Response", zap.Any("response", apiResponse.Detail))

	apiFullName := fmt.Sprintf("%s %s %s", apiResponse.NINData.FirstName, apiResponse.NINData.MiddleName, apiResponse.NINData.Surname)
	fullName := user.FullName

	if !compareNames(apiFullName, fullName) {
		n.logger.Error("User's name does not match NIN data", zap.String("apiFullName", apiFullName), zap.String("FullName", fullName))
		return &NINVerificationResult{
			Success:         true,
			NameMatched:     false,
			ResponseCode:    "400",
			ResponseMessage: "Name does not match BVN name",
		}, nil
	}

	n.logger.Info("User name matches NIN data", zap.String("apiFullName", apiFullName), zap.String("FullName", fullName))

	return &NINVerificationResult{
		Success:         true,
		NameMatched:     true,
		ResponseCode:    "200",
		ResponseMessage: "NIN verification succesful",
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
