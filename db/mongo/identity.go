package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/encryption"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (m *mongoStore) initIdentityIndexes(ctx context.Context) error {
	ctx = m.ensureCtx(ctx)

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "bvn_hash", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys:    bson.D{{Key: "nin_hash", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	}

	if _, err := m.col(userColl).Indexes().CreateMany(ctx, indexes); err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) GetUserByBVNHash(ctx context.Context, hash string) (*models.User, error) {
	return m.getUserByIdentityHash(ctx, "bvn_hash", hash)
}

func (m *mongoStore) GetUserByNINHash(ctx context.Context, hash string) (*models.User, error) {
	return m.getUserByIdentityHash(ctx, "nin_hash", hash)
}

func (m *mongoStore) getUserByIdentityHash(ctx context.Context, field, hash string) (*models.User, error) {
	ctx = m.ensureCtx(ctx)
	if hash == "" {
		return nil, mongo.ErrNoDocuments
	}

	user := &models.User{}
	err := m.col(userColl).FindOne(ctx, bson.M{field: hash}).Decode(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (m *mongoStore) ListOtherVerifiedUsersByDOB(ctx context.Context, excludeUserID string, dob time.Time) ([]models.User, error) {
	ctx = m.ensureCtx(ctx)

	filter := bson.M{
		"birth_date": dob,
		"$or": []bson.M{
			{"has_bvn": true},
			{"has_nin": true},
		},
	}
	if excludeUserID != "" {
		filter["id"] = bson.M{"$ne": excludeUserID}
	}

	cur, err := m.col(userColl).Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	users := make([]models.User, 0)
	for cur.Next(ctx) {
		var user models.User
		if err := cur.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (m *mongoStore) BackfillIdentityHashes(ctx context.Context) error {
	ctx = m.ensureCtx(ctx)

	if err := m.backfillIdentityHashField(ctx, identityBackfillConfig{
		label:          "bvn",
		hasField:       "has_bvn",
		encryptedField: "bvn",
		hashField:      "bvn_hash",
		getEncrypted: func(user models.User) string {
			return user.BVN
		},
		lookupExisting: m.GetUserByBVNHash,
	}); err != nil {
		return err
	}

	if err := m.backfillIdentityHashField(ctx, identityBackfillConfig{
		label:          "nin",
		hasField:       "has_nin",
		encryptedField: "nin",
		hashField:      "nin_hash",
		getEncrypted: func(user models.User) string {
			return user.NIN
		},
		lookupExisting: m.GetUserByNINHash,
	}); err != nil {
		return err
	}

	return nil
}

type identityBackfillConfig struct {
	label          string
	hasField       string
	encryptedField string
	hashField      string
	getEncrypted   func(models.User) string
	lookupExisting func(context.Context, string) (*models.User, error)
}

func (m *mongoStore) backfillIdentityHashField(ctx context.Context, cfg identityBackfillConfig) error {
	filter := bson.M{
		cfg.hasField:       true,
		cfg.encryptedField: bson.M{"$ne": ""},
		"$or": []bson.M{
			{cfg.hashField: bson.M{"$exists": false}},
			{cfg.hashField: ""},
		},
	}

	cur, err := m.col(userColl).Find(ctx, filter)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var user models.User
		if err := cur.Decode(&user); err != nil {
			return err
		}

		plain, err := encryption.DecryptString(cfg.getEncrypted(user))
		if err != nil {
			return fmt.Errorf("failed to decrypt %s for user %s: %w", cfg.label, user.ID, err)
		}
		hash := encryption.HashIdentityID(plain)

		existing, err := cfg.lookupExisting(ctx, hash)
		switch {
		case err == nil && existing.ID != "" && existing.ID != user.ID:
			return fmt.Errorf("duplicate %s found for users %s and %s", cfg.label, existing.ID, user.ID)
		case err != nil && err != mongo.ErrNoDocuments:
			return err
		}

		_, err = m.col(userColl).UpdateOne(ctx, bson.M{"id": user.ID}, bson.M{
			"$set": bson.M{cfg.hashField: hash},
		})
		if err != nil {
			return err
		}
	}

	if err := cur.Err(); err != nil {
		return err
	}

	return nil
}
