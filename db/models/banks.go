package models

import (
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TransferResponse struct {
	UserID string `json:"user_id" bson:"user_id"`
	Status string `json:"status" bson:"status"` // "success" or "failed"

	// Optional fields
	Email          *string `json:"email,omitempty" bson:"email,omitempty"`
	Username       *string `json:"username,omitempty" bson:"username,omitempty"`
	Phone          *string `json:"phone,omitempty" bson:"phone,omitempty"`
	CustomerName   *string `json:"customer_name,omitempty" bson:"customer_name,omitempty"`
	Bank_Name      *string `json:"bank_name,omitempty" bson:"bank_name,omitempty"`
	Account_Number *string `json:"account_number,omitempty" bson:"account_number,omitempty"`
	Account_Name   *string `json:"account_name,omitempty" bson:"account_name,omitempty"`

	FullName               string    `json:"full_name" bson:"full_name"`
	Amount                 string    `json:"amount" bson:"amount"`
	TransactionProduct     string    `json:"transaction_product" bson:"transaction_product"`
	TransactionDescription string    `json:"transaction_description" bson:"transaction_description"`
	Reason                 string    `json:"message" bson:"message"`
	Order_ID               int       `json:"order_id" bson:"order_id"`
	Transaction_ID         string    `json:"transaction_id" bson:"transaction_id"`
	Session_ID             string    `json:"session_id" bson:"session_id"`
	CreatedAt              time.Time `json:"created_at" bson:"created_at"`
}

type TransferRecipient struct {
	UserID    string                     `json:"user_id" bson:"user_id"`
	Recipient []TransferRecipientDetails `json:"recipient" bson:"recipient"`
	CreatedAt time.Time                  `json:"created_at" bson:"created_at"`
}

type TransferRecipientDetails struct {
	Username string `json:"username" bson:"username"`
	Email    string `json:"email" bson:"email"`
	Phone    string `json:"phone" bson:"phone"`
	FullName string `json:"full_name" bson:"full_name"`
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
	ID            string `json:"id" bson:"id"`
	AccountName   string `json:"account_name" bson:"account_name"`
	AccountNumber string `json:"account_number" bson:"account_number"`
	BankName      string `json:"bank_name" bson:"bank_name"`
	NIPCode       string `json:"nip_code" bson:"nip_code"`
}
type BankDetails struct {
	Name    string `json:"name" bson:"name"`
	NIPCode string `json:"nip_code" bson:"nip_code"`
}

type DepositResponse struct {
	UserID                 string    `json:"user_id" bson:"user_id"`
	Status                 string    `json:"status" bson:"status"`                                   // "success" or "failed"
	Amount                 string    `json:"amount" bson:"amount"`                                   // amount recieved
	WalletType             string    `json:"walletType" bson:"walletType"`                           // Nigerian NGN wallet
	Bank_Name              string    `json:"bank_name" bson:"bank_name"`                             // sender's bank name
	Account_Name           string    `json:"account_name" bson:"account_name"`                       // sender's account name
	Account_No             string    `json:"account_no" bson:"account_no"`                           // sender's account number
	TransactionProduct     string    `json:"transaction_product" bson:"transaction_product"`         // *Virtual account
	TransactionDescription string    `json:"transaction_description" bson:"transaction_description"` // description based on the method of deposit
	Message                string    `json:"message" bson:"message"`                                 // map to narration
	Order_ID               int       `json:"order_id" bson:"order_id"`                               // orderID created
	Transaction_ID         string    `json:"transaction_id" bson:"transaction_id"`                   // transactionID created
	Session_ID             string    `json:"session_id" bson:"session_id"`                           // map to paymentReference
	CreatedAt              time.Time `json:"created_at" bson:"created_at"`                           // ISO datetime string
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
