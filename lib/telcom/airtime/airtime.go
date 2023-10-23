package airtime

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

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/randomgen"
	"go.uber.org/zap"
)

var (
	api   = os.Getenv("EASYACCESS")
	token = os.Getenv("EASYACCESS_AUTH")
)

type AirtimeConn struct {
	logger *zap.Logger
	db     db.DataStore
}

func NewAirtimeConn(logger *zap.Logger, store db.DataStore) *AirtimeConn {
	return &AirtimeConn{
		logger: logger,
		db:     store,
	}
}

func (a *AirtimeConn) BuyAirtime(airtime models.AirtimeInfo) (*models.AirtimeResponse, error) {

	id, err := randomgen.GenerateOrderID()
	if err != nil {
		a.logger.Error("unable to generate orderID", zap.Any("error:", "failed to generate orderID"))
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

	apiResponse := models.AirtimeApiResponse{}

	jsonerr := json.NewDecoder(resp.Body).Decode(&apiResponse)
	log.Println(apiResponse)
	fmt.Printf("%+v\n", apiResponse)
	if jsonerr == io.EOF {
		log.Println("No response from body")
		return nil, errors.New("empty response from server")
	} else if jsonerr != nil {
		log.Println("other error:", jsonerr)
		return nil, errors.New("error returned from server")
	}

	// check to see if the buy was successful. The response is printed to the log
	if apiResponse.Success_Response == "false" {
		log.Print(apiResponse.Message)
		return nil, errors.New("failed to buy airtime")
	}

	transactionID := randomgen.GenerateTransactionID("vtu")
	amount := strconv.Itoa(apiResponse.Amount)
	product := apiResponse.Network + " " + airtime.Product

	result := &models.AirtimeResponse{
		OrderID:         id,
		Amount:          amount,
		Network:         apiResponse.Network,
		Description:     apiResponse.Message,
		Phone_no:        apiResponse.Phone_no,
		Product:         product,
		Recipient:       airtime.Recipient,
		ReferenceNumber: apiResponse.Reference,
		Status:          apiResponse.Status,
		TransactionID:   transactionID,
	}

	// save transaction
	if err := a.saveTransaction(result); err != nil {
		a.logger.Error("error saving transaction, an error occurred!", zap.String("error:", fmt.Sprint(err)))
		return result, err
	}

	return result, nil
}

func (a *AirtimeConn) GetTransactionDetail(id string) (models.AirtimeResponse, error) {
	result, err := a.getTransacationDetails(id)
	if err != nil {
		return models.AirtimeResponse{}, err
	}

	return result, nil
}

func (a *AirtimeConn) QueryTransaction(id string) (*models.AirtimeResponse, error) {
	resp, err := a.queryTransaction(id)
	if err != nil {
		return &models.AirtimeResponse{}, err
	}
	defer resp.Body.Close()

	apiResponse := models.EduApiResponse{}
	result := &models.AirtimeResponse{}
	jsonerr := json.NewDecoder(resp.Body).Decode(&apiResponse)
	if jsonerr != nil {
		a.logger.Error("Error querying API...", zap.Error(jsonerr))
		return nil, errors.New("invalid id")
	}

	return result, nil

}

func (a *AirtimeConn) GetUserTransaction(user string) ([]models.AirtimeResponse, error) {
	resp, err := a.getAllTransactions(user)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *AirtimeConn) GetAllTransactions() ([]models.AirtimeResponse, error) {
	result, err := a.getAllTransactions("")
	if err != nil {
		a.logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("Database request error: " + err.Error())
	}

	return result, nil
}

func (a *AirtimeConn) buy(data models.AirtimeInfo) (*http.Response, error) {

	formdata := url.Values{
		"network":      {data.Network},
		"amount":       {data.Amount},
		"mobileno":     {data.Phone_no},
		"airtime_type": {data.AirtimeType},
	}

	body := bytes.NewBufferString(formdata.Encode())
	url := fmt.Sprintf("%s/%s.php", api, "airtime")

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("AuthorizationToken", token)
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *AirtimeConn) saveTransaction(detail *models.AirtimeResponse) error {
	err := a.db.SaveAirtimeTransaction(detail)
	return err
}

func (a *AirtimeConn) getTransacationDetails(id string) (models.AirtimeResponse, error) {
	result, err := a.db.GetAirtimeTransactionDetails(id)
	return result, err
}

func (a *AirtimeConn) getAllTransactions(user string) ([]models.AirtimeResponse, error) {
	results, err := a.db.GetAllAirtimeTransactions(user)
	return results, err
}

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
