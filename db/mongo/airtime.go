package mongo

import (
	"context"
	"log"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/models/telcom"
	"go.mongodb.org/mongo-driver/mongo"
)

// SaveTransaction saves a data transaction to the database.
func (m *mongoStore) SaveDataTransaction(details interface{}) error {

	err := m.saveTransaction(dataColl, details)
	if err != nil {
		return err
	}

	return nil
}

// GetTransactionDetails returns a data transaction detail.
func (m *mongoStore) GetDataTransactionDetails(id string) (telcom.DataResult, error) {
	res := telcom.DataResult{}

	findResult := m.getTransaction(id, dataColl)
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

// GetAllTransaction returns all the data transactions associated to a user, if an empty string is passed it returns all data transactions.
func (m *mongoStore) GetAllDataTransactions(user string) ([]telcom.DataResult, error) {
	ctx := context.Background()
	res := []telcom.DataResult{}

	cur, err := m.getAllTransaction(dataColl, user)
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

	findResult := m.getTransaction(id, dataColl)
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

	cur, err := m.getAllTransaction(dataColl, user)
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

	findResult := m.getTransaction(id, dataColl)
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

	cur, err := m.getAllTransaction(dataColl, user)
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
	err := m.saveTransaction(airColl, details)
	if err != nil {
		return err
	}
	return nil
}

func (m *mongoStore) GetAirtimeTransactionDetails(id string) (telcom.AirtimeResponse, error) {
	res := telcom.AirtimeResponse{}

	result := m.getTransaction(id, eduColl)

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

	cur, err := m.getAllTransaction(dataColl, user)
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
