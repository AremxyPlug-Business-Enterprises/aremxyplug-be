package sqlstore

import (
	"context"
	"fmt"

	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
)

func (s *SqlStore) GetProducts(id int) ([]models.Product, error) {
	rows, err := s.db.Query("SELECT product_id, network_id, plan_type FROM products WHERE network_id = ?", id)
	if err != nil {
		s.logger.Error("Error querying products", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		err := rows.Scan(&p.Product_ID, &p.Network_ID, &p.Plan_Type)
		if err != nil {
			s.logger.Error("Error scanning product row", zap.Error(err))
			return nil, err
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating over product rows", zap.Error(err))
		return nil, err
	}

	s.logger.Info("Products retrieved successfully", zap.Int("count", len(products)))
	return products, nil
}

func (s *SqlStore) CreatePlan(plan models.Plan) (int, error) {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return 0, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	var maxID *int
	err = tx.QueryRowContext(ctx, "SELECT MAX(plan_id) FROM plans").Scan(&maxID)
	if err != nil {
		s.logger.Error("Failed to get max plan_id", zap.Error(err))
		return 0, fmt.Errorf("failed to get max plan_id: %v", err)
	}

	nextID := 1
	if maxID != nil {
		nextID = *maxID + 1
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO plans 
        (plan_id, product_id, amount, validity, size)
        VALUES (?, ?, ?, ?, ?)`,
		nextID,
		plan.ProductID,
		plan.Amount,
		plan.Validity,
		plan.Size,
	)
	if err != nil {
		s.logger.Error("Failed to insert plan", zap.Error(err))
		return 0, fmt.Errorf("insert failed: %v", err)
	}

	if err = tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return 0, fmt.Errorf("commit failed: %v", err)
	}

	s.logger.Info("Plan created successfully", zap.Int("planID", nextID))
	return nextID, nil
}

func (s *SqlStore) UpdatePlan(planID int, updatedPlan models.Plan) error {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM plans WHERE plan_id = ?)",
		planID).Scan(&exists)
	if err != nil {
		s.logger.Error("Failed to check plan existence", zap.Error(err))
		return fmt.Errorf("existence check failed: %v", err)
	}
	if !exists {
		s.logger.Warn("Plan does not exist", zap.Int("planID", planID))
		return fmt.Errorf("plan ID %d does not exist", planID)
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE plans SET
            amount = ?,
            validity = ?,
            size = ?
        WHERE plan_id = ?`,
		updatedPlan.Amount,
		updatedPlan.Validity,
		updatedPlan.Size,
		planID)
	if err != nil {
		s.logger.Error("Failed to update plan", zap.Error(err))
		return fmt.Errorf("update failed: %v", err)
	}

	if err = tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("commit failed: %v", err)
	}

	s.logger.Info("Plan updated successfully", zap.Int("planID", planID))
	return nil
}

func (s *SqlStore) DeletePlan(planID int) error {
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM plans WHERE plan_id = ?)",
		planID).Scan(&exists)
	if err != nil {
		s.logger.Error("Failed to check plan existence", zap.Error(err))
		return fmt.Errorf("existence check failed: %v", err)
	}
	if !exists {
		s.logger.Warn("Plan does not exist", zap.Int("planID", planID))
		return fmt.Errorf("plan ID %d does not exist", planID)
	}

	_, err = tx.ExecContext(ctx,
		"DELETE FROM plans WHERE plan_id = ?",
		planID)
	if err != nil {
		s.logger.Error("Failed to delete plan", zap.Error(err))
		return fmt.Errorf("delete failed: %v", err)
	}

	if err = tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("commit failed: %v", err)
	}

	s.logger.Info("Plan deleted successfully", zap.Int("planID", planID))
	return nil
}

func (s *SqlStore) GetPlansByProductID(productID int) ([]models.Plan, error) {
	rows, err := s.db.Query(`
        SELECT plan_id, product_id, amount, validity, size
        FROM plans
        WHERE product_id = ?`, productID)
	if err != nil {
		s.logger.Error("Failed to retrieve plans", zap.Int("productID", productID), zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve plans for product ID %d: %v", productID, err)
	}
	defer rows.Close()

	var plans []models.Plan
	for rows.Next() {
		var plan models.Plan
		err := rows.Scan(
			&plan.PlanID,
			&plan.ProductID,
			&plan.Amount,
			&plan.Validity,
			&plan.Size,
		)
		if err != nil {
			s.logger.Error("Failed to scan plan row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan plan row: %v", err)
		}
		plans = append(plans, plan)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating over plan rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating over plan rows: %v", err)
	}

	s.logger.Info("Plans retrieved successfully", zap.Int("productID", productID), zap.Int("count", len(plans)))
	return plans, nil
}
