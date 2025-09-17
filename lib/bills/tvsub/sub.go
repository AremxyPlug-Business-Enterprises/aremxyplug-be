package tvsub

import (
	"bytes"
	"encoding/json"
	"errors"
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
	"github.com/aremxyplug-be/lib/randomgen"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	api = os.Getenv("VTPASS_SANDBOX")
	pk  = os.Getenv("APIKey")
	sk  = os.Getenv("SK")
)

var (
	ErrInvalidCardNumber  = errors.New("invalid card number")
	ErrBillerNotAvailable = errors.New("biller not available")
)

type TvConn struct {
	db     db.UtilitiesStore
	logger *zap.Logger
}

func NewTvConn(db db.UtilitiesStore, Logger *zap.Logger) *TvConn {
	return &TvConn{
		db:     db,
		logger: Logger,
	}
}

// buy tvsubscription
// first verifiy the smartcard number
func (t *TvConn) BuySub(data TvInfo) (*models.TV_Result, error) {

	data.RequestID = randomgen.GenerateRequestID()
	data.SubType = "change"
	orderID, err := randomgen.GenerateOrderID()
	if err != nil {
		return nil, t.logAndReturnError("error generating orderID", err)
	}
	transactionID := randomgen.GenerateTransactionID("tv")

	resp, err := t.buySub(data)
	if err != nil {
		t.logger.Error("Buying failed", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	apiResponse := tvAPI{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, t.logAndReturnError("error decoding response body", err)
	}
	log.Printf("%+v", apiResponse)

	// if apiResponse.Code != "000" {
	// 	t.logger.Error("error processing payment", zap.Any("apiresponse", apiResponse))
	// 	return nil, t.logAndReturnError("error processing payment", errors.New(""))
	// }
	fmt.Printf("%+v\n", apiResponse)

	status := ""
	switch apiResponse.Content.Transactions.Status {
	case "delivered":
		status = "success"
	case "pending":
		status = "pending"
	case "failed":
		status = "failed"
	default:
		status = "failed"
	}

	var token *string
	if data.DecoderType == "showmax" {
		token = &apiResponse.PurchasedCode
	} else {
		token = nil
	}

	decoderType := cases.Title(language.English).String(data.DecoderType)

	transacProd := "TV Subscription"
	transDesc := decoderType + " " + "Subscription"
	pkge := cases.Title(language.English).String(data.Package)

	result := &models.TV_Result{
		UserID:                 data.UserID,
		Status:                 status,
		DecoderType:            data.DecoderType,
		Package:                pkge,
		IucNumber:              data.SmartCard_Number,
		Phone:                  data.Phone,
		Email:                  data.Email,
		FullName:               data.Name,
		TransactionProduct:     transacProd,
		TransactionDescription: transDesc,
		OrderID:                orderID,
		TransactionID:          transactionID,
		RequestID:              apiResponse.RequestID,
		ReferenceNumber:        apiResponse.Content.Transactions.TransactionID,
		Token:                  token,
		Amount:                 data.Amount,
		CreatedAt:              time.Now().UTC(),
		TXN:                    data.TXN,
	}

	if err := t.saveTransaction(result); err != nil {
		return nil, t.logAndReturnError("error saving transaction to database", err)
	}

	return result, nil
}

// query tvsubscription
func (t *TvConn) QueryTransaction(requestID string) (models.TV_Result, error) {

	resp, err := t.queryTransaction(requestID)
	if err != nil {
		return models.TV_Result{}, t.logAndReturnError("error communicating with server", err)
	}
	defer resp.Body.Close()

	apiResponse := &tvAPI{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return models.TV_Result{}, t.logAndReturnError("error decoding response body", err)
	}

	if apiResponse.Code != "000" {
		return models.TV_Result{}, nil
	}

	result, err := t.getTransactionDetails(apiResponse.RequestID)
	if err != nil {
		return models.TV_Result{}, t.logAndReturnError("failed to get user's transactions", err)
	}

	return result, nil

}

// get tvsubscription transaction history
func (t *TvConn) GetUserTransactions(user string) ([]models.TV_Result, error) {

	result, err := t.getAllTransaction("user")
	if err != nil {
		return nil, t.logAndReturnError("failed to get user's transactions", err)
	}

	return result, nil

}

func (t *TvConn) GetTransactionDetails(id string) (models.TV_Result, error) {

	result, err := t.getTransactionDetails(id)
	if err != nil {
		return models.TV_Result{}, t.logAndReturnError("failed to get transaction details", err)
	}

	return result, nil
}

// func to be used by admin to return all transaction in database
func (t *TvConn) GetAllTransactions() ([]models.TV_Result, error) {

	result, err := t.getAllTransaction("")
	if err != nil {

		return nil, t.logAndReturnError("failed to get transactions from database", err)
	}

	return result, nil

}

func (t *TvConn) VerifyCard(service, iucNumber string) (verifyResponse, error) {
	formdata := url.Values{
		"billersCode": {iucNumber},
		"serviceID":   {service},
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

	log.Println(url)
	log.Printf("%s\n%s", pk, sk)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return verifyResponse{}, err
	}
	defer resp.Body.Close()

	// Read body once
	bdy, err := io.ReadAll(resp.Body)
	if err != nil {
		return verifyResponse{}, err
	}

	// Parse into base struct
	apiResponse := serverResponse{}
	if err := json.Unmarshal(bdy, &apiResponse); err != nil {
		return verifyResponse{}, err
	}

	cnt := content{}

	if err := json.Unmarshal(apiResponse.Content, &cnt); err == nil {
		// ✅ object case
		if apiResponse.Code != "000" || cnt.Error != "" {
			return verifyResponse{}, ErrInvalidCardNumber
		}
		return verifyResponse{
			Name:  cnt.CustomerName,
			Phone: cnt.CustomerNumber,
		}, nil
	}

	// Otherwise, maybe it's a string
	var str string
	if err := json.Unmarshal(apiResponse.Content, &str); err == nil {
		if apiResponse.Code != "000" {
			return verifyResponse{}, ErrBillerNotAvailable
		}
		// return empty verifyResponse but not an error
		return verifyResponse{}, nil
	}

	return verifyResponse{}, fmt.Errorf("unexpected content format: %s", string(apiResponse.Content))
}

func (t *TvConn) buySub(data TvInfo) (*http.Response, error) {

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

func (t *TvConn) saveTransaction(details *models.TV_Result) error {
	err := t.db.SaveTVSubcriptionTransaction(details)
	if err != nil {
		return err
	}
	return nil
}

func (t *TvConn) getTransactionDetails(id string) (models.TV_Result, error) {
	result, err := t.db.GetTvSubscriptionDetails(id)
	if err != nil {
		return models.TV_Result{}, err
	}
	return result, nil
}

func (t *TvConn) getAllTransaction(user string) ([]models.TV_Result, error) {
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
