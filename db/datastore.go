package db

import "github.com/aremxyplug-be/db/models"

type DataStore interface {
	AremxyStore
}

type AremxyStore interface {
	SaveUser(user models.User) error
	GetUserByEmail(email string) (*models.User, error)
	CreateMessage(message *models.Message) error
	SaveTransaction(details *models.DataResult) error
	GetTransactionDetails(id string) (result models.DataResult, err error)
	GetAllTransactions(user string) ([]models.DataResult, error)
}
