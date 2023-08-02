package emailclient

import "github.com/aremxyplug-be/db/models"

// EmailClient interface
type EmailClient interface {
	Send(email *models.Message) error
}
