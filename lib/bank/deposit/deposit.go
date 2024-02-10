package deposit

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/mongo"
	"github.com/aremxyplug-be/lib/balance"
	"github.com/aremxyplug-be/lib/randomgen"
	"go.uber.org/zap"
)

/*
var (
	api    = os.Getenv("ANCHOR_SANDBOX")
	apikey = os.Getenv("ANCHORAPI_KEY")
)
*/

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

func (c *Config) Deposit(virtualNuban string) error {
	// using the list payment endpoint.
	url := fmt.Sprintf("%s/%s?%s=%s", api, "payments", "virtualNubanId", virtualNuban)

	if virtualNuban == "" {
		c.logger.Error("missing virtualNuban")
		return ErrEmptyVirtualNuban
	}

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
		c.logger.Error(err.Error())
		return ErrAPIConnectionFailed
	}
	defer resp.Body.Close()

	apiResponse := paymentResponse{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return JSONError(err)
	}
	c.logger.Log(c.logger.Level(), string(body))

	/*
		if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
			// log the error
			c.logger.Error(err.Error())
			return JSONError(err)

		}
	*/

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return JSONError(err)
	}
	log.Printf("%+v", apiResponse)

	// create a separate collection for saving virtualNubans and their associated deposit ID

	// range through the response and save only newly created deposits
	// should track the deposits with their id
	// how do I create or track a single deposit?

	// the value of the new deposits should be added to the previous balance of the user.
	paymentData := apiResponse.Data
	for _, data := range paymentData {

		orderID, err := randomgen.GenerateOrderID()
		if err != nil {
			// log the error
			return err
		}

		transctionID := randomgen.GenerateTransactionID("dep")
		attributes := data.Attributes
		virtualNuban := data.Relationships.VirtualNuban.Data.ID
		deposit := depositID{
			VirtualNuban: virtualNuban,
			ID:           data.ID,
		}

		log.Printf("%s", virtualNuban)
		log.Printf("%s", deposit.ID)

		if err := c.db.SaveDepositID(deposit); err != nil {
			if err == mongo.ErrDepositIDExist {
				continue
			}

			c.logger.Error(err.Error())
			return DBConnectionError(err)
		}

		bal, err := c.db.GetBalance(virtualNuban)
		if err != nil {
			c.logger.Error(err.Error())
			return DBConnectionError(err)
		}
		log.Println(bal)

		deposit_amount := data.Attributes.Amount
		log.Println(deposit_amount)

		newBalance, depositAmount := balance.NewBalanceDeposit(bal, deposit_amount)
		log.Println(newBalance)
		userBalance := models.Balance{
			VirtualNuban: virtualNuban,
			Balance:      newBalance,
		}
		if err = c.db.SaveBalance(virtualNuban, userBalance); err != nil {
			// log the error and return
			return DBConnectionError(err)
		}

		log.Printf("%+v", data)

		result := models.DepositResponse{
			Amount:         fmt.Sprintf("%v", depositAmount),
			WalletType:     "Nigerian NGN Wallet",
			Bank_Name:      attributes.CounterParty.Bank.Name,
			Account_Name:   attributes.CounterParty.AccountName,
			Account_No:     attributes.CounterParty.AccountNumber,
			Product:        "Virtual Account",
			Description:    "NGN Wallet Top Up",
			Message:        data.Attributes.Narration,
			Order_ID:       orderID,
			Transaction_ID: transctionID,
			Session_ID:     data.Attributes.PaymentReference,
		}

		log.Printf("%+v", result)

		if err := c.saveTransaction(result); err != nil {
			// log the error and return
			return DBConnectionError(err)
		}

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
