package mongo

import (
	"context"
	"log"

	"github.com/aremxyplug-be/db/models"
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
func (m *mongoStore) GetDataTransactionDetails(id string) (models.DataResult, error) {
	res := models.DataResult{}

	findResult := m.getTransaction(id, dataColl)
	err := findResult.Decode(&res)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.DataResult{}, nil
		}
		// write for errors
		log.Println(err)
		return models.DataResult{}, err
	}

	return res, nil

}

// GetAllTransaction returns all the data transactions associated to a user, if an empty string is passed it returns all data transactions.
func (m *mongoStore) GetAllDataTransactions(user string) ([]models.DataResult, error) {
	ctx := context.Background()
	res := []models.DataResult{}

	cur, err := m.getAllTransaction(dataColl, user)
	if err != nil {
		return []models.DataResult{}, err
	}

	for cur.Next(ctx) {
		resp := models.DataResult{}
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

func (m *mongoStore) SaveAirtimeTransaction(details *models.AirtimeResponse) error {
	err := m.saveTransaction(airColl, details)
	if err != nil {
		return err
	}
	return nil
}

func (m *mongoStore) GetAirtimeTransactionDetails(id string) (models.AirtimeResponse, error) {
	res := models.AirtimeResponse{}

	result := m.getTransaction(id, eduColl)

	err := result.Decode(&res)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.AirtimeResponse{}, nil
		}
		// return error
		return models.AirtimeResponse{}, err
	}

	return res, nil
}

func (m *mongoStore) GetAllAirtimeTransactions(user string) ([]models.AirtimeResponse, error) {
	ctx := context.Background()
	res := []models.AirtimeResponse{}

	cur, err := m.getAllTransaction(dataColl, user)
	if err != nil {
		return []models.AirtimeResponse{}, err
	}

	for cur.Next(ctx) {
		resp := models.AirtimeResponse{}
		if err := cur.Decode(&resp); err != nil {
			return nil, err
		}
		res = append(res, resp)
	}
	defer cur.Close(ctx)

	return res, nil
}
