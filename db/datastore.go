package db

import (
	"context"
	"time"

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
	CheckID(ctx context.Context, id int) (int64, error)
	SaveOTP(ctx context.Context, data models.OTP) error
	GetOTP(ctx context.Context, channel, target string) (models.OTP, error)
	SaveSMS(ctx context.Context, data models.SMSOTP) error
	GetSMS(ctx context.Context, phone string) (models.SMSOTP, error)
	GetPin(ctx context.Context, userID string) (string, error)
	UpdatePin(ctx context.Context, data models.UserPin) error
	SavePin(ctx context.Context, data models.UserPin) error
	CreateUserReferral(ctx context.Context, newUserID, referralCode string) error
	GetReferredUsers(ctx context.Context, referrerID string) ([]models.ReferredUserInfo, error)
	CreatePointDoc(ctx context.Context, userID string) error
	RedeemPoints(ctx context.Context, userID string, pointsToRedeem int, redeemRate int) (amountRedeemed int, e error)
	GetPoint(ctx context.Context, userID string) (models.PointSummary, error)
	UpdatePointAndTransactionTime(ctx context.Context, userID string, pointsEarned int) error
	CreatePointRedeemDoc(ctx context.Context, redeem models.PointRedeem) error
	LogPointTransaction(ctx context.Context, transaction models.PointTransaction) error
	GetPointTransactions(ctx context.Context, userID string, page int) ([]models.PointTransaction, error)
	UpdatePointAfterVerify(ctx context.Context, userID string) error
	GetPointRedeemDetails(ctx context.Context, orderID string) (models.PointRedeem, error)
	GetTotalPointsRedeemed(ctx context.Context, userID string) (int, error)
}

type BankStore interface {
	SaveBankList(ctx context.Context, banklist models.BankDetails) error
	GetBankDetail(ctx context.Context, bankName string) (models.BankDetails, error)
	SaveVirtualAccount(ctx context.Context, account models.AccountDetails) error
	GetVirtualNuban(ctx context.Context, id string) (models.AccountDetails, error)
	SaveCounterParty(ctx context.Context, counterparty interface{}) error
	SaveTransfer(ctx context.Context, transfer models.TransferResponse) error
	GetCounterParty(ctx context.Context, accountNumber, bankname string) (models.CounterParty, error)
	GetTransferDetails(ctx context.Context, id string) (models.TransferResponse, error)
	GetAllTransferHistory(ctx context.Context, user string) ([]models.TransferResponse, error)
	GetDepositDetails(ctx context.Context, id string) (any, error)
	GetAllDepositHistory(ctx context.Context, user string) ([]models.DepositResponse, error)
	GetAllBankTransactions(ctx context.Context, user string) ([]interface{}, error)
	SaveDeposit(ctx context.Context, detail any) error
	GetDepositID(ctx context.Context, virtualNuban string) (result interface{}, err error)
	SaveDepositID(ctx context.Context, detail interface{}) error
	GetBalance(ctx context.Context, userID string) (balance decimal.Decimal, err error)
	SaveBalance(ctx context.Context, userID string, balance models.Balance) error
	UpdateBalance(ctx context.Context, userID string, balance decimal.Decimal) error
	GetBalanceDetails(ctx context.Context, id string) (models.Balance, error)
	CreateInitialBalance(ctx context.Context, userID, virtualNuban string) error
	SaveTransferRecipient(ctx context.Context, userID, username, email, phone, fullName string) error
	GetTransferRecipients(ctx context.Context, userID string) ([]models.TransferRecipientDetails, error)
	DeleteTransferRecipient(ctx context.Context, userID, email string) error
	UpdateBank(ctx context.Context, bank models.BankDetails) error
	GetBankByNIPCode(ctx context.Context, nipCode string) (*models.BankDetails, error)
	DeleteBankByNIPCode(ctx context.Context, nipCode string) error
	GetAllBanks(ctx context.Context) ([]models.BankDetails, error)
	UpsertBankByNIPCode(ctx context.Context, bank models.BankDetails) error
}

