package sqlstore

import (
	"context"
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
func (s *SqlStore) GetTVSubs(table string) ([]models.TVSub, error) {
	ctx := context.Background()

	if err := validateTableName(table); err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}

	query := fmt.Sprintf(
		"SELECT plan_id, package, package_name, amount FROM %s",
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

// CreatePlan inserts a new plan into specified table
func (s *SqlStore) CreateTvSub(table string, plan models.TVSub) (int, error) {
	ctx := context.Background()

	if err := validateTableName(table); err != nil {
		return 0, fmt.Errorf("invalid table name: %w", err)
	}

	query := fmt.Sprintf(
		`INSERT INTO %s (package, package_name, amount)
		VALUES (?, ?, ?)`,
		sanitizeTableName(table),
	)

	result, err := s.db.ExecContext(ctx, query,
		plan.Package,
		plan.PackageName,
		plan.Amount,
	)
	if err != nil {
		s.logger.Error("Failed to insert plan", zap.String("table", table), zap.Error(err))
		return 0, fmt.Errorf("insert into %s failed: %w", table, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		s.logger.Error("Failed to get last insert ID", zap.String("table", table), zap.Error(err))
		return 0, fmt.Errorf("failed to get insert ID from %s: %w", table, err)
	}

	s.logger.Info("Plan created", zap.String("table", table), zap.Int64("planID", id))
	return int(id), nil
}

// UpdatePlan modifies fields of an existing plan in specified table
func (s *SqlStore) UpdateTvSub(table string, planID int, upd models.TVSubUpdate) error {
	ctx := context.Background()

	if err := validateTableName(table); err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}

	setClauses := []string{}
	args := []interface{}{}

	if upd.Package != nil {
		setClauses = append(setClauses, "package = ?")
		args = append(args, *upd.Package)
	}
	if upd.PackageName != nil {
		setClauses = append(setClauses, "package_name = ?")
		args = append(args, *upd.PackageName)
	}
	if upd.Amount != nil {
		setClauses = append(setClauses, "amount = ?")
		args = append(args, *upd.Amount)
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no fields provided for update")
	}

	args = append(args, planID)
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE plan_id = ?",
		sanitizeTableName(table),
		strings.Join(setClauses, ", "),
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
func (s *SqlStore) DeleteTvSub(table string, planID int) error {
	ctx := context.Background()

	if err := validateTableName(table); err != nil {
		return fmt.Errorf("invalid table name: %w", err)
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE plan_id = ?",
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
	return fmt.Sprintf("`%s`", strings.ReplaceAll(table, "`", ""))
}
