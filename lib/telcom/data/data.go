package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/models/telcom"
	"github.com/aremxyplug-be/lib/randomgen"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	api   = os.Getenv("DONTECH")
	token = "Token " + os.Getenv("DONTECH_AUTH")
	vtapi = os.Getenv("VTPASS_SANDBOX")
	pk    = os.Getenv("APIKey")
	sk    = os.Getenv("SK")
)

type DataConn struct {
	Dbconn db.TelcomStore
	Logger *zap.Logger
}

func NewData(DbConn db.TelcomStore, logger *zap.Logger) *DataConn {
	return &DataConn{
		Dbconn: DbConn,
		Logger: logger,
	}
}

// BuyData makes a call to the api to initiate a purchase
func (d *DataConn) BuyData(data telcom.DataInfo) (*telcom.DataResult, error) {
	data.Ported_number = true

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&data); err != nil {
		return nil, d.logAndReturnError("unable to encode data", err)
	}
	id, err := randomgen.GenerateOrderID()
	if err != nil {
		d.Logger.Error("Could not generate orderID...", zap.Error(err))
		return nil, d.logAndReturnError("Could not generate orderID", err)
	}

	req, err := http.NewRequest("POST", api+"/data/", &buf)
	if err != nil {
		return nil, err
	}
	//req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Add("Authorization", token)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	apiResponse := telcom.APIResponse{}

	log.Println(resp.StatusCode)
	if resp.StatusCode == http.StatusCreated {

		if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
			if err == io.EOF {
				return nil, d.logAndReturnError("Empty response body retured from server", err)
			}
			return nil, d.logAndReturnError("error while decoding json", err)
		}

		transactionID := randomgen.GenerateTransactionID("dat")
		result := &telcom.DataResult{
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

		return result, nil
	} else {
		d.Logger.Error("Api Call Error: %s", zap.String("status", fmt.Sprint((resp.Status))))
		body, err := json.Marshal(resp.Body)
		if err != nil {
			return nil, err
		}
		log.Print(string(body))
		return nil, fmt.Errorf("%v", resp.Status)
	}

}

func (d *DataConn) BuySpecData(data models.SpectranetInfo) (*models.SpectranetResult, error) {

	data.RequestID = randomgen.GenerateRequestID()
	orderid, err := randomgen.GenerateOrderID()
	if err != nil {
		return nil, d.logAndReturnError("unable to generate orderid", err)
	}
	transactionID := randomgen.GenerateTransactionID("dat")
	resp, err := d.buySpecData(data)
	if err != nil {
		d.Logger.Error("error returned from server", zap.Any("error:", err))
		return nil, err
	}
	defer resp.Body.Close()

	apiResponse := models.SpectranetApiResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		if err == io.EOF {
			return nil, d.logAndReturnError("Empty response body retured from server", err)
		}
		return nil, d.logAndReturnError("error while decoding json", err)
	}

	trans_content := apiResponse.Content.Transcations
	result := &models.SpectranetResult{
		Network:         data.Network,
		Product:         data.Product,
		Plan:            data.Plan,
		Phone_Number:    trans_content.Phone_Number,
		No_of_Pins:      trans_content.Quantity,
		Amount:          trans_content.Amount,
		ProductDesc:     trans_content.Type,
		Description:     data.Product,
		TranscationID:   transactionID,
		OrderID:         orderid,
		ReferenceNumber: trans_content.TransactionID,
		RequestID:       apiResponse.RequestID,
	}

	if err := d.saveTransacation(result); err != nil {
		return nil, d.logAndReturnError("error while saving to database", err)
	}

	fmt.Printf("%+v\n", apiResponse)

	return result, nil

}

func (d *DataConn) BuySmileData(data models.SmileInfo) (*models.SmileResult, error) {

	data.RequestID = randomgen.GenerateRequestID()
	orderid, err := randomgen.GenerateOrderID()
	if err != nil {
		return nil, d.logAndReturnError("unable to generate orderid", err)
	}
	transactionID := randomgen.GenerateTransactionID("dat")
	resp, err := d.buySmileData(data)
	if err != nil {
		return nil, d.logAndReturnError("error returned from server", err)
	}

	defer resp.Body.Close()

	apiResponse := models.SmileAPIresponse{}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		if err == io.EOF {
			return nil, d.logAndReturnError("Empty response body retured from server", err)
		}
		return nil, d.logAndReturnError("error while decoding json", err)
	}

	trans_content := apiResponse.Content.Transcations
	result := &models.SmileResult{
		Network:         data.Network,
		ProductPlan:     trans_content.Product_Desc,
		Email:           data.Email,
		AccountID:       data.AccountID,
		Phone_Number:    data.AccountID,
		Amount:          trans_content.Amount,
		Product:         trans_content.Type,
		Description:     trans_content.Product_Desc,
		TranscationID:   transactionID,
		OrderID:         orderid,
		ReferenceNumber: trans_content.TransactionID,
		RequestID:       apiResponse.RequestID,
	}

	if err := d.saveTransacation(result); err != nil {
		return nil, d.logAndReturnError("error while saving to database", err)
	}

	return result, nil
}

