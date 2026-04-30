package tvsub

import (
	"bytes"
	"context"
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
	dbmodels "github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/randomgen"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	api = os.Getenv("VTPASS")
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
func (t *TvConn) BuySub(ctx context.Context, data TvInfo) (*models.TV_Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	data.RequestID = randomgen.GenerateRequestID()
	data.SubType = "change"
	orderID, err := randomgen.GenerateOrderID()
	if err != nil {
		return nil, t.logAndReturnError("error generating orderID", err)
	}
	transactionID := randomgen.GenerateTransactionID("tv")

	resp, requestSnapshot, failureType, err := t.buySub(ctx, data)
	if err != nil {
		t.saveProviderAudit(ctx, data, &models.TV_Result{
			UserID:        data.UserID,
			OrderID:       orderID,
			TransactionID: transactionID,
			RequestID:     data.RequestID,
		}, requestSnapshot, dbmodels.ProviderAuditResponseSnapshot{}, "failed", failureType, err.Error(), "", "", nil)
		t.logger.Error("Buying failed", zap.Error(err))
		return nil, err
	}
	if resp.Body == nil {
		t.saveProviderAudit(ctx, data, &models.TV_Result{
			UserID:        data.UserID,
			OrderID:       orderID,
			TransactionID: transactionID,
			RequestID:     data.RequestID,
		}, requestSnapshot, dbmodels.ProviderAuditResponseSnapshot{
			StatusCode: resp.StatusCode,
			Headers:    t.headerSnapshot(resp.Header),
		}, "failed", "decode_error", "response body is nil", "", "", nil)
		return nil, t.logAndReturnError("error decoding response body", errors.New("response body is nil"))
	}
	defer resp.Body.Close()

	apiResponse := tvAPI{}
	rawBody, responseHeaders, err := t.readResponse(resp)
	responseSnapshot := dbmodels.ProviderAuditResponseSnapshot{
		StatusCode: resp.StatusCode,
		Headers:    responseHeaders,
		Body:       string(rawBody),
	}
	if err != nil {
		t.saveProviderAudit(ctx, data, &models.TV_Result{
			UserID:        data.UserID,
			OrderID:       orderID,
			TransactionID: transactionID,
			RequestID:     data.RequestID,
		}, requestSnapshot, responseSnapshot, "failed", "decode_error", err.Error(), "", "", nil)
		return nil, t.logAndReturnError("error decoding response body", err)
	}

	decoded, err := t.decodeRawResponse(rawBody, &apiResponse)
	if err != nil {
		responseSnapshot.Decoded = decoded
		t.saveProviderAudit(ctx, data, &models.TV_Result{
			UserID:        data.UserID,
			OrderID:       orderID,
			TransactionID: transactionID,
			RequestID:     data.RequestID,
		}, requestSnapshot, responseSnapshot, "failed", "decode_error", err.Error(), "", "", nil)
		return nil, t.logAndReturnError("error decoding response body", err)
	}
	responseSnapshot.Decoded = decoded
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
		CardName:               data.CardName,
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
		Profit_Margin:          data.Profit_Margin,
	}

	if status == "failed" || apiResponse.Code != "000" {
		t.saveProviderAudit(ctx, data, result, requestSnapshot, responseSnapshot, status, "provider_error", "provider returned unsuccessful status", apiResponse.Content.Transactions.Status, apiResponse.Response, map[string]interface{}{
			"code":           apiResponse.Code,
			"purchased_code": apiResponse.PurchasedCode,
		})
	} else {
		t.saveProviderAudit(ctx, data, result, requestSnapshot, responseSnapshot, status, "", "", apiResponse.Content.Transactions.Status, apiResponse.Response, map[string]interface{}{
			"code":           apiResponse.Code,
			"purchased_code": apiResponse.PurchasedCode,
		})
	}

	if err := t.saveTransaction(ctx, result); err != nil {
		return nil, t.logAndReturnError("error saving transaction to database", err)
	}

	return result, nil
}

// query tvsubscription
func (t *TvConn) QueryTransaction(ctx context.Context, requestID string) (models.TV_Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	resp, err := t.queryTransaction(ctx, requestID)
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

	result, err := t.getTransactionDetails(ctx, apiResponse.RequestID)
	if err != nil {
		return models.TV_Result{}, t.logAndReturnError("failed to get user's transactions", err)
	}

	return result, nil

}

// get tvsubscription transaction history
func (t *TvConn) GetUserTransactions(ctx context.Context, user string) ([]models.TV_Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := t.getAllTransaction(ctx, "user")
	if err != nil {
		return nil, t.logAndReturnError("failed to get user's transactions", err)
	}

	return result, nil

}

