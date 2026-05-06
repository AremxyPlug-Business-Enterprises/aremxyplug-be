package sqlstore

import (
	"context"
	"database/sql"

	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
)

func (s *SqlStore) GetProductLock(ctx context.Context, key string) (models.ProductLock, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	lock := models.ProductLock{}
	query := `
		SELECT key, enabled, COALESCE(description, ''), COALESCE(updated_at, NOW())
		FROM product_locks
		WHERE key = $1
	`

	err := s.db.QueryRowContext(ctx, query, key).Scan(
		&lock.Key,
		&lock.Enabled,
		&lock.Description,
		&lock.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// Missing rows default to enabled to avoid disabling products during partial rollout.
			return models.ProductLock{Key: key, Enabled: true}, nil
		}
		s.logger.Error("failed to retrieve product lock", zap.String("key", key), zap.Error(err))
		return models.ProductLock{}, err
	}

	return lock, nil
}
