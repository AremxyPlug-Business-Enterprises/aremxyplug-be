package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
)

// GET: Retrieve a record by ID
func (s *SqlStore) GetEduRecord(ctx context.Context, id int) (models.EduRecord, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	query := "SELECT id, amount, name, profit_margin FROM edu_pins WHERE id = $1"
	row := s.db.QueryRowContext(ctx, query, id)

	record := models.EduRecord{}
	err := row.Scan(&record.ID, &record.Amount, &record.Name, &record.Profit_Margin)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.Warn("No record found", zap.Int("ID", id))
			return models.EduRecord{}, fmt.Errorf("no record found with ID %d", id)
		}
		s.logger.Error("Failed to scan record row", zap.Error(err))
		return models.EduRecord{}, fmt.Errorf("failed to scan record row: %v", err)
	}

	s.logger.Info("Record retrieved successfully", zap.Int("ID", record.ID))
	return record, nil
}

// UPDATE: Update amount and name by ID
func (s *SqlStore) UpdateEduRecord(ctx context.Context, id int, edu models.EduRecord) error {
	if ctx == nil {
		ctx = context.Background()
	}
	query := "UPDATE edu_pins SET amount = $1, name = $2 WHERE id = $3"
	result, err := s.db.ExecContext(ctx, query, edu.Amount, edu.Name, id)
	if err != nil {
		s.logger.Error("Failed to update record", zap.Int("ID", id), zap.Error(err))
		return fmt.Errorf("update failed for record ID %d: %v", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error("Failed to check affected rows", zap.Int("ID", id), zap.Error(err))
		return fmt.Errorf("rows affected check failed for record ID %d: %v", id, err)
	}

	if rowsAffected == 0 {
		s.logger.Warn("No record updated", zap.Int("ID", id))
		return fmt.Errorf("no record updated for ID %d", id)
	}

	s.logger.Info("Record updated successfully", zap.Int("ID", id))
	return nil
}

// DELETE: Delete a record by ID
func (s *SqlStore) DeleteEduRecord(ctx context.Context, id int) error {
	if ctx == nil {
		ctx = context.Background()
	}
	query := "DELETE FROM edu_pins WHERE id = $1"
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		s.logger.Error("Failed to delete record", zap.Int("ID", id), zap.Error(err))
		return fmt.Errorf("delete failed for record ID %d: %v", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error("Failed to check affected rows", zap.Int("ID", id), zap.Error(err))
		return fmt.Errorf("rows affected check failed for record ID %d: %v", id, err)
	}

	if rowsAffected == 0 {
		s.logger.Warn("No record deleted", zap.Int("ID", id))
		return fmt.Errorf("no record deleted for ID %d", id)
	}

	s.logger.Info("Record deleted successfully", zap.Int("ID", id))
	return nil
}
