package mongo

import (
	"context"
	"time"

	"github.com/aremxyplug-be/db/models"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Transaction index model
type TxnIndex struct {
	TxnID      string `bson:"txn"`
	Collection string `bson:"collection"`
}

// SaveTxnIndex inserts a mapping from TXNID to its source collection
func (m *mongoStore) SaveTxnIndex(txID, collection string) error {
	ctx := context.Background()
	idx := TxnIndex{TxnID: txID, Collection: collection}
	_, err := m.col("txn_index").InsertOne(ctx, idx)
	return err
}

// GetTxnCollection returns the collection name for a given TXNID
func (m *mongoStore) GetTxnCollection(txID string) (string, error) {
	ctx := context.Background()
	var idx TxnIndex
	err := m.col("txn_index").FindOne(ctx, bson.M{"txn": txID}).Decode(&idx)
	if err != nil {
		return "", err
	}
	return idx.Collection, nil
}

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

func (m *mongoStore) UpdateRecieptByTxID(txnID string) error {
	ctx := context.Background()
	status := "failed"

	// Fallback: scan all collections if not found in index
	collections := []string{dataColl, airColl, tvColl, eduColl, electricColl, transferColl} // add more as needed
	var updateErr error
	for _, coll := range collections {
		filter := bson.M{"txn": txnID}
		update := bson.M{"$set": bson.M{"status": status}}
		result, err := m.col(coll).UpdateOne(ctx, filter, update)
		if err != nil {
			updateErr = err
			continue
		}
		if result.ModifiedCount > 0 {
			return nil // Updated successfully in this collection
		}
	}
	if updateErr != nil {
		return updateErr
	}
	return nil // Not found, but no error
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

func (m *mongoStore) UpdateUserBalanceFromRedis(userID string, balance decimal.Decimal) error {
	ctx := context.Background()
	filter := bson.M{"user_id": userID}

	bal, err := primitive.ParseDecimal128(balance.String())
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

func (m *mongoStore) UpdateReceiptExternalRef(externalRef, status string) error {
	_, err := m.col(transferColl).UpdateOne(context.Background(), bson.M{"reference": externalRef}, bson.M{
		"$set": bson.M{"status": status},
	})
	return err
}

func (m *mongoStore) GetUserFromVirtualNuban(virtualNuban string) (string, error) {
	var result models.AccountDetails
	err := m.col(virtualColl).FindOne(context.Background(), bson.M{"virtual_nuban": virtualNuban}).Decode(&result)
	if err != nil {
		return "", err
	}
	return result.User_ID, nil
}
