package electric

import (
	"context"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/sqlstore"
	"go.uber.org/zap"
)

type ElectricProducts interface {
	GetElectricDetails(ctx context.Context, discoType string) (*models.ElectricDetails, error)
}

type ElectricServiceImpl struct {
	logger *zap.Logger
	store  *sqlstore.SqlStore
}

func NewElectricService(store *sqlstore.SqlStore, logger *zap.Logger) ElectricProducts {
	return &ElectricServiceImpl{
		logger: logger,
		store:  store,
	}
}

func (s *ElectricServiceImpl) GetElectricDetails(ctx context.Context, discoType string) (*models.ElectricDetails, error) {
	s.logger.Info("Fetching electric details", zap.String("discoType", discoType))
	details, err := s.store.GetElectricDetails(ctx, discoType)
	if err != nil {
		s.logger.Error("Error fetching electric details", zap.String("discoType", discoType), zap.Error(err))
		return nil, err
	}
	s.logger.Info("Fetched electric details successfully", zap.String("discoType", discoType))
	return details, nil
}
