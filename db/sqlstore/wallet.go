package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aremxyplug-be/db/models"
)

func (s *SqlStore) GetWalletCharges(ctx context.Context) (models.WalletCharges, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var charges models.WalletCharges
	query := "SELECT api_charge, service_charge FROM wallet_charges LIMIT 1"

	row := s.db.QueryRowContext(ctx, query)
	err := row.Scan(&charges.APICharge, &charges.ServiceCharge)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.WalletCharges{}, fmt.Errorf("wallet charges record not found")
		}
		return models.WalletCharges{}, fmt.Errorf("failed to scan wallet charges: %v", err)
	}

	return charges, nil
}
