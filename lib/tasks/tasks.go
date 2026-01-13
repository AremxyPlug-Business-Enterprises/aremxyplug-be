package tasks

import (
	"fmt"
	"strconv"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
)

// Service is the TaskProgress service.
type Service struct {
	store db.TaskStore
	// Note: This package does NOT publish events directly (to avoid circular imports).
	// The caller (worker/processor) receives the result and may publish a task.completed event via events.Emitter.
}

// NewService constructs the service.
func NewService(store db.DataStore) *Service {
	return &Service{store: store}
}

// ApplyEvent processes a normalized TaskEvent.
// Returns (completed bool, taskCode TaskType, progressDoc *ProgressDoc, err error).
// IMPORTANT: this function does NOT publish task.completed events; caller should handle publishing.
func (s *Service) ApplyEvent(userID string, te models.TaskEvent) (bool, models.TaskType, *models.ProgressDoc, error) {
	def, ok := models.TaskRegistry[te.Task]
	if !ok {
		return false, "", nil, fmt.Errorf("task def not found: %s", te.Task)
	}

	// One-time task
	if def.OneTime {
		// For one-time we treat Amount as the observed value (e.g. points redeemed or wallet fund).
		doc, err := s.store.SetProgressForOneTime(userID, def, te.Amount, te.TxID)
		if err != nil {
			return false, def.Code, nil, err
		}
		// if progress >= target and not completed -> try mark completed
		if doc.Progress >= def.Target && !doc.Completed {
			ok, err := s.store.TryMarkCompleted(userID, def.Code)
			if err != nil {
				return false, def.Code, doc, err
			}
			if ok {
				return true, def.Code, doc, nil
			}
		}
		return false, def.Code, doc, nil
	}

	// Cumulative task
	if !def.OneTime {
		doc, err := s.store.IncrementCumulative(userID, def, te.Amount, te.TxID)
		if err != nil {
			return false, def.Code, nil, err
		}
		if doc.Progress >= def.Target && !doc.Completed {
			ok, err := s.store.TryMarkCompleted(userID, def.Code)
			if err != nil {
				return false, def.Code, doc, err
			}
			if ok {
				return true, def.Code, doc, nil
			}
		}
		return false, def.Code, doc, nil
	}

	return false, def.Code, nil, fmt.Errorf("unhandled task type")
}

// Helper: AmountFromString converts a simple numeric string to int64 (best-effort).
func AmountFromString(s string) int64 {
	if s == "" {
		return 0
	}
	// try parse integer-like first
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return v
	}
	// try float then floor
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int64(f)
	}
	return 0
}
