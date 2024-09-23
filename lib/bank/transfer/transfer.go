package transfer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/randomgen"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

var (
	/*
		api        = os.Getenv("ANCHOR_SANDBOX")
		apikey     = os.Getenv("ANCHORAPI_KEY")
		deposit_id = os.Getenv("DEPOSIT_ID")
	*/
	api        = os.Getenv("ANCHOR_API")
	apikey     = os.Getenv("ANCHORAPI_PROD")
	deposit_id = os.Getenv("DEPOSIT_ID_LIVE")
)

type Config struct {
	db     db.DataStore
	logger *zap.Logger
}

func NewConfig(store db.DataStore, logger *zap.Logger) *Config {
	return &Config{
		db:     store,
		logger: logger,
	}
}

// this endpoint should auto automatically initialize
func (c *Config) ListBanks() error {
	url := fmt.Sprintf("%s/%s", api, "banks")

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("accept", "application/json")
	req.Header.Add("x-anchor-key", apikey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.logger.Error(err.Error())
		return ErrCreatingHTTPRequest
	}
	defer res.Body.Close()

	apiResponse := bankLists{}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.logger.Error(err.Error())
		return err
	}
	c.logger.Log(c.logger.Level(), string(body))
	// bankLists := []models.BankDetails{}

	/*
		if err := json.NewDecoder(res.Body).Decode(&apiResponse); err != nil {
			// do something with error
			c.logger.Error(err.Error())
			return JSONError(err)
		}
	*/
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		c.logger.Error(err.Error())
		return JSONError(err)
	}

	for _, bank := range apiResponse.BanksData {
		// save bank to database.
		bankList := models.BankDetails{
			Name:    bank.Atrributes.Name,
			NIPCode: bank.Atrributes.NIPCode,
		}

		if err := c.db.SaveBankList(bankList); err != nil {
			return DBConnectionError(err)
		}

	}

	return nil
}

func (c *Config) TransferToBank(info models.TransferInfo) (models.TransferResponse, error) {

	// first check if the details is already in the database. if it is just procced to the point of transfer
	counterparty, err := c.getCounterParty(info.Account_Number, info.Bank_name)
	if err == mongo.ErrNoDocuments {
		bankDetail, _ := c.db.GetBankDetail(info.Bank_name)
		details, err := c.verifyAccount(bankDetail.NIPCode, info.Account_Number)
		if err != nil {
			return models.TransferResponse{}, JSONError(err)
		}
		counterparty, err = c.createCounterParty(details)
		if err != nil {
			return models.TransferResponse{}, JSONError(err)
		}
	} else if err != nil {
		return models.TransferResponse{}, DBConnectionError(err)
	}

	orderID, err := randomgen.GenerateOrderID()
	if err != nil {
		return models.TransferResponse{}, ErrGeneratingOrderID
	}
	transactionID := randomgen.GenerateTransactionID("TRF")
	url := fmt.Sprintf("%s/%s", api, "transfers")
	amount := info.Amount * 100

	payload := intiateTransfer{
		Data: transferData{
			Attributes: transferDataAttributes{
				Amount:   amount,
				Currency: "NGN",
			},
			Relationships: relationships{
				DestinationAcc: destination{
					Data: struct {
						Type string `json:"type"`
					}{
						Type: "SubAccount",
					},
				},
				Account: account{
					Data: data{
						ID:   deposit_id, // the ID of the deposit account
						Type: "DepositAccount",
					},
				},
				CounterParty: counterParty{
					Data: data{
						ID:   counterparty.ID,
						Type: "CounterParty",
					},
				},
			},
			Type: "NIPTransfer",
		},
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return models.TransferResponse{}, JSONError(err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return models.TransferResponse{}, ErrCreatingHTTPRequest
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-anchor-key", apikey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return models.TransferResponse{}, ErrAPIConnectionFailed
	}
	defer resp.Body.Close()

	apiResponse := transferResult{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error(err.Error())
		return models.TransferResponse{}, JSONError(err)
	}
	c.logger.Log(c.logger.Level(), string(body))
	if resp.StatusCode != http.StatusCreated {
		c.logger.Error(resp.Status)
		return models.TransferResponse{}, ErrAccountValidationFailed
	}
	/*
		if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
			c.logger.Error(err.Error())
			return models.TransferResponse{}, JSONError(err)
		}
	*/
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		c.logger.Error(err.Error())
		return models.TransferResponse{}, JSONError(err)
	}

	result := models.TransferResponse{
		Bank_Name:      counterparty.BankName,
		Account_Name:   counterparty.AccountName,
		Account_No:     counterparty.AccountNumber,
		Product:        "Money Transfer",
		Description:    "",
		Reason:         info.Reason,
		Order_ID:       orderID,
		Transaction_ID: transactionID,
		// sessionID is gotten from the webhook
	}

	if err := c.saveTransaction(result); err != nil {
		c.logger.Error(err.Error())
		return result, DBConnectionError(err)
	}

	return result, nil

}

