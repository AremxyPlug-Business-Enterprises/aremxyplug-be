package edu

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
	api   = os.Getenv("EASYACCESS")
	token = os.Getenv("EASYACCESS_AUTH")
)

type EduConn struct {
	db     db.UtilitiesStore
	logger *zap.Logger
}

func NewEdu(DbConn db.UtilitiesStore, logger *zap.Logger) *EduConn {
	return &EduConn{
		db:     DbConn,
		logger: logger,
	}
}

func (edu *EduConn) BuyEduPin(ctx context.Context, eduInfo EduInfo) (*models.EduResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	examType := eduInfo.Exam_Type
	pinNumber := strconv.Itoa(eduInfo.Quantity)

	resp, requestSnapshot, failureType, err := edu.buyPin(ctx, examType, pinNumber)
	if err != nil {
		edu.saveProviderAudit(ctx, eduInfo, &models.EduResponse{}, requestSnapshot, dbmodels.ProviderAuditResponseSnapshot{}, "failed", failureType, err.Error(), "", "", nil)
		return nil, err
	}

	id, err := randomgen.GenerateOrderID()
	if err != nil {
		// check error
		edu.logger.Error("Could not generate orderID...", zap.Error(err))
		return nil, errors.New("api call error")
	}

	if resp.Body == nil {
		edu.saveProviderAudit(ctx, eduInfo, &models.EduResponse{}, requestSnapshot, dbmodels.ProviderAuditResponseSnapshot{
			StatusCode: resp.StatusCode,
			Headers:    edu.headerSnapshot(resp.Header),
		}, "failed", "decode_error", "response body is nil", "", "", nil)
		return nil, errors.New("response body is nil")
	}
	defer resp.Body.Close()
	apiResponse := EduApiResponse{}
	body, responseHeaders, err := edu.readResponse(resp)
	responseSnapshot := dbmodels.ProviderAuditResponseSnapshot{
		StatusCode: resp.StatusCode,
		Headers:    responseHeaders,
		Body:       string(body),
	}
	if err != nil {
		log.Println(err)
		edu.saveProviderAudit(ctx, eduInfo, &models.EduResponse{}, requestSnapshot, responseSnapshot, "failed", "decode_error", err.Error(), "", "", nil)
		return nil, errors.New("could not unmarshal response body")
	}
	log.Println(string(body))
	log.Println("Status: ", resp.Status)
	decoded, err := edu.decodeRawResponse(body, &apiResponse)
	if err != nil {
		responseSnapshot.Decoded = decoded
		if err == io.EOF {
			log.Println("No response from body")
			// edu.logger.Error("Empty response body", zap.Error(err))
			edu.saveProviderAudit(ctx, eduInfo, &models.EduResponse{}, requestSnapshot, responseSnapshot, "failed", "decode_error", "empty response from server", "", "", nil)
			return nil, errors.New("empty response from server")
		} else {
			log.Println("other error:", err)
			// edu.logger.Error("error returned from server: ", zap.Error(err))
			edu.saveProviderAudit(ctx, eduInfo, &models.EduResponse{}, requestSnapshot, responseSnapshot, "failed", "decode_error", err.Error(), "", "", nil)
			return nil, errors.New("could not unmarshal response body")
		}

	}
	responseSnapshot.Decoded = decoded
	log.Printf("%+v", apiResponse)

	/*
		err = json.NewDecoder(resp.Body).Decode(&apiResponse)
		log.Println(apiResponse)
		if err == io.EOF {
			log.Println("No response from body")
			// edu.logger.Error("Empty response body", zap.Error(err))
			return nil, errors.New("empty response from server")
		} else if err != nil {
			log.Println("other error:", err)
			// edu.logger.Error("error returned from server: ", zap.Error(err))
			return nil, errors.New("error returned from server")
		}
	*/

	if apiResponse.Success_Response == "false" {
		log.Println(apiResponse.Message)
		edu.saveProviderAudit(ctx, eduInfo, &models.EduResponse{}, requestSnapshot, responseSnapshot, "failed", "provider_error", "provider returned unsuccessful status", apiResponse.Status, apiResponse.Message, nil)
		return nil, errors.New("failed while purchasing edu pin")
	}

	transactionID := randomgen.GenerateTransactionID("edu")
	pins := []string{
		apiResponse.Pin1,
		apiResponse.Pin2,
		apiResponse.Pin3,
		apiResponse.Pin4,
		apiResponse.Pin5,
		apiResponse.Pin6,
		apiResponse.Pin7,
		apiResponse.Pin8,
		apiResponse.Pin9,
		apiResponse.Pin10,
	}
	var pinGenerated []string
	for _, pin := range pins {
		if pin != "" {
			pinGenerated = append(pinGenerated, pin)
		}
	}

	status := "success"

	// examType should be upper cases
	examType = cases.Upper(language.English).String(examType)
	txnDesc := fmt.Sprintf("%s E-PINs", examType)
	txnProduct := "Education Pins"

	// associate the responses for the api
	result := &models.EduResponse{
		UserID:                 eduInfo.UserID,
		Amount:                 apiResponse.Amount,
		Exam_Type:              examType,
		Quantity:               eduInfo.Quantity,
		PhoneNumber:            eduInfo.Phone_Number,
		ReferenceNumber:        apiResponse.Reference,
		FullName:               eduInfo.Name,
		Email:                  eduInfo.Email,
		TransactionProduct:     txnProduct,
		Status:                 status,
		TransactionDescription: txnDesc,
		OrderID:                id,
		Pin_Generated:          pinGenerated,
		CreatedAt:              time.Now().UTC(),
		TransactionID:          transactionID,
		TXN:                    eduInfo.TXN,
	}

	log.Printf("%+v", result)

	edu.saveProviderAudit(ctx, eduInfo, result, requestSnapshot, responseSnapshot, "success", "", "", apiResponse.Status, apiResponse.Message, map[string]interface{}{
		"reference": apiResponse.Reference,
	})

	// write to database
	if err := edu.saveTransaction(ctx, result); err != nil {
		edu.logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("database insert error")
	}

	return result, nil

}

