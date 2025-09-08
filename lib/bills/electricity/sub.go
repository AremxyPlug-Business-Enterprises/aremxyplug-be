package electricity

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

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
	db     db.UtilitiesStore
	logger *zap.Logger
}

func NewElectricConn(db db.UtilitiesStore, logger *zap.Logger) *ElectricConn {
	return &ElectricConn{
		db:     db,
		logger: logger,
	}
}

// pay electricity bill
func (e *ElectricConn) PayBill(data ElectricInfo) (*models.ElectricResult, error) {

	data.RequestID = randomgen.GenerateRequestID()
	orderID, err := randomgen.GenerateOrderID()
	if err != nil {
		return nil, e.logAndReturnError("error generating orderID", err)
	}
	transactionID := randomgen.GenerateTransactionID("ele")

	resp, err := e.payBill(data)
	if err != nil {
		return nil, e.logAndReturnError("error communicating with server", err)
	}
	defer resp.Body.Close()

	apiResponse := electricAPI{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, e.logAndReturnError("error decoding response body", err)
	}
	// log.Printf("%+v", apiResponse)
	// if apiResponse.Code != "000" {
	// 	e.logger.Error("error processing payment", zap.Any("apiresponse", apiResponse))
	// 	return nil, e.logAndReturnError("error processing payment", errors.New(""))
	// }
	transDetails := apiResponse.Content
	description := data.DiscoType + " " + data.Meter_Type
	status := ""

	switch transDetails.Transactions.Status {
	case "delivered":
		status = "success"
	case "pending":
		status = "pending"
	case "failed":
		status = "failed"
	default:
		status = "failed"
	}

	token := apiResponse.Token

	billGenerated := ""
	if token != "" {
		parts := strings.Split(apiResponse.Token, ":")
		token_generated := strings.TrimSpace(parts[1])
		billGenerated = token_generated
	}

	transctionProd := "Electricity Bills"
	amount := strconv.Itoa(data.Amount)

	result := &models.ElectricResult{
		Status:                 status,
		UserID:                 data.UserID,
		Amount:                 amount,
		DiscoType:              data.DiscoType,
		MeterType:              data.Meter_Type,
		MeterNumber:            transDetails.Transactions.UniqueElement,
		Phone:                  data.Phone,
		VerifiedName:           data.VerifiedName,
		FullName:               data.FullName,
		BillGenerated:          billGenerated,
		Email:                  data.Email,
		TransactionProduct:     transctionProd,
		TransactionDescription: description,
		OrderID:                orderID,
		TransactionID:          transactionID,
		RequestID:              apiResponse.RequestID,
		ReferenceNumber:        transDetails.Transactions.TransactionID,
		CreatedAt:              time.Now().UTC(),
		TXN:                    data.TXN,
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

	apiResponse := electricAPI{}
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
func (e *ElectricConn) GetUserTransactions(username string) ([]models.ElectricResult, error) {
	result, err := e.getAllTransaction(username)
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

func (e *ElectricConn) payBill(data ElectricInfo) (*http.Response, error) {

	amount := strconv.Itoa(data.Amount)
	phone := data.Phone

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

func (e *ElectricConn) getAllTransaction(username string) ([]models.ElectricResult, error) {
	result, err := e.db.GetAllElectricSubTransactions(username)
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

func (e *ElectricConn) logAndReturnError(errorMsg string, err error) error {
	e.logger.Error(errorMsg, zap.Error(err))
	return errors.New(errorMsg)
}

// return an error message for when the meter number is not correct
func (e *ElectricConn) VerifyMeterNo(discoType, meterNo, meterType string) (verifyResponse, error) {

	if meterNo == "" {
		return verifyResponse{}, errors.New("no meter number provided")
	}

	formdata := url.Values{
		"serviceID":   {discoType},
		"billersCode": {meterNo},
		"type":        {meterType},
	}

	body := bytes.NewBufferString(formdata.Encode())
	url := fmt.Sprintf("%s/%s", api, "merchant-verify")

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return verifyResponse{}, err
	}
	req.Header.Set("api-key", pk)
	req.Header.Set("secret-key", sk)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return verifyResponse{}, err
	}

	apiResponse := serverResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return verifyResponse{}, err
	}
	e.logger.Info("verify meter response", zap.String("code", apiResponse.Code))
	if apiResponse.Code != "000" {
		e.logger.Error("error verifying meter number", zap.Any("apiresponse", apiResponse))
		return verifyResponse{}, errors.New("invalid meter number")
	}

	if apiResponse.Content.Err != "" {
		return verifyResponse{}, errors.New(apiResponse.Content.Err)
	}
	return verifyResponse{
		Name:     apiResponse.Content.Name,
		Meter_No: apiResponse.Content.Meter_Number,
	}, nil
}
