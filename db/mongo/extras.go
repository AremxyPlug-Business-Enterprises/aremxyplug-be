package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aremxyplug-be/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

func (m *mongoStore) UpdateReferralCount(referrersCode string) error {
	// TODO: using the referral code as the filter, update the count field on the user document
	ctx := context.Background()

	filter := bson.D{primitive.E{Key: "ref_code", Value: referrersCode}}
	updateFilter := bson.D{
		{Key: "$inc", Value: bson.D{{Key: "count", Value: 1}}},
	}

	updateResult, err := m.col("").UpdateOne(ctx, filter, updateFilter)
	if err != nil || updateResult.MatchedCount == 0 {
		m.logger.Error("failed to update user's referral count", zap.Error(err))
		return errors.New("failed to update user's referral count")
	}

	return nil
}

func (m *mongoStore) CreateUserReferral(newUserID, referralCode string) error {
	ctx := context.Background()

	referral := models.Referral{
		UserID:     newUserID,
		ReferrerID: "", // default, updated if referralCode is valid
		ReferredAt: time.Now().UTC(),
		IsActive:   true,
	}

	// If a referral code is provided, try to find the referrer
	if referralCode != "" {
		var referrer models.User
		err := m.col("users").FindOne(ctx, bson.M{"invitation_code": referralCode}).Decode(&referrer)
		if err == nil {
			referral.ReferrerID = referrer.ID

			// OPTIONAL: Immediately update the referrer's count (if you're storing it)
			if err := m.UpdateReferralCount(referrer.ID); err != nil {
				m.logger.Error("failed to update referrer count", zap.Error(err))
				return fmt.Errorf("failed to update referrer count: %w", err)
			}
		}
	}

	_, err := m.col("referrals").InsertOne(ctx, referral)
	if err != nil {
		return fmt.Errorf("failed to create referral record: %w", err)
	}

	return nil
}

func (m *mongoStore) GetReferredUsers(referrerID string) ([]models.ReferredUserInfo, error) {
	ctx := context.Background()

	pipeline := mongo.Pipeline{
		// Match referrals where this user is the referrer
		{{Key: "$match", Value: bson.M{"referrerID": referrerID}}},

		// Join with users collection to get referred user's details
		{{
			Key: "$lookup", Value: bson.M{
				"from":         "users",
				"localField":   "user_id",
				"foreignField": "id",
				"as":           "user_info",
			},
		}},

		// Unwind the user_info array to get a flat document
		{{Key: "$unwind", Value: "$user_info"}},

		// Project desired fields
		{{
			Key: "$project", Value: bson.M{
				"user_id":     "$user_id",
				"full_name":   "$user_info.fullname",
				"email":       "$user_info.email",
				"is_active":   "$is_active",
				"referred_at": "$referred_at",
			},
		}},
	}

	cursor, err := m.col("referrals").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation error: %w", err)
	}
	defer cursor.Close(ctx)

	var results []models.ReferredUserInfo
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("cursor decode error: %w", err)
	}

	return results, nil
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

func (m *mongoStore) GetPoint(userID string) (models.Points, error) {

	ctx := context.Background()

	filter := bson.D{primitive.E{Key: "user_id", Value: userID}}
	points := models.Points{}

	result := m.col("").FindOne(ctx, filter)
	if err := result.Decode(&points); err != nil {
		return models.Points{}, nil
	}

	return points, nil
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
