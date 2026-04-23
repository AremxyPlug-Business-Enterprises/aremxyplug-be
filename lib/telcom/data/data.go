package data

import (
	"bytes"
	"context"
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
	"github.com/aremxyplug-be/db/models/telcom"
	"github.com/aremxyplug-be/lib/randomgen"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	dontechapi      = os.Getenv("DONTECH")
	dontechToken    = "Token " + os.Getenv("DONTECH_AUTH")
	easyaccessapi   = os.Getenv("EASYACCESS")
	easyaccessToken = os.Getenv("EASYACCESS_AUTH")
	api247          = os.Getenv("247API_BASE_URL")
	apiKey247       = "Token " + os.Getenv("247API_API_KEY")
	vtapi           = os.Getenv("VTPASS_SANDBOX")
	pk              = os.Getenv("APIKey")
	sk              = os.Getenv("SK")
)

type DataConn struct {
	dbConn db.TelcomStore
	logger *zap.Logger
}

func NewData(dbConn db.TelcomStore, logger *zap.Logger) *DataConn {
	return &DataConn{
		dbConn: dbConn,
		logger: logger,
	}
}

// BuyData makes a call to the api to initiate a purchase
func (d *DataConn) BuyData(ctx context.Context, data DataInfo) (*telcom.DataResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	switch data.ProviderID {
	case 1: // Dontech
		return d.buyDontechData(ctx, data)
	case 2: // Easyaccessapi
		return d.buyEasyaccessData(ctx, data)
	case 3: // 247api
		return d.buy247Data(ctx, data)
	default:
		return nil, errors.New("Invalid Provider ID")
	}

}

func (d *DataConn) providerName(providerID int) string {
	switch providerID {
	case 1:
		return "dontech"
	case 2:
		return "easyaccess"
	case 3:
		return "247api"
	default:
		return "unknown"
	}
}

func (d *DataConn) networkName(networkID int) (string, error) {
	switch networkID {
	case 1:
		return "MTN", nil
	case 2:
		return "GLO", nil
	case 3:
		return "9MOBILE", nil
	case 4:
		return "AIRTEL", nil
	default:
		return "", errors.New("Invalid Network ID")
	}
}

func (d *DataConn) newDataResult(data DataInfo, networkStr, referenceNumber string) (*telcom.DataResult, error) {
	orderID, err := randomgen.GenerateOrderID()
	if err != nil {
		d.logger.Error("Could not generate orderID...", zap.Error(err))
		return nil, d.logAndReturnError("Could not generate orderID", err)
	}

	transactionDesc := data.Plan_Name + " " + data.PlanSize
	return &telcom.DataResult{
		UserID:                 data.UserID,
		Network:                networkStr,
		NetworkProduct:         data.Plan_Name,
		PhoneNumber:            data.Mobile_Num,
		ReferenceNumber:        referenceNumber,
		Plan_Amount:            data.Amount,
		PlanName:               data.Plan_Name,
		Validity:               data.Validity,
		CreatedAt:              time.Now().UTC(),
		OrderID:                orderID,
		FullName:               data.FullName,
		TransactionProduct:     "Data Top-up",
		TransactionDescription: transactionDesc,
		TransactionID:          randomgen.GenerateTransactionID("dat"),
		RecipientName:          data.Name,
		Profit_Margin:          data.Profit_Margin,
	}, nil
}

func (d *DataConn) newAuditRequestSnapshot(method, endpoint string, headers map[string]string, body string, query, form map[string]string) telcom.AuditRequestSnapshot {
	return telcom.AuditRequestSnapshot{
		Method:  method,
		URL:     endpoint,
		Headers: d.sanitizeAuditHeaders(headers),
		Query:   query,
		Form:    form,
		Body:    body,
	}
}

