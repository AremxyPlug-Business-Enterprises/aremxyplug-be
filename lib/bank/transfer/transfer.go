package transfer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/randomgen"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	api         = os.Getenv("ANCHOR_SANDBOX")
	apikey      = os.Getenv("ANCHORAPI_KEY")
	customer_id = os.Getenv("CUSTOMER_ID")
)

type Config struct {
	db db.DataStore
}

func NewConfig(store db.DataStore) *Config {
	return &Config{
		db: store,
	}
}

// this endpoint should auto automatically initialize
func (c *Config) listBanks() error {
	url := fmt.Sprintf("%s/%s", api, "banks")

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("accept", "application/json")
	req.Header.Add("x-anchor-key", apikey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	apiResponse := bankLists{}
	bankLists := []models.BankDetails{}

	if err := json.NewDecoder(res.Body).Decode(&apiResponse); err != nil {
		// do something with error
	}

	for _, bank := range apiResponse.BanksData {
		// save bank to database.
		bankList := models.BankDetails{
			Name:    bank.Atrributes.Name,
			NIPCode: bank.Atrributes.NIPCode,
		}
		bankLists = append(bankLists, bankList)
	}

	if err := c.db.SaveBankList(bankLists); err != nil {
		return DBConnectionError(err)
	}

	return nil
}

func (c *Config) TransferToBank(info models.TransferInfo) (models.TransferResponse, error) {

	// first check if the details is already in the database. if it is just procced to the point of transfer
	details := verifyAccountResponse{}
	counterparty, err := c.getCounterParty(info.Account_Number, info.Bank_name)
	if err == mongo.ErrNoDocuments {
		bankDetail, _ := c.db.GetBankDetail(info.Bank_name)
		details, _ = c.verifyAccount(bankDetail.NIPCode, info.Account_Number)
		counterparty, _ = c.createCounterParty(details)
	} else if err != nil {
		return models.TransferResponse{}, DBConnectionError(err)
	}

	orderID, err := randomgen.GenerateOrderID()
	if err != nil {

	}
	transactionID := randomgen.GenerateTransactionID("TRF")
	url := fmt.Sprintf("%s/%s", api, "transfers")

	payload := intiateTransfer{
		Data: transferData{
			Attributes: transferDataAttributes{
				Amount:   1000,
				Currency: "NGN",
			},
		},
		Type: "NIPTransfer",
		CounterParty: counterParty{
			Data: data{
				ID:   counterparty.ID,
				Type: "CounterParty",
			},
		},
		Relationships: relationships{
			DestinationAcc: destination{
				Data: struct {
					Type string `json:"type"`
				}{
					Type: "SubAccount",
				},
			},
		},
		Account: account{
			Data: data{
				ID:   "", // the ID of the deposit account
				Type: "DepositAccount",
			},
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

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return models.TransferResponse{}, JSONError(err)
	}

	result := models.TransferResponse{
		Bank_Name:      details.Data.Attributes.Bank.Name,
		Account_Name:   details.Data.AccountName,
		Account_No:     details.Data.AccountNumber,
		Product:        "Money Transfer",
		Description:    "",
		Reason:         info.Reason,
		Order_ID:       orderID,
		Transaction_ID: transactionID,
		// sessionID is gotten from the webhook
	}

	if err := c.saveTransaction(result); err != nil {
		return result, DBConnectionError(err)
	}

	return result, nil

}

func (c *Config) verifyAccount(sortCode, accNumber string) (verifyAccountResponse, error) {

	url := fmt.Sprintf("%s/%s/%s", api, sortCode, accNumber)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return verifyAccountResponse{}, ErrCreatingHTTPRequest
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("x-anchor-key", apikey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return verifyAccountResponse{}, ErrAPIConnectionFailed
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		// return that account wasn't found
		return verifyAccountResponse{}, ErrAccountValidationFailed
	}

	response := verifyAccountResponse{}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return verifyAccountResponse{}, JSONError(err)
	}

	return response, nil

}

func (c *Config) createCounterParty(info verifyAccountResponse) (models.CounterParty, error) {

	url := fmt.Sprintf("%s/%s", api, "counterparties")

	payload := counterPartyPayload{}
	payload.Data.Type = "CounterParty"
	payload.Data.Attributes.AccountName = info.Data.AccountName
	payload.Data.Attributes.BankCode = info.Data.Attributes.Bank.NipCode
	payload.Data.Attributes.VerifyName = true
	payload.Data.Attributes.AccountNumber = info.Data.AccountNumber
	payload.Data.Relationships.Bank.Data.ID = customer_id
	payload.Data.Relationships.Bank.Data.Type = "DepositAccount"

	requestBody, err := json.Marshal(payload)
	if err != nil {
		return models.CounterParty{}, JSONError(err)
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

	apiResponse := counterPartyAPIResponse{}

	if resp.StatusCode != http.StatusCreated {
		return models.CounterParty{}, ErrCounterpartyCreationFailed
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return models.CounterParty{}, JSONError(err)
	}

	result := models.CounterParty{
		ID:            apiResponse.Data.ID,
		AccountName:   apiResponse.Data.AccountName,
		AccountNumber: apiResponse.Data.AccontNumber,
		BankName:      apiResponse.Data.Bank.Name,
		NIPCode:       apiResponse.Data.Bank.NipCode,
	}

	if err := c.saveCounterParty(result); err != nil {
		return result, DBConnectionError(err)
	}

	return result, nil
}

// endpoint to verify a transfer from the API, we will save all transactions regardless.
func verifyTransfer(id string) (transferResult, error) {

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

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
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
	err := c.saveCounterParty(conterparty)
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
