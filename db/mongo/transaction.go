package mongo

import (
	"context"
	"time"

	"github.com/aremxyplug-be/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

func (m *mongoStore) GetTransactions(filter map[string]interface{}, page, pageSize int) ([]models.Transaction, int, error) {
	// Define the initial project stage (only include needed fields)
	ctx := context.Background()

	// Filtering
	matchConditions := bson.D{}

	if userID, ok := filter["user_id"].(string); ok && userID != "" {
		matchConditions = append(matchConditions, bson.E{Key: "user_id", Value: userID})
	}

	if product, ok := filter["product"].(string); ok && product != "" {
		matchConditions = append(matchConditions, bson.E{Key: "product", Value: product})
	}

	if searchTerm, ok := filter["search"].(string); ok && searchTerm != "" {
		regex := bson.D{{Key: "$regex", Value: searchTerm}, {Key: "$options", Value: "i"}}
		matchConditions = append(matchConditions, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "description", Value: regex}},
			bson.D{{Key: "product", Value: regex}},
		}})
	}

	dateRange := bson.D{}
	if start, ok := filter["start_date"].(time.Time); ok {
		dateRange = append(dateRange, bson.E{Key: "$gte", Value: start})
	}
	if end, ok := filter["end_date"].(time.Time); ok {
		dateRange = append(dateRange, bson.E{Key: "$lte", Value: end})
	}
	if len(dateRange) > 0 {
		matchConditions = append(matchConditions, bson.E{Key: "created_at", Value: dateRange})
	}

	projectStage := bson.D{{Key: "$project", Value: bson.D{
		{Key: "product", Value: 1}, {Key: "description", Value: 1},
		{Key: "order_id", Value: 1}, {Key: "created_at", Value: 1}, {Key: "status", Value: 1},
	}}}

	basePipeline := mongo.Pipeline{}
	if len(matchConditions) > 0 {
		basePipeline = append(basePipeline, bson.D{{Key: "$match", Value: matchConditions}})
	}
	basePipeline = append(basePipeline, projectStage)

	pipeline := make(mongo.Pipeline, len(basePipeline))
	copy(pipeline, basePipeline)

	collections := []string{"airtime", "data", "transfer", "edu", "tv-sub", "electric-sub"}
	for _, coll := range collections[1:] {
		unionStages := bson.A{}
		for _, stage := range basePipeline {
			unionStages = append(unionStages, stage)
		}

		pipeline = append(pipeline, bson.D{{
			Key: "$unionWith",
			Value: bson.M{
				"coll":     coll,
				"pipeline": unionStages,
			},
		}})
	}

	// Sorting and pagination
	pipeline = append(pipeline,
		bson.D{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}}}},
		bson.D{{Key: "$skip", Value: int64((page - 1) * pageSize)}},
		bson.D{{Key: "$limit", Value: int64(pageSize)}},
	)

	if len(pipeline) > 0 {
		if stage1, err := bson.Marshal(pipeline[0]); err != nil {
			m.logger.Error("Stage 1 marshaling error", zap.Error(err))
		} else {
			m.logger.Info("Stage 1: %x", zap.ByteString("stage 1", stage1))
		}
	}

	if len(pipeline) > len(basePipeline) {
		lastStage := pipeline[len(basePipeline)]
		if stageData, err := bson.Marshal(lastStage); err != nil {
			m.logger.Error("Union stage error", zap.Error(err))
		} else {
			m.logger.Info("Union stage", zap.ByteString("union stage", stageData))
		}
	}

	pipelineDoc := bson.D{{Key: "pipeline", Value: pipeline}}

	if debugJSON, err := bson.MarshalExtJSON(pipelineDoc, false, false); err != nil {
		m.logger.Error("Pipeline marshaling error", zap.Error(err))
	} else {
		m.logger.Info("Pipeline JSON", zap.ByteString("pieline", debugJSON))
	}

	cursor, err := m.col(collections[0]).Aggregate(ctx, pipeline)
	if err != nil {
		m.logger.Error("Aggregation error:", zap.Error(err))
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var results []models.Transaction
	if err := cursor.All(ctx, &results); err != nil {
		m.logger.Error("Cursor error: %v", zap.Error(err))
		return nil, 0, err
	}

	var total int
	for _, coll := range collections {
		count, err := m.col(coll).CountDocuments(ctx, bson.D(matchConditions))
		if err != nil {
			m.logger.Error("CountDocuments error for collection", zap.String("collection", coll), zap.Error(err))
			return nil, 0, err
		}
		total += int(count)
	}

	m.logger.Info("Transactions fetched successfully. Total: %d", zap.Int("total document", total))
	return results, total, nil
}