func (t *TvConn) GetTransactionDetails(ctx context.Context, id string) (models.TV_Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := t.getTransactionDetails(ctx, id)
	if err != nil {
		return models.TV_Result{}, t.logAndReturnError("failed to get transaction details", err)
	}

	return result, nil
}

// func to be used by admin to return all transaction in database
func (t *TvConn) GetAllTransactions(ctx context.Context) ([]models.TV_Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	result, err := t.getAllTransaction(ctx, "")
	if err != nil {

		return nil, t.logAndReturnError("failed to get transactions from database", err)
	}

	return result, nil

}

func (t *TvConn) VerifyCard(ctx context.Context, service, iucNumber string) (verifyResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	formdata := url.Values{
		"billersCode": {iucNumber},
		"serviceID":   {service},
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

func (t *TvConn) buySub(ctx context.Context, data TvInfo) (*http.Response, dbmodels.ProviderAuditRequestSnapshot, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

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
	requestSnapshot := t.newAuditRequestSnapshot(http.MethodPost, url, map[string]string{
		"api-key":      pk,
		"secret-key":   sk,
		"Content-Type": "application/x-www-form-urlencoded",
	}, formdata.Encode(), nil, t.valuesSnapshot(formdata))

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

func (t *TvConn) saveTransaction(ctx context.Context, details *models.TV_Result) error {
	if ctx == nil {
		ctx = context.Background()
	}
	err := t.db.SaveTVSubcriptionTransaction(ctx, details)
	if err != nil {
		return err
	}
	return nil
}

func (t *TvConn) getTransactionDetails(ctx context.Context, id string) (models.TV_Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := t.db.GetTvSubscriptionDetails(ctx, id)
	if err != nil {
		return models.TV_Result{}, err
	}
	return result, nil
}

func (t *TvConn) getAllTransaction(ctx context.Context, user string) ([]models.TV_Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := t.db.GetAllTvSubTransactions(ctx, user)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (t *TvConn) queryTransaction(ctx context.Context, requestID string) (*http.Response, error) {
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
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (t *TvConn) newAuditRequestSnapshot(method, endpoint string, headers map[string]string, body string, query, form map[string]string) dbmodels.ProviderAuditRequestSnapshot {
	return dbmodels.ProviderAuditRequestSnapshot{
		Method:  method,
		URL:     endpoint,
		Headers: t.sanitizeAuditHeaders(headers),
		Query:   query,
		Form:    form,
		Body:    body,
	}
}

func (t *TvConn) sanitizeAuditHeaders(headers map[string]string) map[string]string {
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

func (t *TvConn) headerSnapshot(headers http.Header) map[string]string {
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

func (t *TvConn) valuesSnapshot(values url.Values) map[string]string {
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

func (t *TvConn) readResponse(resp *http.Response) ([]byte, map[string]string, error) {
	if resp == nil {
		return nil, nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, t.headerSnapshot(resp.Header), err
	}

	return body, t.headerSnapshot(resp.Header), nil
}

func (t *TvConn) decodeRawResponse(raw []byte, target interface{}) (map[string]interface{}, error) {
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

func (t *TvConn) saveProviderAudit(ctx context.Context, data TvInfo, result *models.TV_Result, request dbmodels.ProviderAuditRequestSnapshot, response dbmodels.ProviderAuditResponseSnapshot, status, failureType, errorMessage, providerStatus, providerMessage string, metadata map[string]interface{}) {
	audit := &dbmodels.ProviderRequestAudit{
		UserID:          data.UserID,
		Product:         "tv-sub",
		ProviderName:    "vtpass",
		Operation:       "pay",
		Status:          status,
		FailureType:     failureType,
		TransactionID:   result.TransactionID,
		OrderID:         result.OrderID,
		ReferenceNumber: result.ReferenceNumber,
		RequestID:       result.RequestID,
		PhoneNumber:     data.Phone,
		SmartcardNumber: data.SmartCard_Number,
		DecoderType:     data.DecoderType,
		PackageName:     data.Package,
		Request:         request,
		Response:        response,
		ErrorMessage:    errorMessage,
		ProviderStatus:  providerStatus,
		ProviderMessage: providerMessage,
		Metadata:        metadata,
		CreatedAt:       time.Now().UTC(),
	}

	if err := t.db.SaveProviderRequestAudit(ctx, audit); err != nil {
		t.logger.Warn("failed to save tv provider audit", zap.Error(err), zap.String("status", status), zap.String("failure_type", failureType))
	}
}

func (d *TvConn) logAndReturnError(errorMsg string, err error) error {
	d.logger.Error(errorMsg, zap.Error(err))
	return errors.New(errorMsg)
}
