package edu

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

type EduConn struct {
	Dbconn db.UtilitiesStore
	Logger *zap.Logger
}

func NewEdu(DbConn db.UtilitiesStore, logger *zap.Logger) *EduConn {
	return &EduConn{
		Dbconn: DbConn,
		Logger: logger,
	}
}

func (edu *EduConn) BuyEduPin(eduInfo models.EduInfo) (*models.EduResponse, error) {

	examType := eduInfo.Exam_Type
	pinNumber := strconv.Itoa(eduInfo.Quantity)

	resp, err := edu.buyPin(examType, pinNumber)
	if err != nil {
		return nil, err
	}

	id, err := randomgen.GenerateOrderID()
	if err != nil {
		// check error
		edu.Logger.Error("Could not generate orderID...", zap.Error(err))
		return nil, errors.New("api call error")
	}

	if resp.Body == nil {
		return nil, errors.New("response body is nil")
	}
	defer resp.Body.Close()
	apiResponse := models.EduApiResponse{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, errors.New("could not unmarshal response body")
	}
	log.Println(string(body))
	log.Println("Status: ", resp.Status)
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		if err == io.EOF {
			log.Println("No response from body")
			// edu.Logger.Error("Empty response body", zap.Error(err))
			return nil, errors.New("empty response from server")
		} else {
			log.Println("other error:", err)
			// edu.Logger.Error("error returned from server: ", zap.Error(err))
			return nil, errors.New("could not unmarshal response body")
		}

	}
	log.Printf("%+v", apiResponse)

	/*
		err = json.NewDecoder(resp.Body).Decode(&apiResponse)
		log.Println(apiResponse)
		if err == io.EOF {
			log.Println("No response from body")
			// edu.Logger.Error("Empty response body", zap.Error(err))
			return nil, errors.New("empty response from server")
		} else if err != nil {
			log.Println("other error:", err)
			// edu.Logger.Error("error returned from server: ", zap.Error(err))
			return nil, errors.New("error returned from server")
		}
	*/

	if apiResponse.Success_Response == "false" {
		log.Println(apiResponse.Message)
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

	// associate the responses for the api
	result := &models.EduResponse{
		Amount:          apiResponse.Amount,
		Phone:           eduInfo.Phone_Number,
		ReferenceNumber: apiResponse.Reference,
		Email:           eduInfo.Email,
		Product:         eduInfo.Exam_Type,
		Status:          apiResponse.Status,
		Description:     apiResponse.Message,
		OrderID:         id,
		Pin_Generated:   pinGenerated,
		CreatedAt:       apiResponse.Date,
		TransactionID:   transactionID,
	}

	log.Printf("%+v", result)

	// write to database
	if err := edu.saveTransaction(result); err != nil {
		edu.Logger.Error("Database error try again...", zap.Error(err))
		return nil, errors.New("database insert error")
	}

	return result, nil

}

func (edu *EduConn) QueryTransaction(id string) (*models.EduResponse, error) {

	resp, err := edu.queryTransaction(id)
	if err != nil {
		// return and check error
		return &models.EduResponse{}, err
	}
	defer resp.Body.Close()

	apiResponse := models.EduApiResponse{}
	result := &models.EduResponse{}
	json.NewDecoder(resp.Body).Decode(&apiResponse)

	return result, nil

}

func (edu *EduConn) GetTransactionDetail(id string) (models.EduResponse, error) {

	resp := models.EduResponse{}
	result, err := edu.getTransactionDetails(id)
	if err != nil {
		edu.Logger.Error("Database error try again...", zap.Error(err))
		return resp, errors.New("Database request error: " + err.Error())
	}

	return result, nil
}

func (edu *EduConn) GetAllTransaction(user string) ([]models.EduResponse, error) {
	resp, err := edu.Dbconn.GetAllEduTransactions(user)
	if err != nil {
		return nil, err
	}

	return resp, nil

}

func (edu *EduConn) Ping() (*http.Response, error) {

	req, err := http.NewRequest("GET", api+"/wallet_balance.php", nil)
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

func (edu *EduConn) buyPin(examType string, pinNumber string) (*http.Response, error) {

	formdata := url.Values{
		"no_of_pins": {pinNumber},
	}

	body := bytes.NewBufferString(formdata.Encode())

	url := fmt.Sprintf("%s/%s_v2.php", api, examType)

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

func (edu *EduConn) saveTransaction(detail *models.EduResponse) error {

	err := edu.Dbconn.SaveEduTransaction(detail)
	if err != nil {
		return err
	}

	return nil
}

func (edu *EduConn) queryTransaction(id string) (*http.Response, error) {

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

func (edu *EduConn) getTransactionDetails(id string) (models.EduResponse, error) {

	res, err := edu.Dbconn.GetEduTransactionDetails(id)
	if err != nil {
		edu.Logger.Error("Error getting details from database...", zap.Error(err))
		return models.EduResponse{}, errors.New("database error")
	}

	return res, nil

}
