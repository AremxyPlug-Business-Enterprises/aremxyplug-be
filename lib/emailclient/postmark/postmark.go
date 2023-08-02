package postmark

import (
	"errors"

	"github.com/aremxyplug-be/config"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/emailclient"
	"github.com/aremxyplug-be/lib/logger"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

const (
	postmarkAPIURL                = "https://api.postmarkapp.com"
	sendEmailWithTemplateEndpoint = "/email/withTemplate/"
)

// Ensure implementation of EmailClient interface
var _ emailclient.EmailClient = (*emailClient)(nil)

type emailClient struct {
	RESTClient *resty.Client
	secret     *config.Secrets
}

// Send generate and send a new email message using postmark API
func (e *emailClient) Send(message *models.Message) error {
	// Build request
	logService := logger.New()
	if message == nil {
		return errors.New("message it's empty")
	}
	request := EmailWithTemplateRequest{
		TemplateAlias: message.TemplateID,
		TemplateModel: map[string]interface{}{
			"Data": message.DataMap,
		},
		From: e.secret.PlatformEmail,
		To:   message.Target,
	}

	if len(message.Attachments) > 0 {
		attachments := make([]EmailTemplateAttachment, len(message.Attachments))
		for i, attachment := range message.Attachments {
			attachments[i] = EmailTemplateAttachment{
				Name:        attachment.Name,
				Content:     attachment.Content,
				ContentType: attachment.ContentType,
			}
		}
		request.Attachments = attachments
	}

	// Execute call to postmark API
	var result EmailWithTemplateResponse
	var errorResponse ErrorResponse
	response, err := e.RESTClient.R().
		SetBody(request).
		SetResult(&result).
		SetError(&errorResponse).
		Post(sendEmailWithTemplateEndpoint)
	if err != nil {
		return err
	}
	if response.IsError() {
		//return fmt.Errorf("postmark call response error with code: %d, message: %s", errorResponse.ErrorCode,
		//	errorResponse.Message)
		logService.Error("Error sending email", zap.String("Message", errorResponse.Message), zap.Int64("Code", int64(errorResponse.ErrorCode)))
		return nil
	}

	// Check https://postmarkapp.com/developer/api/overview#error-codes for error codes
	if result.ErrorCode > 0 {
		//return fmt.Errorf("postmark call response error with message_id: %s, message_body: %s", result.MessageID, result.Message)
		logService.Error("Error sending email", zap.String("Message", errorResponse.Message), zap.Int64("Code", int64(errorResponse.ErrorCode)))
		return nil
	}
	return nil
}

// New return a new instance of a Postmark definition for EmailClient interface
func New(secrets *config.Secrets) emailclient.EmailClient {
	// Build REST client
	restClient := resty.New()
	restClient.SetBaseURL(postmarkAPIURL)
	restClient.SetHeader("Content-Type", "application/json")
	restClient.SetHeader("Accept", "application/json")
	restClient.SetHeader("X-Postmark-Server-Token", secrets.PostmarkKey)
	restClient.SetDebug(true)

	// Define service attributes
	emailClient := emailClient{
		RESTClient: restClient,
		secret:     secrets,
	}

	return &emailClient
}
