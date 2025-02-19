package termii

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
)

var (
	apiKey  = os.Getenv("TERMII_API_KEY")
	baseUrl = os.Getenv("TERMII_SMS_BASE_URL")
)

type SMSConn struct {
	dbconn db.Extras
	logger *zap.Logger
}

func NewSMSConn(store db.DataStore, logger *zap.Logger) *SMSConn {
	return &SMSConn{
		dbconn: store,
		logger: logger,
	}
}

func (s *SMSConn) SendSMS(phoneNo string) error {
	s.logger.Info("Sending SMS", zap.String("phoneNo", phoneNo))

	message_text := fmt.Sprintf("Your pin is < 123456 >, it expires in %d minutes", 5)

	payload := smsRequest{
		APIKey:         apiKey,
		MessageType:    "NUMERIC",
		To:             phoneNo,
		From:           "AREMXYPLUG",
		Channel:        "generic",
		PinAttempts:    5,
		PinTimeToLive:  5,
		PinLength:      6,
		PinPlaceholder: "< 123456 >",
		MessageText:    message_text,
		PinType:        "NUMERIC",
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("Failed to marshal payload", zap.Error(err))
		return err
	}

	url := fmt.Sprintf("%s/send", baseUrl)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		s.logger.Error("Failed to create new request", zap.Error(err))
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("Failed to send request", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("API call returned non-OK status", zap.Int("statusCode", resp.StatusCode))
		return errors.New("error during API call")
	}

	apiResponse := smsClientResponse{}

	// Log the response Body
	s.logger.Info("response status code", zap.Int("status code", resp.StatusCode))
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read response body", zap.Error(err))
		return err
	}
	s.logger.Info("Response Body", zap.ByteString("body", bodyBytes))

	// Decode the response body
	if err := json.NewDecoder(bytes.NewBuffer(bodyBytes)).Decode(&apiResponse); err != nil {
		s.logger.Error("Failed to decode response", zap.Error(err))
		return err
	}

	/*
		if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
			s.logger.Error("Failed to decode response", zap.Error(err))
			return err
		}
	*/

	otpDetails := models.SMSOTP{
		PinID: apiResponse.PinID,
		Phone: apiResponse.To,
	}

	if err := s.dbconn.SaveSMS(otpDetails); err != nil {
		s.logger.Error("Failed to save SMS OTP details", zap.Error(err))
		return err
	}

	s.logger.Info("SMS sent and saved successfully", zap.String("pinID", apiResponse.PinID))
	return nil
}

func (s *SMSConn) VerifyToken(otp, phone string) error {
	s.logger.Info("Verifying token", zap.String("phone", phone), zap.String("otp", otp))

	otpDetails, err := s.dbconn.GetSMS(phone)
	if err != nil {
		s.logger.Error("Failed to get SMS OTP details", zap.Error(err))
		return err
	}

	payload := verifyTokenRequest{
		APIKey: apiKey,
		PinID:  otpDetails.PinID,
		Pin:    otp,
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("Failed to marshal payload", zap.Error(err))
		return err
	}

	url := fmt.Sprintf("%s/verify", baseUrl)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		s.logger.Error("Failed to create new request", zap.Error(err))
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("Failed to send request", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	apiResponse := verifyTokenResponse{}

	// Log the response Body
	s.logger.Info("response status code", zap.Int("status code", resp.StatusCode))
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read response body", zap.Error(err))
		return err
	}
	s.logger.Info("Response Body", zap.ByteString("body", bodyBytes))

	// Decode the response body
	if err := json.NewDecoder(bytes.NewBuffer(bodyBytes)).Decode(&apiResponse); err != nil {
		s.logger.Error("Failed to decode response", zap.Error(err))
		return err
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("API call returned non-OK status", zap.Int("statusCode", resp.StatusCode))
		return errors.New("error during API call")
	}

	// if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
	// 	s.logger.Error("Failed to decode response", zap.Error(err))
	// 	return err
	// }

	if !apiResponse.Verified {
		s.logger.Warn("Verification failed", zap.String("phone", phone), zap.String("otp", otp))
		return errors.New("verification failed")
	}

	s.logger.Info("Token verified successfully", zap.String("phone", phone))
	return nil
}
