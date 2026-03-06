package airtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aremxyplug-be/db"
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

func (a *AirtimeConn) BuyAirtime(airtime AirtimeInfo) (*telcom.AirtimeResponse, error) {

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

	resp, err := a.buy(airtime)
	if err != nil {
		a.logger.Error("error returned from server", zap.Any("error:", err))
		return nil, err
	}
	if resp.Body == nil {
		a.logger.Error("empty resp body", zap.String("error:", "response body is nil!"))
		return nil, errors.New("empty response body")
	}
	log.Println(resp.Status)
	defer resp.Body.Close()

	apiResponse := vendResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		if err == io.EOF {
			return nil, logAndReturnError(a.logger, "Empty response from server")
		}
		return nil, logAndReturnError(a.logger, "Error returned from server")
	}

	log.Printf("%+v\n", apiResponse)
	status := "success"

	// check to see if the buy was successful. The response is printed to the log
	if !apiResponse.Status {
		status = "failed"
	}

	result.Status = status
	result.ReferenceNumber = strconv.Itoa(apiResponse.Data.RechargeID)

	// save transaction
	if err := a.saveTransaction(result); err != nil {
		return result, logAndReturnError(a.logger, "error saving transaction, an error occurred")
	}

	return result, nil
}

func (a *AirtimeConn) GetTransactionDetail(id string) (telcom.AirtimeResponse, error) {
	result, err := a.getTransacationDetails(id)
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

func (a *AirtimeConn) GetUserTransaction(username string) ([]telcom.AirtimeResponse, error) {
	resp, err := a.getAllTransactions(username)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *AirtimeConn) GetAllTransactions() ([]telcom.AirtimeResponse, error) {
	result, err := a.getAllTransactions("")
	if err != nil {
		a.logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("Database request error: " + err.Error())
	}

	return result, nil
}

func (a *AirtimeConn) buy(data AirtimeInfo) (*http.Response, error) {

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
		return nil, errors.New("invalid network code")
	}

	amount, err := strconv.Atoi(data.Amount)
	if err != nil {
		a.logger.Error("Error converting amount to int", zap.Error(err))
		return nil, err
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
		return nil, err
	}

	req, err := http.NewRequest("POST", api, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Bearer", token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *AirtimeConn) saveTransaction(detail *telcom.AirtimeResponse) error {
	err := a.db.SaveAirtimeTransaction(detail)
	return err
}

func (a *AirtimeConn) getTransacationDetails(id string) (telcom.AirtimeResponse, error) {
	result, err := a.db.GetAirtimeTransactionDetails(id)
	return result, err
}

func (a *AirtimeConn) getAllTransactions(userID string) ([]telcom.AirtimeResponse, error) {
	results, err := a.db.GetAllAirtimeTransactions(userID)
	return results, err
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
