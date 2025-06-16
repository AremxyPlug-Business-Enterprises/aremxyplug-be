package edu

import (
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/sqlstore"
	"go.uber.org/zap"
)

type Edurecords interface {
	GetRecord(id int) (models.EduRecord, error)
	UpdateRecord(id int, edu models.EduRecord) error
	DeleteRecord(id int) error
}

type EduServiceImpl struct {
	logger *zap.Logger
	store  *sqlstore.SqlStore
}

func NewEduService(store *sqlstore.SqlStore, logger *zap.Logger) Edurecords {
	return &EduServiceImpl{
		logger: logger,
		store:  store,
	}
}

func (s *EduServiceImpl) GetRecord(id int) (models.EduRecord, error) {
	s.logger.Info("Fetching records")
	record, err := s.store.GetEduRecord(id)
	if err != nil {
		s.logger.Error("Error fetching records", zap.Error(err))
		return models.EduRecord{}, err
	}
	s.logger.Info("Fetched records successfully", zap.Int("count", 1))
	return record, nil
}

func (s *EduServiceImpl) UpdateRecord(id int, edu models.EduRecord) error {
	s.logger.Info("Updating record", zap.Int("ID", id))
	err := s.store.UpdateEduRecord(id, edu)
	if err != nil {
		s.logger.Error("Error updating record", zap.Error(err))
		return err
	}
	s.logger.Info("Updated record successfully", zap.Int("ID", id))
	return nil
}

func (s *EduServiceImpl) DeleteRecord(id int) error {
	s.logger.Info("Deleting record", zap.Int("ID", id))
	err := s.store.DeleteEduRecord(id)
	if err != nil {
		s.logger.Error("Error deleting record", zap.Error(err))
		return err
	}
	s.logger.Info("Deleted record successfully", zap.Int("ID", id))
	return nil
}
