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

// SaveTransaction saves a data transaction to the database.
func (d *mongoStore) SaveTransaction(details *models.DataResult) error {
	ctx := context.Background()

	_, err := d.mongoClient.Database(d.databaseName).Collection("data").InsertOne(ctx, details)
	if err != nil {
		return err
	}

	return nil
}

// GetTransactionDetails returns a data transaction detail.
func (d *mongoStore) GetTransactionDetails(id string) (result models.DataResult, err error) {
	ctx := context.Background()
	res := models.DataResult{}
	oID, err := strconv.Atoi(id)
	if err != nil {
		return models.DataResult{}, err
	}

	filter := bson.D{primitive.E{Key: "order_id", Value: oID}}

	coll := d.mongoClient.Database(d.databaseName).Collection("data")
	e := coll.FindOne(ctx, filter).Decode(&res)
	if e != nil {
		if e == mongo.ErrNoDocuments {
			return models.DataResult{}, nil
		}
		// write for errors
		log.Println(e)
	}

	return res, nil

}

// GetAllTransaction returns all the data transactions associated to a user, if an empty string is passed it returns all data transactions.
func (d *mongoStore) GetAllTransactions(user string) ([]models.DataResult, error) {
	ctx := context.Background()
	res := []models.DataResult{}

	var filter bson.D

	if user == "" {
		filter = bson.D{}
	} else {
		filter = bson.D{primitive.E{Key: "username", Value: user}}
	}

	coll := d.mongoClient.Database(d.databaseName).Collection("data")
	cur, err := coll.Find(ctx, filter)
	if err != nil {
		// write for errors
		log.Println(err)
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
