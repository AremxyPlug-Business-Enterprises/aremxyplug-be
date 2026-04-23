package termii

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"go.uber.org/zap"
)

var (
	whatsAppBaseURL = os.Getenv("TERMII_WHATSAPP_BASE_URL")
	whatsAppSender  = os.Getenv("TERMII_WHATSAPP_SENDER")
	whatsAppChannel = os.Getenv("TERMII_WHATSAPP_CHANNEL")
	whatsAppType    = os.Getenv("TERMII_WHATSAPP_TYPE")
)

type WhatsAppClient struct {
	logger *zap.Logger
}

func NewWhatsAppClient(logger *zap.Logger) *WhatsAppClient {
	return &WhatsAppClient{
		logger: logger,
	}
}

func (w *WhatsAppClient) SendWhatsAppToken(ctx context.Context, otp, phone string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	w.logger.Info("Sending WhatsApp OTP with Termii", zap.String("phone", phone))

	payload := whatsAppRequest{
		APIKey:  apiKey,
		To:      phone,
		From:    whatsAppSender,
		SMS:     otp,
		Channel: whatsAppChannel,
		Type:    whatsAppType,
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		w.logger.Error("Failed to marshal WhatsApp payload", zap.Error(err))
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, whatsAppBaseURL, bytes.NewReader(requestBody))
	if err != nil {
		w.logger.Error("Failed to create WhatsApp request", zap.Error(err))
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		w.logger.Error("Failed to send WhatsApp request", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		w.logger.Error("Failed to read WhatsApp response body", zap.Error(err))
		return err
	}

	w.logger.Info("WhatsApp response received", zap.Int("status_code", resp.StatusCode), zap.String("body", string(bodyBytes)))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("termii whatsapp send failed with status %d", resp.StatusCode)
	}

	apiResponse := whatsAppResponse{}
	if err := json.NewDecoder(bytes.NewBuffer(bodyBytes)).Decode(&apiResponse); err != nil {
		w.logger.Error("Failed to decode WhatsApp response body", zap.Error(err))
		return err
	}

	if apiResponse.Code != "" && apiResponse.Code != "ok" {
		return fmt.Errorf("termii whatsapp send failed: %s", apiResponse.Message)
	}

	w.logger.Info("WhatsApp OTP sent successfully", zap.String("phone", phone), zap.String("message_id", apiResponse.MessageID))
	return nil
}
