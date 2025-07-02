package pointredeem

import (
	"fmt"
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

	redeemDoc := models.PointRedeem{
		UserID:                 userID,
		Points_Redeemed:        points,
		Amount_Redeemed:        float64(redeemedAmount),
		Redeemed_Rate:          redeemRateStr,
		TransactionProduct:     "",
		TransactionDescription: "",
		TransactionID:          transactionID,
		OrderID:                orderID,
		CreatedAt:              time.Now().UTC(),
	}

	if err := p.db.CreatePointRedeemDoc(redeemDoc); err != nil {
		return models.PointRedeem{}, nil
	}

	return redeemDoc, nil

}

func (p *PointConfig) GetPoints(userID string) (models.Points, error) {

	point, err := p.db.GetPoint(userID)
	if err != nil {
		return models.Points{}, err
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
