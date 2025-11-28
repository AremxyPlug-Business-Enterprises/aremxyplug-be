package mongo

import (
	"context"
	"errors"
	"time"

	"github.com/aremxyplug-be/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrAlreadyCompleted = errors.New("task already completed")

func (m *mongoStore) GetProgress(userID string, task models.TaskType) (*models.ProgressDoc, error) {
	ctx := context.Background()
	var doc models.ProgressDoc
	err := m.col(tasksColl).FindOne(ctx, bson.M{"user_id": userID, "task_code": task}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &doc, nil
}

// IncrementCumulative atomically increments a cumulative task and returns the updated doc.
func (m *mongoStore) IncrementCumulative(userID string, def models.TaskDef, delta int64, txID string) (*models.ProgressDoc, error) {

	ctx := context.Background()

	now := time.Now().UTC()
	filter := bson.M{"user_id": userID, "task_code": def.Code}
	update := bson.M{
		"$inc": bson.M{"progress": delta},
		"$set": bson.M{
			"type":       "cumulative",
			"target":     def.Target,
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"created_at": now,
			"completed":  false,
			"progress":   0, // $inc will add delta
		},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var doc models.ProgressDoc
	if err := m.col(tasksColl).FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// SetProgressForOneTime sets progress for one-time tasks (only increases if higher), returns doc.
func (m *mongoStore) SetProgressForOneTime(userID string, def models.TaskDef, value int64, txID string) (*models.ProgressDoc, error) {

	ctx := context.Background()

	now := time.Now().UTC()
	filter := bson.M{"user_id": userID, "task_code": def.Code}
	update := bson.M{
		"$setOnInsert": bson.M{
			"created_at": now,
			"completed":  false,
			"progress":   0,
			"type":       "one_time",
			"target":     def.Target,
		},
		"$set": bson.M{
			"updated_at": now,
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var doc models.ProgressDoc
	if err := m.col(tasksColl).FindOneAndUpdate(ctx, filter, update, opts).Decode(&doc); err != nil {
		return nil, err
	}

	// only update progress if provided value is greater
	if doc.Progress < value {
		_, err := m.col(tasksColl).UpdateOne(ctx, bson.M{"_id": doc.ID}, bson.M{"$set": bson.M{"progress": value, "updated_at": now}})
		if err != nil {
			return nil, err
		}
		doc.Progress = value
		doc.UpdatedAt = now
	}
	return &doc, nil
}

// TryMarkCompleted tries to atomically set completed=true; returns true if it changed.
func (m *mongoStore) TryMarkCompleted(userID string, task models.TaskType) (bool, error) {

	ctx := context.Background()

	now := time.Now().UTC()
	filter := bson.M{"user_id": userID, "task_code": task, "completed": false}
	update := bson.M{"$set": bson.M{"completed": true, "completed_at": now, "updated_at": now}}
	res, err := m.col(tasksColl).UpdateOne(ctx, filter, update)
	if err != nil {
		return false, err
	}
	if res.ModifiedCount == 1 {
		return true, nil
	}
	// If ModifiedCount == 0, the document either didn't exist or already completed.
	// Check existence to give useful error:
	var doc models.ProgressDoc
	err = m.col(tasksColl).FindOne(ctx, bson.M{"user_id": userID, "task_code": task}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}
	if doc.Completed {
		return false, ErrAlreadyCompleted
	}
	return false, nil
}
