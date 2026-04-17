package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
)

var validTableName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,63}$`)

func validateTableName(table string) error {
	if !validTableName.MatchString(table) {
		return errors.New("invalid table name format")
	}
	return nil
}

// GetPlans retrieves all plans from specified table
func (s *SqlStore) GetTVSubs(ctx context.Context, table string) ([]models.TVSub, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := validateTableName(table); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}

	query := fmt.Sprintf(
		"SELECT plan_id, package, package_name, amount, profit_margin FROM %s",
		sanitizeTableName(table),
	)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		s.logger.Error("Error querying plans", zap.String("table", table), zap.Error(err))
		return nil, fmt.Errorf("failed to query %s plans: %w", table, err)
	}
	defer rows.Close()

	var plans []models.TVSub
	for rows.Next() {
		var p models.TVSub
		if err := rows.Scan(
			&p.ID,
			&p.Package,
			&p.PackageName,
			&p.Amount,
			&p.Profit_Margin,
		); err != nil {
			s.logger.Error("Error scanning plan row", zap.String("table", table), zap.Error(err))
			return nil, fmt.Errorf("failed to scan %s plan: %w", table, err)
		}
		plans = append(plans, p)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Error iterating plan rows", zap.String("table", table), zap.Error(err))
		return nil, fmt.Errorf("row iteration error for %s: %w", table, err)
	}

	s.logger.Info("Plans retrieved", zap.String("table", table), zap.Int("count", len(plans)))
	return plans, nil
}

func (s *SqlStore) GetTVSubByPackage(ctx context.Context, table string, packageName string) (*models.TVSub, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := validateTableName(table); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}

	query := fmt.Sprintf(
		"SELECT plan_id, package, package_name, amount, profit_margin FROM %s WHERE package = $1",
		sanitizeTableName(table),
	)
	row := s.db.QueryRowContext(ctx, query, packageName)

	var p models.TVSub
	if err := row.Scan(
		&p.ID,
		&p.Package,
		&p.PackageName,
		&p.Amount,
		&p.Profit_Margin,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Warn("Plan not found", zap.String("table", table), zap.String("packageName", packageName))
			return nil, nil
		}
		s.logger.Error("Error scanning plan row", zap.String("table", table), zap.String("packageName", packageName), zap.Error(err))
		return nil, fmt.Errorf("failed to scan %s plan: %w", table, err)
	}
	s.logger.Info("Plan retrieved", zap.String("table", table), zap.String("packageName", packageName))
	return &p, nil
}

// CreatePlan inserts a new plan into specified table
func (s *SqlStore) CreateTvSub(ctx context.Context, table string, plan models.TVSub) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := validateTableName(table); err != nil {
		return 0, fmt.Errorf("invalid table name: %w", err)
	}

	query := fmt.Sprintf(
		`INSERT INTO %s (package, package_name, amount, profit_margin)
		VALUES ($1, $2, $3, $4)
		RETURNING plan_id`,
		sanitizeTableName(table),
	)

	var id int
	err := s.db.QueryRowContext(ctx, query,
		plan.Package,
		plan.PackageName,
		plan.Amount,
		plan.Profit_Margin,
	).Scan(&id)
	if err != nil {
		s.logger.Error("Failed to insert plan", zap.String("table", table), zap.Error(err))
		return 0, fmt.Errorf("insert into %s failed: %w", table, err)
	}

	s.logger.Info("Plan created", zap.String("table", table), zap.Int("planID", id))
	return id, nil
}

// UpdatePlan modifies fields of an existing plan in specified table
func (s *SqlStore) UpdateTvSub(ctx context.Context, table string, planID int, upd models.TVSubUpdate) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := validateTableName(table); err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}

	setClauses := []string{}
	args := []interface{}{}
	argPos := 1

	if upd.Package != nil {
		setClauses = append(setClauses, fmt.Sprintf("package = $%d", argPos))
		args = append(args, *upd.Package)
		argPos++
	}
	if upd.PackageName != nil {
		setClauses = append(setClauses, fmt.Sprintf("package_name = $%d", argPos))
		args = append(args, *upd.PackageName)
		argPos++
	}
	if upd.Amount != nil {
		setClauses = append(setClauses, fmt.Sprintf("amount = $%d", argPos))
		args = append(args, *upd.Amount)
		argPos++
	}
	if upd.Profit_Margin != nil {
		setClauses = append(setClauses, fmt.Sprintf("profit_margin = $%d", argPos))
		args = append(args, *upd.Profit_Margin)
		argPos++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no fields provided for update")
	}

	args = append(args, planID)
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE plan_id = $%d",
		sanitizeTableName(table),
		strings.Join(setClauses, ", "),
		argPos,
	)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		s.logger.Error("Failed to update plan", zap.String("table", table), zap.Error(err))
		return fmt.Errorf("update failed for %s: %w", table, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error("Failed to check affected rows", zap.String("table", table), zap.Error(err))
		return fmt.Errorf("rows affected check failed for %s: %w", table, err)
	}

	if rowsAffected == 0 {
		s.logger.Warn("Plan does not exist", zap.String("table", table), zap.Int("planID", planID))
		return fmt.Errorf("%s plan ID %d does not exist", table, planID)
	}

	s.logger.Info("Plan updated", zap.String("table", table), zap.Int("planID", planID))
	return nil
}

// DeletePlan removes a plan by ID from specified table
func (s *SqlStore) DeleteTvSub(ctx context.Context, table string, planID int) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := validateTableName(table); err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE plan_id = $1",
		sanitizeTableName(table),
	)

	result, err := s.db.ExecContext(ctx, query, planID)
	if err != nil {
		s.logger.Error("Failed to delete plan", zap.String("table", table), zap.Error(err))
		return fmt.Errorf("delete failed for %s: %w", table, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error("Failed to check affected rows", zap.String("table", table), zap.Error(err))
		return fmt.Errorf("rows affected check failed for %s: %w", table, err)
	}

	if rowsAffected == 0 {
		s.logger.Warn("Plan does not exist", zap.String("table", table), zap.Int("planID", planID))
		return fmt.Errorf("%s plan ID %d does not exist", table, planID)
	}

	s.logger.Info("Plan deleted", zap.String("table", table), zap.Int("planID", planID))
	return nil
}

// sanitizeTableName provides additional safety layer (quotes table name)
func sanitizeTableName(table string) string {
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(table, `"`, ""))
}