func (edu *EduConn) QueryTransaction(ctx context.Context, id string) (*models.EduResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	resp, err := edu.queryTransaction(ctx, id)
	if err != nil {
		// return and check error
		return &models.EduResponse{}, err
	}
	defer resp.Body.Close()

	apiResponse := EduApiResponse{}
	result := &models.EduResponse{}
	json.NewDecoder(resp.Body).Decode(&apiResponse)

	return result, nil

}

func (edu *EduConn) GetTransactionDetail(ctx context.Context, id string) (models.EduResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	resp := models.EduResponse{}
	result, err := edu.getTransactionDetails(ctx, id)
	if err != nil {
		edu.logger.Error("Database error try again...", zap.Error(err))
		return resp, errors.New("Database request error: " + err.Error())
	}

	return result, nil
}

func (edu *EduConn) GetAllTransaction(ctx context.Context, user string) ([]models.EduResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := edu.db.GetAllEduTransactions(ctx, user)
	if err != nil {
		return nil, err
	}

	return resp, nil

}

func (edu *EduConn) Ping(ctx context.Context) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", api+"/wallet_balance.php", nil)
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Set("AuthorizationToken", token)
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

func (edu *EduConn) buyPin(ctx context.Context, examType string, pinNumber string) (*http.Response, dbmodels.ProviderAuditRequestSnapshot, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	formdata := url.Values{
		"no_of_pins": {pinNumber},
	}

	body := bytes.NewBufferString(formdata.Encode())

	url := fmt.Sprintf("%s/%s_v2.php", api, examType)
	requestSnapshot := edu.newAuditRequestSnapshot(http.MethodPost, url, map[string]string{
		"AuthorizationToken": token,
		"cache-control":      "no-cache",
		"Content-Type":       "application/x-www-form-urlencoded",
	}, formdata.Encode(), nil, edu.valuesSnapshot(formdata))

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, requestSnapshot, "request_build_error", err
	}
	req.Header.Set("AuthorizationToken", token)
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, requestSnapshot, "transport_error", err
	}

	return resp, requestSnapshot, "", nil
}

func (edu *EduConn) saveTransaction(ctx context.Context, detail *models.EduResponse) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if edu == nil {
		return errors.New("edu is nil")
	}
	if edu.logger == nil {
		return errors.New("logger is nil")
	}
	if edu.db == nil {
		return errors.New("db is nil")
	}

	log.Println(detail)

	err := edu.db.SaveEduTransaction(ctx, detail)
	if err != nil {
		return err
	}

	return nil
}

func (edu *EduConn) queryTransaction(ctx context.Context, id string) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(&id)

	req, err := http.NewRequestWithContext(ctx, "POST", api+"query_transaction.php", &buf)
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

func (edu *EduConn) newAuditRequestSnapshot(method, endpoint string, headers map[string]string, body string, query, form map[string]string) dbmodels.ProviderAuditRequestSnapshot {
	return dbmodels.ProviderAuditRequestSnapshot{
		Method:  method,
		URL:     endpoint,
		Headers: edu.sanitizeAuditHeaders(headers),
		Query:   query,
		Form:    form,
		Body:    body,
	}
}

func (edu *EduConn) sanitizeAuditHeaders(headers map[string]string) map[string]string {
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

func (edu *EduConn) headerSnapshot(headers http.Header) map[string]string {
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

func (edu *EduConn) valuesSnapshot(values url.Values) map[string]string {
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

func (edu *EduConn) readResponse(resp *http.Response) ([]byte, map[string]string, error) {
	if resp == nil {
		return nil, nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, edu.headerSnapshot(resp.Header), err
	}

	return body, edu.headerSnapshot(resp.Header), nil
}

func (edu *EduConn) decodeRawResponse(raw []byte, target interface{}) (map[string]interface{}, error) {
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

func (edu *EduConn) saveProviderAudit(ctx context.Context, info EduInfo, result *models.EduResponse, request dbmodels.ProviderAuditRequestSnapshot, response dbmodels.ProviderAuditResponseSnapshot, status, failureType, errorMessage, providerStatus, providerMessage string, metadata map[string]interface{}) {
	audit := &dbmodels.ProviderRequestAudit{
		UserID:          info.UserID,
		Product:         "edu",
		ProviderName:    "easyaccess",
		Operation:       "buy",
		Status:          status,
		FailureType:     failureType,
		TransactionID:   result.TransactionID,
		OrderID:         result.OrderID,
		ReferenceNumber: result.ReferenceNumber,
		PhoneNumber:     info.Phone_Number,
		ExamType:        info.Exam_Type,
		Quantity:        info.Quantity,
		Request:         request,
		Response:        response,
		ErrorMessage:    errorMessage,
		ProviderStatus:  providerStatus,
		ProviderMessage: providerMessage,
		Metadata:        metadata,
		CreatedAt:       time.Now().UTC(),
	}

	if err := edu.db.SaveProviderRequestAudit(ctx, audit); err != nil {
		edu.logger.Warn("failed to save edu provider audit", zap.Error(err), zap.String("status", status), zap.String("failure_type", failureType))
	}
}

func (edu *EduConn) getTransactionDetails(ctx context.Context, id string) (models.EduResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	res, err := edu.db.GetEduTransactionDetails(ctx, id)
	if err != nil {
		edu.logger.Error("Error getting details from database...", zap.Error(err))
		return models.EduResponse{}, errors.New("database error")
	}

	return res, nil

}
