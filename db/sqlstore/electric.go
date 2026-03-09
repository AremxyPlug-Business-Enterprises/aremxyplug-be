package sqlstore

import (
	"database/sql"

	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
)

func (s *SqlStore) GetElectricDetails(discoType string) (*models.ElectricDetails, error) {
	query := "SELECT disco_type, profit_margin FROM electric_providers WHERE disco_type = $1"
	rows := s.db.QueryRow(query, discoType)

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
