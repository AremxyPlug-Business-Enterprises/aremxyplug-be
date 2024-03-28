package pointredeem

import "github.com/aremxyplug-be/db"

type PointConfig struct {
	db db.Extras
}

func NewPointConfig(store db.Extras) *PointConfig {
	return &PointConfig{
		db: store,
	}
}

func (p *PointConfig) RedeemPoints(userID string, points int) bool {

	yes := p.db.CanRedeemPoints(userID, points)
	if !yes {
		return false
	}

	return yes
}

func (p *PointConfig) UpdatePoints(userID string, points int) error {

	err := p.db.UpdatePoint(userID, points)
	if err != nil {
		return err
	}

	return nil
}

func (p *PointConfig) UserPoints(userID string) error {
	err := p.db.CreatePointDoc(userID)
	if err != nil {
		return err
	}

	return nil
}
