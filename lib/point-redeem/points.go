package pointredeem

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/lib/randomgen"
)

type PointConfig struct {
	db db.Extras
}

func NewPointConfig(store db.Extras) *PointConfig {
	return &PointConfig{
		db: store,
	}
}

func (p *PointConfig) RedeemPoints(ctx context.Context, userID string, points int) (models.PointRedeem, error) {

	const (
		MaxRedeemCap = 100
		MinRedeem    = 10
	)

	// Validate minimum points to redeem
	if points < MinRedeem {
		return models.PointRedeem{}, fmt.Errorf("minimum redeem is %d points, you requested %d", MinRedeem, points)
	}

	// Get total points already redeemed by this user
	totalRedeemed, err := p.db.GetTotalPointsRedeemed(ctx, userID)
	if err != nil {
		return models.PointRedeem{}, err
	}

	// Check if redemption would exceed the cap
	if totalRedeemed+points > MaxRedeemCap {
		remainingCap := MaxRedeemCap - totalRedeemed
		return models.PointRedeem{}, fmt.Errorf("redemption cap exceeded: you have already redeemed %d points, can only redeem up to %d more", totalRedeemed, remainingCap)
	}

	redeemRate := 1
	redeemRateStr := fmt.Sprintf("%d PTS ~ ₦1", redeemRate)

	transactionID := randomgen.GenerateTransactionID("pnt")
	orderID, _ := randomgen.GenerateOrderID()

	redeemedAmount, err := p.db.RedeemPoints(ctx, userID, points, redeemRate)
	if err != nil {
		return models.PointRedeem{}, err
	}

	amount_redeemed := strconv.Itoa(redeemedAmount)
	point_redeemed := strconv.Itoa(points)

	redeemDoc := models.PointRedeem{
		UserID:                 userID,
		Points_Redeemed:        point_redeemed,
		Amount_Redeemed:        amount_redeemed,
		Redeemed_Rate:          redeemRateStr,
		TransactionProduct:     "Point Redeem",
		TransactionDescription: "NGN Wallet Top Up",
		TransactionID:          transactionID,
		OrderID:                orderID,
		CreatedAt:              time.Now().UTC(),
		Status:                 "success",
	}

	if err := p.db.CreatePointRedeemDoc(ctx, redeemDoc); err != nil {
		return models.PointRedeem{}, err
	}

	return redeemDoc, nil

}

func (p *PointConfig) GetPoints(ctx context.Context, userID string) (models.PointSummary, error) {

	point, err := p.db.GetPoint(ctx, userID)
	if err != nil {
		return models.PointSummary{}, err
	}

	return point, nil

}

func (p *PointConfig) UserPoints(ctx context.Context, userID string) error {
	err := p.db.CreatePointDoc(ctx, userID)
	if err != nil {
		return err
	}

	return nil
}

func (p *PointConfig) CreatePointTransaction(ctx context.Context, txn models.PointTransaction) error {
	txn.CreatedAt = time.Now().UTC()
	txn.TransactionID = randomgen.GenerateTransactionID("pnt")

	err := p.db.LogPointTransaction(ctx, txn)
	if err != nil {
		return fmt.Errorf("failed to create point transaction: %v", err)
	}

	return nil
}
