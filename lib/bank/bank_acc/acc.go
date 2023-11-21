package bankacc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
)

var (
	api         = os.Getenv("ANCHOR_SANDBOX")
	apikey      = os.Getenv("ANCHORAPI_KEY")
	customer_id = os.Getenv("CUSTOMER_ID")
)

type BankConfig struct {
	dbConn db.DataStore
	logger *zap.Logger
}

// initialize BankConfig.

func NewBankConfig(store db.DataStore, logger *zap.Logger) *BankConfig {
	return &BankConfig{
		dbConn: store,
		logger: logger,
	}
}

func (b *BankConfig) VirtualAccount() (models.AccountDetails, error) {

	// create a new virtual accout for new users as soon as their account is confirmed
	// should be called at the moment that a user's account is verified

	url := fmt.Sprintf("%s/%s", api, "virtualnuban")

	payload := virtualNubanPayload{}
	payload.Data.Type = "VirtualNuban"
	payload.Data.Attributes.Provider = "providus"
	payload.Data.Attributes.VirtualAccount.BVN = ""
	payload.Data.Attributes.VirtualAccount.Name = ""
	payload.Data.Attributes.VirtualAccount.Email = ""
	payload.Data.Attributes.VirtualAccount.Permanent = true
	payload.Data.Relationships.SettlementAccount.Data.Type = "DepositAccount"
	payload.Data.Relationships.SettlementAccount.Data.ID = ""

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return models.AccountDetails{}, JSONError(err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return models.AccountDetails{}, ErrCreatingHTTPRequest
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("accept", "application/json")
	req.Header.Set("x-anchor-key", apikey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return models.AccountDetails{}, ErrAPIConnectionFailed
	}

	defer resp.Body.Close()

	apiResponse := virtualAccountResponse{}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return models.AccountDetails{}, JSONError(err)
	}

	result := models.AccountDetails{
		Account_Name: apiResponse.Data.Attributes.AccountName,
		Account_No:   apiResponse.Data.Attributes.AccountNumber,
		Bank_Name:    apiResponse.Data.Attributes.Bank.Name,
	}

	if err := b.saveAccount(result); err != nil {
		return models.AccountDetails{}, DBConnectionError(err)
	}

	return result, nil

}

func (b *BankConfig) saveAccount(account models.AccountDetails) error {
	err := b.dbConn.SaveVirtualAccount(account)
	if err != nil {
		return err
	}

	return nil
}
