package db

import (
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/models/telcom"
)

type DataStore interface {
	Extras
	BankStore
	UserStore
	TelcomStore
	UtilitiesStore
}

type Extras interface {
	SaveOTP(data models.OTP) error
	GetOTP(email string) (models.OTP, error)
	GetPin(userID string) (string, error)
	UpdatePin(data models.UserPin) error
	SavePin(data models.UserPin) error
	UpdateReferralCount(referralCode string) error
	CreateUserReferral(userID, refcode string) error
	GetReferral(userID string) (string, error)
	UpdatePoint(userID string, points int) error
	CreatePointDoc(userID string) error
	CanRedeemPoints(userID string, points int) bool
	GetPoint(userID string) (models.Points, error)
}

type BankStore interface {
	SaveBankList(banklist models.BankDetails) error
	GetBankDetail(bankName string) (models.BankDetails, error)
	SaveVirtualAccount(account models.AccountDetails) error
	GetVirtualNuban(name string) (models.AccountDetails, error)
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

type UserStore interface {
	SaveUser(user models.User) error
	GetUserByEmail(email string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	GetUserByUsernameOrEmail(email string, username string) (*models.User, error)
	GetUserByID(id string) (*models.User, error)
	CreateMessage(message *models.Message) error
	UpdateUserPassword(email string, password string) error
	UpdateBVNField(user models.User) error
}

type TelcomStore interface {
	SaveDataTransaction(details interface{}) error
	GetDataTransactionDetails(id string) (telcom.DataResult, error)
	GetAllDataTransactions(username string) ([]telcom.DataResult, error)
	GetSpecTransDetails(id string) (telcom.SpectranetResult, error)
	GetAllSpecDataTransactions(username string) ([]telcom.SpectranetResult, error)
	GetSmileTransDetails(id string) (telcom.SmileResult, error)
	GetAllSmileDataTransactions(username string) ([]telcom.SmileResult, error)
	SaveAirtimeTransaction(details *telcom.AirtimeResponse) error
	GetAirtimeTransactionDetails(id string) (telcom.AirtimeResponse, error)
	GetAllAirtimeTransactions(username string) ([]telcom.AirtimeResponse, error)
	SaveTelcomRecipient(userID string, data telcom.Recipient) error
	GetTelcomRecipients(username string) (telcom.TelcomRecipient, error)
	EditTelcomRecipient(userID string, data telcom.Recipient) error
	DeleteTelcomRecipient(recipientID int, userID string) error
}

type UtilitiesStore interface {
	SaveEduTransaction(details *models.EduResponse) error
	GetEduTransactionDetails(id string) (models.EduResponse, error)
	GetAllEduTransactions(user string) ([]models.EduResponse, error)
	SaveTVSubcriptionTransaction(details *models.BillResult) error
	GetTvSubscriptionDetails(id string) (models.BillResult, error)
	GetAllTvSubTransactions(user string) ([]models.BillResult, error)
	SaveElectricTransaction(details *models.ElectricResult) error
	GetElectricSubDetails(id string) (models.ElectricResult, error)
	GetAllElectricSubTransactions(user string) ([]models.ElectricResult, error)
}
