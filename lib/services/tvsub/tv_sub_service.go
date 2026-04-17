package tvsub

import (
	"context"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/sqlstore"
	"go.uber.org/zap"
)

type TVSubService interface {
	GetTVSubs(ctx context.Context, tableName string) ([]models.TVSub, error)
	CreateTVSub(ctx context.Context, tableName string, data models.TVSub) (int, error)
	UpdateTVSub(ctx context.Context, tableName string, id int, data models.TVSubUpdate) error
	DeleteTVSub(ctx context.Context, tableName string, id int) error
	GetTVSubByPackageName(ctx context.Context, tableName string, packageName string) (*models.TVSub, error)
}

type TVSubServiceImpl struct {
	logger *zap.Logger
	store  *sqlstore.SqlStore
}

func NewTVSubService(logger *zap.Logger, store *sqlstore.SqlStore) TVSubService {
	return &TVSubServiceImpl{
		logger: logger,
		store:  store,
	}
}

func (s *TVSubServiceImpl) GetTVSubs(ctx context.Context, tableName string) ([]models.TVSub, error) {
	s.logger.Info("Fetching TV subscriptions")
	tvSubs, err := s.store.GetTVSubs(ctx, tableName)
	if err != nil {
		s.logger.Error("Error fetching TV subscriptions", zap.Error(err))
		return nil, err
	}
	s.logger.Info("Fetched TV subscriptions successfully", zap.Int("count", len(tvSubs)))
	return tvSubs, nil

}

func (s *TVSubServiceImpl) GetTVSubByPackageName(ctx context.Context, tableName string, packageName string) (*models.TVSub, error) {
	s.logger.Info("Fetching TV subscription by package name", zap.String("packageName", packageName))
	tvSub, err := s.store.GetTVSubByPackage(ctx, tableName, packageName)
	if err != nil {
		s.logger.Error("Error fetching TV subscription by package name", zap.String("packageName", packageName), zap.Error(err))
		return nil, err
	}
	s.logger.Info("Fetched TV subscription by package name successfully", zap.String("packageName", packageName))
	return tvSub, nil
}

func (s *TVSubServiceImpl) CreateTVSub(ctx context.Context, tableName string, data models.TVSub) (int, error) {
	s.logger.Info("Creating TV subscription")
	id, err := s.store.CreateTvSub(ctx, tableName, data)
	if err != nil {
		s.logger.Error("Error creating TV subscription", zap.Error(err))
		return 0, err
	}
	s.logger.Info("Created TV subscription successfully", zap.Int("id", id))
	return id, nil
}

func (s *TVSubServiceImpl) UpdateTVSub(ctx context.Context, tableName string, id int, data models.TVSubUpdate) error {
	s.logger.Info("Updating TV subscription", zap.Int("id", id))
	err := s.store.UpdateTvSub(ctx, tableName, id, data)
	if err != nil {
		s.logger.Error("Error updating TV subscription", zap.Error(err))
		return err
	}
	s.logger.Info("Updated TV subscription successfully", zap.Int("id", id))
	return nil
}

func (s *TVSubServiceImpl) DeleteTVSub(ctx context.Context, tableName string, id int) error {
	s.logger.Info("Deleting TV subscription", zap.Int("id", id))
	err := s.store.DeleteTvSub(ctx, tableName, id)
	if err != nil {
		s.logger.Error("Error deleting TV subscription", zap.Error(err))
		return err
	}
	s.logger.Info("Deleted TV subscription successfully", zap.Int("id", id))
	return nil
}
