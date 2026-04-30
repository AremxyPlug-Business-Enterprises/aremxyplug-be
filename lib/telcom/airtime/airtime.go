package airtime

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
	dbmodels "github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/models/telcom"
	"github.com/aremxyplug-be/lib/randomgen"
	"go.uber.org/zap"
)

var (
	smsServersAPI   = strings.TrimSpace(os.Getenv("SMSSERVERS_BASE_URL"))
	smsServersToken = strings.TrimSpace(os.Getenv("SMSSERVERS_API_KEY"))
	vtpassAPI       = strings.TrimSpace(os.Getenv("VTPASS_SANDBOX"))
	vtpassPK        = strings.TrimSpace(os.Getenv("APIKey"))
	vtpassSK        = strings.TrimSpace(os.Getenv("SK"))
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

	orderID, err := randomgen.GenerateOrderID()
	if err != nil {
		a.logger.Error("unable to generate orderID", zap.String("error", "failed to generate orderID"))
		return nil, err
	}

	network, err := a.networkName(airtime.Network)
	if err != nil {
		return nil, err
	}

	airtime.Reference = randomgen.GenerateRequestID()
	result := &telcom.AirtimeResponse{
		UserID:                 airtime.UserID,
		OrderID:                orderID,
		Amount:                 airtime.Amount,
		Network:                network,
		NetworkProduct:         network + " VTU",
		TransactionProduct:     "Airtime Top-up",
		TransactionDescription: network + " VTU",
		Phone_no:               airtime.Phone_no,
		FullName:               airtime.FullName,
		RecipientName:          airtime.Recipient,
		TransactionID:          randomgen.GenerateTransactionID("vtu"),
		CreatedAt:              time.Now().UTC(),
		UserReference:          airtime.Reference,
		Discount_perecent:      airtime.Discount_percent,
		Discount_amount:        airtime.Discount_amount,
		Profit_Margin:          airtime.Profit_Margin,
		Provider_Discount:      airtime.Provider_Discount,
	}

	providerName, operation, resp, requestSnapshot, failureType, err := a.executeProviderPurchase(ctx, airtime)
	if err != nil {
		a.saveProviderAudit(ctx, airtime, result, providerName, operation, requestSnapshot, dbmodels.ProviderAuditResponseSnapshot{}, "failed", failureType, err.Error(), "", "", nil)
		a.logger.Error("error returned from provider", zap.String("provider", providerName), zap.Error(err))
		return nil, err
	}

	if resp.Body == nil {
		a.saveProviderAudit(ctx, airtime, result, providerName, operation, requestSnapshot, dbmodels.ProviderAuditResponseSnapshot{
			StatusCode: resp.StatusCode,
			Headers:    a.headerSnapshot(resp.Header),
		}, "failed", "decode_error", "response body is nil", "", "", nil)
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
		a.saveProviderAudit(ctx, airtime, result, providerName, operation, requestSnapshot, responseSnapshot, "failed", "decode_error", err.Error(), "", "", nil)
		return nil, err
	}

	switch providerName {
	case "smsservers":
		return a.handleSMSServersResponse(ctx, airtime, result, providerName, operation, requestSnapshot, responseSnapshot, rawBody)
	case "vtpass":
		return a.handleVTPassResponse(ctx, airtime, result, providerName, operation, requestSnapshot, responseSnapshot, rawBody)
	default:
		return nil, fmt.Errorf("unsupported airtime provider: %s", airtime.ProviderName)
	}
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

func (a *AirtimeConn) executeProviderPurchase(ctx context.Context, data AirtimeInfo) (string, string, *http.Response, dbmodels.ProviderAuditRequestSnapshot, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	switch providerName := a.normalizeProviderName(data.ProviderName); providerName {
	case "smsservers":
		resp, requestSnapshot, failureType, err := a.buySMSServers(ctx, data)
		return providerName, "vend", resp, requestSnapshot, failureType, err
	case "vtpass":
		resp, requestSnapshot, failureType, err := a.buyVTPass(ctx, data)
		return providerName, "pay", resp, requestSnapshot, failureType, err
	default:
		return providerName, "", nil, dbmodels.ProviderAuditRequestSnapshot{}, "provider_error", fmt.Errorf("unsupported airtime provider: %s", data.ProviderName)
	}
}

func (a *AirtimeConn) buySMSServers(ctx context.Context, data AirtimeInfo) (*http.Response, dbmodels.ProviderAuditRequestSnapshot, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	productCode := a.smsServersProductCode(data.Network)
	if productCode == "" {
		a.logger.Error("Invalid network code", zap.String("network", data.Network))
		return nil, dbmodels.ProviderAuditRequestSnapshot{}, "request_build_error", errors.New("invalid network code")
	}

	amount, err := strconv.Atoi(data.Amount)
	if err != nil {
		a.logger.Error("Error converting amount to int", zap.Error(err))
		return nil, dbmodels.ProviderAuditRequestSnapshot{}, "request_build_error", err
	}

	payload := vendRequest{
		ProductCode:   productCode,
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

	requestSnapshot := a.newAuditRequestSnapshot(http.MethodPost, smsServersAPI, map[string]string{
		"Content-Type": "application/json",
		"Bearer":       smsServersToken,
	}, string(body), nil, nil)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, smsServersAPI, bytes.NewBuffer(body))
	if err != nil {
		return nil, requestSnapshot, "request_build_error", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Bearer", smsServersToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, requestSnapshot, "transport_error", err
	}

	return resp, requestSnapshot, "", nil
}

func (a *AirtimeConn) buyVTPass(ctx context.Context, data AirtimeInfo) (*http.Response, dbmodels.ProviderAuditRequestSnapshot, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	serviceID := a.vtpassServiceID(data.Network)
	if serviceID == "" {
		return nil, dbmodels.ProviderAuditRequestSnapshot{}, "request_build_error", errors.New("invalid network code")
	}

	formdata := url.Values{
		"request_id": {data.Reference},
		"serviceID":  {serviceID},
		"amount":     {data.Amount},
		"phone":      {data.Phone_no},
	}

	body := bytes.NewBufferString(formdata.Encode())
	endpoint := fmt.Sprintf("%s/%s", strings.TrimRight(vtpassAPI, "/"), "pay")
	requestSnapshot := a.newAuditRequestSnapshot(http.MethodPost, endpoint, map[string]string{
		"api-key":      vtpassPK,
		"secret-key":   vtpassSK,
		"Content-Type": "application/x-www-form-urlencoded",
	}, formdata.Encode(), nil, a.valuesSnapshot(formdata))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, requestSnapshot, "request_build_error", err
	}
	req.Header.Set("api-key", vtpassPK)
	req.Header.Set("secret-key", vtpassSK)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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

func (a *AirtimeConn) valuesSnapshot(values url.Values) map[string]string {
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

func (a *AirtimeConn) saveProviderAudit(ctx context.Context, airtime AirtimeInfo, result *telcom.AirtimeResponse, providerName, operation string, request dbmodels.ProviderAuditRequestSnapshot, response dbmodels.ProviderAuditResponseSnapshot, status, failureType, errorMessage, providerStatus, providerMessage string, metadata map[string]interface{}) {
	audit := &dbmodels.ProviderRequestAudit{
		UserID:          airtime.UserID,
		Product:         "airtime",
		ProviderName:    providerName,
		Operation:       operation,
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

func (a *AirtimeConn) handleSMSServersResponse(ctx context.Context, airtime AirtimeInfo, result *telcom.AirtimeResponse, providerName, operation string, requestSnapshot dbmodels.ProviderAuditRequestSnapshot, responseSnapshot dbmodels.ProviderAuditResponseSnapshot, rawBody []byte) (*telcom.AirtimeResponse, error) {
	apiResponse := vendResponse{}
	decoded, err := a.decodeRawResponse(rawBody, &apiResponse)
	if err != nil {
		responseSnapshot.Decoded = decoded
		if err == io.EOF {
			a.saveProviderAudit(ctx, airtime, result, providerName, operation, requestSnapshot, responseSnapshot, "failed", "decode_error", "Empty response from server", "", "", nil)
			return nil, logAndReturnError(a.logger, "Empty response from server")
		}

		a.saveProviderAudit(ctx, airtime, result, providerName, operation, requestSnapshot, responseSnapshot, "failed", "decode_error", err.Error(), "", "", nil)
		return nil, logAndReturnError(a.logger, "Error returned from server")
	}
	responseSnapshot.Decoded = decoded

	status := "success"
	if !apiResponse.Status {
		status = "failed"
	}

	result.Status = status
	result.ReferenceNumber = strconv.Itoa(apiResponse.Data.RechargeID)
	providerStatus := strconv.FormatBool(apiResponse.Status)
	providerMessage := apiResponse.ServerMessage
	metadata := map[string]interface{}{
		"error_code":  apiResponse.ErrorCode,
		"text_status": apiResponse.TextStatus,
	}

	if status == "failed" {
		a.saveProviderAudit(ctx, airtime, result, providerName, operation, requestSnapshot, responseSnapshot, status, "provider_error", "provider returned unsuccessful status", providerStatus, providerMessage, metadata)
	} else {
		a.saveProviderAudit(ctx, airtime, result, providerName, operation, requestSnapshot, responseSnapshot, status, "", "", providerStatus, providerMessage, metadata)
	}

	if err := a.saveTransaction(ctx, result); err != nil {
		return result, logAndReturnError(a.logger, "error saving transaction, an error occurred")
	}

	return result, nil
}

func (a *AirtimeConn) handleVTPassResponse(ctx context.Context, airtime AirtimeInfo, result *telcom.AirtimeResponse, providerName, operation string, requestSnapshot dbmodels.ProviderAuditRequestSnapshot, responseSnapshot dbmodels.ProviderAuditResponseSnapshot, rawBody []byte) (*telcom.AirtimeResponse, error) {
	apiResponse := vtpassAirtimeResponse{}
	decoded, err := a.decodeRawResponse(rawBody, &apiResponse)
	if err != nil {
		responseSnapshot.Decoded = decoded
		if err == io.EOF {
			a.saveProviderAudit(ctx, airtime, result, providerName, operation, requestSnapshot, responseSnapshot, "failed", "decode_error", "Empty response from server", "", "", nil)
			return nil, logAndReturnError(a.logger, "Empty response from server")
		}

		a.saveProviderAudit(ctx, airtime, result, providerName, operation, requestSnapshot, responseSnapshot, "failed", "decode_error", err.Error(), "", "", nil)
		return nil, logAndReturnError(a.logger, "Error returned from server")
	}
	responseSnapshot.Decoded = decoded

	status := a.mapVTPassStatus(apiResponse.Content.Transactions.Status)
	if apiResponse.Code != "000" {
		status = "failed"
	}

	result.Status = status
	result.ReferenceNumber = apiResponse.Content.Transactions.TransactionID
	metadata := map[string]interface{}{
		"code":                 apiResponse.Code,
		"response_description": apiResponse.ResponseDescription,
		"commission":           apiResponse.Content.Transactions.Commission,
		"total_amount":         apiResponse.Content.Transactions.TotalAmount,
		"amount":               apiResponse.Amount,
		"transaction_date":     apiResponse.TransactionDate,
	}

	if status == "failed" {
		a.saveProviderAudit(ctx, airtime, result, providerName, operation, requestSnapshot, responseSnapshot, status, "provider_error", "provider returned unsuccessful status", apiResponse.Content.Transactions.Status, apiResponse.ResponseDescription, metadata)
	} else {
		a.saveProviderAudit(ctx, airtime, result, providerName, operation, requestSnapshot, responseSnapshot, status, "", "", apiResponse.Content.Transactions.Status, apiResponse.ResponseDescription, metadata)
	}

	if err := a.saveTransaction(ctx, result); err != nil {
		return result, logAndReturnError(a.logger, "error saving transaction, an error occurred")
	}

	return result, nil
}

func (a *AirtimeConn) normalizeProviderName(providerName string) string {
	normalized := strings.ToLower(strings.TrimSpace(providerName))
	switch normalized {
	case "smsservers", "sms servers", "simservers", "sim servers":
		return "smsservers"
	case "vtpass", "vt pass":
		return "vtpass"
	default:
		return normalized
	}
}

func (a *AirtimeConn) networkName(network string) (string, error) {
	switch network {
	case "1":
		return "MTN", nil
	case "2":
		return "AIRTEL", nil
	case "3":
		return "GLO", nil
	case "4":
		return "9MOBILE", nil
	default:
		return "", errors.New("invalid network code")
	}
}

func (a *AirtimeConn) smsServersProductCode(network string) string {
	switch network {
	case "1":
		return "mtn_custom"
	case "2":
		return "airtel_custom"
	case "3":
		return "glo_custom"
	case "4":
		return "9mobile_custom"
	default:
		return ""
	}
}

func (a *AirtimeConn) vtpassServiceID(network string) string {
	switch network {
	case "1":
		return "mtn"
	case "2":
		return "airtel"
	case "3":
		return "glo"
	case "4":
		return "9mobile"
	default:
		return ""
	}
}

func (a *AirtimeConn) mapVTPassStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "delivered":
		return "success"
	case "pending":
		return "pending"
	default:
		return "failed"
	}
}

// international airtime

// buy airtime

// get history

// query transaction

// smile airtime

// buy airtimme

// query transaction
