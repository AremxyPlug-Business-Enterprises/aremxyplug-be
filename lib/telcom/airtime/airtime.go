package airtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aremxyplug-be/db"
	dbmodels "github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/models/telcom"
	"github.com/aremxyplug-be/lib/randomgen"
	"go.uber.org/zap"
)

var (
	api   = os.Getenv("SMSSERVERS_BASE_URL")
	token = os.Getenv("SMSSERVERS_API_KEY")
)

type AirtimeConn struct {
	logger *zap.Logger
	db     db.TelcomStore
}

func NewAirtimeConn(store db.TelcomStore, logger *zap.Logger) *AirtimeConn {
	return &AirtimeConn{
		logger: logger,
		db:     store,
	}
}

func (a *AirtimeConn) BuyAirtime(ctx context.Context, airtime AirtimeInfo) (*telcom.AirtimeResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	id, err := randomgen.GenerateOrderID()
	if err != nil {
		a.logger.Error("unable to generate orderID", zap.Any("error:", "failed to generate orderID"))
		return nil, err
	}

	network := ""

	switch airtime.Network {
	case "1":
		network = "MTN"
	case "2":
		network = "AIRTEL"
	case "3":
		network = "GLO"
	case "4":
		network = "9MOBILE"

	default:
		network = "UNKNOWN"
	}

	airtime.Reference = randomgen.GenerateRequestID()
	transactionID := randomgen.GenerateTransactionID("vtu")
	amount := airtime.Amount
	product := network + " " + "VTU"
	description := product
	transactionProduct := "Airtime Top-up"
	discountPercentage := airtime.Discount_percent
	discountAmount := airtime.Discount_amount

	result := &telcom.AirtimeResponse{
		UserID:                 airtime.UserID,
		OrderID:                id,
		Amount:                 amount,
		Network:                network,
		NetworkProduct:         product,
		TransactionProduct:     transactionProduct,
		TransactionDescription: description,
		Phone_no:               airtime.Phone_no,
		FullName:               airtime.FullName,
		RecipientName:          airtime.Recipient,
		TransactionID:          transactionID,
		CreatedAt:              time.Now().UTC(),
		UserReference:          airtime.Reference,
		Discount_perecent:      discountPercentage,
		Discount_amount:        discountAmount,
		Profit_Margin:          airtime.Profit_Margin,
		Provider_Discount:      airtime.Provider_Discount,
	}

	resp, requestSnapshot, failureType, err := a.buy(ctx, airtime)
	if err != nil {
		a.saveProviderAudit(ctx, airtime, result, requestSnapshot, dbmodels.ProviderAuditResponseSnapshot{}, "failed", failureType, err.Error(), "", "", nil)
		a.logger.Error("error returned from server", zap.Any("error:", err))
		return nil, err
	}
	if resp.Body == nil {
		a.saveProviderAudit(ctx, airtime, result, requestSnapshot, dbmodels.ProviderAuditResponseSnapshot{
			StatusCode: resp.StatusCode,
			Headers:    a.headerSnapshot(resp.Header),
		}, "failed", "decode_error", "response body is nil", "", "", nil)
		a.logger.Error("empty resp body", zap.String("error:", "response body is nil!"))
		return nil, errors.New("empty response body")
	}
	defer resp.Body.Close()

	rawBody, responseHeaders, err := a.readResponse(resp)
	responseSnapshot := dbmodels.ProviderAuditResponseSnapshot{
		StatusCode: resp.StatusCode,
		Headers:    responseHeaders,
		Body:       string(rawBody),
	}
	if err != nil {
		a.saveProviderAudit(ctx, airtime, result, requestSnapshot, responseSnapshot, "failed", "decode_error", err.Error(), "", "", nil)
		return nil, err
	}

	apiResponse := vendResponse{}
	decoded, err := a.decodeRawResponse(rawBody, &apiResponse)
	if err != nil {
		responseSnapshot.Decoded = decoded
		if err == io.EOF {
			a.saveProviderAudit(ctx, airtime, result, requestSnapshot, responseSnapshot, "failed", "decode_error", "Empty response from server", "", "", nil)
			return nil, logAndReturnError(a.logger, "Empty response from server")
		}
		a.saveProviderAudit(ctx, airtime, result, requestSnapshot, responseSnapshot, "failed", "decode_error", err.Error(), "", "", nil)
		return nil, logAndReturnError(a.logger, "Error returned from server")
	}
	responseSnapshot.Decoded = decoded

	log.Printf("%+v\n", apiResponse)
	status := "success"

	// check to see if the buy was successful. The response is printed to the log
	if !apiResponse.Status {
		status = "failed"
	}

	result.Status = status
	result.ReferenceNumber = strconv.Itoa(apiResponse.Data.RechargeID)
	providerStatus := strconv.FormatBool(apiResponse.Status)
	providerMessage := apiResponse.ServerMessage

	// save transaction
	if status == "failed" {
		a.saveProviderAudit(ctx, airtime, result, requestSnapshot, responseSnapshot, "failed", "provider_error", "provider returned unsuccessful status", providerStatus, providerMessage, map[string]interface{}{
			"error_code":  apiResponse.ErrorCode,
			"text_status": apiResponse.TextStatus,
		})
	} else {
		a.saveProviderAudit(ctx, airtime, result, requestSnapshot, responseSnapshot, "success", "", "", providerStatus, providerMessage, map[string]interface{}{
			"error_code":  apiResponse.ErrorCode,
			"text_status": apiResponse.TextStatus,
		})
	}
	if err := a.saveTransaction(ctx, result); err != nil {
		return result, logAndReturnError(a.logger, "error saving transaction, an error occurred")
	}

	return result, nil
}

