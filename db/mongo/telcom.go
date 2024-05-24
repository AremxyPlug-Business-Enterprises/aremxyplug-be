package mongo

import (
	"context"
	"log"
	"time"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/models/telcom"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// SaveTransaction saves a data transaction to the database.
func (m *mongoStore) SaveDataTransaction(details interface{}) error {

	err := m.saveToDB(dataColl, details)
	if err != nil {
		return err
	}

	return nil
}

// getRecordDetails returns a data transaction detail.
func (m *mongoStore) GetDataTransactionDetails(id string) (telcom.DataResult, error) {
	res := telcom.DataResult{}

	findResult := m.getRecord(id, dataColl)
	err := findResult.Decode(&res)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return telcom.DataResult{}, nil
		}
		// write for errors
		log.Println(err)
		return telcom.DataResult{}, err
	}

	return res, nil

}

// getAllRecords returns all the data transactions associated to a user, if an empty string is passed it returns all data transactions.
func (m *mongoStore) GetAllDataTransactions(user string) ([]telcom.DataResult, error) {
	ctx := context.Background()
	res := []telcom.DataResult{}

	cur, err := m.getAllRecords(dataColl, user)
	if err != nil {
		return []telcom.DataResult{}, err
	}

	for cur.Next(ctx) {
		resp := telcom.DataResult{}
		if err := cur.Decode(&resp); err != nil {
			return nil, err
		}
		res = append(res, resp)
	}
	defer cur.Close(ctx)

	return res, nil

}

func (m *mongoStore) GetSpecTransDetails(id string) (models.SpectranetResult, error) {
	res := models.SpectranetResult{}

	findResult := m.getRecord(id, dataColl)
	err := findResult.Decode(&res)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.SpectranetResult{}, nil
		}
		// write for errors
		log.Println(err)
		return models.SpectranetResult{}, err
	}

	return res, nil
}

func (m *mongoStore) GetAllSpecDataTransactions(user string) ([]models.SpectranetResult, error) {
	ctx := context.Background()
	res := []models.SpectranetResult{}

	cur, err := m.getAllRecords(dataColl, user)
	if err != nil {
		return []models.SpectranetResult{}, err
	}

	for cur.Next(ctx) {
		resp := models.SpectranetResult{}
		if err := cur.Decode(&resp); err != nil {
			return nil, err
		}
		res = append(res, resp)
	}
	defer cur.Close(ctx)

	return res, nil
}

func (m *mongoStore) GetSmileTransDetails(id string) (models.SmileResult, error) {
	res := models.SmileResult{}

	findResult := m.getRecord(id, dataColl)
	err := findResult.Decode(&res)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.SmileResult{}, nil
		}
		// write for errors
		log.Println(err)
		return models.SmileResult{}, err
	}

	return res, nil
}

func (m *mongoStore) GetAllSmileDataTransactions(user string) ([]models.SmileResult, error) {
	ctx := context.Background()
	res := []models.SmileResult{}

	cur, err := m.getAllRecords(dataColl, user)
	if err != nil {
		return []models.SmileResult{}, err
	}

	for cur.Next(ctx) {
		resp := models.SmileResult{}
		if err := cur.Decode(&resp); err != nil {
			return nil, err
		}
		res = append(res, resp)
	}
	defer cur.Close(ctx)

	return res, nil
}

func (m *mongoStore) SaveAirtimeTransaction(details *telcom.AirtimeResponse) error {
	err := m.saveToDB(airColl, details)
	if err != nil {
		return err
	}
	return nil
}

func (m *mongoStore) GetAirtimeTransactionDetails(id string) (telcom.AirtimeResponse, error) {
	res := telcom.AirtimeResponse{}

	result := m.getRecord(id, eduColl)

	err := result.Decode(&res)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return telcom.AirtimeResponse{}, nil
		}
		// return error
		return telcom.AirtimeResponse{}, err
	}

	return res, nil
}

func (m *mongoStore) GetAllAirtimeTransactions(user string) ([]telcom.AirtimeResponse, error) {
	ctx := context.Background()
	res := []telcom.AirtimeResponse{}

	cur, err := m.getAllRecords(dataColl, user)
	if err != nil {
		return []telcom.AirtimeResponse{}, err
	}

	for cur.Next(ctx) {
		resp := telcom.AirtimeResponse{}
		if err := cur.Decode(&resp); err != nil {
			return nil, err
		}
		res = append(res, resp)
	}
	defer cur.Close(ctx)

	return res, nil
}

func (m *mongoStore) SaveTelcomRecipient(data telcom.TelcomRecipient) error {

	err := m.saveToDB("airtime-Recipient", data)

	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) GetTelcomRecipients(userID string) ([]telcom.TelcomRecipient, error) {

	ctx := context.Background()
	res := make([]telcom.TelcomRecipient, 0)

	cur, err := m.getTelcomRecipientRecord("telcom-Recipient", userID)
	if err != nil {
		return []telcom.TelcomRecipient{}, err
	}

	for cur.Next(ctx) {
		resp := telcom.TelcomRecipient{}
		if err := cur.Decode(&resp); err != nil {
			return nil, err
		}
		res = append(res, resp)
	}
	defer cur.Close(ctx)

	return res, nil

}

func (m *mongoStore) EditTelcomRecipient(userID string, data telcom.TelcomRecipient) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.D{primitive.E{Key: "userID", Value: userID}}

	// Define update fields based on what is provided
	updateFields := bson.M{}
	if data.Phone_no != "" {
		updateFields["phone"] = data.Phone_no
	}
	if data.Name != "" {
		updateFields["name"] = data.Name
	}

	// Prepare the update statement
	updateFilter := bson.M{"$set": updateFields}

	_, err := m.col("").UpdateOne(ctx, filter, updateFilter)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) DeleteTelcomRecipient(name, userID string) error {

	ctx := context.Background()

	filter := bson.M{
		"name":   name,
		"userID": userID,
	}

	delResult := m.col("").FindOneAndDelete(ctx, filter)
	if delResult.Err() != nil {
		return delResult.Err()
	}

	return nil
}

func (m *mongoStore) getTelcomRecipientRecord(collectionName, userID string) (*mongo.Cursor, error) {
	ctx := context.Background()
	var filter bson.D

	if userID == "" {
		filter = bson.D{}
	} else {
		filter = bson.D{primitive.E{Key: "userID", Value: userID}}
	}

	cur, err := m.col(collectionName).Find(ctx, filter)
	return cur, err

}