func (d *DataConn) sanitizeAuditHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	sanitized := make(map[string]string, len(headers))
	for key, value := range headers {
		switch http.CanonicalHeaderKey(key) {
		case "Authorization", "Authorizationtoken", "Api-Key", "Secret-Key":
			sanitized[key] = "[REDACTED]"
		default:
			sanitized[key] = value
		}
	}
	return sanitized
}

func (d *DataConn) headerSnapshot(headers http.Header) map[string]string {
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

func (d *DataConn) valuesSnapshot(values url.Values) map[string]string {
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

func (d *DataConn) readResponse(resp *http.Response) ([]byte, map[string]string, error) {
	if resp == nil {
		return nil, nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, d.headerSnapshot(resp.Header), err
	}

	return body, d.headerSnapshot(resp.Header), nil
}

func (d *DataConn) decodeRawResponse(raw []byte, target interface{}) (map[string]interface{}, error) {
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

func (d *DataConn) saveFailureAudit(ctx context.Context, data DataInfo, result *telcom.DataResult, request telcom.AuditRequestSnapshot, response telcom.AuditResponseSnapshot, failureType, errorMessage, providerStatus, providerMessage string, metadata map[string]interface{}) {
	audit := &telcom.DataFailureAudit{
		UserID:          data.UserID,
		ProviderID:      data.ProviderID,
		ProviderName:    d.providerName(data.ProviderID),
		FailureType:     failureType,
		Network:         result.Network,
		PlanID:          data.PlanID,
		PlanName:        data.Plan_Name,
		PhoneNumber:     data.Mobile_Num,
		TransactionID:   result.TransactionID,
		OrderID:         result.OrderID,
		ReferenceNumber: result.ReferenceNumber,
		Request:         request,
		Response:        response,
		ErrorMessage:    errorMessage,
		ProviderStatus:  providerStatus,
		ProviderMessage: providerMessage,
		Metadata:        metadata,
		CreatedAt:       time.Now().UTC(),
	}

	if err := d.dbConn.SaveDataFailureAudit(ctx, audit); err != nil {
		d.logger.Warn("failed to save data provider failure audit", zap.Error(err), zap.String("provider", audit.ProviderName), zap.String("failure_type", failureType))
	}
}

func (d *DataConn) buyDontechData(ctx context.Context, data DataInfo) (*telcom.DataResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	reqData := struct {
		Network      int    `json:"network"`
		Plan         int    `json:"plan"`
		MobileNumber string `json:"mobile_number"`
		PortedNumber bool   `json:"Ported_number"`
	}{
		Network:      data.Network,
		Plan:         data.PlanID,
		MobileNumber: data.Mobile_Num,
		PortedNumber: true,
	}
	networkStr, err := d.networkName(data.Network)
	if err != nil {
		return nil, err
	}

	result, err := d.newDataResult(data, networkStr, "")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&reqData); err != nil {
		return nil, d.logAndReturnError("unable to encode data", err)
	}

	endpoint := dontechapi + "/data/"
	requestSnapshot := d.newAuditRequestSnapshot(
		http.MethodPost,
		endpoint,
		map[string]string{
			"Authorization": dontechToken,
			"Content-Type":  "application/json",
		},
		buf.String(),
		nil,
		nil,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(buf.Bytes()))
	if err != nil {
		d.saveFailureAudit(ctx, data, result, requestSnapshot, telcom.AuditResponseSnapshot{}, "request_build_error", err.Error(), "", "", nil)
		return nil, err
	}
	req.Header.Add("Authorization", dontechToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		d.saveFailureAudit(ctx, data, result, requestSnapshot, telcom.AuditResponseSnapshot{}, "transport_error", err.Error(), "", "", nil)
		return nil, err
	}
	defer resp.Body.Close()

	rawBody, responseHeaders, err := d.readResponse(resp)
	responseSnapshot := telcom.AuditResponseSnapshot{
		StatusCode: resp.StatusCode,
		Headers:    responseHeaders,
		Body:       string(rawBody),
	}
	if err != nil {
		d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "decode_error", err.Error(), "", "", nil)
		return nil, d.logAndReturnError("error while reading response body", err)
	}

	if resp.StatusCode != http.StatusCreated {
		d.logger.Error("Api Call Error", zap.String("status", fmt.Sprint(resp.Status)))
		d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "http_error", resp.Status, "", "", nil)
		return nil, fmt.Errorf("%v", resp.Status)
	}

	apiResponse := dontechAPIResponse{}
	decoded, err := d.decodeRawResponse(rawBody, &apiResponse)
	if err != nil {
		responseSnapshot.Decoded = decoded
		if err == io.EOF {
			d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "decode_error", "Empty response body retured from server", "", "", nil)
			return nil, d.logAndReturnError("Empty response body retured from server", err)
		}
		d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "decode_error", err.Error(), "", "", nil)
		return nil, d.logAndReturnError("error while decoding json", err)
	}
	responseSnapshot.Decoded = decoded

	if apiResponse.Status != "successful" {
		d.logger.Error("server response error", zap.Any("apiresponse", apiResponse))
		result.Status = "failed"
		result.RecipientName = apiResponse.Ident
		result.ApiID = strconv.Itoa(apiResponse.Id)
		d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "provider_error", "provider returned unsuccessful status", apiResponse.Status, "", map[string]interface{}{"api_id": apiResponse.Id})
		if err := d.saveTransaction(ctx, result); err != nil {
			d.logger.Error("Database error try again...", zap.Error(err))
			return nil, errors.New("Database Insert Error...")
		}
		return result, nil
	}

	result.RecipientName = apiResponse.Ident
	result.ApiID = strconv.Itoa(apiResponse.Id)
	result.Status = "success"
	if err := d.saveTransaction(ctx, result); err != nil {
		d.logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("Database Insert Error...")
	}

	return result, nil
}

