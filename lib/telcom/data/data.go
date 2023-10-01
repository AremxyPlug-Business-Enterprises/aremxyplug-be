package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/randomgen"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	api   = os.Getenv("DONTECH")
	token = "Token " + os.Getenv("DONTECH_AUTH")
)

type DataConn struct {
	Dbconn db.DataStore
	Logger *zap.Logger
}

func NewData(DbConn db.DataStore, logger *zap.Logger) *DataConn {
	return &DataConn{
		Dbconn: DbConn,
		Logger: logger,
	}
}

// BuyData makes a call to the api to initiate a purchase
func (d *DataConn) BuyData(data models.DataInfo) (*models.DataResult, error) {
	data.Ported_number = true

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(&data)
	id, err := randomgen.GenerateOrderID()
	if err != nil {
		// check error
		d.Logger.Error("Could not generate orderID...", zap.Error(err))
		return nil, errors.New("Api Call Error")
	}

	req, err := http.NewRequest("POST", api+"/data/", &buf)
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Add("Authorization", token)
	req.Header.Add("Content-Type", "application/json")
	//resp, err := d.RestyClient.R().SetBody(info).SetAuthToken(token).Post("/user/")
	if err != nil {
		// return err
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	apiResponse := models.APIResponse{}
	if resp.StatusCode == http.StatusCreated {

		if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
			if err == io.EOF {
				log.Println("No response from body")
				// edu.Logger.Error("Empty response body", zap.Error(err))
				return nil, errors.New("empty response from server")
			} else {
				log.Println("other error:", err)
				d.Logger.Error("error returned from server: ", zap.Error(err))
				return nil, errors.New("error returned from server...")
			}
		}

		transactionID := randomgen.GenerateTransactionID("dat")
		result := &models.DataResult{
			Network:         apiResponse.Plan_network,
			Phone_Number:    apiResponse.Mobile_number,
			ReferenceNumber: apiResponse.Ident,
			Plan_Amount:     apiResponse.Plan_amount,
			PlanName:        apiResponse.Plan_Name,
			CreatedAt:       time.Now().String(),
			OrderID:         id,
			TransactionID:   transactionID,
			Status:          apiResponse.Status,
			Name:            data.Name,
			ApiID:           apiResponse.Id,
		}
		if err := d.saveTransacation(result); err != nil {
			d.Logger.Error("Database error try again...", zap.Error(err))
			return nil, errors.New("Database Insert Error...")
		}

		log.Println(resp.StatusCode)
		return result, nil
	} else {
		d.Logger.Error("Api Call Error: %v", zap.String("status", fmt.Sprint((resp.StatusCode))))
		return nil, errors.New("Api Call Error")
	}

}

// GetTransactionDetail takes a  id and returns the details of the transaction
func (d *DataConn) GetTransactionDetail(id string) (models.DataResult, error) {
	resp := models.DataResult{}
	res, err := d.getTransactionDetails(id)
	if err != nil {
		// write error
		d.Logger.Error("Database error try again...", zap.Error(err))
		return resp, errors.New("Database request error: " + err.Error())
	}

	return res, nil
}

// GetUserTransactions return all the data transactions associated to a user
func (d *DataConn) GetUserTransactions(user string) ([]models.DataResult, error) {

	res, err := d.getAllTransactions(user)
	if err != nil {
		// write error
		d.Logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("Database request error: " + err.Error())
	}

	return res, err
}

// PingUser is a test function to ping the api
func (d *DataConn) PingUser(w http.ResponseWriter) (*http.Response, error) {

	req, err := http.NewRequest("GET", api+"/user/", nil)
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Set("Authorization", "Token "+token)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	statusCode := res.StatusCode

	log.Println(api)

	log.Println("StatusCode: ", statusCode)

	return res, nil
}

// GetAllTransactions returns a list of all data transactions.
func (d *DataConn) GetAllTransactions() ([]models.DataResult, error) {
	var user string
	result, err := d.Dbconn.GetAllDataTransactions(user)
	if err != nil {
		d.Logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("Database request error: " + err.Error())
	}

	return result, nil
}

func (d *DataConn) QueryTransaction(id int) error {

	pid := strconv.Itoa(id)

	req, err := http.NewRequest("POST", api+"/data/"+pid, nil)
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Add("Authorization", "Token "+token)
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		// return err
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		d.Logger.Error("Error querying API...", zap.Error(err))
		return errors.New("Invalid Id...")
	}

	return nil

}

// saveTranscation saves the details of a transaction to database
func (d *DataConn) saveTransacation(details *models.DataResult) error {
	err := d.Dbconn.SaveDataTransaction(details)
	return err
}

// getTransacationDetails returns the details of a transaction
func (d *DataConn) getTransactionDetails(id string) (models.DataResult, error) {
	result, err := d.Dbconn.GetDataTransactionDetails(id)
	return result, err
}

// getAllTransaction returns all transactions, if an empty string is passed, it returns all transaction in the database
func (d *DataConn) getAllTransactions(user string) ([]models.DataResult, error) {
	results, err := d.Dbconn.GetAllDataTransactions(user)
	return results, err
}