func (c *Config) verifyAccount(sortCode, accNumber string) (verifyAccountResponse, error) {

	url := fmt.Sprintf("%s/%s/%s/%s/%s", api, "payments", "verify-account", sortCode, accNumber)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.logger.Error(err.Error())
		return verifyAccountResponse{}, ErrCreatingHTTPRequest
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("x-anchor-key", apikey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.logger.Error(err.Error())
		return verifyAccountResponse{}, ErrAPIConnectionFailed
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.logger.Error(err.Error())
		return verifyAccountResponse{}, err
	}
	c.logger.Log(c.logger.Level(), string(body))

	if res.StatusCode != http.StatusOK {
		// return that account wasn't found
		c.logger.Log(c.logger.Level(), res.Status)
		return verifyAccountResponse{}, ErrAccountValidationFailed
	}

	response := verifyAccountResponse{}

	/*
		if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
			c.logger.Error(err.Error())
			return verifyAccountResponse{}, JSONError(err)
		}
	*/
	if err := json.Unmarshal(body, &response); err != nil {
		c.logger.Error(err.Error())
		return verifyAccountResponse{}, err
	}

	return response, nil

}

func (c *Config) createCounterParty(info verifyAccountResponse) (models.CounterParty, error) {

	url := fmt.Sprintf("%s/%s", api, "counterparties")

	payload := counterPartyPayload{}
	payload.Data.Type = "CounterParty"
	payload.Data.Attributes.AccountName = info.Data.Attributes.AccountName
	payload.Data.Attributes.BankCode = info.Data.Attributes.Bank.NipCode
	payload.Data.Attributes.VerifyName = true
	payload.Data.Attributes.AccountNumber = info.Data.Attributes.AccountNumber
	payload.Data.Relationships.Bank.Data.ID = deposit_id
	payload.Data.Relationships.Bank.Data.Type = "DepositAccount"

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return models.CounterParty{}, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return models.CounterParty{}, ErrCreatingHTTPRequest
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-anchor-key", apikey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return models.CounterParty{}, ErrAPIConnectionFailed
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error(err.Error())
		return models.CounterParty{}, JSONError(err)
	}
	c.logger.Log(c.logger.Level(), string(body))
	apiResponse := counterPartyAPIResponse{}

	if resp.StatusCode != http.StatusCreated {
		c.logger.Log(c.logger.Level(), resp.Status)
		return models.CounterParty{}, ErrCounterpartyCreationFailed
	}
	/*
		if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
			c.logger.Error(err.Error())
			return models.CounterParty{}, err
		}
	*/
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		c.logger.Error(err.Error())
		return models.CounterParty{}, err
	}

	result := models.CounterParty{
		ID:            apiResponse.Data.ID,
		AccountName:   apiResponse.Data.Attributes.AccountName,
		AccountNumber: apiResponse.Data.Attributes.AccountNumber,
		BankName:      apiResponse.Data.Attributes.Bank.Name,
		NIPCode:       apiResponse.Data.Attributes.Bank.NipCode,
	}

	if err := c.saveCounterParty(result); err != nil {
		c.logger.Error(err.Error())
		return result, DBConnectionError(err)
	}

	return result, nil
}

// endpoint to verify a transfer from the API, we will save all transactions regardless.
func (c *Config) verifyTransfer(id string) (transferResult, error) {

	url := fmt.Sprintf("%s/%s/%s", api, "verify", id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return transferResult{}, ErrCreatingHTTPRequest
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("x-anchor-key", apikey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return transferResult{}, ErrAPIConnectionFailed
	}

	defer res.Body.Close()

	// at this point return the transfer status

	result := transferResult{}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.logger.Error(err.Error())
		return transferResult{}, JSONError(err)
	}
	c.logger.Log(c.logger.Level(), string(body))

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		c.logger.Error(err.Error())
		return transferResult{}, JSONError(err)
	}

	return result, nil
}

func (c *Config) saveTransaction(details models.TransferResponse) error {
	err := c.db.SaveTransfer(details)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) saveCounterParty(conterparty models.CounterParty) error {
	err := c.db.SaveCounterParty(conterparty)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) getCounterParty(accountname, bankname string) (models.CounterParty, error) {
	counterparty, err := c.db.GetCounterParty(accountname, bankname)
	if err != nil {
		return models.CounterParty{}, err
	}

	return counterparty, nil
}
