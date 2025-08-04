package deposit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/mongo"
	"github.com/aremxyplug-be/lib/balance"
	"github.com/aremxyplug-be/lib/randomgen"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

var (
	api    = os.Getenv("ANCHOR_API")
	apikey = os.Getenv("ANCHORAPI_PROD")
)

type Config struct {
	db     db.DataStore
	logger *zap.Logger
}

type depositID struct {
	VirtualNuban string `json:"virtualNuban" bson:"virtualNuban"`
	ID           string `json:"id" bson:"ID"`
}

func NewDepositConfig(db db.DataStore, logger *zap.Logger) *Config {
	return &Config{
		db:     db,
		logger: logger,
	}
}
func (c *Config) Deposit(virtualaccountid string, userID string) error {
	// using the list payment endpoint.
	url := fmt.Sprintf("%s/%s?%s=%s", api, "payments", "virtualNubanId", virtualaccountid)

	if virtualaccountid == "" {
		c.logger.Error("Deposit failed: missing account number")
		return ErrEmptyVirtualNuban
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.logger.Error("Deposit failed: unable to create new request", zap.Error(err))
		return ErrNewRequestFailed
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("x-anchor-key", apikey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.logger.Error("Deposit failed: API connection error", zap.Error(err))
		return ErrAPIConnectionFailed
	}
	defer resp.Body.Close()

	apiResponse := paymentResponse{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Deposit failed: unable to read response body", zap.Error(err))
		return JSONError(err)
	}
	c.logger.Debug("API Response Body", zap.String("body", string(body)))

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		c.logger.Error("Deposit failed: unable to unmarshal JSON response", zap.Error(err))
		return JSONError(err)
	}
	c.logger.Debug("Parsed API Response", zap.Any("response", apiResponse))

	paymentData := apiResponse.Data
	for _, data := range paymentData {
		orderID, err := randomgen.GenerateOrderID()
		if err != nil {
			c.logger.Error("Deposit failed: unable to generate order ID", zap.Error(err))
			return err
		}

		transctionID := randomgen.GenerateTransactionID("dep")
		attributes := data.Attributes
		virtualNuban := data.Relationships.VirtualNuban.Data.ID
		deposit := depositID{
			VirtualNuban: virtualNuban,
			ID:           data.ID,
		}

		if err := c.db.SaveDepositID(deposit); err != nil {
			if err == mongo.ErrDepositIDExist {
				c.logger.Info("Deposit ID already exists, skipping", zap.String("depositID", data.ID))
				continue
			}
			c.logger.Error("Deposit failed: unable to save deposit ID", zap.Error(err))
			return DBConnectionError(err)
		}

		bal, err := c.db.GetBalance(userID)
		if err != nil {
			c.logger.Error("Deposit failed: unable to fetch balance", zap.Error(err))
			return DBConnectionError(err)
		}
		c.logger.Debug("Fetched Balance", zap.Any("balance", bal))

		deposit_amount := data.Attributes.Amount * 0.01
		newBalance, depositAmount := balance.NewBalanceDeposit(bal, decimal.NewFromFloatWithExponent(deposit_amount, -2))
		parsedBalance, _ := primitive.ParseDecimal128(newBalance.String())
		c.logger.Debug("New Balance Calculated", zap.String("newBalance", newBalance.String()))

		userBalance := models.Balance{
			VirtualNuban: virtualNuban,
			Balance:      parsedBalance,
			UserID:       userID,
			UpdateAt:     time.Now().UTC(),
		}
		if err := c.db.SaveBalance(userID, userBalance); err != nil {
			c.logger.Error("Deposit failed: unable to save user balance", zap.Error(err))
			return DBConnectionError(err)
		}

		createdAt, err := time.Parse("2006-01-02T15:04:05", data.Attributes.CreatedAt)
		if err != nil {
			c.logger.Error("Deposit failed: unable to parse createdAt", zap.Error(err))
			return err
		}

		result := models.DepositResponse{
			UserID:                 userID,
			Status:                 "success",
			Amount:                 fmt.Sprintf("%v", depositAmount),
			WalletType:             "Nigerian NGN Wallet",
			Bank_Name:              attributes.CounterParty.Bank.Name,
			Account_Name:           attributes.CounterParty.AccountName,
			Account_No:             attributes.CounterParty.AccountNumber,
			TransactionProduct:     "Virtual Account",
			TransactionDescription: "NGN Wallet Top Up",
			Message:                data.Attributes.Narration,
			Order_ID:               orderID,
			Transaction_ID:         transctionID,
			Session_ID:             data.Attributes.PaymentReference,
			CreatedAt:              createdAt,
		}

		if err := c.saveTransaction(result); err != nil {
			c.logger.Error("Deposit failed: unable to save transaction", zap.Error(err))
			return DBConnectionError(err)
		}
		c.logger.Info("Deposit transaction saved successfully", zap.Any("transaction", result))
	}

	return nil
}

// write to save transaction to the database
func (c *Config) saveTransaction(detail models.DepositResponse) error {
	err := c.db.SaveDeposit(detail)
	if err != nil {
		return err
	}
	return nil
}
