package deposit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/balance"
	"github.com/aremxyplug-be/lib/randomgen"
)

var (
	api    = os.Getenv("ANCHOR_SANDBOX")
	apikey = os.Getenv("ANCHORAPI_KEY")
)

type Config struct {
	db db.DataStore
}

type depositID struct {
	VirtualNuban string `json:"virtualNuban"`
	ID           string `json:"id"`
}

func NewDepositConfig(db db.DataStore) *Config {
	return &Config{db: db}
}

func (c *Config) Deposit(virtualNuban string) error {
	// using the list payment endpoint.
	url := fmt.Sprintf("%s/%s?%s=%s", api, "payments", "virtualNubanId", virtualNuban)

	// first get the user's associated account number from the database
	// add the virtualNuban to the request

	orderID, err := randomgen.GenerateOrderID()
	if err != nil {
		// log the error
		return err
	}

	transctionID := randomgen.GenerateTransactionID("dep")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// log the error
		return ErrNewRequestFailed
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("x-anchor-key", apikey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// log the error
		return ErrAPIConnectionFailed
	}
	defer resp.Body.Close()

	apiResponse := paymentResponse{}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		// log the error
		return JSONError(err)

	}

	// create a separate collection for saving virtualNubans and their associated deposit ID

	// range through the response and save only newly created deposits
	// should track the deposits with their id
	// how do I create or track a single deposit?

	// the value of the new deposits should be added to the previous balance of the user.
	paymentData := apiResponse.Data
	for _, data := range paymentData {
		attributes := data.Attributes
		virtualNuban := data.Relationships.VirtualNuban.Data.ID
		result, err := c.db.GetDepositID(virtualNuban)
		if err != nil {
			// do something with the error
			// log the error
			return DBConnectionError(err)
		}
		var id string
		if deposit_struct, ok := result.(depositID); ok {
			id = deposit_struct.ID

		}
		if id != "" {
			continue
		} else if id == "" {
			// first run through the database to see if that particular depositID already
			// if it does exist skip that deposit and run for the next, if doesn't exist add the amount to the users balance and continue.
			bal, err := c.db.GetBalance(virtualNuban)
			if err != nil {
				// log the error
				return DBConnectionError(err)
			}

			deposit_amount := data.Amount

			newBalance := balance.NewBalanceDeposit(bal, deposit_amount)
			userBalance := models.Balance{
				VirtualNuban: virtualNuban,
				Balance:      newBalance,
			}
			if err = c.db.SaveBalance(virtualNuban, userBalance); err != nil {
				// log the error and return
				return DBConnectionError(err)
			}

			deposit := depositID{
				VirtualNuban: virtualNuban,
				ID:           data.ID,
			}

			c.db.SaveDepositID(deposit)

			result := models.DepositResponse{
				Amount:         fmt.Sprintf("%v", deposit_amount),
				WalletType:     "Nigerian NGN Wallet",
				Bank_Name:      attributes.CounterParty.Bank.Name,
				Account_Name:   attributes.CounterParty.AccountName,
				Account_No:     attributes.CounterParty.AccountNumber,
				Product:        "Virtual Account",
				Description:    "NGN Wallet Top Up",
				Message:        data.Narration,
				Order_ID:       orderID,
				Transaction_ID: transctionID,
				Session_ID:     data.PaymentReference,
			}

			if err := c.saveTransaction(result); err != nil {
				// log the error and return
				return DBConnectionError(err)
			}
		}

		// then save the transaction to the database as well

	}

	// return the balance.
	return nil
}

// before any transaction, there should first be a check to see if the user has the ability to carry out the transaction.

// write to save transaction to the database
func (c *Config) saveTransaction(detail models.DepositResponse) error {
	err := c.db.SaveDeposit(detail)
	if err != nil {
		return err
	}
	return nil
}
