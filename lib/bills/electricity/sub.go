package electricity

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/randomgen"
	"go.uber.org/zap"
)

var (
	api = os.Getenv("VTPASS_SANDBOX")
	pk  = os.Getenv("APIKey")
	sk  = os.Getenv("SK")
)

type ElectricConn struct {
	db     db.DataStore
	logger *zap.Logger
}

func NewElectricConn(db db.DataStore, logger *zap.Logger) *ElectricConn {
	return &ElectricConn{
		db:     db,
		logger: logger,
	}
}

// pay electricity bill
func (e *ElectricConn) PayBill(data models.ElectricInfo) (*models.ElectricResult, error) {

	data.RequestID = randomgen.GenerateRequestID()
	orderID, err := randomgen.GenerateOrderID()
	if err != nil {
		return nil, e.logAndReturnError("error generating orderID", err)
	}
	transactionID := randomgen.GenerateTransactionID("ele")

	validNo, err := e.verifyMeterNo(data.DiscoType, data.Meter_No, data.Meter_Type)
	if err != nil {
		return nil, e.logAndReturnError("error verifying meter number", err)
	}
	if !validNo {
		return nil, e.logAndReturnError("meter number is not valid", errors.New("invalid meter number"))
	}

	resp, err := e.payBill(data)
	if err != nil {
		return nil, e.logAndReturnError("error communicating with server", err)
	}
	defer resp.Body.Close()

	apiResponse := models.ElectricAPI{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, e.logAndReturnError("error decoding response body", err)
	}
	log.Println(apiResponse)
	transDetails := apiResponse.Contents.Transactions
	phone := strconv.Itoa(data.Phone)
	description := data.DiscoType + data.Meter_Type

	result := &models.ElectricResult{
		DiscoType:     data.DiscoType,
		MeterType:     data.Meter_Type,
		MeterNumber:   transDetails.Meter_No,
		Phone:         phone,
		Email:         data.Email,
		Product:       transDetails.Type,
		Description:   description,
		OrderID:       orderID,
		TransactionID: transactionID,
		RequestID:     apiResponse.RequestID,
	}

	if err := e.saveTransaction(result); err != nil {
		return nil, e.logAndReturnError("error saving transaction to database", err)
	}

	return result, nil
}

// query eletricity bill
func (e *ElectricConn) QueryTransaction(id string) (models.ElectricResult, error) {

	resp, err := e.queryTransaction(id)
	if err != nil {
		return models.ElectricResult{}, e.logAndReturnError("error communicating with server", err)
	}
	defer resp.Body.Close()

	apiResponse := models.ElectricAPI{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return models.ElectricResult{}, e.logAndReturnError("error decoding response body", err)
	}

	if apiResponse.Code != "000" {
		return models.ElectricResult{}, nil
	}

	result, err := e.getTransactionDetails(apiResponse.RequestID)
	if err != nil {
		return models.ElectricResult{}, e.logAndReturnError("failed to get user's transactions", err)
	}

	return result, nil
}

// get transaction history
func (e *ElectricConn) GetUserTransactions(user string) ([]models.ElectricResult, error) {
	result, err := e.getAllTransaction("user")
	if err != nil {
		return nil, e.logAndReturnError("failed to get user's transactions", err)
	}

	return result, nil
}

func (e *ElectricConn) GetTransactionDetails(id string) (models.ElectricResult, error) {
	result, err := e.getTransactionDetails(id)
	if err != nil {
		return models.ElectricResult{}, e.logAndReturnError("failed to get transaction details", err)
	}

	return result, nil
}

// GetAllTransaction returns all transactions, to be used by admin
func (e *ElectricConn) GetAllTransactions() ([]models.ElectricResult, error) {

	result, err := e.getAllTransaction("")
	if err != nil {
		return nil, e.logAndReturnError("failed to get transactions from database", err)
	}

	return result, nil

}

func (e *ElectricConn) payBill(data models.ElectricInfo) (*http.Response, error) {

	amount := strconv.Itoa(data.Amount)
	phone := strconv.Itoa(data.Phone)

	formdata := url.Values{
		"request_id":     {data.RequestID},
		"serviceID":      {data.DiscoType},
		"billersCode":    {data.Meter_No},
		"variation_code": {data.Meter_Type},
		"amount":         {amount},
		"phone":          {phone},
	}

	body := bytes.NewBufferString(formdata.Encode())
	url := fmt.Sprintf("%s/%s", api, "pay")

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

func (e *ElectricConn) saveTransaction(details *models.ElectricResult) error {
	err := e.db.SaveElectricTransaction(details)
	if err != nil {
		return err
	}
	return nil
}

func (e *ElectricConn) getTransactionDetails(id string) (models.ElectricResult, error) {
	result, err := e.db.GetElectricSubDetails(id)
	if err != nil {
		return models.ElectricResult{}, err
	}
	return result, nil
}

func (e *ElectricConn) getAllTransaction(user string) ([]models.ElectricResult, error) {
	result, err := e.db.GetAllElectricSubTransactions(user)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (e *ElectricConn) queryTransaction(requestID string) (*http.Response, error) {

	formdata := url.Values{
		"request_id": {requestID},
	}

	body := bytes.NewBufferString(formdata.Encode())
	url := fmt.Sprintf("%s/%s", api, "requery")

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, e.logAndReturnError("failed to create request", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, e.logAndReturnError("error communicating with server", err)
	}

	return resp, nil
}

func (d *ElectricConn) logAndReturnError(errorMsg string, err error) error {
	d.logger.Error(errorMsg, zap.Error(err))
	return errors.New(errorMsg)
}

func (d *ElectricConn) verifyMeterNo(discoType, meterNo, meterType string) (bool, error) {

	formdata := url.Values{
		"serviceID":      {discoType},
		"billersCode":    {meterNo},
		"variation_code": {meterType},
	}

	body := bytes.NewBufferString(formdata.Encode())
	url := fmt.Sprintf("%s/%s", api, "merchant-verify")

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return false, err
	}
	req.Header.Set("api-key", pk)
	req.Header.Set("secret-key", sk)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	_, err = client.Do(req)
	if err != nil {
		return false, err
	}

	return true, nil
}
