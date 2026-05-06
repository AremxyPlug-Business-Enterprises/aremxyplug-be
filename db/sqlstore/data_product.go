package sqlstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
)

func (s *SqlStore) GetProducts(id int) ([]models.Product, error) {
	return s.GetProductsWithContext(context.Background(), id)
}

func (s *SqlStore) GetProductsWithContext(ctx context.Context, id int) ([]models.Product, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	rows, err := s.db.QueryContext(ctx,
		"SELECT product_id, network_id, plan_type FROM products WHERE network_id = $1 ORDER BY sort ASC",
		id)
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
	return s.CreatePlanWithContext(context.Background(), plan)
}

func (s *SqlStore) CreatePlanWithContext(ctx context.Context, plan models.Plan) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var insertedID int
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO plans (product_id, amount, validity, size)
		VALUES ($1, $2, $3, $4)
		RETURNING plan_id`,
		plan.ProductID,
		plan.Amount,
		plan.Validity,
		plan.Size,
	).Scan(&insertedID)
	if err != nil {
		s.logger.Error("Failed to insert plan", zap.Error(err))
		return 0, fmt.Errorf("insert failed: %v", err)
	}

	s.logger.Info("Plan created successfully", zap.Int("planID", insertedID))
	return insertedID, nil
}

func (s *SqlStore) UpdatePlan(planID int, updatedPlan models.PlanUpdate) error {
	return s.UpdatePlanWithContext(context.Background(), planID, updatedPlan)
}

func (s *SqlStore) UpdatePlanWithContext(ctx context.Context, planID int, updatedPlan models.PlanUpdate) error {
	if ctx == nil {
		ctx = context.Background()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Track clauses and values for the SET part
	setClauses := []string{}
	args := []interface{}{}

	argPos := 1

	if updatedPlan.Amount != 0 {
		setClauses = append(setClauses, fmt.Sprintf("amount = $%d", argPos))
		args = append(args, updatedPlan.Amount)
		argPos++
	}
	if updatedPlan.Validity != "" {
		setClauses = append(setClauses, fmt.Sprintf("validity = $%d", argPos))
		args = append(args, updatedPlan.Validity)
		argPos++
	}
	if updatedPlan.Size != "" {
		setClauses = append(setClauses, fmt.Sprintf("size = $%d", argPos))
		args = append(args, updatedPlan.Size)
		argPos++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no fields provided for update")
	}

	// Add the planID to the args for WHERE clause
	args = append(args, planID)

	// Build the final query
	query := fmt.Sprintf(
		"UPDATE plans SET %s WHERE plan_id = $%d",
		strings.Join(setClauses, ", "),
		argPos,
	)

	var exists bool
	err = tx.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM plans WHERE plan_id = $1)",
		planID).Scan(&exists)
	if err != nil {
		s.logger.Error("Failed to check plan existence", zap.Error(err))
		return fmt.Errorf("existence check failed: %v", err)
	}
	if !exists {
		s.logger.Warn("Plan does not exist", zap.Int("planID", planID))
		return fmt.Errorf("plan ID %d does not exist", planID)
	}

	_, err = tx.ExecContext(ctx, query, args...)
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
	return s.DeletePlanWithContext(context.Background(), planID)
}

func (s *SqlStore) DeletePlanWithContext(ctx context.Context, planID int) error {
	if ctx == nil {
		ctx = context.Background()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM plans WHERE plan_id = $1)",
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
		"DELETE FROM plans WHERE plan_id = $1",
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
	return s.GetPlansByProductIDWithContext(context.Background(), productID)
}

func (s *SqlStore) GetPlansByProductIDWithContext(ctx context.Context, productID int) ([]models.Plan, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT 
			p.id, 
			p.product_id, 
			p.amount, 
			p.validity, 
			p.size, 
			pr.plan_type 
		FROM plans p
		INNER JOIN products pr ON p.product_id = pr.product_id
		INNER JOIN api_providers ap ON p.provider_id = ap.id
		WHERE 
			p.product_id = $1 AND 
			p.available = TRUE AND 
			LOWER(p.status) = 'active' AND 
			LOWER(ap.status) = 'active' AND 
			ap.available = TRUE
		ORDER BY
			CASE
				WHEN LOWER(TRIM(REPLACE(p.size, ' ', ''))) LIKE '%tb'
				THEN CAST(REPLACE(LOWER(TRIM(REPLACE(p.size, ' ', ''))), 'tb', '') AS NUMERIC(20,6)) * 1024 * 1024
				WHEN LOWER(TRIM(REPLACE(p.size, ' ', ''))) LIKE '%gb'
				THEN CAST(REPLACE(LOWER(TRIM(REPLACE(p.size, ' ', ''))), 'gb', '') AS NUMERIC(20,6)) * 1024
				WHEN LOWER(TRIM(REPLACE(p.size, ' ', ''))) LIKE '%mb'
				THEN CAST(REPLACE(LOWER(TRIM(REPLACE(p.size, ' ', ''))), 'mb', '') AS NUMERIC(20,6))
				WHEN LOWER(TRIM(REPLACE(p.size, ' ', ''))) LIKE '%kb'
				THEN CAST(REPLACE(LOWER(TRIM(REPLACE(p.size, ' ', ''))), 'kb', '') AS NUMERIC(20,6)) / 1024
				ELSE 0
			END ASC,
			CASE
				WHEN LOWER(p.validity) LIKE '%daily%' THEN 1
				WHEN LOWER(p.validity) LIKE '%weekly%' THEN 7
				WHEN LOWER(p.validity) LIKE '%7 day%' THEN 7
				WHEN LOWER(p.validity) LIKE '%30 day%' THEN 30
				WHEN LOWER(p.validity) LIKE '%1 month%' THEN 30
				ELSE 9999 
			END ASC
	`, productID)
	if err != nil {
		s.logger.Error("Failed to retrieve plans", zap.Int("productID", productID), zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve plans for product ID %d: %v", productID, err)
	}
	defer rows.Close()

	var plans []models.Plan
	for rows.Next() {
		var plan models.Plan
		err := rows.Scan(
			&plan.ID,
			&plan.ProductID,
			&plan.Amount,
			&plan.Validity,
			&plan.Size,
			&plan.PlanType,
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

func (s *SqlStore) GetPlanByID(planID int) (*models.Plan, error) {
	return s.GetPlanByIDWithContext(context.Background(), planID)
}

func (s *SqlStore) GetPlanByIDWithContext(ctx context.Context, planID int) (*models.Plan, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var plan models.Plan
	err := s.db.QueryRowContext(ctx, `
		SELECT 
			p.plan_id, 
			p.product_id, 
			p.amount, 
			p.validity,
			p.provider_id, 
			p.size,
			p.profit_margin,
			pr.plan_type
		FROM plans p
		INNER JOIN products pr ON p.product_id = pr.product_id
		INNER JOIN api_providers ap ON p.provider_id = ap.id
		WHERE 
			p.id = $1 AND
			p.available = TRUE AND
			LOWER(p.status) = 'active' AND
			LOWER(ap.status) = 'active' AND
			ap.available = TRUE`, planID).Scan(
		&plan.PlanID,
		&plan.ProductID,
		&plan.Amount,
		&plan.Validity,
		&plan.ProviderID,
		&plan.Size,
		&plan.ProfitMargin,
		&plan.PlanType,
	)
	if err != nil {
		s.logger.Error("Failed to retrieve plan", zap.Int("planID", planID), zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve plan with ID %d: %v", planID, err)
	}

	s.logger.Info("Plan retrieved successfully", zap.Int("planID", int(plan.PlanID.Int64)), zap.Float64("profitMargin", plan.ProfitMargin))
	return &plan, nil
}
