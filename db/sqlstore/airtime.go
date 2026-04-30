package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
)

func (s *SqlStore) GetAirtimeProduct(ctx context.Context, network string) (models.AirtimeProduct, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	activeQuery := `
		SELECT
			a.id,
			a.provider_id,
			a.network,
			a.provider_discount_percent,
			a.customer_discount_percent,
			a.profit_margin
		FROM airtime a
		INNER JOIN airtime_provider ap ON a.provider_id = ap.id
		WHERE
			UPPER(a.network) = $1 AND
			a.available = TRUE AND
			ap.available = TRUE
		ORDER BY a.id ASC
		LIMIT 1
	`

	var ap models.AirtimeProduct
	err := s.db.QueryRowContext(ctx, activeQuery, network).Scan(
		&ap.ID,
		&ap.ProviderID,
		&ap.Network,
		&ap.Provider_Discount,
		&ap.Customer_Discount,
		&ap.Profit_Margin,
	)
	if err != nil {
		if usesLegacyAirtimeSchema(err) {
			s.logger.Warn("Falling back to legacy airtime schema", zap.String("network", network), zap.Error(err))
			return s.getLegacyAirtimeProduct(ctx, network)
		}

		if err == sql.ErrNoRows {
			s.logger.Warn("No active airtime product found", zap.String("network", network))
			return models.AirtimeProduct{}, fmt.Errorf("no active airtime product found for network %s: %w", network, err)
		}

		s.logger.Error("Failed to retrieve airtime product", zap.String("network", network), zap.Error(err))
		return models.AirtimeProduct{}, err
	}

	return ap, nil
}

func (s *SqlStore) getLegacyAirtimeProduct(ctx context.Context, network string) (models.AirtimeProduct, error) {
	legacyQuery := `
		SELECT network, provider_discount_percent, customer_discount_percent, profit_margin
		FROM airtime
		WHERE network = $1
	`

	var ap models.AirtimeProduct
	err := s.db.QueryRowContext(ctx, legacyQuery, network).Scan(
		&ap.Network,
		&ap.Provider_Discount,
		&ap.Customer_Discount,
		&ap.Profit_Margin,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.Warn("No legacy airtime product found", zap.String("network", network))
			return models.AirtimeProduct{}, fmt.Errorf("no airtime product found for network %s: %w", network, err)
		}

		s.logger.Error("Failed to retrieve airtime product from legacy schema", zap.String("network", network), zap.Error(err))
		return models.AirtimeProduct{}, err
	}

	return ap, nil
}

func usesLegacyAirtimeSchema(err error) bool {
	var sqlStateErr interface{ SQLState() string }
	if !errors.As(err, &sqlStateErr) {
		return false
	}

	switch sqlStateErr.SQLState() {
	case "42P01", "42703":
		return true
	default:
		return false
	}
}
