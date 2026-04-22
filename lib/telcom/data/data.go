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
	var networkStr string
	switch data.Network {
	case 1:
		networkStr = "MTN"
	case 2:
		networkStr = "GLO"
	case 3:
		networkStr = "9MOBILE"
	case 4:
		networkStr = "AIRTEL"
	default:
		return nil, errors.New("Invalid Network ID")
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&reqData); err != nil {
		return nil, d.logAndReturnError("unable to encode data", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", dontechapi+"/data/", &buf)
	if err != nil {
		return nil, err
	}
	//req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Add("Authorization", dontechToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	apiResponse := dontechAPIResponse{}

	transactionID := randomgen.GenerateTransactionID("dat")
	orderID, _ := randomgen.GenerateOrderID()
	transactionDesc := data.Plan_Name + " " + data.PlanSize
	result := &telcom.DataResult{
		UserID:                 data.UserID,
		Network:                networkStr,
		NetworkProduct:         data.Plan_Name,
		PhoneNumber:            data.Mobile_Num,
		Plan_Amount:            data.Amount,
		PlanName:               data.Plan_Name,
		Validity:               data.Validity,
		CreatedAt:              time.Now().UTC(),
		OrderID:                orderID,
		FullName:               data.FullName,
		TransactionProduct:     "Data Top-up",
		TransactionDescription: transactionDesc,
		TransactionID:          transactionID,
		RecipientName:          data.Name,
		Profit_Margin:          data.Profit_Margin,
	}

	log.Println(resp.StatusCode)
	if resp.StatusCode == http.StatusCreated {

		if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
			if err == io.EOF {
				return nil, d.logAndReturnError("Empty response body retured from server", err)
			}
			return nil, d.logAndReturnError("error while decoding json", err)
		}

		if apiResponse.Status != "successful" {
			d.logger.Error("server response error", zap.Any("apiresponse", apiResponse))
			result.Status = "failed"
			result.RecipientName = apiResponse.Ident
			rID := strconv.Itoa(apiResponse.Id)
			result.ApiID = rID
			if err := d.saveTransaction(ctx, result); err != nil {
				d.logger.Error("Database error try again...", zap.Error(err))
				return nil, errors.New("Database Insert Error...")
			}
			return result, nil
		}

		result.RecipientName = apiResponse.Ident
		rID := strconv.Itoa(apiResponse.Id)
		result.ApiID = rID
		result.Status = "success"
		if err := d.saveTransaction(ctx, result); err != nil {
			d.logger.Error("Database error try again...", zap.Error(err))
			return nil, errors.New("Database Insert Error...")
		}

		return result, nil
	} else {
		d.logger.Error("Api Call Error: %s", zap.String("status", fmt.Sprint((resp.Status))))
		body, err := json.Marshal(resp.Body)
		if err != nil {
			return nil, err
		}
		log.Print(string(body))
		return nil, fmt.Errorf("%v", resp.Status)
	}

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
	var networkStr string
	switch data.Network {
	case 1:
		network = "01" //
		networkStr = "MTN"
	case 2:
		network = "02" // GLO
		networkStr = "GLO"
	case 3:
		network = "04" // 9MOBILE
		networkStr = "9MOBILE"
	case 4:
		network = "03" // AIRTEL
		networkStr = "AIRTEL"
	default:
		return nil, errors.New("Invalid Network ID")
	}

	planID := strconv.Itoa(data.PlanID)

	formdata := url.Values{
		"network":          {network},
		"mobileno":         {data.Mobile_Num},
		"dataplan":         {planID},
		"client_reference": {requestID},
	}

	body := bytes.NewBufferString(formdata.Encode())

	url := fmt.Sprintf("%s/%s.php", easyaccessapi, "data")

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("AuthorizationToken", easyaccessToken)
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	transactionID := randomgen.GenerateTransactionID("dat")
	id, err := randomgen.GenerateOrderID()
	if err != nil {
		d.logger.Error("Could not generate orderID...", zap.Error(err))
		return nil, d.logAndReturnError("Could not generate orderID", err)
	}

	transactionDesc := data.Plan_Name + " " + data.PlanSize

	result := &telcom.DataResult{
		UserID:                 data.UserID,
		Network:                networkStr,
		NetworkProduct:         data.Plan_Name,
		PhoneNumber:            data.Mobile_Num,
		ReferenceNumber:        requestID,
		Plan_Amount:            data.Amount,
		PlanName:               data.Plan_Name,
		Validity:               data.Validity,
		CreatedAt:              time.Now().UTC(),
		OrderID:                id,
		FullName:               data.FullName,
		TransactionProduct:     "Data Top-up",
		TransactionDescription: transactionDesc,
		TransactionID:          transactionID,
		RecipientName:          data.Name,
		Profit_Margin:          data.Profit_Margin,
		// ApiID:                  apiID,
	}

	apiResponse := easyaccessResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		if err == io.EOF {
			return nil, d.logAndReturnError("Empty response body retured from server", err)
		}
		return nil, d.logAndReturnError("error while decoding json", err)
	}
	if apiResponse.Status != "Successful" {
		d.logger.Error("failed to purchase data", zap.Any("apiresponse", apiResponse))
		result.Status = "failed"
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
	networkStr := ""
	switch data.Network {
	case 1:
		network = "1" // MTN
		networkStr = "MTN"
	case 2:
		network = "3" // GLO
		networkStr = "GLO"
	case 3:
		network = "4" // 9MOBILE
		networkStr = "9MOBILE"
	case 4:
		network = "2" // AIRTEL
		networkStr = "AIRTEL"
	default:
		return nil, errors.New("Invalid Network ID")
	}

	requestID := randomgen.GenerateRequestID()
	planID := strconv.Itoa(data.PlanID)

	query := url.Values{
		"network":    {network},
		"phone":      {data.Mobile_Num},
		"bypass":     {"false"},
		"request-id": {requestID},
		"data_plan":  {planID},
	}

	url := fmt.Sprintf("%s/%s?%s", api247, "data", query.Encode())

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", apiKey247)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	transactionID := randomgen.GenerateTransactionID("dat")
	id, err := randomgen.GenerateOrderID()
	if err != nil {
		d.logger.Error("Could not generate orderID...", zap.Error(err))
		return nil, d.logAndReturnError("Could not generate orderID", err)
	}

	transactionDesc := data.Plan_Name + " " + data.PlanSize

	status := ""

	result := &telcom.DataResult{
		UserID:                 data.UserID,
		Network:                networkStr,
		NetworkProduct:         data.Plan_Name,
		PhoneNumber:            data.Mobile_Num,
		ReferenceNumber:        requestID,
		Plan_Amount:            data.Amount,
		PlanName:               data.Plan_Name,
		Validity:               data.Validity,
		CreatedAt:              time.Now().UTC(),
		OrderID:                id,
		FullName:               data.FullName,
		TransactionProduct:     "Data Top-up",
		TransactionDescription: transactionDesc,
		TransactionID:          transactionID,
		RecipientName:          data.Name,
		Profit_Margin:          data.Profit_Margin,
		// ApiID:                  apiID,
	}

	apiResponse := api247Response{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		if err == io.EOF {
			return nil, d.logAndReturnError("Empty response body retured from server", err)
		}
		return nil, d.logAndReturnError("error while decoding json", err)
	}
	if apiResponse.Status != "successful" {
		d.logger.Error("failed to purchase data", zap.Any("apiresponse", apiResponse))
		status = "failed"
		result.Status = status
		if err := d.saveTransaction(ctx, result); err != nil {
			d.logger.Error("Database error try again...", zap.Error(err))
			return nil, errors.New("Database Insert Error...")
		}
		return result, nil
	}

	status = "success"
	result.Status = status
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
