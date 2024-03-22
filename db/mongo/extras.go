package mongo

import (
	"context"
	"errors"

	"github.com/aremxyplug-be/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (m *mongoStore) UpdateReferralCount(referralCode string) error {
	// TODO: using the referral code as the filter, update the count field on the user document
	ctx := context.Background()

	filter := bson.D{primitive.E{Key: "ref_code", Value: referralCode}}
	updateFilter := bson.D{}

	updateResult, err := m.col("").UpdateOne(ctx, filter, updateFilter)
	if err != nil || updateResult.MatchedCount == 0 {
		return errors.New("failed to update user's referral count")
	}

	return nil
}

func (m *mongoStore) CreateUserReferral(userID, refcode string) error {
	// TODO: create a referral document for user using userID and refCode
	ctx := context.Background()

	refDoc := models.Referral{
		UserID:  userID,
		RefCode: refcode,
		Count:   0,
	}

	_, err := m.col("").InsertOne(ctx, refDoc)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) UpdatePoint(userID string, points int) error {
	// TODO: update the point doucument using the userID as the filter and adding the points to the previous point balance
	ctx := context.Background()

	filter := bson.D{primitive.E{Key: "user_id", Value: userID}}
	updateFilter := bson.D{}

	updateResult, err := m.col("").UpdateOne(ctx, filter, updateFilter)
	if err != nil || updateResult.MatchedCount == 0 {
		return errors.New("failed to update user's point balance")
	}

	return nil
}

func (m *mongoStore) CreatePointDoc(userID string) error {
	// TODO: Create a document on the collection points for the user on signUp
	ctx := context.Background()

	point := models.Points{
		UserID:  userID,
		Balance: 0,
	}

	_, err := m.col("").InsertOne(ctx, point)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) CanRedeemPoints(userID string, points int) bool {
	// TODO: first get the user point from the database and then compare with the points to redeem
	ctx := context.Background()

	pointDoc := models.Points{}

	filter := bson.D{primitive.E{Key: "user_id", Value: userID}}

	result := m.col("").FindOne(ctx, filter)
	if err := result.Decode(&pointDoc); err != nil {
		return false
	}
	return true
}
