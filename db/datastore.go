package db

import (
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/models/telcom"
	"github.com/shopspring/decimal"
)

type DataStore interface {
	Extras
	BankStore
	UserStore
	TelcomStore
	UtilitiesStore
}

type Extras interface {
	CheckID(id int) (int64, error)
	SaveOTP(data models.OTP) error
	GetOTP(email string) (models.OTP, error)
	SaveSMS(data models.SMSOTP) error
	GetSMS(phone string) (models.SMSOTP, error)
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
	GetVirtualNuban(id string) (models.AccountDetails, error)
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
	GetBalance(userID string) (balance decimal.Decimal, err error)
	SaveBalance(userID string, balance models.Balance) error
	UpdateBalance(userID string, balance decimal.Decimal) error
	GetBalanceDetails(id string) (models.Balance, error)
	CreateInitialBalance(userID, virtualNuban string) error
}

type UserStore interface {
	SaveUser(user models.User) error
	GetUserByEmail(email string) (*models.User, error)
	GetUserByPhone(phone string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	GetUserByUsernameOrEmail(email string, username string) (*models.User, error)
	GetUserByID(id string) (*models.User, error)
	CreateMessage(message *models.Message) error
	UpdateUserPassword(email string, password string) error
	UpdateBVNField(user models.User) error
	UpdateNINField(user models.User) error
	VerifyUser(identifier string) (*models.User, error)
	GetUserByUsernameOrEmailOrPhone(username, email, phone string) (*models.User, error)
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
	GetAllElectricSubTransactions(username string) ([]models.ElectricResult, error)
}
