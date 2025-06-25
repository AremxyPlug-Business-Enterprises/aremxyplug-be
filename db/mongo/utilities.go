package mongo

import (
	"context"

	"github.com/aremxyplug-be/db/models"
	"go.mongodb.org/mongo-driver/mongo"
)

func (m *mongoStore) SaveTVSubcriptionTransaction(details *models.TV_Result) error {
	err := m.saveToDB(tvColl, details)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) GetTvSubscriptionDetails(id string) (models.TV_Result, error) {
	res := models.TV_Result{}

	result := m.getRecord(id, tvColl)

	err := result.Decode(&res)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.TV_Result{}, nil
		}
		// return error
		return models.TV_Result{}, err
	}

	return res, nil
}

func (m *mongoStore) GetAllTvSubTransactions(userID string) ([]models.TV_Result, error) {
	ctx := context.Background()
	res := []models.TV_Result{}

	cur, err := m.getAllRecords(tvColl, userID)
	if err != nil {
		return []models.TV_Result{}, err
	}

	for cur.Next(ctx) {
		resp := models.TV_Result{}
		if err := cur.Decode(&resp); err != nil {
			return nil, err
		}
		res = append(res, resp)
	}
	defer cur.Close(ctx)

	return res, nil
}

func (m *mongoStore) SaveElectricTransaction(details *models.ElectricResult) error {
	err := m.saveToDB(electricColl, details)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) GetElectricSubDetails(id string) (models.ElectricResult, error) {
	res := models.ElectricResult{}

	result := m.getRecord(id, electricColl)

	err := result.Decode(&res)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.ElectricResult{}, nil
		}
		// return error
		return models.ElectricResult{}, err
	}

	return res, nil
}

func (m *mongoStore) GetAllElectricSubTransactions(userID string) ([]models.ElectricResult, error) {
	ctx := context.Background()
	res := []models.ElectricResult{}

	cur, err := m.getAllRecords(electricColl, userID)
	if err != nil {
		return []models.ElectricResult{}, err
	}

	for cur.Next(ctx) {
		resp := models.ElectricResult{}
		if err := cur.Decode(&resp); err != nil {
			return nil, err
		}
		res = append(res, resp)
	}
	defer cur.Close(ctx)

	return res, nil
}

// SaveEduTransactions saves the result of the edu transaction to the database.
func (m *mongoStore) SaveEduTransaction(details *models.EduResponse) error {
	err := m.saveToDB(eduColl, details)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) GetEduTransactionDetails(id string) (models.EduResponse, error) {
	res := models.EduResponse{}

	result := m.getRecord(id, eduColl)

	err := result.Decode(&res)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.EduResponse{}, nil
		}
		// return error
		return models.EduResponse{}, err
	}

	return res, nil

}

func (m *mongoStore) GetAllEduTransactions(userID string) ([]models.EduResponse, error) {
	ctx := context.Background()
	res := []models.EduResponse{}

	cur, err := m.getAllRecords(eduColl, userID)
	if err != nil {
		return []models.EduResponse{}, err
	}

	for cur.Next(ctx) {
		resp := models.EduResponse{}
		if err := cur.Decode(&resp); err != nil {
			return nil, err
		}
		res = append(res, resp)
	}
	defer cur.Close(ctx)

	return res, nil
}
