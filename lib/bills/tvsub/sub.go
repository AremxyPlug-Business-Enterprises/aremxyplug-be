package tvsub

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

type TvConn struct {
	db     db.UtilitiesStore
	logger *zap.Logger
}

type responses struct {
	Code string `json:"code"`
}

func NewTvConn(db db.UtilitiesStore, Logger *zap.Logger) *TvConn {
	return &TvConn{
		db:     db,
		logger: Logger,
	}
}

// buy tvsubscription
// first verifiy the smartcard number
func (t *TvConn) BuySub(data models.TvInfo) (*models.BillResult, error) {

	data.RequestID = randomgen.GenerateRequestID()
	orderID, err := randomgen.GenerateOrderID()
	if err != nil {
		return nil, t.logAndReturnError("error generating orderID", err)
	}
	transactionID := randomgen.GenerateTransactionID("tv")
	valid, err := verifyCard(data.SmartCard_Number, data.DecoderType)
	if err != nil || !valid {
		// return unable to verify the card number
		t.logger.Error("Verification failed", zap.Error(err))
		return nil, err
	}
	resp, err := t.buySub(data)
	if err != nil {
		t.logger.Error("Buying failed", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	apiResponse := &models.TvAPI{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		log.Print(err)
	}
	log.Println(apiResponse)
	fmt.Printf("%v\n", apiResponse)

	result := &models.BillResult{
		DecoderType:   data.DecoderType,
		Package:       data.Package,
		IucNumber:     data.SmartCard_Number,
		Phone:         data.Phone,
		Email:         data.Email,
		Product:       apiResponse.Content.Transcations.Type,
		Description:   apiResponse.Content.Transcations.Product_Desc,
		OrderID:       orderID,
		TranscationID: transactionID,
		RequestID:     apiResponse.RequestID,
		Amount:        apiResponse.Content.Transcations.Amount,
	}

	if err := t.saveTransaction(result); err != nil {
		t.logAndReturnError("error saving transaction to database", err)
	}

	return result, nil
}

// query tvsubscription
func (t *TvConn) QueryTransaction(requestID string) (models.BillResult, error) {

	resp, err := t.queryTransaction(requestID)
	if err != nil {
		return models.BillResult{}, t.logAndReturnError("error communicating with server", err)
	}
	defer resp.Body.Close()

	apiResponse := &models.TvAPI{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return models.BillResult{}, t.logAndReturnError("error decoding response body", err)
	}

	if apiResponse.Code != "000" {
		return models.BillResult{}, nil
	}

	result, err := t.getTransactionDetails(apiResponse.RequestID)
	if err != nil {
		return models.BillResult{}, t.logAndReturnError("failed to get user's transactions", err)
	}

	return result, nil

}

// get tvsubscription transaction history
func (t *TvConn) GetUserTransactions(user string) ([]models.BillResult, error) {

	result, err := t.getAllTransaction("user")
	if err != nil {
		return nil, t.logAndReturnError("failed to get user's transactions", err)
	}

	return result, nil

}

func (t *TvConn) GetTransactionDetails(id string) (models.BillResult, error) {

	result, err := t.getTransactionDetails(id)
	if err != nil {
		return models.BillResult{}, t.logAndReturnError("failed to get transaction details", err)
	}

	return result, nil
}

// func to be used by admin to return all transaction in database
func (t *TvConn) GetAllTransactions() ([]models.BillResult, error) {

	result, err := t.getAllTransaction("")
	if err != nil {

		return nil, t.logAndReturnError("failed to get transactions from database", err)
	}

	return result, nil

}

func verifyCard(iucNumber, service string) (bool, error) {
	formdata := url.Values{
		"billersCode": {iucNumber},
		"serviceID":   {service},
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

	log.Println(url)
	log.Printf("%s\n%s", pk, sk)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	apiResponse := responses{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return false, err
	}
	code := apiResponse.Code
	log.Println(code)
	if code != "000" {
		return false, errors.New("smart card number is not valid")
	}

	return true, nil
}

func (t *TvConn) buySub(data models.TvInfo) (*http.Response, error) {

	amount := strconv.Itoa(data.Amount)

	formdata := url.Values{
		"request_id":        {data.RequestID},
		"serviceID":         {data.DecoderType},
		"billersCode":       {data.SmartCard_Number},
		"variation_code":    {data.Package},
		"amount":            {amount},
		"phone":             {data.Phone},
		"subscription_type": {data.SubType},
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

func (t *TvConn) saveTransaction(details *models.BillResult) error {
	err := t.db.SaveTVSubcriptionTransaction(details)
	if err != nil {
		return err
	}
	return nil
}

func (t *TvConn) getTransactionDetails(id string) (models.BillResult, error) {
	result, err := t.db.GetTvSubscriptionDetails(id)
	if err != nil {
		return models.BillResult{}, err
	}
	return result, nil
}

func (t *TvConn) getAllTransaction(user string) ([]models.BillResult, error) {
	result, err := t.db.GetAllTvSubTransactions(user)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *TvConn) queryTransaction(requestID string) (*http.Response, error) {

	formdata := url.Values{
		"request_id": {requestID},
	}

	body := bytes.NewBufferString(formdata.Encode())
	url := fmt.Sprintf("%s/%s", api, "requery")

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (d *TvConn) logAndReturnError(errorMsg string, err error) error {
	d.logger.Error(errorMsg, zap.Error(err))
	return errors.New(errorMsg)
}
