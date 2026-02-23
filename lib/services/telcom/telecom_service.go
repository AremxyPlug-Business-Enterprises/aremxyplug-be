package telecom

import (
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/sqlstore"
	"go.uber.org/zap"
)

type TelecomProducts interface {
	GetProducts(id int) ([]models.Product, error)
	CreatePlan(plan models.Plan) (int, error)
	UpdatePlan(id int, data models.PlanUpdate) error
	DeletePlan(id int) error
	GetPlans(id int) ([]models.Plan, error)
	GetPlanByID(planID int) (*models.Plan, error)
	GetAirtimeProduct(network string) (models.AirtimeProduct, error)
	GetEduRecord(id int) (models.EduRecord, error)
}

type TelecomServiceImpl struct {
	logger *zap.Logger
	store  *sqlstore.SqlStore
}

func NewTelecomService(store *sqlstore.SqlStore, logger *zap.Logger) TelecomProducts {
	return &TelecomServiceImpl{
		logger: logger,
		store:  store,
	}
}

func (s *TelecomServiceImpl) GetProducts(id int) ([]models.Product, error) {
	s.logger.Info("Fetching products")
	products, err := s.store.GetProducts(id)
	if err != nil {
		s.logger.Error("Error fetching products", zap.Error(err))
		return nil, err
	}
	s.logger.Info("Fetched products successfully", zap.Int("count", len(products)))
	return products, nil
}

func (s *TelecomServiceImpl) CreatePlan(plan models.Plan) (int, error) {
	s.logger.Info("Creating plan")
	id, err := s.store.CreatePlan(plan)
	if err != nil {
		s.logger.Error("Error creating plan", zap.Error(err))
		return 0, err
	}
	s.logger.Info("Created plan successfully", zap.Int("id", id))
	return id, nil
}

func (s *TelecomServiceImpl) UpdatePlan(id int, data models.PlanUpdate) error {
	s.logger.Info("Updating plan", zap.Int("id", id))
	err := s.store.UpdatePlan(id, data)
	if err != nil {
		s.logger.Error("Error updating plan", zap.Error(err))
		return err
	}
	s.logger.Info("Updated plan successfully", zap.Int("id", id))
	return nil
}

func (s *TelecomServiceImpl) DeletePlan(id int) error {
	s.logger.Info("Deleting plan", zap.Int("id", id))
	err := s.store.DeletePlan(id)
	if err != nil {
		s.logger.Error("Error deleting plan", zap.Error(err))
		return err
	}
	s.logger.Info("Deleted plan successfully", zap.Int("id", id))
	return nil
}

func (s *TelecomServiceImpl) GetPlans(id int) ([]models.Plan, error) {
	s.logger.Info("Fetching plans")
	plans, err := s.store.GetPlansByProductID(id)
	if err != nil {
		s.logger.Error("Error fetching plans", zap.Error(err))
		return nil, err
	}
	s.logger.Info("Fetched plans successfully", zap.Int("count", len(plans)))
	return plans, nil
}

func (s *TelecomServiceImpl) GetPlanByID(planID int) (*models.Plan, error) {
	s.logger.Info("Fetching plan by ID", zap.Int("id", planID))
	plan, err := s.store.GetPlanByID(planID)
	if err != nil {
		s.logger.Error("Error fetching plan by ID", zap.Error(err))
		return nil, err
	}
	s.logger.Info("Fetched plan by ID successfully", zap.Int("id", planID))
	return plan, nil
}

func (s *TelecomServiceImpl) GetAirtimeProduct(network string) (models.AirtimeProduct, error) {
	s.logger.Info("Fetching airtime product", zap.String("network", network))
	airtimeProduct, err := s.store.GetAirtimeProduct(network)
	if err != nil {
		s.logger.Error("Error fetching airtime product", zap.Error(err))
		return models.AirtimeProduct{}, err
	}
	s.logger.Info("Fetched airtime product successfully", zap.String("network", network))
	return airtimeProduct, nil
}

func (s *TelecomServiceImpl) GetEduRecord(id int) (models.EduRecord, error) {
	s.logger.Info("Fetching records")
	record, err := s.store.GetEduRecord(id)
	if err != nil {
		s.logger.Error("Error fetching records", zap.Error(err))
		return models.EduRecord{}, err
	}
	s.logger.Info("Fetched records successfully", zap.Int("count", 1))
	return record, nil
}
