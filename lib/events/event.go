package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/redis"
	"github.com/aremxyplug-be/lib/tasks"
)

// ----------------------------------------------------------------------
// BUSINESS EVENT STRUCT (this is the incoming raw event)
// ----------------------------------------------------------------------

type Event struct {
	ID            string                 `json:"id" bson:"_id,omitempty"`
	Version       string                 `json:"version"`
	Type          string                 `json:"type"`
	UserID        string                 `json:"user_id"`
	Amount        string                 `json:"amount,omitempty"`
	TxID          string                 `json:"tx_id,omitempty"`
	PointsAwarded int64                  `json:"points_awarded,omitempty"`
	Meta          map[string]interface{} `json:"meta,omitempty"`
	TS            time.Time              `json:"ts"`
	Published     bool                   `json:"published"`
}

type Processor struct {
	redis   *redis.RedisConn
	TaskSvc *tasks.Service
	Logger  *zap.Logger
}

func NewProcessor(redis *redis.RedisConn, taskSvc *tasks.Service, logger *zap.Logger) *Processor {
	return &Processor{
		redis:   redis,
		TaskSvc: taskSvc,
		Logger:  logger,
	}
}

func (p *Processor) ProcessEvent(ctx context.Context, ev *Event) error {
	if ev == nil {
		p.Logger.Error("nil event passed to ProcessEvent")
		return fmt.Errorf("nil event")
	}

	p.Logger.Debug("processing incoming event",
		zap.String("id", ev.ID),
		zap.String("type", ev.Type),
		zap.String("user", ev.UserID),
		zap.Time("ts", ev.TS),
	)

	// 1. Convert business event → TaskEvent
	te := mapEventToTaskEvent(ev)
	if te == nil {
		p.Logger.Debug("event is not task-related; skipping", zap.String("type", ev.Type), zap.String("user", ev.UserID))
		return nil // not a task-related event
	}
	p.Logger.Debug("mapped event to task event", zap.Any("task_event", te), zap.String("user", ev.UserID))

	// 2. Apply task logic
	completed, taskCode, progressDoc, err := p.TaskSvc.ApplyEvent(ev.UserID, *te)
	if err != nil {
		p.Logger.Error("task apply failed",
			zap.Error(err),
			zap.String("user", ev.UserID),
			zap.String("task", string(taskCode)),
			zap.Any("task_event", te),
		)
		return err
	}
	p.Logger.Info("task apply succeeded",
		zap.String("user", ev.UserID),
		zap.String("task", string(taskCode)),
		zap.Bool("completed", completed),
		zap.Any("progress_doc", progressDoc),
	)

	// 3. Publish real-time progress update
	progressPayload := map[string]interface{}{
		"type":      "task.progress",
		"task":      taskCode,
		"progress":  nil,
		"target":    nil,
		"completed": false,
	}

	if progressDoc != nil {
		progressPayload["progress"] = progressDoc.Progress
		progressPayload["target"] = progressDoc.Target
		progressPayload["completed"] = progressDoc.Completed
	}

	p.Logger.Debug("publishing user progress update", zap.String("user", ev.UserID), zap.Any("payload", progressPayload))
	if err := p.redis.PublishUserEvent(ctx, ev.UserID, progressPayload); err != nil {
		p.Logger.Warn("failed to publish user progress",
			zap.Error(err),
			zap.String("user", ev.UserID),
		)
	} else {
		p.Logger.Debug("published user progress", zap.String("user", ev.UserID))
	}

	// 4. Emit task.completed event
	if completed {
		completedEvent := &Event{
			Version:   "1",
			Type:      "task.completed",
			UserID:    ev.UserID,
			Meta:      map[string]interface{}{"task_code": string(taskCode)},
			TS:        time.Now().UTC(),
			Published: false,
		}

		p.Logger.Debug("saving and publishing task.completed event", zap.String("user", ev.UserID), zap.String("task", string(taskCode)))
		if err := p.redis.SaveAndPublish(ctx, completedEvent); err != nil {
			p.Logger.Error("failed to publish task.completed",
				zap.Error(err),
				zap.String("user", ev.UserID),
				zap.String("task", string(taskCode)),
			)
		} else {
			p.Logger.Info("task completed",
				zap.String("user", ev.UserID),
				zap.String("task", string(taskCode)),
			)
		}
	}

	return nil
}

func mapEventToTaskEvent(ev *Event) *models.TaskEvent {
	switch ev.Type {

	case "signup.completed":
		return &models.TaskEvent{
			Task:   models.TaskSignup,
			Amount: 1,
		}

	case "kyc.completed":
		return &models.TaskEvent{
			Task:   models.TaskKYC,
			Amount: 1,
		}

	case "wallet.funded":
		amt := tasks.AmountFromString(ev.Amount)
		return &models.TaskEvent{
			Task:   models.TaskFundWallet,
			Amount: amt,
			TxID:   ev.TxID,
		}

	case "transaction.completed", "purchase.completed":
		amt := tasks.AmountFromString(ev.Amount)
		return &models.TaskEvent{
			Task:   models.TaskTransactionVolume,
			Amount: amt,
			TxID:   ev.TxID,
		}

	case "points.redeemed":
		return &models.TaskEvent{
			Task:   models.TaskPointRedeem,
			Amount: ev.PointsAwarded,
			TxID:   ev.TxID,
		}

	default:
		return nil
	}
}

func MarshalEvent(ev *Event) ([]byte, error) {
	return json.Marshal(ev)
}
