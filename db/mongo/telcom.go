package mongo

import (
	"context"
	"log"
	"sort"

	"github.com/aremxyplug-be/db/models/telcom"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
func (m *mongoStore) GetAllDataTransactions(username string) ([]telcom.DataResult, error) {
	ctx := context.Background()
	res := []telcom.DataResult{}

	cur, err := m.getAllRecords(dataColl, username)
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

func (m *mongoStore) GetAllSpecDataTransactions(username string) ([]telcom.SpectranetResult, error) {
	ctx := context.Background()
	res := []telcom.SpectranetResult{}

	cur, err := m.getAllRecords(dataColl, username)
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

func (m *mongoStore) GetAllSmileDataTransactions(username string) ([]telcom.SmileResult, error) {
	ctx := context.Background()
	res := []telcom.SmileResult{}

	cur, err := m.getAllRecords(dataColl, username)
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

func (m *mongoStore) GetAllAirtimeTransactions(username string) ([]telcom.AirtimeResponse, error) {
	ctx := context.Background()
	res := []telcom.AirtimeResponse{}

	cur, err := m.getAllRecords(dataColl, username)
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

	filter := bson.D{primitive.E{Key: "userID", Value: userID}}
	projection := bson.M{"recipients.id": 1}
	telcomRecipient := telcom.TelcomRecipient{}

	err := coll.FindOne(ctx, filter, options.FindOne().SetProjection(projection)).Decode(&telcomRecipient)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			data.ID = 0
			telcomRecipient = telcom.TelcomRecipient{
				UserID:    userID,
				Recipient: append(telcomRecipient.Recipient, data),
			}

			_, err := coll.InsertOne(ctx, telcomRecipient)
			if err != nil {
				return err
			}
			return nil
		}
	}

	maxID := 0
	for _, recipient := range telcomRecipient.Recipient {
		if recipient.ID > 0 {
			maxID = recipient.ID
		}
	}

	data.ID = maxID + 1

	updateFilter := bson.D{{Key: "$push", Value: bson.D{primitive.E{Key: "recipients", Value: data}}}}

	_, err = coll.UpdateOne(ctx, filter, updateFilter)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) GetTelcomRecipients(userID string) (telcom.TelcomRecipient, error) {

	ctx := context.Background()
	recipients := telcom.TelcomRecipient{}
	coll := m.col("telcom-recipient")

	filter := bson.D{primitive.E{Key: "userID", Value: userID}}
	res := coll.FindOne(ctx, filter)

	if err := res.Decode(&recipients); err != nil {
		if err == mongo.ErrNoDocuments {
			return telcom.TelcomRecipient{}, nil
		}
		return telcom.TelcomRecipient{}, err
	}

	return recipients, nil

}

func (m *mongoStore) EditTelcomRecipient(userID string, data telcom.Recipient) error {

	ctx := context.Background()
	coll := m.col("telcom-recipient")
	telcomRecipient := telcom.TelcomRecipient{}

	filter := bson.D{primitive.E{Key: "userID", Value: userID}}

	findResult := coll.FindOne(ctx, filter)
	if err := findResult.Decode(&telcomRecipient); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil
		}
		return err
	}

	recipientToUpdate := telcom.Recipient{}
	for i := range telcomRecipient.Recipient {
		if telcomRecipient.Recipient[i].ID == data.ID {
			recipientToUpdate = telcomRecipient.Recipient[i]
			break
		}
	}

	// Define update fields based on what is provided
	updateFields := bson.M{}
	if data.Name != "" && recipientToUpdate.Name != data.Name {
		updateFields["recipients.$.name"] = data.Name
	}
	if data.Phone_no != "" && recipientToUpdate.Phone_no != data.Phone_no {
		updateFields["recipients.$.phone"] = data.Phone_no
	}

	if len(updateFields) > 0 {
		// Prepare the update statement
		updateFilter := bson.M{"$set": updateFields}

		filter := bson.M{
			"userID":        userID,
			"recipients.id": data.ID,
		}

		_, err := coll.UpdateOne(ctx, filter, updateFilter)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *mongoStore) DeleteTelcomRecipient(recipientID int, userID string) error {

	ctx := context.Background()
	coll := m.col("telcom-recipient")

	filter := bson.M{
		"userID": userID,
	}
	projection := bson.M{"recipients": 1}
	telcomRecipient := telcom.TelcomRecipient{}

	delResult := coll.FindOne(ctx, filter, options.FindOne().SetProjection(projection))
	if err := delResult.Decode(&telcomRecipient); err != nil {
		return err
	}

	updatedRecipients := []telcom.Recipient{}
	for _, recipient := range telcomRecipient.Recipient {
		if recipient.ID != recipientID {
			updatedRecipients = append(updatedRecipients, recipient)
		}
	}

	sort.SliceStable(updatedRecipients, func(i, j int) bool {
		return updatedRecipients[i].ID < updatedRecipients[j].ID
	})
	for i := range updatedRecipients {
		updatedRecipients[i].ID = i
	}

	updateFilter := bson.D{primitive.E{Key: "$set", Value: bson.D{primitive.E{Key: "recipients", Value: updatedRecipients}}}}
	_, err := coll.UpdateOne(ctx, filter, updateFilter)
	if err != nil {
		return err
	}

	return nil

}

func (m *mongoStore) getRecipientRecords() (*mongo.Cursor, error) {
	ctx := context.Background()

	filter := bson.D{}
	cur, err := m.col("").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	return cur, nil

}