type UserStore interface {
	SaveUser(ctx context.Context, user models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByPhone(ctx context.Context, phone string) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByUsernameOrEmail(ctx context.Context, email string, username string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	CreateMessage(ctx context.Context, message *models.Message) error
	UpdateUserPassword(ctx context.Context, email string, password string) error
	UpdateUserPasswordByID(ctx context.Context, id string, password string) error
	UpdateBVNField(ctx context.Context, user models.User) error
	UpdateNINField(ctx context.Context, user models.User) error
	GetUserByBVNHash(ctx context.Context, hash string) (*models.User, error)
	GetUserByNINHash(ctx context.Context, hash string) (*models.User, error)
	ListOtherVerifiedUsersByDOB(ctx context.Context, excludeUserID string, dob time.Time) ([]models.User, error)
	BackfillIdentityHashes(ctx context.Context) error
	VerifyUser(ctx context.Context, identifier string) (*models.User, error)
	UpdatePhone(ctx context.Context, id, phone string) error
	UpdateEmail(ctx context.Context, id, email string) error
	GetUserByUsernameOrEmailOrPhone(ctx context.Context, username, email, phone string) (*models.User, error)
	UpdateUserAddress(ctx context.Context, userID, gender, dob, address, postalCode string) error
	UpdateUserBeta(ctx context.Context, id string, beta bool) error
}

type TelcomStore interface {
	SaveDataTransaction(ctx context.Context, details interface{}) error
	GetDataTransactionDetails(ctx context.Context, id string) (telcom.DataResult, error)
	GetAllDataTransactions(ctx context.Context, username string) ([]telcom.DataResult, error)
	GetSpecTransDetails(ctx context.Context, id string) (telcom.SpectranetResult, error)
	GetAllSpecDataTransactions(ctx context.Context, username string) ([]telcom.SpectranetResult, error)
	GetSmileTransDetails(ctx context.Context, id string) (telcom.SmileResult, error)
	GetAllSmileDataTransactions(ctx context.Context, username string) ([]telcom.SmileResult, error)
	SaveAirtimeTransaction(ctx context.Context, details *telcom.AirtimeResponse) error
	GetAirtimeTransactionDetails(ctx context.Context, id string) (telcom.AirtimeResponse, error)
	GetAllAirtimeTransactions(ctx context.Context, username string) ([]telcom.AirtimeResponse, error)
	SaveTelcomRecipient(ctx context.Context, userID string, data telcom.Recipient) error
	GetTelcomRecipients(ctx context.Context, username string) (telcom.TelcomRecipient, error)
	EditTelcomRecipient(ctx context.Context, userID string, data telcom.Recipient) error
	DeleteTelcomRecipient(ctx context.Context, recipientID int, userID string) error
}

type UtilitiesStore interface {
	SaveEduTransaction(ctx context.Context, details *models.EduResponse) error
	GetEduTransactionDetails(ctx context.Context, id string) (models.EduResponse, error)
	GetAllEduTransactions(ctx context.Context, user string) ([]models.EduResponse, error)
	SaveTVSubcriptionTransaction(ctx context.Context, details *models.TV_Result) error
	GetTvSubscriptionDetails(ctx context.Context, id string) (models.TV_Result, error)
	GetAllTvSubTransactions(ctx context.Context, user string) ([]models.TV_Result, error)
	SaveElectricTransaction(ctx context.Context, details *models.ElectricResult) error
	GetElectricSubDetails(ctx context.Context, id string) (models.ElectricResult, error)
	GetAllElectricSubTransactions(ctx context.Context, username string) ([]models.ElectricResult, error)
}

type TransactionStore interface {
	GetTransactions(ctx context.Context, filter map[string]interface{}, page, pageSize int) (models.TransactionResponse, error)
	GetSalesSummary(ctx context.Context, category string, filter map[string]interface{}, page int) (models.SalesSummary, error)
	GetWalletSummary(ctx context.Context, filter map[string]interface{}, page int) (models.TransactionResponse, error)
	GetSalesOverview(ctx context.Context, filter map[string]interface{}) (models.SalesSummary, error)
	GetChart(ctx context.Context, filter map[string]interface{}, rangeType string) (models.StatsResponse, error)
}

type WebhookStore interface {
	GetReceiptByExternalRef(ctx context.Context, externalRef string) (models.TransferResponse, error)
	GetReceiptByTxID(ctx context.Context, txID string) (models.TransferResponse, error)
	UpdateReceiptFinal(ctx context.Context, txID, status, sessionID string) error
	UpdateRecieptByTxID(ctx context.Context, txnID string) error
	UpdateUserBalanceFromRedis(ctx context.Context, userID string, balance decimal.Decimal) error
	UpdateReceiptStatus(ctx context.Context, txID, status string) error
	UpdateReceiptExternalRef(ctx context.Context, externalRef, status string) error
	GetUserFromVirtualNuban(ctx context.Context, virtualNuban string) (string, error)
}

type TaskStore interface {
	// GetProgress retrieves the current progress document for a user and task.
	GetProgress(ctx context.Context, userID string, task models.TaskType) (*models.ProgressDoc, error)
	// IncrementCumulative atomically increments a cumulative task and returns the updated doc.
	IncrementCumulative(ctx context.Context, userID string, def models.TaskDef, delta int64, txID string) (*models.ProgressDoc, error)
	// SetProgressForOneTime sets the progress for a one-time task and returns the updated doc.
	SetProgressForOneTime(ctx context.Context, userID string, def models.TaskDef, value int64, txID string) (*models.ProgressDoc, error)
	// TryMarkCompleted attempts to mark a task as completed for a user.
	TryMarkCompleted(ctx context.Context, userID string, task models.TaskType) (bool, error)
	// ListUserProgress retrieves all task progress documents for a user.
	ListUserProgress(ctx context.Context, userID string) ([]models.ProgressDoc, error)
}
