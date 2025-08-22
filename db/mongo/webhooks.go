package mongo

import (
	"context"
	"strconv"
	"time"

	"github.com/aremxyplug-be/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (m *mongoStore) GetReceiptByExternalRef(externalRef string) (models.TransferResponse, error) {
	var receipt models.TransferResponse
	err := m.col(transferColl).FindOne(context.Background(), bson.M{"reference": externalRef}).Decode(&receipt)
	if err != nil {
		return models.TransferResponse{}, err
	}
	return receipt, nil
}

func (m *mongoStore) GetReceiptByTxID(txID string) (models.TransferResponse, error) {
	var receipt models.TransferResponse
	err := m.col(transferColl).FindOne(context.Background(), bson.M{"txn": txID}).Decode(&receipt)
	if err != nil {
		return models.TransferResponse{}, err
	}
	return receipt, nil
}

func (m *mongoStore) UpdateReceiptFinal(txID, status, sessionID string) error {
	setStatus := ""

	if status == "SUCCESS" {
		setStatus = "success"
	}
	if status == "FAILED" {
		setStatus = "failed"
	}
	_, err := m.col(transferColl).UpdateOne(context.Background(), bson.M{"txn": txID}, bson.M{
		"$set": bson.M{
			"status":     setStatus,
			"session_id": sessionID,
		},
	})
	return err
}

func (m *mongoStore) UpdateUserBalanceFromRedis(userID string, balance float64) error {
	ctx := context.Background()
	filter := bson.M{"user_id": userID}

	balanceToParse := strconv.FormatFloat(balance, 'f', 2, 64)

	bal, err := primitive.ParseDecimal128(balanceToParse)
	if err != nil {
		m.logger.Error(err.Error())
		return err
	}

	updateFilter := bson.D{{Key: "$set", Value: bson.D{
		{Key: "balance", Value: bal},
		{Key: "updated_at", Value: time.Now().UTC()},
	}}}

	_, err = m.col(balColl).UpdateOne(ctx, filter, updateFilter)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) UpdateReceiptStatus(txID, status string) error {
	_, err := m.col(transferColl).UpdateOne(context.Background(), bson.M{"txn": txID}, bson.M{
		"$set": bson.M{"status": status},
	})
	return err
}

func (m *mongoStore) UpdateReceiptExternalRef(txID, externalRef string) error {
	_, err := m.col(transferColl).UpdateOne(context.Background(), bson.M{"txn": txID}, bson.M{
		"$set": bson.M{"reference": externalRef},
	})
	return err
}