func (d *DataConn) buyEasyaccessData(ctx context.Context, data DataInfo) (*telcom.DataResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// 	01 for MTN
	// 02 for GLO
	// 03 for AIRTEL
	// 04 for 9MOBILE
	requestID := randomgen.GenerateRequestID()

	var network string
	switch data.Network {
	case 1:
		network = "01" //
	case 2:
		network = "02" // GLO
	case 3:
		network = "04" // 9MOBILE
	case 4:
		network = "03" // AIRTEL
	default:
		return nil, errors.New("Invalid Network ID")
	}

	networkStr, err := d.networkName(data.Network)
	if err != nil {
		return nil, err
	}

	result, err := d.newDataResult(data, networkStr, requestID)
	if err != nil {
		return nil, err
	}

	planID := strconv.Itoa(data.PlanID)

	formdata := url.Values{
		"network":          {network},
		"mobileno":         {data.Mobile_Num},
		"dataplan":         {planID},
		"client_reference": {requestID},
	}

	body := bytes.NewBufferString(formdata.Encode())

	endpoint := fmt.Sprintf("%s/%s.php", easyaccessapi, "data")
	requestSnapshot := d.newAuditRequestSnapshot(
		http.MethodPost,
		endpoint,
		map[string]string{
			"AuthorizationToken": easyaccessToken,
			"cache-control":      "no-cache",
			"Content-Type":       "application/x-www-form-urlencoded",
		},
		formdata.Encode(),
		nil,
		d.valuesSnapshot(formdata),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		d.saveFailureAudit(ctx, data, result, requestSnapshot, telcom.AuditResponseSnapshot{}, "request_build_error", err.Error(), "", "", nil)
		return nil, err
	}
	req.Header.Set("AuthorizationToken", easyaccessToken)
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		d.saveFailureAudit(ctx, data, result, requestSnapshot, telcom.AuditResponseSnapshot{}, "transport_error", err.Error(), "", "", nil)
		return nil, err
	}
	defer resp.Body.Close()

	rawBody, responseHeaders, err := d.readResponse(resp)
	responseSnapshot := telcom.AuditResponseSnapshot{
		StatusCode: resp.StatusCode,
		Headers:    responseHeaders,
		Body:       string(rawBody),
	}
	if err != nil {
		d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "decode_error", err.Error(), "", "", nil)
		return nil, d.logAndReturnError("error while reading response body", err)
	}

	if resp.StatusCode != http.StatusOK {
		d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "http_error", resp.Status, "", "", nil)
		return nil, fmt.Errorf("%v", resp.Status)
	}

	apiResponse := easyaccessResponse{}
	decoded, err := d.decodeRawResponse(rawBody, &apiResponse)
	if err != nil {
		responseSnapshot.Decoded = decoded
		if err == io.EOF {
			d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "decode_error", "Empty response body retured from server", "", "", nil)
			return nil, d.logAndReturnError("Empty response body retured from server", err)
		}
		d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "decode_error", err.Error(), "", "", nil)
		return nil, d.logAndReturnError("error while decoding json", err)
	}
	responseSnapshot.Decoded = decoded

	if apiResponse.Status != "Successful" {
		d.logger.Error("failed to purchase data", zap.Any("apiresponse", apiResponse))
		result.Status = "failed"
		d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "provider_error", "provider returned unsuccessful status", apiResponse.Status, apiResponse.Message, map[string]interface{}{
			"reference":        apiResponse.Reference,
			"client_reference": apiResponse.Client_reference,
			"transaction_date": apiResponse.Transaction_date,
		})
		if err := d.saveTransaction(ctx, result); err != nil {
			d.logger.Error("Database error try again...", zap.Error(err))
			return nil, errors.New("Database Insert Error...")
		}
		return result, nil
	}

	result.Status = "success"
	if err := d.saveTransaction(ctx, result); err != nil {
		d.logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("Database Insert Error...")
	}

	return result, nil
}

