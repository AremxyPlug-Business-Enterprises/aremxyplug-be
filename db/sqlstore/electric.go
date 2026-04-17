package sqlstore

import (
	"context"
	"database/sql"

	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
)

func (s *SqlStore) GetElectricDetails(ctx context.Context, discoType string) (*models.ElectricDetails, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	query := "SELECT disco_type, profit_margin FROM electric_providers WHERE disco_type = $1"
	rows := s.db.QueryRowContext(ctx, query, discoType)

	details := &models.ElectricDetails{}
	if err := rows.Scan(&details.DiscoType, &details.Profit_Margin); err != nil {
		if err == sql.ErrNoRows {
			s.logger.Warn("Electric details not found", zap.String("discoType", discoType))
			return nil, nil
		}
		return nil, err
	}
	return details, nil

}
