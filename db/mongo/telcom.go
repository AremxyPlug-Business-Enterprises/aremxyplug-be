package mongo

import (
	"context"
	"fmt"
	"log"

	"github.com/aremxyplug-be/db/models/telcom"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
func (m *mongoStore) GetAllDataTransactions(userID string) ([]telcom.DataResult, error) {
	ctx := context.Background()
	res := []telcom.DataResult{}

	cur, err := m.getAllRecords(dataColl, userID)
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

func (m *mongoStore) GetSpecTransDetails(id string) (telcom.SpectranetResult, error) {
	res := telcom.SpectranetResult{}

	findResult := m.getRecord(id, dataColl)
	err := findResult.Decode(&res)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return telcom.SpectranetResult{}, nil
		}
		// write for errors
		log.Println(err)
		return telcom.SpectranetResult{}, err
	}

	return res, nil
}

func (m *mongoStore) GetAllSpecDataTransactions(userID string) ([]telcom.SpectranetResult, error) {
	ctx := context.Background()
	res := []telcom.SpectranetResult{}

	cur, err := m.getAllRecords(dataColl, userID)
	if err != nil {
		return []telcom.SpectranetResult{}, err
	}

	for cur.Next(ctx) {
		resp := telcom.SpectranetResult{}
		if err := cur.Decode(&resp); err != nil {
			return nil, err
		}
		res = append(res, resp)
	}
	defer cur.Close(ctx)

	return res, nil
}

func (m *mongoStore) GetSmileTransDetails(id string) (telcom.SmileResult, error) {
	res := telcom.SmileResult{}

	findResult := m.getRecord(id, dataColl)
	err := findResult.Decode(&res)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return telcom.SmileResult{}, nil
		}
		// write for errors
		log.Println(err)
		return telcom.SmileResult{}, err
	}

	return res, nil
}

func (m *mongoStore) GetAllSmileDataTransactions(userID string) ([]telcom.SmileResult, error) {
	ctx := context.Background()
	res := []telcom.SmileResult{}

	cur, err := m.getAllRecords(dataColl, userID)
	if err != nil {
		return []telcom.SmileResult{}, err
	}

	for cur.Next(ctx) {
		resp := telcom.SmileResult{}
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

	result := m.getRecord(id, airColl)

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

func (m *mongoStore) GetAllAirtimeTransactions(userID string) ([]telcom.AirtimeResponse, error) {
	ctx := context.Background()
	res := []telcom.AirtimeResponse{}

	cur, err := m.getAllRecords(airColl, userID)
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

func (m *mongoStore) SaveTelcomRecipient(userID string, data telcom.Recipient) error {
	ctx := context.Background()
	coll := m.col("telcom-recipient")

	// 1) Check if active recipient with same phone exists -> reject
	activeFilter := bson.M{
		"user_id": userID,
		"recipients": bson.M{
			"$elemMatch": bson.M{
				"phone":  data.Phone_no,
				"active": true,
			},
		},
	}
	if err := coll.FindOne(ctx, activeFilter).Err(); err == nil {
		return fmt.Errorf("recipient with phone %s already exists", data.Phone_no)
	} else if err != mongo.ErrNoDocuments {
		return err
	}

	// 2) Check if inactive recipient exists -> reactivate & update fields (reuse ID)
	inactiveFilter := bson.M{
		"user_id": userID,
		"recipients": bson.M{
			"$elemMatch": bson.M{
				"phone":  data.Phone_no,
				"active": false,
			},
		},
	}
	// Build set to update fields on the matched array slot
	setFields := bson.M{
		"recipients.$.active":  true,
		"recipients.$.name":    data.Name,
		"recipients.$.network": data.Network,
	}
	res, err := coll.UpdateOne(ctx, inactiveFilter, bson.M{"$set": setFields})
	if err != nil {
		return err
	}
	if res.ModifiedCount > 0 {
		// Reactivated successfully — done
		return nil
	}

	// 3) No existing entry — allocate new ID then push
	id, err := m.nextRecipientID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to generate recipient id: %w", err)
	}
	data.ID = id
	data.Active = true // ensure active set

	upsertFilter := bson.M{"user_id": userID}
	update := bson.M{
		"$setOnInsert": bson.M{"user_id": userID},
		"$push":        bson.M{"recipients": data},
	}
	opts := options.Update().SetUpsert(true)
	_, err = coll.UpdateOne(ctx, upsertFilter, update, opts)
	return err
}

func (m *mongoStore) nextRecipientID(ctx context.Context, userID string) (int, error) {

	counterColl := m.col("recipient_counters")
	filter := bson.M{"_id": "recipient_" + userID}
	update := bson.M{"$inc": bson.M{"seq": 1}}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var result struct {
		Seq int `bson:"seq"`
	}
	err := counterColl.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return 0, err
	}
	return result.Seq, nil
}

func (m *mongoStore) GetTelcomRecipients(userID string) (telcom.TelcomRecipient, error) {
	ctx := context.Background()
	coll := m.col("telcom-recipient")

	// fetch full doc
	filter := bson.M{"user_id": userID}
	doc := telcom.TelcomRecipient{}
	err := coll.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return telcom.TelcomRecipient{UserID: userID}, nil
		}
		return telcom.TelcomRecipient{}, err
	}

	// Filter active recipients in Go
	active := make([]telcom.Recipient, 0, len(doc.Recipient))
	for _, r := range doc.Recipient {
		if r.Active {
			active = append(active, r)
		}
	}
	doc.Recipient = active
	return doc, nil
}

func (m *mongoStore) EditTelcomRecipient(userID string, data telcom.Recipient) error {
	ctx := context.Background()
	coll := m.col("telcom-recipient")

	// If changing phone, ensure another active recipient does not already use it
	if data.Phone_no != "" {
		conflictFilter := bson.M{
			"user_id": userID,
			"recipients": bson.M{
				"$elemMatch": bson.M{
					"phone":  data.Phone_no,
					"active": true,
					"id":     bson.M{"$ne": data.ID}, // not the same recipient
				},
			},
		}
		if err := coll.FindOne(ctx, conflictFilter).Err(); err == nil {
			return fmt.Errorf("another active recipient already uses phone %s", data.Phone_no)
		} else if err != mongo.ErrNoDocuments {
			return err
		}
	}

	// Build update set
	set := bson.M{}
	if data.Name != "" {
		set["recipients.$.name"] = data.Name
	}
	if data.Phone_no != "" {
		set["recipients.$.phone"] = data.Phone_no
	}
	if data.Network != "" {
		set["recipients.$.network"] = data.Network
	}

	if len(set) == 0 {
		return nil
	}

	filter := bson.M{
		"user_id":       userID,
		"recipients.id": data.ID,
	}
	_, err := coll.UpdateOne(ctx, filter, bson.M{"$set": set})
	return err
}

func (m *mongoStore) DeleteTelcomRecipient(recipientID int, userID string) error {
	ctx := context.Background()
	coll := m.col("telcom-recipient")

	filter := bson.M{
		"user_id":       userID,
		"recipients.id": recipientID,
	}
	update := bson.M{"$set": bson.M{"recipients.$.active": false}}
	res, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("recipient id %d not found for user %s", recipientID, userID)
	}
	return nil
}