func (d *DataConn) buy247Data(ctx context.Context, data DataInfo) (*telcom.DataResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// "1": {"id": 1, "network": "MTN"},
	// "2": {"id": 2, "network": "Airtel"},
	// "3": {"id": 3, "network": "Glo"},
	// "4": {"id": 4, "network": "9mobile"}

	var network string
	switch data.Network {
	case 1:
		network = "1" // MTN
	case 2:
		network = "3" // GLO
	case 3:
		network = "4" // 9MOBILE
	case 4:
		network = "2" // AIRTEL
	default:
		return nil, errors.New("Invalid Network ID")
	}

	networkStr, err := d.networkName(data.Network)
	if err != nil {
		return nil, err
	}

	requestID := randomgen.GenerateRequestID()
	result, err := d.newDataResult(data, networkStr, requestID)
	if err != nil {
		return nil, err
	}

	planID := strconv.Itoa(data.PlanID)

	query := url.Values{
		"network":    {network},
		"phone":      {data.Mobile_Num},
		"bypass":     {"false"},
		"request-id": {requestID},
		"data_plan":  {planID},
	}

	endpoint := fmt.Sprintf("%s/%s?%s", api247, "data", query.Encode())
	requestSnapshot := d.newAuditRequestSnapshot(
		http.MethodPost,
		endpoint,
		map[string]string{
			"Authorization": apiKey247,
		},
		"",
		d.valuesSnapshot(query),
		nil,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		d.saveFailureAudit(ctx, data, result, requestSnapshot, telcom.AuditResponseSnapshot{}, "request_build_error", err.Error(), "", "", nil)
		return nil, err
	}

	req.Header.Set("Authorization", apiKey247)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		d.saveFailureAudit(ctx, data, result, requestSnapshot, telcom.AuditResponseSnapshot{}, "transport_error", err.Error(), "", "", nil)
		return nil, err
	}
	defer resp.Body.Close()

	rawBody, responseHeaders, err := d.readResponse(resp)
	responseSnapshot := telcom.AuditResponseSnapshot{
		StatusCode: resp.StatusCode,
		Headers:    responseHeaders,
		Body:       string(rawBody),
	}
	if err != nil {
		d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "decode_error", err.Error(), "", "", nil)
		return nil, d.logAndReturnError("error while reading response body", err)
	}

	if resp.StatusCode != http.StatusOK {
		d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "http_error", resp.Status, "", "", nil)
		return nil, fmt.Errorf("%v", resp.Status)
	}

	apiResponse := api247Response{}
	decoded, err := d.decodeRawResponse(rawBody, &apiResponse)
	if err != nil {
		responseSnapshot.Decoded = decoded
		if err == io.EOF {
			d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "decode_error", "Empty response body retured from server", "", "", nil)
			return nil, d.logAndReturnError("Empty response body retured from server", err)
		}
		d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "decode_error", err.Error(), "", "", nil)
		return nil, d.logAndReturnError("error while decoding json", err)
	}
	responseSnapshot.Decoded = decoded

	if apiResponse.Status != "successful" {
		d.logger.Error("failed to purchase data", zap.Any("apiresponse", apiResponse))
		result.Status = "failed"
		d.saveFailureAudit(ctx, data, result, requestSnapshot, responseSnapshot, "provider_error", "provider returned unsuccessful status", apiResponse.Status, apiResponse.Message, map[string]interface{}{
			"response":   apiResponse.Response,
			"data_size":  apiResponse.DataSize,
			"data_type":  apiResponse.DataType,
			"request_id": apiResponse.RequestID,
		})
		if err := d.saveTransaction(ctx, result); err != nil {
			d.logger.Error("Database error try again...", zap.Error(err))
			return nil, errors.New("Database Insert Error...")
		}
		return result, nil
	}

	result.Status = "success"
	result.ApiID = apiResponse.RequestID

	if err := d.saveTransaction(ctx, result); err != nil {
		d.logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("Database Insert Error...")
	}

	return result, nil

}

