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
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

var (
	ErrMatchedCount = errors.New("failed to update user's referral count")
	ErrPointCount   = errors.New("not enough point to redeem")
)

var (
	pointColl       = "points"
	pointRedeemColl = "point-redeem"
)

func (m *mongoStore) updateReferralCount(referrersCode string) error {
	// TODO: using the referral code as the filter, update the count field on the user document
	ctx := context.Background()

	filter := bson.D{primitive.E{Key: "username", Value: referrersCode}}
	updateFilter := bson.D{
		{Key: "$inc", Value: bson.D{{Key: "count", Value: 1}}},
	}

	updateResult, err := m.col("user").UpdateOne(ctx, filter, updateFilter)
	if err != nil {
		m.logger.Error("error communicating with database", zap.Error(err))
		return errors.New("an error occurred while updating referral count")
	}
	if updateResult.MatchedCount == 0 {
		m.logger.Error("failed to update user's referral count", zap.Any("matchedCount", "no matched document updated"))
		return ErrMatchedCount
	}

	return nil
}

func (m *mongoStore) CreateUserReferral(newUserID, referralCode string) error {
	ctx := context.Background()

	referral := models.Referral{
		UserID:     newUserID,
		ReferredAt: time.Now().UTC(),
		IsActive:   true,
	}

	// If a referral code is provided, resolve the actual user ID
	if referralCode != "" {
		var referrer models.User
		err := m.col("user").FindOne(ctx, bson.M{"invitation_code": referralCode}).Decode(&referrer)
		if err != nil {
			m.logger.Warn("referral code not found or invalid", zap.String("code", referralCode), zap.Error(err))
		} else {
			referral.ReferrerID = referrer.ID

			// Update referrer's referral count
			if err := m.updateReferralCount(referrer.ID); err != nil {
				m.logger.Error("failed to update referrer count", zap.Error(err))
				if err == ErrMatchedCount {
					m.logger.Error("no matched document", zap.Error(err))
				}
				return fmt.Errorf("failed to update referrer count: %w", err)
			}
		}
	}

	// Save the referral record
	if _, err := m.col("referrals").InsertOne(ctx, referral); err != nil {
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
				"from":         "user",
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
		UserID:    userID,
		Balance:   0,
		CreatedAt: time.Now().UTC(),
	}

	_, err := m.col(pointColl).InsertOne(ctx, point)
	if err != nil {
		return err
	}

	return nil
}

func (m *mongoStore) RedeemPoints(userID string, pointsToRedeem int, redeemRate int) (amountRedeemed int, e error) {
	ctx := context.Background()

	filter := bson.M{"user_id": userID}
	points := models.Points{}

	// Get user's point balance
	err := m.col(pointColl).FindOne(ctx, filter).Decode(&points)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch points: %v", err)
	}

	// Check if the user has enough points
	if pointsToRedeem > points.Balance {
		return 0, ErrPointCount
	}

	// Calculate amount redeemed (assuming 1 point = redeemRate Naira)
	amountRedeemed = pointsToRedeem * redeemRate

	// Deduct points from user's points document
	_, err = m.col(pointColl).UpdateOne(ctx, filter, bson.M{
		"$inc": bson.M{"balance": -pointsToRedeem},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to update points: %v", err)
	}

	// Convert amountRedeemed to Decimal128
	amountDecimal, err := primitive.ParseDecimal128(fmt.Sprintf("%d", amountRedeemed))
	if err != nil {
		return 0, fmt.Errorf("failed to convert amount to Decimal128: %v", err)
	}

	// Update user's balance
	_, err = m.col(balColl).UpdateOne(ctx, filter, bson.M{
		"$inc": bson.M{"balance": amountDecimal},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to update user balance: %v", err)
	}

	return amountRedeemed, nil
}

func (m *mongoStore) CreatePointRedeemDoc(redeem models.PointRedeem) error {
	ctx := context.Background()

	// Create a new point redeem document
	redeem.CreatedAt = time.Now().UTC()

	// Insert the redeem document into the point-redeem collection
	_, err := m.col(pointRedeemColl).InsertOne(ctx, redeem)
	if err != nil {
		return fmt.Errorf("failed to create point redeem document: %v", err)
	}

	return nil
}

// Update user's point balance after transaction
func (m *mongoStore) UpdatePointAndTransactionTime(userID string, pointsEarned int) error {

	ctx := context.Background()
	// Update Points Document
	filter := bson.M{"user_id": userID}
	update := bson.D{
		{Key: "$inc", Value: bson.D{{Key: "balance", Value: pointsEarned}}},
		{Key: "$setOnInsert", Value: bson.D{{Key: "created_at", Value: time.Now()}}},
	}
	opts := options.Update().SetUpsert(true)
	_, err := m.col(pointColl).UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to update points: %v", err)
	}

	// Update User's last_transaction field
	userColl, err := m.userColl()
	if err != nil {

	}
	_, err = userColl.UpdateOne(ctx, bson.M{"id": userID}, bson.D{
		{Key: "$set", Value: bson.D{{Key: "last_transaction", Value: time.Now()}}},
	})
	if err != nil {
		return fmt.Errorf("failed to update user last_transaction: %v", err)
	}

	return nil
}
