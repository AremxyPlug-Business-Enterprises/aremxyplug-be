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
	SaveDataTransaction(details interface{}) error
	GetDataTransactionDetails(id string) (models.DataResult, error)
	GetAllDataTransactions(user string) ([]models.DataResult, error)
	GetSpecTransDetails(id string) (models.SpectranetResult, error)
	GetAllSpecDataTransactions(user string) ([]models.SpectranetResult, error)
	GetSmileTransDetails(id string) (models.SmileResult, error)
	GetAllSmileDataTransactions(user string) ([]models.SmileResult, error)
	SaveEduTransaction(details *models.EduResponse) error
	GetEduTransactionDetails(id string) (models.EduResponse, error)
	GetAllEduTransactions(user string) ([]models.EduResponse, error)
	SaveAirtimeTransaction(details *models.AirtimeResponse) error
	GetAirtimeTransactionDetails(id string) (models.AirtimeResponse, error)
	GetAllAirtimeTransactions(user string) ([]models.AirtimeResponse, error)
	SaveTVSubcriptionTransaction(details *models.BillResult) error
	GetTvSubscriptionDetails(id string) (models.BillResult, error)
	GetAllTvSubTransactions(user string) ([]models.BillResult, error)
	SaveElectricTransaction(details *models.ElectricResult) error
	GetElectricSubDetails(id string) (models.ElectricResult, error)
	GetAllElectricSubTransactions(user string) ([]models.ElectricResult, error)
	SaveOTP(data models.OTP) error
	GetOTP(email string) (models.OTP, error)
}
hjbhjb