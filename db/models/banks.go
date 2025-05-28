package models

import (
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TransferInfo struct {
	UserID         string
	Bank_name      string  `json:"bank_name"`
	Account_Number string  `json:"account_number"`
	Account_Name   string  `json:"account_name"`
	Amount         float64 `json:"amount"`
	Reason         string  `json:"message"`
}

type TransferResponse struct {
	UserID         string    `json:"user_id" bson:"user_id"`
	Bank_Name      string    `json:"bank_name" bson:"bank_name"`
	Account_Number string    `json:"account_number" bson:"account_number"`
	Account_Name   string    `json:"account_name" bson:"account_name"`
	Account_No     string    `json:"account_no" bson:"account_no"`
	Name           string    `json:"name" bson:"name"`
	Amount         string    `json:"amount" bson:"amount"` // amount sent
	Product        string    `json:"product" bson:"product"`
	Description    string    `json:"description" bson:"description"`
	Reason         string    `json:"message" bson:"message"`
	Order_ID       int       `json:"order_id" bson:"order_id"`
	Transaction_ID string    `json:"transaction_id" bson:"transaction_id"`
	Session_ID     string    `json:"session_id" bson:"session_id"`
	CreatedAt      time.Time `json:"created_at" bson:"created_at"` // ISO datetime string
}

type AccountDetails struct {
	Bank_Name        string    `json:"bank_name" bson:"bank_name"`
	User_ID          string    `json:"user_id" bson:"user_id"`
	Account_Name     string    `json:"account_name" bson:"account_name"`
	Account_No       string    `json:"account_no" bson:"account_no"`
	VirtualAccountID string    `json:"virtualaccountid" bson:"virtualaccountid"`
	CreatedAt        time.Time `json:"created_at" bson:"created_at"`
}

type CounterParty struct {
	ID            string `json:"id"`
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
	BankName      string `json:"bank_name"`
	NIPCode       string `json:"nipCode"`
}
type BankDetails struct {
	Name    string `json:"name" bson:"name"`
	NIPCode string `json:"nipCode" bson:"nipCode"`
}

type DepositResponse struct {
	UserID         string    `json:"user_id" bson:"user_id"`
	Amount         string    `json:"amount" bson:"amount"`                 // amount recieved
	WalletType     string    `json:"walletType" bson:"walletType"`         // Nigerian NGN wallet
	Bank_Name      string    `json:"bank_name" bson:"bank_name"`           // sender's bank name
	Account_Name   string    `json:"account_name" bson:"account_name"`     // sender's account name
	Account_No     string    `json:"account_no" bson:"account_no"`         // sender's account number
	Product        string    `json:"product" bson:"product"`               // *Virtual account
	Description    string    `json:"description" bson:"description"`       // description based on the method of deposit
	Message        string    `json:"message" bson:"message"`               // map to narration
	Order_ID       int       `json:"order_id" bson:"order_id"`             // orderID created
	Transaction_ID string    `json:"transaction_id" bson:"transaction_id"` // transactionID created
	Session_ID     string    `json:"session_id" bson:"session_id"`         // map to paymentReference
	CreatedAt      time.Time `json:"created_at" bson:"created_at"`         // ISO datetime string
}

type Balance struct {
	VirtualNuban string               `json:"virtualNuban" bson:"virtualNuban"`
	UserID       string               `json:"user_id" bson:"user_id"`
	Balance      primitive.Decimal128 `json:"balance" bson:"balance"`
	CreatedAt    time.Time            `json:"created_at" bson:"created_at"`
	UpdateAt     time.Time            `json:"update_at" bson:"update_at"`
}

func (b Balance) Decimal() (decimal.Decimal, error) {
	return decimal.NewFromString(b.Balance.String())
}