func (a *AirtimeConn) GetTransactionDetail(ctx context.Context, id string) (telcom.AirtimeResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := a.getTransacationDetails(ctx, id)
	if err != nil {
		return telcom.AirtimeResponse{}, err
	}

	return result, nil
}

/*
func (a *AirtimeConn) QueryTransaction(id string) (*telcom.AirtimeResponse, error) {
	resp, err := a.queryTransaction(id)
	if err != nil {
		return &telcom.AirtimeResponse{}, err
	}
	defer resp.Body.Close()

	apiResponse := telcom.AirtimeApiResponse{}
	result := &telcom.AirtimeResponse{}
	jsonerr := json.NewDecoder(resp.Body).Decode(&apiResponse)
	if jsonerr != nil {
		a.logger.Error("Error querying API...", zap.Error(jsonerr))
		return nil, errors.New("invalid id")
	}

	return result, nil

}
*/

func (a *AirtimeConn) GetUserTransaction(ctx context.Context, username string) ([]telcom.AirtimeResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := a.getAllTransactions(ctx, username)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *AirtimeConn) GetAllTransactions(ctx context.Context) ([]telcom.AirtimeResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := a.getAllTransactions(ctx, "")
	if err != nil {
		a.logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("Database request error: " + err.Error())
	}

	return result, nil
}

func (a *AirtimeConn) buy(ctx context.Context, data AirtimeInfo) (*http.Response, dbmodels.ProviderAuditRequestSnapshot, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	productcode := ""

	switch data.Network {
	case "1":
		productcode = "mtn_custom"
	case "2":
		productcode = "airtel_custom"
	case "3":
		productcode = "glo_custom"
	case "4":
		productcode = "9mobile_custom"

	default:
		productcode = ""
	}

	if productcode == "" {
		a.logger.Error("Invalid network code", zap.String("network", data.Network))
		return nil, dbmodels.ProviderAuditRequestSnapshot{}, "request_build_error", errors.New("invalid network code")
	}

	amount, err := strconv.Atoi(data.Amount)
	if err != nil {
		a.logger.Error("Error converting amount to int", zap.Error(err))
		return nil, dbmodels.ProviderAuditRequestSnapshot{}, "request_build_error", err
	}

	payload := vendRequest{
		ProductCode:   productcode,
		Amount:        amount,
		PhoneNumber:   data.Phone_no,
		Action:        "vend",
		UserReference: data.Reference,
		BypassNetwork: "yes",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		a.logger.Error("Error marshalling payload", zap.Error(err))
		return nil, dbmodels.ProviderAuditRequestSnapshot{}, "request_build_error", err
	}

	requestSnapshot := a.newAuditRequestSnapshot(http.MethodPost, api, map[string]string{
		"Content-Type": "application/json",
		"Bearer":       token,
	}, string(body), nil, nil)

	req, err := http.NewRequestWithContext(ctx, "POST", api, bytes.NewBuffer(body))
	if err != nil {
		return nil, requestSnapshot, "request_build_error", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Bearer", token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, requestSnapshot, "transport_error", err
	}

	return resp, requestSnapshot, "", nil
}