// GetTransactionDetail takes a  id and returns the details of the transaction
func (d *DataConn) GetTransactionDetail(id string) (telcom.DataResult, error) {
	resp := telcom.DataResult{}
	res, err := d.getTransactionDetails(id)
	if err != nil {
		return resp, d.logAndReturnError("error while communicating with database", err)
	}

	return res, nil
}

// GetUserTransactions return all the data transactions associated to a user
func (d *DataConn) GetUserTransactions(user string) ([]telcom.DataResult, error) {

	res, err := d.getAllTransactions(user)
	if err != nil {
		return res, d.logAndReturnError("error while communicating with database", err)
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
func (d *DataConn) GetAllTransactions() ([]telcom.DataResult, error) {
	var user string
	result, err := d.getAllTransactions(user)
	if err != nil {
		d.Logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("Database request error: " + err.Error())
	}

	return result, nil
}

func (d *DataConn) GetSpecTransDetails(requestID string) (models.SpectranetResult, error) {
	resp := models.SpectranetResult{}
	res, err := d.getSpecDataDetails(requestID)
	if err != nil {
		return resp, d.logAndReturnError("error while communicating with database", err)
	}

	return res, nil
}

func (d *DataConn) GetSpecUserTransactions(user string) ([]models.SpectranetResult, error) {

	res, err := d.getAllSpecTransactions(user)
	if err != nil {
		d.Logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("database request error: " + err.Error())
	}

	return res, err
}

func (d *DataConn) GetAllSpecTransactions() ([]models.SpectranetResult, error) {
	var user string
	result, err := d.getAllSpecTransactions(user)
	if err != nil {
		d.Logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("Database request error: " + err.Error())
	}

	return result, nil
}

func (d *DataConn) GetSmileTransDetails(requestID string) (models.SmileResult, error) {
	resp := models.SmileResult{}
	res, err := d.getSmileDataDetails(requestID)
	if err != nil {
		// write error
		d.Logger.Error("Database error try again...", zap.Error(err))
		return resp, errors.New("Database request error: " + err.Error())
	}

	return res, nil
}

func (d *DataConn) GetSmileUserTransactions(user string) ([]models.SmileResult, error) {

	res, err := d.getAllSmileTransactions(user)
	if err != nil {
		// write error
		d.Logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("database request error: " + err.Error())
	}

	return res, err
}

func (d *DataConn) GetAllSmileTransactions() ([]models.SmileResult, error) {
	var user string
	result, err := d.getAllSmileTransactions(user)
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

func (d *DataConn) buySmileData(data models.SmileInfo) (*http.Response, error) {

	formdata := url.Values{
		"request_id":     {data.RequestID},
		"serviceID":      {data.Product},
		"billersCode":    {data.AccountID},
		"variation_code": {data.Product_plan},
		"phone":          {data.Phone_Number},
	}

	body := bytes.NewBufferString(formdata.Encode())
	url := fmt.Sprintf("%s/%s", vtapi, "pay")

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-key", pk)
	req.Header.Set("secret-key", sk)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (d *DataConn) buySpecData(data models.SpectranetInfo) (*http.Response, error) {

	amount := strconv.Itoa(data.Amount)

	formdata := url.Values{
		"request_id":     {data.RequestID},
		"serviceID":      {data.Network},
		"billersCode":    {data.Phone_Number},
		"variation_code": {data.Plan},
		"amount":         {amount},
		"phone":          {data.Phone_Number},
		"quantity":       {data.No_of_Pins},
	}

	body := bytes.NewBufferString(formdata.Encode())
	url := fmt.Sprintf("%s/%s", vtapi, "pay")

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-key", pk)
	req.Header.Set("secret-key", sk)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// saveTranscation saves the details of a transaction to database
func (d *DataConn) saveTransacation(details interface{}) error {
	err := d.Dbconn.SaveDataTransaction(details)
	return err
}

// getTransacationDetails returns the details of a transaction
func (d *DataConn) getTransactionDetails(id string) (telcom.DataResult, error) {
	result, err := d.Dbconn.GetDataTransactionDetails(id)
	return result, err
}

// getAllTransaction returns all transactions, if an empty string is passed, it returns all transaction in the database
func (d *DataConn) getAllTransactions(user string) ([]telcom.DataResult, error) {
	results, err := d.Dbconn.GetAllDataTransactions(user)
	return results, err
}

// get transactions history
func (d *DataConn) getSpecDataDetails(requestID string) (models.SpectranetResult, error) {
	result, err := d.Dbconn.GetSpecTransDetails(requestID)
	return result, err
}

func (d *DataConn) getAllSpecTransactions(user string) ([]models.SpectranetResult, error) {
	result, err := d.Dbconn.GetAllSpecDataTransactions(user)
	return result, err
}

func (d *DataConn) getSmileDataDetails(id string) (models.SmileResult, error) {
	result, err := d.Dbconn.GetSmileTransDetails(id)
	return result, err
}

func (d *DataConn) getAllSmileTransactions(user string) ([]models.SmileResult, error) {
	result, err := d.Dbconn.GetAllSmileDataTransactions(user)
	return result, err
}

func (d *DataConn) logAndReturnError(errorMsg string, err error) error {
	d.Logger.Error(errorMsg, zap.Error(err))
	return errors.New(errorMsg)
}
