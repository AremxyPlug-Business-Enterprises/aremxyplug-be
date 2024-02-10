package db

import "github.com/aremxyplug-be/db/models"

type DataStore interface {
	AremxyStore
}

type AremxyStore interface {
	SaveUser(user models.User) error
	GetUserByEmail(email string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	GetUserByUsernameOrEmail(email string, username string) (*models.User, error)
	GetUserByID(id string) (*models.User, error)
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
	SaveBankList(banklist models.BankDetails) error
	GetBankDetail(bankName string) (models.BankDetails, error)
	SaveVirtualAccount(account models.AccountDetails) error
	GetVirtualNuban(name string) (string, error)
	SaveCounterParty(counterparty interface{}) error
	SaveTransfer(transfer models.TransferResponse) error
	GetCounterParty(accountNumber, bankname string) (models.CounterParty, error)
	GetTransferDetails(id string) (models.TransferResponse, error)
	GetAllTransferHistory(user string) ([]models.TransferResponse, error)
	GetDepositDetails(id string) (models.DepositResponse, error)
	GetAllDepositHistory(user string) ([]models.DepositResponse, error)
	GetAllBankTransactions(user string) ([]interface{}, error)
	SaveDeposit(detail models.DepositResponse) error
	GetDepositID(virtualNuban string) (result interface{}, err error)
	SaveDepositID(detail interface{}) error
	GetBalance(virtualNuban string) (balance float64, err error)
	SaveBalance(virtualNuban string, balance models.Balance) error
	UpdateBalance(virtualNuban string, balance float64) error
}

// Path: db/datastore.go
