package mongo

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/errorvalues"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

// New returns a new instance of DataStore and Client
// response can contain error
func New(connectURI, databaseName string, logger *zap.Logger) (db.DataStore, *mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5+time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectURI))
	if err != nil {
		return nil, nil, err
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, nil, err
	}

	return &mongoStore{mongoClient: client, databaseName: databaseName, logger: logger}, client, nil
}

var _ db.DataStore = &mongoStore{}

var (
	dataColl = "data"
	eduColl  = "edu"
)

type mongoStore struct {
	mongoClient  *mongo.Client
	databaseName string
	logger       *zap.Logger
}

func (m *mongoStore) col(collectionName string) *mongo.Collection {
	return m.mongoClient.Database(m.databaseName).Collection(collectionName)
}

func (m *mongoStore) SaveUser(user models.User) error {
	_, err := m.col(models.UserCollectionName).InsertOne(context.Background(), user)
	if err != nil {
		return errorvalues.Format(errorvalues.DatabaseError, err)
	}

	return nil
}

func (m *mongoStore) GetUserByEmail(email string) (*models.User, error) {
	filter := bson.M{
		"email": email,
	}
	user := &models.User{}
	err := m.col(models.UserCollectionName).FindOne(context.Background(), filter).Decode(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (d *mongoStore) CreateMessage(message *models.Message) error {
	ctx := context.Background()
	var modelInDB models.Message
	err := d.mongoClient.
		Database(d.databaseName).
		Collection(models.MessagesCollectionName).
		FindOne(ctx, bson.M{"id": message.ID}).
		Decode(&modelInDB)
	if err != nil {
		// If error is not mongo.ErrNoDocuments return it
		// In case is mongo.ErrNoDocuments proceed with storing the model
		if err != mongo.ErrNoDocuments {
			return err
		}
	}
	// If model exist in DB skip the creation, return with no errors
	if modelInDB.ID != "" {
		return nil
	}

	_, err = d.mongoClient.
		Database(d.databaseName).
		Collection(models.MessagesCollectionName).
		InsertOne(ctx, message)
	if err != nil {
		return err
	}

	return nil
}

// update user password
func (d *mongoStore) UpdateUserPassword(email string, password string) error {
	ctx := context.Background()
	filter := bson.M{"email": email}
	update := bson.M{"$set": bson.M{"password": password}}
	_, err := d.mongoClient.
		Database(d.databaseName).
		Collection(models.UserCollectionName).
		UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

// SaveTransaction saves a data transaction to the database.
func (m *mongoStore) SaveDataTransaction(details *models.DataResult) error {

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

	if cur.Next(ctx) {
		resp := models.DataResult{}
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
	err := m.saveTransaction(eduColl, details)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) GetEduTransactionDetails(id string) (models.EduResponse, error) {
	res := models.EduResponse{}

	result := m.getTransaction(id, eduColl)

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

func (m *mongoStore) GetAllEduTransactions(user string) ([]models.EduResponse, error) {
	ctx := context.Background()
	res := []models.EduResponse{}

	cur, err := m.getAllTransaction(dataColl, user)
	if err != nil {
		return []models.EduResponse{}, err
	}

	if cur.Next(ctx) {
		resp := models.EduResponse{}
		if err := cur.Decode(&resp); err != nil {
			return nil, err
		}
		res = append(res, resp)
	}
	defer cur.Close(ctx)

	return res, nil
}

func (m *mongoStore) getTransaction(id, collectionName string) *mongo.SingleResult {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	oID, err := strconv.Atoi(id)
	if err != nil {
		return &mongo.SingleResult{}
	}

	filter := bson.D{primitive.E{Key: "order_id", Value: oID}}

	result := m.col(collectionName).FindOne(ctx, filter)

	return result

}

func (m *mongoStore) saveTransaction(collectionName string, details interface{}) error {
	ctx := context.Background()

	_, err := m.col(collectionName).InsertOne(ctx, details)

	return err
}

func (m *mongoStore) getAllTransaction(collectionName, user string) (*mongo.Cursor, error) {
	ctx := context.Background()
	var filter bson.D

	if user == "" {
		filter = bson.D{}
	} else {
		filter = bson.D{primitive.E{Key: "username", Value: user}}
	}

	cur, err := m.col(collectionName).Find(ctx, filter)
	return cur, err
}
