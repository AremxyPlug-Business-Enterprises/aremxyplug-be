package transfer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/redis"
	"github.com/aremxyplug-be/lib/events"
	"github.com/aremxyplug-be/lib/randomgen"
	"github.com/shopspring/decimal"
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
	db        db.DataStore
	logger    *zap.Logger
	redis     *redis.RedisConn
	processor *events.Processor
}

func NewConfig(store db.DataStore, logger *zap.Logger, redis *redis.RedisConn, processor *events.Processor) *Config {
	return &Config{
		db:        store,
		logger:    logger,
		redis:     redis,
		processor: processor,
	}
}

// func (c *Config) TransferToAremxyPlug(data AremxyPlugTransfer) (models.TransferResponse, error) {

// 	user, err := c.db.GetUserByUsernameOrEmail(data.Email, data.Username)
// 	if err != nil {
// 		c.logger.Error(err.Error())
// 		return models.TransferResponse{}, err
// 	}

// 	// now retrieve the virtual account details of the user
// 	virtualAccount, err := c.db.GetVirtualNuban(user.ID)
// 	if err != nil {
// 		c.logger.Error(err.Error())
// 		return models.TransferResponse{}, err
// 	}

// 	// now initiate the transfer to the AremxyPlug account
// 	transferResponse, err := c.TransferToBank(TransferInfo{
// 		UserID:         user.ID,
// 		Account_Name:   virtualAccount.Account_Name, // assuming virtualAccount has AccountName field
// 		Account_Number: virtualAccount.Account_No,   // assuming virtualAccount has AccountNumber field
// 		Bank_name:      virtualAccount.Bank_Name,    // assuming virtualAccount has BankName field
// 		Reason:         data.Reason,
// 		FullName:       data.FullName,
// 		Amount:         data.Amount, // assuming data has Amount field
// 		Email:          data.Email,
// 		Phone:          data.Phone,
// 		Username:       data.Username,
// 		RecipientName:  data.Name,
// 		Source:         "aremxyplug",
// 		TXN:            data.TXN,
// 	})
// 	if err != nil {
// 		c.logger.Error(err.Error())
// 		return models.TransferResponse{}, err
// 	}

// 	c.logger.Info("Transfer processed successfully", zap.Any("response", transferResponse))

// 	transferResponse.Email = &user.Email
// 	transferResponse.Username = &user.Username
// 	transferResponse.Phone = &user.PhoneNumber
// 	transferResponse.CustomerName = &user.FullName

// 	return transferResponse, nil
// }

