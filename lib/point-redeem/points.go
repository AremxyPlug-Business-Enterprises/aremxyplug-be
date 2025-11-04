package pointredeem

import (
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

func (p *PointConfig) RedeemPoints(userID string, points int) (models.PointRedeem, error) {

	redeemRate := 1

	// write the as a complete string
	redeemRateStr := fmt.Sprintf("%d", redeemRate)

	transactionID := randomgen.GenerateTransactionID("pnt")
	orderID, _ := randomgen.GenerateOrderID()

	redeemedAmount, err := p.db.RedeemPoints(userID, points, redeemRate)
	if err != nil {
		// depending on the error returned
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

	if err := p.db.CreatePointRedeemDoc(redeemDoc); err != nil {
		return models.PointRedeem{}, err
	}

	return redeemDoc, nil

}

func (p *PointConfig) GetPoints(userID string) (models.PointSummary, error) {

	point, err := p.db.GetPoint(userID)
	if err != nil {
		return models.PointSummary{}, err
	}

	return point, nil

}

func (p *PointConfig) UserPoints(userID string) error {
	err := p.db.CreatePointDoc(userID)
	if err != nil {
		return err
	}

	return nil
}

func (p *PointConfig) CreatePointTransaction(txn models.PointTransaction) error {
	txn.CreatedAt = time.Now().UTC()
	txn.TransactionID = randomgen.GenerateTransactionID("pnt")

	err := p.db.LogPointTransaction(txn)
	if err != nil {
		return fmt.Errorf("failed to create point transaction: %v", err)
	}

	return nil
}
