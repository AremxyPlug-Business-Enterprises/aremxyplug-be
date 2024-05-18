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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
	airColl  = "airtime"
	tvColl   = "tv-sub"
)

type mongoStore struct {
	mongoClient  *mongo.Client
	databaseName string
	logger       *zap.Logger
}

func (m *mongoStore) col(collectionName string) *mongo.Collection {
	return m.mongoClient.Database(m.databaseName).Collection(collectionName)
}

func (m *mongoStore) otpColl() (*mongo.Collection, error) {
	col := m.mongoClient.Database(m.databaseName).Collection("OTP")
	ctx := context.Background()
	indexModel := mongo.IndexModel{
		Keys:    bson.D{primitive.E{Key: "expireAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}

	_, err := col.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return nil, err
	}

	return col, nil
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

func (m *mongoStore) GetUserByUsername(username string) (*models.User, error) {
	filter := bson.M{
		"username": username,
	}
	user := &models.User{}
	err := m.col(models.UserCollectionName).FindOne(context.Background(), filter).Decode(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (m *mongoStore) GetUserByID(id string) (*models.User, error) {

	filter := bson.M{
		"id": id,
	}
	user := &models.User{}
	err := m.col(models.UserCollectionName).FindOne(context.Background(), filter).Decode(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (d *mongoStore) GetUserByUsernameOrEmail(email string, username string) (*models.User, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"email": email},
			{"username": username},
		},
	}
	user := &models.User{}
	err := d.mongoClient.
		Database(d.databaseName).
		Collection(models.UserCollectionName).
		FindOne(context.Background(), filter).
		Decode(user)
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

func (m *mongoStore) getRecord(id, collectionName string) *mongo.SingleResult {
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

func (m *mongoStore) saveToDB(collectionName string, details interface{}) error {
	ctx := context.Background()

	_, err := m.col(collectionName).InsertOne(ctx, details)
	if err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

func (m *mongoStore) getAllRecords(collectionName, user string) (*mongo.Cursor, error) {
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

func (m *mongoStore) SaveOTP(data models.OTP) error {
	ctx := context.Background()
	data.ExpireAt = time.Now().Add(time.Duration(5) * time.Minute)

	col, err := m.otpColl()
	if err != nil {
		return err
	}

	_, err = col.InsertOne(ctx, data)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) GetOTP(email string) (models.OTP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	data := models.OTP{}
	filter := bson.D{primitive.E{Key: "email", Value: email}}
	opts := options.FindOne().SetSort(bson.D{{Key: "expiresAt", Value: -1}})

	result := m.col("OTP").FindOne(ctx, filter, opts)
	err := result.Decode(&data)
	if err == mongo.ErrNoDocuments {
		return models.OTP{}, nil
	} else if err != nil {
		return models.OTP{}, err
	}

	return data, nil
}
