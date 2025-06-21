package sqlstore

import (
	"database/sql"
	"fmt"

	"github.com/aremxyplug-be/db/models"
	"go.uber.org/zap"
)

// GET: Retrieve a record by ID
func (s *SqlStore) GetEduRecord(id int) (models.EduRecord, error) {
	query := "SELECT id, amount, name FROM your_table_name WHERE id = ?"
	row := s.db.QueryRow(query, id)

	record := models.EduRecord{}
	err := row.Scan(&record.ID, &record.Amount, &record.Name)
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
func (s *SqlStore) UpdateEduRecord(id int, edu models.EduRecord) error {
	query := "UPDATE your_table_name SET amount = ?, name = ? WHERE id = ?"
	result, err := s.db.Exec(query, edu.Amount, edu.Name, id)
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
func (s *SqlStore) DeleteEduRecord(id int) error {
	query := "DELETE FROM your_table_name WHERE id = ?"
	result, err := s.db.Exec(query, id)
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