func (c *Config) TransferToAremxyPlug(ctx context.Context, data AremxyPlugTransfer) (models.TransferResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	user, err := c.db.GetUserByUsernameOrEmail(ctx, data.Email, data.Username)
	if err != nil {
		c.logger.Error(err.Error())
		return models.TransferResponse{}, err
	}
	trfOrderID, _ := randomgen.GenerateOrderID()
	depOrderID, _ := randomgen.GenerateOrderID()
	trfTransactionID := randomgen.GenerateTransactionID("trf")
	depTransactionID := randomgen.GenerateTransactionID("dep")

	bal, err := c.db.GetBalance(ctx, user.ID)
	if err != nil {
		c.logger.Error(err.Error())
		return models.TransferResponse{}, err
	}

	// newBalance, _ := balance.NewBalanceDeposit(bal, decimal.NewFromFloatWithExponent(data.Amount, -2))

	newBalance := bal.Add(decimal.NewFromFloatWithExponent(data.Amount, -2))

	if err := c.db.UpdateBalance(ctx, user.ID, newBalance); err != nil {
		c.logger.Error(err.Error())
		return models.TransferResponse{}, err
	}

	trf := models.TransferResponse{
		Status:                 "success",
		Amount:                 fmt.Sprintf("%v", data.Amount),
		UserID:                 data.UserID,
		FullName:               data.FullName,
		Email:                  &user.Email,
		Username:               &user.Username,
		Phone:                  &user.PhoneNumber,
		CustomerName:           &user.FullName,
		TransactionProduct:     "Internal Transfer",
		TransactionDescription: "From NGN Wallet",
		Reason:                 data.Reason,
		Order_ID:               trfOrderID,
		Transaction_ID:         trfTransactionID,
		TXN:                    data.TXN,
		CreatedAt:              time.Now().UTC(),
	}

	if err := c.saveTransaction(ctx, trf); err != nil {
		c.logger.Error(err.Error())
		return trf, DBConnectionError(err)
	}

	// save the new deposit receipt for the reciever's account
	dept := models.InternalDepositResponse{
		UserID:                 user.ID,
		Amount:                 fmt.Sprintf("%v", data.Amount),
		WalletType:             "Nigerian NGN Wallet",
		SenderName:             data.FullName,
		TransactionProduct:     "Internal Deposit",
		TransactionDescription: "NGN Wallet Top Up",
		Message:                data.Reason,
		Reference:              trfTransactionID,
		Order_ID:               depOrderID,
		Transaction_ID:         depTransactionID,
		Status:                 "success",
		CreatedAt:              time.Now().UTC(),
	}

	// need to use redis to update balance here as well
	if err := c.db.SaveDeposit(ctx, dept); err != nil {
		c.logger.Error(err.Error())
		return models.TransferResponse{}, err
	}

	if c.processor != nil {
		// Process deposit event for receiver
		go func(uID string, amt string, txID string) {
			baseCtx := ctx
			if baseCtx == nil {
				baseCtx = context.Background()
			}
			baseCtx = context.WithoutCancel(baseCtx)

			taskCtx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
			defer cancel()
			ev := &events.Event{
				Type:      "transaction.completed",
				UserID:    user.ID, // receiver user ID
				Amount:    fmt.Sprintf("%v", data.Amount),
				TxID:      depTransactionID,
				TS:        time.Now().UTC(),
				Published: false,
			}
			if err := c.processor.ProcessEvent(taskCtx, ev); err != nil {
				c.logger.Error("failed to process transaction.completed event", zap.Error(err), zap.String("user_id", uID))
			}
		}(user.ID, fmt.Sprintf("%v", data.Amount), depTransactionID)

		// Process transfer event for sender
		go func(uID string, amt string, txID string) {
			baseCtx := ctx
			if baseCtx == nil {
				baseCtx = context.Background()
			}
			baseCtx = context.WithoutCancel(baseCtx)

			taskCtx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
			defer cancel()
			ev := &events.Event{
				Type:      "transaction.completed",
				UserID:    data.UserID, // sender user ID
				Amount:    fmt.Sprintf("%v", data.Amount),
				TxID:      trfTransactionID,
				TS:        time.Now().UTC(),
				Published: false,
			}
			if err := c.processor.ProcessEvent(taskCtx, ev); err != nil {
				c.logger.Error("failed to process transaction.completed event", zap.Error(err), zap.String("user_id", uID))
			}
		}(data.UserID, fmt.Sprintf("%v", data.Amount), trfTransactionID)
	}

	return trf, nil

}

