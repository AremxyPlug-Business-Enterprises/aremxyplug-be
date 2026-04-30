package electricity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	dbmodels "github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/randomgen"
	"go.uber.org/zap"
)

var (
	api = os.Getenv("VTPASS")
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
func (e *ElectricConn) PayBill(ctx context.Context, data ElectricInfo) (*models.ElectricResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	data.RequestID = randomgen.GenerateRequestID()
	orderID, err := randomgen.GenerateOrderID()
	if err != nil {
		return nil, e.logAndReturnError("error generating orderID", err)
	}
	transactionID := randomgen.GenerateTransactionID("ele")

	resp, requestSnapshot, failureType, err := e.payBill(ctx, data)
	if err != nil {
		e.saveProviderAudit(ctx, data, &models.ElectricResult{
			UserID:        data.UserID,
			OrderID:       orderID,
			TransactionID: transactionID,
			RequestID:     data.RequestID,
		}, requestSnapshot, dbmodels.ProviderAuditResponseSnapshot{}, "failed", failureType, err.Error(), "", "", nil)
		return nil, e.logAndReturnError("error communicating with server", err)
	}
	if resp.Body == nil {
		e.saveProviderAudit(ctx, data, &models.ElectricResult{
			UserID:        data.UserID,
			OrderID:       orderID,
			TransactionID: transactionID,
			RequestID:     data.RequestID,
		}, requestSnapshot, dbmodels.ProviderAuditResponseSnapshot{
			StatusCode: resp.StatusCode,
			Headers:    e.headerSnapshot(resp.Header),
		}, "failed", "decode_error", "response body is nil", "", "", nil)
		return nil, e.logAndReturnError("error decoding response body", errors.New("response body is nil"))
	}
	defer resp.Body.Close()

	apiResponse := electricAPI{}
	rawBody, responseHeaders, err := e.readResponse(resp)
	responseSnapshot := dbmodels.ProviderAuditResponseSnapshot{
		StatusCode: resp.StatusCode,
		Headers:    responseHeaders,
		Body:       string(rawBody),
	}
	if err != nil {
		e.saveProviderAudit(ctx, data, &models.ElectricResult{
			UserID:        data.UserID,
			OrderID:       orderID,
			TransactionID: transactionID,
			RequestID:     data.RequestID,
		}, requestSnapshot, responseSnapshot, "failed", "decode_error", err.Error(), "", "", nil)
		return nil, e.logAndReturnError("error decoding response body", err)
	}

	decoded, err := e.decodeRawResponse(rawBody, &apiResponse)
	if err != nil {
		responseSnapshot.Decoded = decoded
		e.saveProviderAudit(ctx, data, &models.ElectricResult{
			UserID:        data.UserID,
			OrderID:       orderID,
			TransactionID: transactionID,
			RequestID:     data.RequestID,
		}, requestSnapshot, responseSnapshot, "failed", "decode_error", err.Error(), "", "", nil)
		return nil, e.logAndReturnError("error decoding response body", err)
	}
	responseSnapshot.Decoded = decoded
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

	if status == "failed" || apiResponse.Code != "000" {
		e.saveProviderAudit(ctx, data, result, requestSnapshot, responseSnapshot, status, "provider_error", "provider returned unsuccessful status", transDetails.Transactions.Status, apiResponse.ResponseDescription, map[string]interface{}{
			"code":                 apiResponse.Code,
			"token_amount":         apiResponse.TokenAmount,
			"exchange_reference":   apiResponse.ExchangeReference,
			"response_description": apiResponse.ResponseDescription,
		})
	} else {
		e.saveProviderAudit(ctx, data, result, requestSnapshot, responseSnapshot, status, "", "", transDetails.Transactions.Status, apiResponse.ResponseDescription, map[string]interface{}{
			"code":                 apiResponse.Code,
			"token_amount":         apiResponse.TokenAmount,
			"exchange_reference":   apiResponse.ExchangeReference,
			"response_description": apiResponse.ResponseDescription,
		})
	}

	if err := e.saveTransaction(ctx, result); err != nil {
		return nil, e.logAndReturnError("error saving transaction to database", err)
	}

	return result, nil
}

// query eletricity bill
func (e *ElectricConn) QueryTransaction(ctx context.Context, id string) (models.ElectricResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	resp, err := e.queryTransaction(ctx, id)
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

	result, err := e.getTransactionDetails(ctx, apiResponse.RequestID)
	if err != nil {
		return models.ElectricResult{}, e.logAndReturnError("failed to get user's transactions", err)
	}

	return result, nil
}

// get transaction history
func (e *ElectricConn) GetUserTransactions(ctx context.Context, username string) ([]models.ElectricResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := e.getAllTransaction(ctx, username)
	if err != nil {
		return nil, e.logAndReturnError("failed to get user's transactions", err)
	}

	return result, nil
}

func (e *ElectricConn) GetTransactionDetails(ctx context.Context, id string) (models.ElectricResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := e.getTransactionDetails(ctx, id)
	if err != nil {
		return models.ElectricResult{}, e.logAndReturnError("failed to get transaction details", err)
	}

	return result, nil
}

// GetAllTransaction returns all transactions, to be used by admin
func (e *ElectricConn) GetAllTransactions(ctx context.Context) ([]models.ElectricResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := e.getAllTransaction(ctx, "")
	if err != nil {
		return nil, e.logAndReturnError("failed to get transactions from database", err)
	}

	return result, nil

}

func (e *ElectricConn) payBill(ctx context.Context, data ElectricInfo) (*http.Response, dbmodels.ProviderAuditRequestSnapshot, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

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
	requestSnapshot := e.newAuditRequestSnapshot(http.MethodPost, url, map[string]string{
		"api-key":      pk,
		"secret-key":   sk,
		"Content-Type": "application/x-www-form-urlencoded",
	}, formdata.Encode(), nil, e.valuesSnapshot(formdata))

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, requestSnapshot, "request_build_error", err
	}
	req.Header.Set("api-key", pk)
	req.Header.Set("secret-key", sk)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, requestSnapshot, "transport_error", err
	}

	return resp, requestSnapshot, "", nil
}

