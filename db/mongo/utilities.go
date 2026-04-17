package mongo

import (
	"context"

	"github.com/aremxyplug-be/db/models"
	"go.mongodb.org/mongo-driver/mongo"
)

func (m *mongoStore) SaveTVSubcriptionTransaction(ctx context.Context, details *models.TV_Result) error {
	ctx = m.ensureCtx(ctx)
	err := m.saveToDB(ctx, tvColl, details)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) GetTvSubscriptionDetails(ctx context.Context, id string) (models.TV_Result, error) {
	ctx = m.ensureCtx(ctx)
	res := models.TV_Result{}

	result := m.getRecord(ctx, id, tvColl)

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

func (m *mongoStore) GetAllTvSubTransactions(ctx context.Context, userID string) ([]models.TV_Result, error) {
	ctx = m.ensureCtx(ctx)
	res := []models.TV_Result{}

	cur, err := m.getAllRecords(ctx, tvColl, userID)
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

func (m *mongoStore) SaveElectricTransaction(ctx context.Context, details *models.ElectricResult) error {
	ctx = m.ensureCtx(ctx)
	err := m.saveToDB(ctx, electricColl, details)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) GetElectricSubDetails(ctx context.Context, id string) (models.ElectricResult, error) {
	ctx = m.ensureCtx(ctx)
	res := models.ElectricResult{}

	result := m.getRecord(ctx, id, electricColl)

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

func (m *mongoStore) GetAllElectricSubTransactions(ctx context.Context, userID string) ([]models.ElectricResult, error) {
	ctx = m.ensureCtx(ctx)
	res := []models.ElectricResult{}

	cur, err := m.getAllRecords(ctx, electricColl, userID)
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
func (m *mongoStore) SaveEduTransaction(ctx context.Context, details *models.EduResponse) error {
	ctx = m.ensureCtx(ctx)
	err := m.saveToDB(ctx, eduColl, details)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) GetEduTransactionDetails(ctx context.Context, id string) (models.EduResponse, error) {
	ctx = m.ensureCtx(ctx)
	res := models.EduResponse{}

	result := m.getRecord(ctx, id, eduColl)

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

func (m *mongoStore) GetAllEduTransactions(ctx context.Context, userID string) ([]models.EduResponse, error) {
	ctx = m.ensureCtx(ctx)
	res := []models.EduResponse{}

	cur, err := m.getAllRecords(ctx, eduColl, userID)
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