func (c *Config) TransferToBank(ctx context.Context, info TransferInfo) (models.TransferResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// first check if the details is already in the database. if it is just procced to the point of transfer
	counterparty, err := c.getCounterParty(ctx, info.Account_Number, info.Bank_name)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.logger.Info("Counterparty not found, creating new one", zap.String("account_number", info.Account_Number), zap.String("bank_name", info.Bank_name))
			bankDetail, err := c.db.GetBankDetail(ctx, info.Bank_name)
			if err != nil {
				c.logger.Error("Failed to get bank details", zap.Error(err))
				return models.TransferResponse{}, err
			}
			c.logger.Info("Bank details for verifying account", zap.Any("bankDetail", bankDetail))
			details, err := c.verifyAccount(ctx, bankDetail.NIPCode, info.Account_Number)
			if err != nil {
				return models.TransferResponse{}, err
			}
			counterparty, err = c.createCounterParty(ctx, details)
			if err != nil {
				return models.TransferResponse{}, err
			}
		} else {
			return models.TransferResponse{}, err
		}
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
				Reason:   info.Reason,
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

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
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
		// need to log the response at this point
		c.logger.Log(c.logger.Level(), resp.Status)
		c.logger.Error("Transfer failed", zap.String("response", string(body)))
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

	apiStatus := apiResponse.Data.Attributes.Status

	status := ""
	switch apiStatus {
	case "PENDING":
		// implement the redis case for this???
		c.logger.Info("Transfer is pending, saving transaction details", zap.String("transaction_id", apiResponse.Data.ID))
		status = "pending"
	case "FAILED":
		c.logger.Info("Transfer is pending, saving transaction details", zap.String("transaction_id", apiResponse.Data.ID))
		status = "failed"
	case "SUCCESS":
		c.logger.Info("Transfer is pending, saving transaction details", zap.String("transaction_id", apiResponse.Data.ID))
		status = "success"
	default:
		c.logger.Info("Transfer is pending, saving transaction details", zap.String("transaction_id", apiResponse.Data.ID))
		status = "pending"
	}

	amt := strconv.Itoa(int(info.Amount))

	result := models.TransferResponse{
		Status:                 status,
		Amount:                 amt,
		UserID:                 info.UserID,
		FullName:               info.FullName,
		TransactionProduct:     "Money Transfer",
		TransactionDescription: "From NGN Wallet",
		Reason:                 info.Reason,
		Order_ID:               orderID,
		Transaction_ID:         transactionID,
		TXN:                    info.TXN,
		Reference:              apiResponse.Data.ID,
		CreatedAt:              time.Now().UTC(),
	}

	switch info.Source {
	case "direct":
		result.Bank_Name = &counterparty.BankName
		result.Account_Name = &counterparty.AccountName
		result.Account_Number = &counterparty.AccountNumber

	case "aremxyplug":
		result.Username = &info.Username
		result.Email = &info.Email
		result.Phone = &info.Phone
		result.CustomerName = &info.RecipientName

	}

	if err := c.saveTransaction(ctx, result); err != nil {
		c.logger.Error(err.Error())
		return result, DBConnectionError(err)
	}

	return result, nil

}

func (c *Config) verifyAccount(ctx context.Context, sortCode, accNumber string) (verifyAccountResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	url := fmt.Sprintf("%s/%s/%s/%s/%s", api, "payments", "verify-account", sortCode, accNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
		c.logger.Error("Account verification failed", zap.String("response", string(body)))
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

func (c *Config) createCounterParty(ctx context.Context, info verifyAccountResponse) (models.CounterParty, error) {
	if ctx == nil {
		ctx = context.Background()
	}

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

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
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

	if err := c.saveCounterParty(ctx, result); err != nil {
		c.logger.Error(err.Error())
		return result, DBConnectionError(err)
	}

	return result, nil
}

// endpoint to verify a transfer from the API, we will save all transactions regardless.
func (c *Config) verifyTransfer(ctx context.Context, id string) (transferResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	url := fmt.Sprintf("%s/%s/%s", api, "verify", id)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

func (c *Config) saveTransaction(ctx context.Context, details models.TransferResponse) error {
	if ctx == nil {
		ctx = context.Background()
	}
	err := c.db.SaveTransfer(ctx, details)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) saveCounterParty(ctx context.Context, conterparty models.CounterParty) error {
	if ctx == nil {
		ctx = context.Background()
	}
	err := c.db.SaveCounterParty(ctx, conterparty)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) getCounterParty(ctx context.Context, accountname, bankname string) (models.CounterParty, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	counterparty, err := c.db.GetCounterParty(ctx, accountname, bankname)
	if err != nil {
		return models.CounterParty{}, err
	}

	return counterparty, nil
}