func (e *ElectricConn) saveTransaction(ctx context.Context, details *models.ElectricResult) error {
	if ctx == nil {
		ctx = context.Background()
	}
	err := e.db.SaveElectricTransaction(ctx, details)
	if err != nil {
		return err
	}
	return nil
}

func (e *ElectricConn) getTransactionDetails(ctx context.Context, id string) (models.ElectricResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := e.db.GetElectricSubDetails(ctx, id)
	if err != nil {
		return models.ElectricResult{}, err
	}
	return result, nil
}

func (e *ElectricConn) getAllTransaction(ctx context.Context, username string) ([]models.ElectricResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := e.db.GetAllElectricSubTransactions(ctx, username)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (e *ElectricConn) queryTransaction(ctx context.Context, requestID string) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	formdata := url.Values{
		"request_id": {requestID},
	}

	body := bytes.NewBufferString(formdata.Encode())
	url := fmt.Sprintf("%s/%s", api, "requery")

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
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

func (e *ElectricConn) newAuditRequestSnapshot(method, endpoint string, headers map[string]string, body string, query, form map[string]string) dbmodels.ProviderAuditRequestSnapshot {
	return dbmodels.ProviderAuditRequestSnapshot{
		Method:  method,
		URL:     endpoint,
		Headers: e.sanitizeAuditHeaders(headers),
		Query:   query,
		Form:    form,
		Body:    body,
	}
}

func (e *ElectricConn) sanitizeAuditHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	sanitized := make(map[string]string, len(headers))
	for key, value := range headers {
		switch http.CanonicalHeaderKey(key) {
		case "Authorization", "Authorizationtoken", "Api-Key", "Secret-Key", "Bearer":
			sanitized[key] = "[REDACTED]"
		default:
			sanitized[key] = value
		}
	}
	return sanitized
}

func (e *ElectricConn) headerSnapshot(headers http.Header) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	out := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}
		out[key] = values[0]
	}
	return out
}

func (e *ElectricConn) valuesSnapshot(values url.Values) map[string]string {
	if len(values) == 0 {
		return nil
	}

	out := make(map[string]string, len(values))
	for key, entries := range values {
		if len(entries) == 0 {
			continue
		}
		out[key] = entries[0]
	}
	return out
}

func (e *ElectricConn) readResponse(resp *http.Response) ([]byte, map[string]string, error) {
	if resp == nil {
		return nil, nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, e.headerSnapshot(resp.Header), err
	}

	return body, e.headerSnapshot(resp.Header), nil
}

func (e *ElectricConn) decodeRawResponse(raw []byte, target interface{}) (map[string]interface{}, error) {
	if len(raw) == 0 {
		return nil, io.EOF
	}

	if err := json.Unmarshal(raw, target); err != nil {
		return nil, err
	}

	decoded := map[string]interface{}{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, nil
	}

	return decoded, nil
}

func (e *ElectricConn) saveProviderAudit(ctx context.Context, data ElectricInfo, result *models.ElectricResult, request dbmodels.ProviderAuditRequestSnapshot, response dbmodels.ProviderAuditResponseSnapshot, status, failureType, errorMessage, providerStatus, providerMessage string, metadata map[string]interface{}) {
	audit := &dbmodels.ProviderRequestAudit{
		UserID:          data.UserID,
		Product:         "electricity",
		ProviderName:    "vtpass",
		Operation:       "pay",
		Status:          status,
		FailureType:     failureType,
		TransactionID:   result.TransactionID,
		OrderID:         result.OrderID,
		ReferenceNumber: result.ReferenceNumber,
		RequestID:       result.RequestID,
		PhoneNumber:     data.Phone,
		MeterNumber:     data.Meter_No,
		Request:         request,
		Response:        response,
		ErrorMessage:    errorMessage,
		ProviderStatus:  providerStatus,
		ProviderMessage: providerMessage,
		Metadata:        metadata,
		CreatedAt:       time.Now().UTC(),
	}

	if err := e.db.SaveProviderRequestAudit(ctx, audit); err != nil {
		e.logger.Warn("failed to save electricity provider audit", zap.Error(err), zap.String("status", status), zap.String("failure_type", failureType))
	}
}

func (e *ElectricConn) logAndReturnError(errorMsg string, err error) error {
	e.logger.Error(errorMsg, zap.Error(err))
	return errors.New(errorMsg)
}

// return an error message for when the meter number is not correct
func (e *ElectricConn) VerifyMeterNo(ctx context.Context, discoType, meterNo, meterType string) (verifyResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

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

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
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
