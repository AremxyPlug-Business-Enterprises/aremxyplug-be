package mongo

import (
	"context"
	"errors"
	"fmt"
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

var (
	dataColl     = "data"
	eduColl      = "edu"
	airColl      = "airtime"
	tvColl       = "tv-sub"
	electricColl = "elect-sub"
	userColl     = "user"
	messagesColl = "messages"
	// verificationsColl = "verifications"
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

	store := &mongoStore{
		mongoClient:  client,
		databaseName: databaseName,
		logger:       logger,
	}

	if err := store.InitIndexes(); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize indexes: %w", err)
	}
	if err := store.BackfillIdentityHashes(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to backfill identity hashes: %w", err)
	}
	if err := store.initIdentityIndexes(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize identity indexes: %w", err)
	}

	return store, client, nil
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

func (m *mongoStore) ensureCtx(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (m *mongoStore) InitIndexes() error {
	ctx := context.Background()
	db := m.mongoClient.Database(m.databaseName)

	// OTP index
	otpIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "expireAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}
	if _, err := db.Collection("OTP").Indexes().CreateOne(ctx, otpIndex); err != nil {
		return fmt.Errorf("failed to create OTP index: %w", err)
	}

	// SMS index
	smsIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "expireAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}
	if _, err := db.Collection("SMS").Indexes().CreateOne(ctx, smsIndex); err != nil {
		return fmt.Errorf("failed to create SMS index: %w", err)
	}

	// User index (if you really need a TTL on user collection)
	userIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "expireAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}
	if _, err := db.Collection(userColl).Indexes().CreateOne(ctx, userIndex); err != nil {
		return fmt.Errorf("failed to create user index: %w", err)
	}

	identityLookupIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "birth_date", Value: 1},
			{Key: "has_bvn", Value: 1},
			{Key: "has_nin", Value: 1},
		},
	}
	if _, err := db.Collection(userColl).Indexes().CreateOne(ctx, identityLookupIndex); err != nil {
		return fmt.Errorf("failed to create identity lookup index: %w", err)
	}

	bankIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "nip_code", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	if _, err := db.Collection(bankColl).Indexes().CreateOne(ctx, bankIndex); err != nil {
		return fmt.Errorf("failed to create bank index: %w", err)
	}

	deptIndex := mongo.IndexModel{
		Keys:    bson.D{primitive.E{Key: "ID", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	if _, err := db.Collection(deptColl).Indexes().CreateOne(ctx, deptIndex); err != nil {
		return fmt.Errorf("failed to create department index: %w", err)
	}

	telecomRecipientIndex := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "recipients.active", Value: 1},
			},
			Options: options.Index().SetSparse(true),
		},
	}

	if _, err := db.Collection("telcom-recipient").Indexes().CreateMany(ctx, telecomRecipientIndex); err != nil {
		return fmt.Errorf("failed to create telcom recipient index: %w", err)
	}

	tasksIndex := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "task_code", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "completed", Value: 1}, {Key: "updated_at", Value: -1}},
		},
	}

	if _, err := db.Collection("tasks").Indexes().CreateMany(ctx, tasksIndex); err != nil {
		return fmt.Errorf("failed to create tasks index: %w", err)
	}

	return nil
}

func (m *mongoStore) SaveUser(ctx context.Context, user models.User) error {
	ctx = m.ensureCtx(ctx)
	user.ExpireAt = time.Now().Add(time.Duration(15) * time.Minute)

	col := m.col(userColl)

	_, err := col.InsertOne(ctx, user)
	if err != nil {
		return errorvalues.Format(errorvalues.DatabaseError, err)
	}

	return nil
}

func (m *mongoStore) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{
		"email": email,
	}
	user := &models.User{}
	err := m.col(userColl).FindOne(ctx, filter).Decode(user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, err
		}
		return nil, err
	}
	return user, nil
}

func (m *mongoStore) GetUserByPhone(ctx context.Context, phone string) (*models.User, error) {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{
		"phonenumber": phone,
	}
	user := &models.User{}
	err := m.col(userColl).FindOne(ctx, filter).Decode(user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, err
		}
		return nil, err
	}
	return user, nil
}