func (d *DataConn) BuySpecData(ctx context.Context, data telcom.SpectranetInfo) (*telcom.SpectranetResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	data.RequestID = randomgen.GenerateRequestID()
	orderid, err := randomgen.GenerateOrderID()
	if err != nil {
		return nil, d.logAndReturnError("unable to generate orderid", err)
	}
	transactionID := randomgen.GenerateTransactionID("dat")
	resp, err := d.buySpecData(ctx, data)
	if err != nil {
		d.logger.Error("error returned from server", zap.Any("error:", err))
		return nil, err
	}
	defer resp.Body.Close()

	apiResponse := telcom.SpectranetApiResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		if err == io.EOF {
			return nil, d.logAndReturnError("Empty response body retured from server", err)
		}
		return nil, d.logAndReturnError("error while decoding json", err)
	}

	trans_content := apiResponse.Content.Transcations
	result := &telcom.SpectranetResult{
		UserID:                 data.UserID,
		Network:                data.Network,
		Product:                data.Product,
		Plan:                   data.Plan,
		Phone_Number:           trans_content.Phone_Number,
		No_of_Pins:             trans_content.Quantity,
		Amount:                 trans_content.Amount,
		TransactionProduct:     "Data Top-up",
		TransactionDescription: data.Product,
		TransactionID:          transactionID,
		OrderID:                orderid,
		ReferenceNumber:        trans_content.TransactionID,
		RequestID:              apiResponse.RequestID,
	}

	if err := d.saveTransaction(ctx, result); err != nil {
		return nil, d.logAndReturnError("error while saving to database", err)
	}

	fmt.Printf("%+v\n", apiResponse)

	return result, nil

}

