package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	api = os.Getenv("DONTECH")
	// token = "Token " + os.Getenv("AUTHTOKEN")
	token = os.Getenv("AUTHTOKEN")
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
	id, err := d.generateOrderID()
	if err != nil {
		// check error
		d.Logger.Error("Could not generate orderID...", zap.Error(err))
		return nil, errors.New("Api Call Error")
	}

	req, err := http.NewRequest("POST", api+"/data/", &buf)
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Add("Authorization", "Token "+token)
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
	result := &models.DataResult{}
	if resp.StatusCode != http.StatusCreated {
		d.Logger.Error("Api Call Error: %v", zap.String("status", fmt.Sprint((resp.StatusCode))))
		return nil, errors.New("Api Call Error")
	}

	json.NewDecoder(resp.Body).Decode(&apiResponse)

	result.Network = apiResponse.Plan_network
	result.Phone_Number = apiResponse.Mobile_number
	result.ReferenceNumber = apiResponse.Ident
	result.Plan_Amount = apiResponse.Plan_amount
	result.PlanName = apiResponse.Plan_Name
	result.CreatedAt = time.Now().String()
	//result.OrderID = apiResponse.Id
	result.OrderID = id
	result.TransactionID = d.generateTransactionID()
	result.Status = apiResponse.Status
	result.Name = data.Name
	log.Println(result)
	if err := d.saveTransacation(result); err != nil {
		d.Logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("Database Insert Error...")
	}

	log.Println(resp.StatusCode)
	return result, nil
}

// GetTransactionDetail takes a  id and returns the details of the transaction
func (d *DataConn) GetTransactionDetail(id string) (models.DataResult, error) {
	// check if id is a validate transaction id.

	resp := models.DataResult{}
	res, err := d.getTransactionDetails(id)
	if err != nil {
		// write error
		d.Logger.Error("Database error try again...", zap.Error(err))
		return resp, errors.New("Database request error: " + err.Error())
	}

	// return transaction details if no errors
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

	// return the list of transactions.
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
		// Log the error
		// Return an empty result and error
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
	//resp, err := d.RestyClient.R().SetBody(info).SetAuthToken(token).Post("/user/")
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

// generateTransactionID generates a unique transaction ID.
func (d *DataConn) generateTransactionID() string {
	seedRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	charset := os.Getenv("CHARSET")

	b := make([]byte, 10)
	for i := range b {
		b[i] = charset[seedRand.Intn(len(charset))]
	}

	return string(b)
}

func (d *DataConn) generateOrderID() (int, error) {
	seedRand := rand.New(rand.NewSource(int64(time.Now().UnixNano())))
	numbset := os.Getenv(("NUMBSET"))

	b := make([]byte, 10)
	for i := range b {
		b[i] = numbset[seedRand.Intn(len(numbset))]
	}

	s := string(b)

	Id, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}

	return Id, nil
}
