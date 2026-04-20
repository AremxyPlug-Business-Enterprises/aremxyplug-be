package bankacc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

/*
var (
	api         = os.Getenv("ANCHOR_SANDBOX")
	apikey      = os.Getenv("ANCHORAPI_KEY")
	deposit_id  = os.Getenv("DEPOSIT_ID")
	customer_id = os.Getenv("CUSTOMER_ID")
)
*/

var (
	api    = os.Getenv("ANCHOR_API")
	apikey = os.Getenv("ANCHORAPI_PROD")
	//deposit_id  = os.Getenv("DEPOSIT_ID_LIVE")
	customer_id = os.Getenv("CUSTOMER_ID_LIVE")
	deposit_id  = os.Getenv("DEPOSIT_ID_LIVE_2")
	bvn         = os.Getenv("BVN_LIVE")
)

type BankConfig struct {
	dbConn db.BankStore
	logger *zap.Logger
}

// initialize BankConfig.

func NewBankConfig(store db.DataStore, logger *zap.Logger) *BankConfig {
	return &BankConfig{
		dbConn: store,
		logger: logger,
	}
}

func (b *BankConfig) VirtualAccount(ctx context.Context, user models.User) (models.AccountDetails, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// create a new virtual accout for new users as soon as their account is confirmed
	// should be called at the moment that a user's account is verified

	url := fmt.Sprintf("%s/%s", api, "virtual-nubans")
	to := cases.Title(language.English)
	full_name := to.String(user.FullName)

	name := fmt.Sprintf("%s/%s", "AP", full_name)

	payload := virtualNubanPayload{}
	payload.Data.Type = "VirtualNuban"
	payload.Data.Attributes.Provider = "ninepsb"
	payload.Data.Attributes.VirtualAccount.BVN = bvn
	payload.Data.Attributes.VirtualAccount.Name = name
	payload.Data.Attributes.VirtualAccount.Email = user.Email
	payload.Data.Attributes.VirtualAccount.Permanent = true
	payload.Data.Relationships.SettlementAccount.Data.Type = "DepositAccount"
	payload.Data.Relationships.SettlementAccount.Data.ID = deposit_id

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return models.AccountDetails{}, JSONError(err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
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

	if resp.StatusCode != http.StatusCreated {
		b.logger.Log(b.logger.Level(), resp.Status)
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return models.AccountDetails{}, JSONError(err)
		}
		b.logger.Log(b.logger.Level(), string(body))
		return models.AccountDetails{}, fmt.Errorf("failed to create account")
	}

	apiResponse := virtualAccountResponse{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.AccountDetails{}, JSONError(err)
	}
	b.logger.Log(b.logger.Level(), string(body))
	/*
		if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
			return models.AccountDetails{}, JSONError(err)
		}
	*/
	bodyValue := string(body)
	if strings.Contains(bodyValue, "errors") {
		b.logger.Log(b.logger.Level(), bodyValue)
		return models.AccountDetails{}, fmt.Errorf("%s", "server error")
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return models.AccountDetails{}, JSONError(err)
	}

	virtualAccount := apiResponse.Data.ID

	result := models.AccountDetails{
		Account_Name:     apiResponse.Data.Attributes.AccountName,
		Account_No:       apiResponse.Data.Attributes.AccountNumber,
		Bank_Name:        apiResponse.Data.Attributes.Bank.Name,
		User_ID:          user.ID,
		VirtualAccountID: virtualAccount,
	}

	if err := b.saveAccount(ctx, result); err != nil {
		b.logger.Error("Error saving account details to the database:", zap.Error(err))
		return models.AccountDetails{}, DBConnectionError(err)
	}
	if err := b.dbConn.CreateInitialBalance(ctx, user.ID, virtualAccount); err != nil {
		b.logger.Error("Error creating initial balance:", zap.Error(err))
		return models.AccountDetails{}, DBConnectionError(err)
	}

	return result, nil

}

func (b *BankConfig) CreateDepositAccount(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	url := fmt.Sprintf("%s/%s", api, "accounts")
	payload := createDeposit{
		Data: createDepositData{
			Attributes: depositAttributes{
				ProductName: "SETTLEMENT",
			},
			Relationships: depositRelationships{
				Customer: depositCustomer{
					Data: depositCustomerData{
						ID:   customer_id,
						Type: "BusinessCustomer",
					},
				},
			},
			Type: "DepositAccount",
		},
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		b.logger.Error(err.Error())
		fmt.Println("Error marshalling json payload:", err)
		return JSONError(err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		b.logger.Error(err.Error())
		fmt.Println("Error creating a http request:", err)
		return ErrCreatingHTTPRequest
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("accept", "application/json")
	req.Header.Set("x-anchor-key", apikey)

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Do(req)
	if err != nil {
		b.logger.Error(err.Error())
		fmt.Println("Error calling external api:", err)
		return ErrAPIConnectionFailed
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// return unsuccessful response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			b.logger.Error(err.Error())
			fmt.Println("Error writing to .env file:", err)
			return JSONError(err)
		}
		b.logger.Log(b.logger.Level(), string(body))
		fmt.Println("Error creating deposit account:", resp.Status)
		return ErrCreatingDepositAccount
	}

	apiResponse := depositCustomerResponse{}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		b.logger.Error(err.Error())
		fmt.Println("Error writing to .env file:", err)
		return JSONError(err)
	}
	b.logger.Log(b.logger.Level(), string(body))

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		b.logger.Error(err.Error())
		return JSONError(err)
	}

	if err := os.Setenv("DEPOSIT_ID_LIVE", apiResponse.Data.ID); err != nil {
		b.logger.Error(err.Error())
		fmt.Println("Error writing to .env file:", err)
		return ErrSettingENV
	}
	envVars := make(map[string]string)
	for _, envVar := range os.Environ() {
		pair := strings.SplitN(envVar, "=", 2)
		envVars[pair[0]] = pair[1]
	}

	err = godotenv.Write(envVars, ".env")
	if err != nil {
		b.logger.Error(err.Error())
		fmt.Println("Error writing to .env file:", err)
		return ErrWritingToENV
	}

	return nil
}

func (b *BankConfig) saveAccount(ctx context.Context, account models.AccountDetails) error {
	if ctx == nil {
		ctx = context.Background()
	}
	err := b.dbConn.SaveVirtualAccount(ctx, account)
	if err != nil {
		return err
	}

	return nil
}