func (a *AirtimeConn) saveTransaction(ctx context.Context, detail *telcom.AirtimeResponse) error {
	if ctx == nil {
		ctx = context.Background()
	}
	err := a.db.SaveAirtimeTransaction(ctx, detail)
	return err
}

func (a *AirtimeConn) getTransacationDetails(ctx context.Context, id string) (telcom.AirtimeResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := a.db.GetAirtimeTransactionDetails(ctx, id)
	return result, err
}

func (a *AirtimeConn) getAllTransactions(ctx context.Context, userID string) ([]telcom.AirtimeResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	results, err := a.db.GetAllAirtimeTransactions(ctx, userID)
	return results, err
}

func (a *AirtimeConn) newAuditRequestSnapshot(method, endpoint string, headers map[string]string, body string, query, form map[string]string) dbmodels.ProviderAuditRequestSnapshot {
	return dbmodels.ProviderAuditRequestSnapshot{
		Method:  method,
		URL:     endpoint,
		Headers: a.sanitizeAuditHeaders(headers),
		Query:   query,
		Form:    form,
		Body:    body,
	}
}

func (a *AirtimeConn) sanitizeAuditHeaders(headers map[string]string) map[string]string {
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

func (a *AirtimeConn) headerSnapshot(headers http.Header) map[string]string {
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

func (a *AirtimeConn) readResponse(resp *http.Response) ([]byte, map[string]string, error) {
	if resp == nil {
		return nil, nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, a.headerSnapshot(resp.Header), err
	}

	return body, a.headerSnapshot(resp.Header), nil
}

func (a *AirtimeConn) decodeRawResponse(raw []byte, target interface{}) (map[string]interface{}, error) {
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

func (a *AirtimeConn) saveProviderAudit(ctx context.Context, airtime AirtimeInfo, result *telcom.AirtimeResponse, request dbmodels.ProviderAuditRequestSnapshot, response dbmodels.ProviderAuditResponseSnapshot, status, failureType, errorMessage, providerStatus, providerMessage string, metadata map[string]interface{}) {
	audit := &dbmodels.ProviderRequestAudit{
		UserID:          airtime.UserID,
		Product:         "airtime",
		ProviderName:    "smsservers",
		Operation:       "vend",
		Status:          status,
		FailureType:     failureType,
		TransactionID:   result.TransactionID,
		OrderID:         result.OrderID,
		ReferenceNumber: result.ReferenceNumber,
		RequestID:       airtime.Reference,
		PhoneNumber:     airtime.Phone_no,
		Network:         result.Network,
		Request:         request,
		Response:        response,
		ErrorMessage:    errorMessage,
		ProviderStatus:  providerStatus,
		ProviderMessage: providerMessage,
		Metadata:        metadata,
		CreatedAt:       time.Now().UTC(),
	}

	if err := a.db.SaveProviderRequestAudit(ctx, audit); err != nil {
		a.logger.Warn("failed to save airtime provider audit", zap.Error(err), zap.String("status", status), zap.String("failure_type", failureType))
	}
}

/*
func (a *AirtimeConn) queryTransaction(id string) (*http.Response, error) {

		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(&id)

		req, err := http.NewRequest("POST", api+"query_transaction.php", &buf)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", token)
		req.Header.Set("cache-control", "no-cache")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		return resp, nil
	}
*/
func logAndReturnError(logger *zap.Logger, errorMsg string) error {
	logger.Error(errorMsg)
	return errors.New(errorMsg)
}

// international airtime

// buy airtime

// get history

// query transaction

// smile airtime

// buy airtimme

// query transaction
