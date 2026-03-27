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
	TransactionStore
	WebhookStore
	TaskStore
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
	CreateUserReferral(newUserID, referralCode string) error
	GetReferredUsers(referrerID string) ([]models.ReferredUserInfo, error)
	CreatePointDoc(userID string) error
	RedeemPoints(userID string, pointsToRedeem int, redeemRate int) (amountRedeemed int, e error)
	GetPoint(userID string) (models.PointSummary, error)
	UpdatePointAndTransactionTime(userID string, pointsEarned int) error
	CreatePointRedeemDoc(redeem models.PointRedeem) error
	LogPointTransaction(transaction models.PointTransaction) error
	GetPointTransactions(userID string, page int) ([]models.PointTransaction, error)
	UpdatePointAfterVerify(userID string) error
	GetPointRedeemDetails(orderID string) (models.PointRedeem, error)
	GetTotalPointsRedeemed(userID string) (int, error)
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
	GetDepositDetails(id string) (any, error)
	GetAllDepositHistory(user string) ([]models.DepositResponse, error)
	GetAllBankTransactions(user string) ([]interface{}, error)
	SaveDeposit(detail any) error
	GetDepositID(virtualNuban string) (result interface{}, err error)
	SaveDepositID(detail interface{}) error
	GetBalance(userID string) (balance decimal.Decimal, err error)
	SaveBalance(userID string, balance models.Balance) error
	UpdateBalance(userID string, balance decimal.Decimal) error
	GetBalanceDetails(id string) (models.Balance, error)
	CreateInitialBalance(userID, virtualNuban string) error
	SaveTransferRecipient(userID, username, email, phone, fullName string) error
	GetTransferRecipients(userID string) ([]models.TransferRecipientDetails, error)
	DeleteTransferRecipient(userID, email string) error
	UpdateBank(bank models.BankDetails) error
	GetBankByNIPCode(nipCode string) (*models.BankDetails, error)
	DeleteBankByNIPCode(nipCode string) error
	GetAllBanks() ([]models.BankDetails, error)
	UpsertBankByNIPCode(bank models.BankDetails) error
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
	UpdateUserPasswordByID(id string, password string) error
	UpdateBVNField(user models.User) error
	UpdateNINField(user models.User) error
	VerifyUser(identifier string) (*models.User, error)
	UpdatePhone(id, phone string) error
	UpdateEmail(id, email string) error
	GetUserByUsernameOrEmailOrPhone(username, email, phone string) (*models.User, error)
	UpdateUserAddress(userID string, gender string, dob string, address string, postalCode string) error
	UpdateUserBeta(id string, beta bool) error
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
	SaveTVSubcriptionTransaction(details *models.TV_Result) error
	GetTvSubscriptionDetails(id string) (models.TV_Result, error)
	GetAllTvSubTransactions(user string) ([]models.TV_Result, error)
	SaveElectricTransaction(details *models.ElectricResult) error
	GetElectricSubDetails(id string) (models.ElectricResult, error)
	GetAllElectricSubTransactions(username string) ([]models.ElectricResult, error)
}

type TransactionStore interface {
	GetTransactions(filter map[string]interface{}, page, pageSize int) (models.TransactionResponse, error)
	GetSalesSummary(category string, filter map[string]interface{}, page int) (models.SalesSummary, error)
	GetWalletSummary(filter map[string]interface{}, page int) (models.TransactionResponse, error)
	GetSalesOverview(filter map[string]interface{}) (models.SalesSummary, error)
	GetChart(filter map[string]interface{}, rangeType string) (models.StatsResponse, error)
}

type WebhookStore interface {
	GetReceiptByExternalRef(externalRef string) (models.TransferResponse, error)
	GetReceiptByTxID(txID string) (models.TransferResponse, error)
	UpdateReceiptFinal(txID, status, sessionID string) error
	UpdateRecieptByTxID(txnID string) error
	UpdateUserBalanceFromRedis(userID string, balance decimal.Decimal) error
	UpdateReceiptStatus(txID, status string) error
	UpdateReceiptExternalRef(externalRef, status string) error
	GetUserFromVirtualNuban(virtualNuban string) (string, error)
}

type TaskStore interface {
	// GetProgress retrieves the current progress document for a user and task.
	GetProgress(userID string, task models.TaskType) (*models.ProgressDoc, error)
	// IncrementCumulative atomically increments a cumulative task and returns the updated doc.
	IncrementCumulative(userID string, def models.TaskDef, delta int64, txID string) (*models.ProgressDoc, error)
	// SetProgressForOneTime sets the progress for a one-time task and returns the updated doc.
	SetProgressForOneTime(userID string, def models.TaskDef, value int64, txID string) (*models.ProgressDoc, error)
	// TryMarkCompleted attempts to mark a task as completed for a user.
	TryMarkCompleted(userID string, task models.TaskType) (bool, error)
	// ListUserProgress retrieves all task progress documents for a user.
	ListUserProgress(userID string) ([]models.ProgressDoc, error)
}