func (m *mongoStore) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{
		"username": username,
	}
	user := &models.User{}
	err := m.col(userColl).FindOne(ctx, filter).Decode(user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, err
		}
		return nil, err
	}
	return user, nil
}

func (m *mongoStore) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{
		"id": id,
	}
	user := &models.User{}
	err := m.col(userColl).FindOne(ctx, filter).Decode(user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (m *mongoStore) GetUserByUsernameOrEmail(ctx context.Context, email string, username string) (*models.User, error) {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{}

	if email != "" {
		filter["email"] = email
	}
	if username != "" {
		filter["username"] = username
	}

	if email == "" && username == "" {
		return nil, errors.New("email or username must be provided")
	}

	user := &models.User{}
	err := m.mongoClient.
		Database(m.databaseName).
		Collection(userColl).
		FindOne(ctx, filter).
		Decode(user)
	if err != nil {
		return nil, err
	}
	return user, nil

}

func (m *mongoStore) GetUserByUsernameOrEmailOrPhone(ctx context.Context, username, email, phone string) (*models.User, error) {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{
		"$or": []bson.M{
			{"email": email},
			{"username": username},
			{"phonenumber": phone},
		},
	}
	user := &models.User{}
	err := m.mongoClient.
		Database(m.databaseName).
		Collection(userColl).
		FindOne(ctx, filter).
		Decode(user)
	if err != nil {
		return nil, err
	}
	return user, nil

}
func (m *mongoStore) CreateMessage(ctx context.Context, message *models.Message) error {
	ctx = m.ensureCtx(ctx)
	var modelInDB models.Message
	err := m.col(messagesColl).
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

	_, err = m.col(messagesColl).
		InsertOne(ctx, message)
	if err != nil {
		return err
	}

	return nil
}

// update user password
func (m *mongoStore) UpdateUserPassword(ctx context.Context, email string, password string) error {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{"email": email}
	update := bson.M{"$set": bson.M{"password": password}}
	_, err := m.mongoClient.
		Database(m.databaseName).
		Collection(userColl).
		UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func (m *mongoStore) UpdateUserPasswordByID(ctx context.Context, id string, password string) error {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{"id": id}
	update := bson.M{"$set": bson.M{"password": password}}
	_, err := m.col(userColl).
		UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func (m *mongoStore) UpdateBVNField(ctx context.Context, user models.User) error {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{"id": user.ID}
	update := bson.M{"$set": bson.M{"bvn": user.BVN, "bvn_hash": user.BVNHash, "bvn_phone": user.BVNPhone, "has_bvn": true}}
	_, err := m.col(userColl).
		UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func (m *mongoStore) UpdateNINField(ctx context.Context, user models.User) error {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{"id": user.ID}
	update := bson.M{"$set": bson.M{"nin": user.NIN, "nin_hash": user.NINHash, "has_nin": true}}
	_, err := m.col(userColl).
		UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func (m *mongoStore) UpdateEmail(ctx context.Context, id, email string) error {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{"id": id}
	update := bson.M{"$set": bson.M{"email": email}}
	coll := m.col(userColl)
	_, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil

}

func (m *mongoStore) UpdatePhone(ctx context.Context, id, phone string) error {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{"id": id}
	update := bson.M{"$set": bson.M{"phonenumber": phone}}
	coll := m.col(userColl)
	_, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) UpdateUserBeta(ctx context.Context, id string, beta bool) error {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{"id": id}
	update := bson.M{"$set": bson.M{"beta": beta}}
	_, err := m.col(userColl).UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) VerifyUser(ctx context.Context, identifier string) (*models.User, error) {
	userColl := m.col(userColl)
	ctx = m.ensureCtx(ctx)

	filter := bson.M{
		"$or": []bson.M{
			{"email": identifier},
			{"phonenumber": identifier},
		},
	}

	user := &models.User{}

	err := userColl.FindOne(ctx, filter).Decode(user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no user found with the identifier: %s", identifier)
		}
		return nil, fmt.Errorf("error querying the database: %w", err)
	}

	if user.IsVerified {
		return nil, errors.New("user is already verified")
	}

	update := bson.M{
		"$set":   bson.M{"is_verified": true},
		"$unset": bson.M{"expireAt": ""},
	}

	updateResult, err := userColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update user document: %w", err)
	}

	if updateResult.MatchedCount == 0 {
		return nil, errors.New("failed to update user document")
	}

	user.IsVerified = true

	return user, nil
}

func (m *mongoStore) UpdateUserAddress(ctx context.Context, userID string, gender string, dob string, address string, postalCode string) error {
	ctx = m.ensureCtx(ctx)

	birth_date, err := time.Parse("2006-01-02", dob)
	if err != nil {
		return fmt.Errorf("failed to parse date of birth: %w", err)

	}

	filter := bson.M{"id": userID}
	update := bson.M{
		"$set": bson.M{
			"birth_date":  birth_date,
			"gender":      gender,
			"address":     address,
			"postal_code": postalCode,
		}}
	_, err = m.col(userColl).
		UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return err
	}
	return nil
}

func (m *mongoStore) getRecord(ctx context.Context, id, collectionName string) *mongo.SingleResult {
	ctx = m.ensureCtx(ctx)
	oID, err := strconv.Atoi(id)
	if err != nil {
		return &mongo.SingleResult{}
	}

	filter := bson.D{primitive.E{Key: "order_id", Value: oID}}

	result := m.col(collectionName).FindOne(ctx, filter)

	return result

}

func (m *mongoStore) saveToDB(ctx context.Context, collectionName string, details interface{}) error {
	ctx = m.ensureCtx(ctx)
	_, err := m.col(collectionName).InsertOne(ctx, details)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) getAllRecords(ctx context.Context, collectionName, userID string) (*mongo.Cursor, error) {
	ctx = m.ensureCtx(ctx)
	var filter bson.D

	if userID == "" {
		filter = bson.D{}
	} else {
		filter = bson.D{primitive.E{Key: "user_id", Value: userID}}
	}

	cur, err := m.col(collectionName).Find(ctx, filter)
	return cur, err
}

func (m *mongoStore) SaveOTP(ctx context.Context, data models.OTP) error {
	ctx = m.ensureCtx(ctx)
	data.ExpireAt = time.Now().Add(time.Duration(5) * time.Minute)

	col := m.col("OTP")

	_, err := col.InsertOne(ctx, data)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) GetOTP(ctx context.Context, email string) (models.OTP, error) {
	ctx = m.ensureCtx(ctx)
	data := models.OTP{}
	filter := bson.D{primitive.E{Key: "email", Value: email}}
	opts := options.FindOne().SetSort(bson.D{{Key: "expireAt", Value: -1}})

	result := m.col("OTP").FindOne(ctx, filter, opts)
	err := result.Decode(&data)
	if err == mongo.ErrNoDocuments {
		return models.OTP{}, errors.New("no record found")
	} else if err != nil {
		return models.OTP{}, err
	}

	return data, nil
}

func (m *mongoStore) SaveSMS(ctx context.Context, data models.SMSOTP) error {
	ctx = m.ensureCtx(ctx)
	data.ExpireAt = time.Now().Add(time.Duration(5) * time.Minute)

	col := m.col("SMS")

	_, err := col.InsertOne(ctx, data)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) GetSMS(ctx context.Context, phone string) (models.SMSOTP, error) {
	ctx = m.ensureCtx(ctx)
	data := models.SMSOTP{}
	filter := bson.D{primitive.E{Key: "phone", Value: phone}}
	opts := options.FindOne().SetSort(bson.D{{Key: "expireAt", Value: -1}})

	result := m.col("SMS").FindOne(ctx, filter, opts)
	err := result.Decode(&data)
	if err == mongo.ErrNoDocuments {
		return models.SMSOTP{}, errors.New("no record found")
	} else if err != nil {
		return models.SMSOTP{}, err
	}

	return data, nil
}

func (m *mongoStore) CheckID(ctx context.Context, id int) (int64, error) {
	ctx = m.ensureCtx(ctx)
	filter := bson.M{"id": id}

	count, err := m.col(userColl).CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return count, err
}