func (d *DataConn) BuySmileData(ctx context.Context, data telcom.SmileInfo) (*telcom.SmileResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	data.RequestID = randomgen.GenerateRequestID()
	orderid, err := randomgen.GenerateOrderID()
	if err != nil {
		return nil, d.logAndReturnError("unable to generate orderid", err)
	}
	transactionID := randomgen.GenerateTransactionID("dat")
	resp, err := d.buySmileData(ctx, data)
	if err != nil {
		return nil, d.logAndReturnError("error returned from server", err)
	}

	defer resp.Body.Close()

	apiResponse := telcom.SmileAPIresponse{}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		if err == io.EOF {
			return nil, d.logAndReturnError("Empty response body retured from server", err)
		}
		return nil, d.logAndReturnError("error while decoding json", err)
	}

	trans_content := apiResponse.Content.Transcations
	result := &telcom.SmileResult{
		UserID:                 data.UserID,
		Network:                data.Network,
		ProductPlan:            trans_content.Product_Desc,
		Email:                  data.Email,
		AccountID:              data.AccountID,
		Phone_Number:           data.AccountID,
		Amount:                 trans_content.Amount,
		TransactionProduct:     "Data Top-up",
		TransactionDescription: trans_content.Product_Desc,
		TransactionID:          transactionID,
		OrderID:                orderid,
		ReferenceNumber:        trans_content.TransactionID,
		RequestID:              apiResponse.RequestID,
	}

	if err := d.saveTransaction(ctx, result); err != nil {
		return nil, d.logAndReturnError("error while saving to database", err)
	}

	return result, nil
}

// GetTransactionDetail takes a  id and returns the details of the transaction
func (d *DataConn) GetTransactionDetail(ctx context.Context, id string) (telcom.DataResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	resp := telcom.DataResult{}
	res, err := d.getTransactionDetails(ctx, id)
	if err != nil {
		return resp, d.logAndReturnError("error while communicating with database", err)
	}

	return res, nil
}

// GetUserTransactions return all the data transactions associated to a user
func (d *DataConn) GetUserTransactions(ctx context.Context, username string) ([]telcom.DataResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	res, err := d.getAllTransactions(ctx, username)
	if err != nil {
		return res, d.logAndReturnError("error while communicating with database", err)
	}

	return res, err
}

// PingUser is a test function to ping the api
func (d *DataConn) PingUser(ctx context.Context, w http.ResponseWriter) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", dontechapi+"/user/", nil)
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Set("Authorization", dontechToken)
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

	log.Println(dontechapi)

	log.Println("StatusCode: ", statusCode)

	return res, nil
}

// GetAllTransactions returns a list of all data transactions.
func (d *DataConn) GetAllTransactions(ctx context.Context) ([]telcom.DataResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var user string
	result, err := d.getAllTransactions(ctx, user)
	if err != nil {
		d.logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("Database request error: " + err.Error())
	}

	return result, nil
}

func (d *DataConn) GetSpecTransDetails(ctx context.Context, requestID string) (telcom.SpectranetResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	resp := telcom.SpectranetResult{}
	res, err := d.getSpecDataDetails(ctx, requestID)
	if err != nil {
		return resp, d.logAndReturnError("error while communicating with database", err)
	}

	return res, nil
}

func (d *DataConn) GetSpecUserTransactions(ctx context.Context, username string) ([]telcom.SpectranetResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	res, err := d.getAllSpecTransactions(ctx, username)
	if err != nil {
		d.logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("database request error: " + err.Error())
	}

	return res, err
}

func (d *DataConn) GetAllSpecTransactions(ctx context.Context) ([]telcom.SpectranetResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var user string
	result, err := d.getAllSpecTransactions(ctx, user)
	if err != nil {
		d.logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("Database request error: " + err.Error())
	}

	return result, nil
}

func (d *DataConn) GetSmileTransDetails(ctx context.Context, requestID string) (telcom.SmileResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	resp := telcom.SmileResult{}
	res, err := d.getSmileDataDetails(ctx, requestID)
	if err != nil {
		// write error
		d.logger.Error("Database error try again...", zap.Error(err))
		return resp, errors.New("Database request error: " + err.Error())
	}

	return res, nil
}

