package db

import "github.com/aremxyplug-be/db/models"

type DataStore interface {
	AremxyStore
}

type AremxyStore interface {
	SaveUser(user models.User) error
	GetUserByEmail(email string) (*models.User, error)
	CreateMessage(message *models.Message) error
	UpdateUserPassword(email string, password string) error
	SaveDataTransaction(details *models.DataResult) error
	GetDataTransactionDetails(id string) (models.DataResult, error)
	GetAllDataTransactions(user string) ([]models.DataResult, error)
	SaveEduTransaction(details *models.EduResponse) error
	GetEduTransactionDetails(id string) (models.EduResponse, error)
	GetAllEduTransactions(user string) ([]models.EduResponse, error)
}