func (d *DataConn) GetSmileUserTransactions(ctx context.Context, username string) ([]telcom.SmileResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	res, err := d.getAllSmileTransactions(ctx, username)
	if err != nil {
		// write error
		d.logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("database request error: " + err.Error())
	}

	return res, err
}

func (d *DataConn) GetAllSmileTransactions(ctx context.Context) ([]telcom.SmileResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var user string
	result, err := d.getAllSmileTransactions(ctx, user)
	if err != nil {
		d.logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("Database request error: " + err.Error())
	}

	return result, nil
}

func (d *DataConn) buySmileData(ctx context.Context, data telcom.SmileInfo) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	formdata := url.Values{
		"request_id":     {data.RequestID},
		"serviceID":      {data.Product},
		"billersCode":    {data.AccountID},
		"variation_code": {data.Product_plan},
		"phone":          {data.Phone_Number},
	}

	body := bytes.NewBufferString(formdata.Encode())
	url := fmt.Sprintf("%s/%s", vtapi, "pay")

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		d.logger.Error("Error creating HTTP request for Smile data", zap.Error(err))
		return nil, err
	}
	req.Header.Set("api-key", pk)
	req.Header.Set("secret-key", sk)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		d.logger.Error("Error sending HTTP request for Smile data", zap.Error(err))
		return nil, err
	}

	return resp, nil
}

func (d *DataConn) buySpecData(ctx context.Context, data telcom.SpectranetInfo) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
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

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		d.logger.Error("Error creating HTTP request for Spectranet data", zap.Error(err))
		return nil, err
	}
	req.Header.Set("api-key", pk)
	req.Header.Set("secret-key", sk)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		d.logger.Error("Error sending HTTP request for Spectranet data", zap.Error(err))
		return nil, err
	}

	return resp, nil
}

// saveTransaction saves the details of a transaction to the database
func (d *DataConn) saveTransaction(ctx context.Context, details interface{}) error {
	if ctx == nil {
		ctx = context.Background()
	}
	err := d.dbConn.SaveDataTransaction(ctx, details)
	if err != nil {
		d.logger.Error("Error saving transaction to database", zap.Any("details", details), zap.Error(err))
	}
	return err
}

// getTransactionDetails returns the details of a transaction
func (d *DataConn) getTransactionDetails(ctx context.Context, id string) (telcom.DataResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := d.dbConn.GetDataTransactionDetails(ctx, id)
	if err != nil {
		d.logger.Error("Error fetching transaction details", zap.String("transactionID", id), zap.Error(err))
	}
	return result, err
}

// getAllTransactions returns all transactions. If an empty string is passed, it returns all transactions in the database
func (d *DataConn) getAllTransactions(ctx context.Context, username string) ([]telcom.DataResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	results, err := d.dbConn.GetAllDataTransactions(ctx, username)
	if err != nil {
		d.logger.Error("Error fetching all transactions", zap.String("username", username), zap.Error(err))
	}
	return results, err
}

// get transactions history
func (d *DataConn) getSpecDataDetails(ctx context.Context, requestID string) (telcom.SpectranetResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := d.dbConn.GetSpecTransDetails(ctx, requestID)
	return result, err
}

func (d *DataConn) getAllSpecTransactions(ctx context.Context, username string) ([]telcom.SpectranetResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := d.dbConn.GetAllSpecDataTransactions(ctx, username)
	return result, err
}

func (d *DataConn) getSmileDataDetails(ctx context.Context, id string) (telcom.SmileResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := d.dbConn.GetSmileTransDetails(ctx, id)
	return result, err
}

func (d *DataConn) getAllSmileTransactions(ctx context.Context, username string) ([]telcom.SmileResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := d.dbConn.GetAllSmileDataTransactions(ctx, username)
	return result, err
}

func (d *DataConn) logAndReturnError(errorMsg string, err error) error {
	d.logger.Error(errorMsg, zap.Error(err))
	return errors.New(errorMsg)
}
