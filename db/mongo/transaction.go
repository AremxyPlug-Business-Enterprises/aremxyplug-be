package mongo

import (
	"context"
	"time"

	"github.com/aremxyplug-be/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

func (m *mongoStore) GetTransactions(filter map[string]interface{}, page, pageSize int) (models.TransactionResponse, error) {
	ctx := context.Background()

	matchConditions := bson.D{}
	if userID, ok := filter["user_id"].(string); ok && userID != "" {
		matchConditions = append(matchConditions, bson.E{Key: "user_id", Value: userID})
	}
	if product, ok := filter["product"].(string); ok && product != "" {
		matchConditions = append(matchConditions, bson.E{Key: "product", Value: product})
	}
	if searchTerm, ok := filter["search"].(string); ok && searchTerm != "" {
		regex := bson.M{"$regex": searchTerm, "$options": "i"}
		matchConditions = append(matchConditions, bson.E{Key: "$or", Value: bson.A{
			bson.M{"description": regex},
			bson.M{"product": regex},
		}})
	}
	dateRange := bson.M{}
	if start, ok := filter["start_date"].(time.Time); ok {
		dateRange["$gte"] = start
	}
	if end, ok := filter["end_date"].(time.Time); ok {
		dateRange["$lte"] = end
	}
	if len(dateRange) > 0 {
		matchConditions = append(matchConditions, bson.E{Key: "created_at", Value: dateRange})
	}

	inflowCollections := []string{"deposit-transaction", "point-redeem"}
	outflowCollections := []string{"airtime", "data", "transfer", "edu", "tv-sub", "electric-sub"}

	collectionsToQuery := append(outflowCollections, inflowCollections...)
	baseCollection := collectionsToQuery[0]
	remainingCollections := collectionsToQuery[1:]

	projectStage := bson.D{{Key: "$project", Value: bson.M{
		"product":     1,
		"description": 1,
		"order_id":    1,
		"created_at":  1,
		"status":      1,
		"amount":      1,
	}}}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: matchConditions}},
		projectStage,
		{{Key: "$addFields", Value: bson.M{"flowType": "outflow"}}},
	}

	for _, coll := range remainingCollections {
		flowType := "outflow"
		for _, inflow := range inflowCollections {
			if coll == inflow {
				flowType = "inflow"
				break
			}
		}
		unionStages := bson.A{
			bson.D{{Key: "$match", Value: matchConditions}},
			bson.D{{Key: "$addFields", Value: bson.M{"flowType": flowType}}},
		}
		pipeline = append(pipeline, bson.D{{Key: "$unionWith", Value: bson.M{"coll": coll, "pipeline": unionStages}}})
	}

	pipeline = append(pipeline,
		bson.D{{Key: "$addFields", Value: bson.M{
			"amountDecimal": bson.M{
				"$convert": bson.M{"input": "$amount", "to": "double", "onError": 0, "onNull": 0},
			},
		}}},
		bson.D{{Key: "$facet", Value: bson.M{
			"transactions": bson.A{
				bson.D{{Key: "$sort", Value: bson.M{"created_at": -1}}},
				bson.D{{Key: "$skip", Value: int64((page - 1) * pageSize)}},
				bson.D{{Key: "$limit", Value: int64(pageSize)}},
			},
			"totals": bson.A{
				bson.D{{Key: "$group", Value: bson.M{
					"_id":          nil,
					"totalCount":   bson.M{"$sum": 1},
					"totalInflow":  bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$flowType", "inflow"}}, "$amountDecimal", 0}}},
					"totalOutflow": bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$flowType", "outflow"}}, "$amountDecimal", 0}}},
				}}},
			},
		}}},
	)

	cursor, err := m.col(baseCollection).Aggregate(ctx, pipeline)
	if err != nil {
		m.logger.Error("Aggregation error:", zap.Error(err))
		return models.TransactionResponse{}, err
	}
	defer cursor.Close(ctx)

	var aggResult []struct {
		Transactions []models.TransactionItem   `bson:"transactions"`
		Totals       []models.TotalsAggregation `bson:"totals"`
	}
	if err := cursor.All(ctx, &aggResult); err != nil || len(aggResult) == 0 {
		return models.TransactionResponse{}, err
	}

	res := models.TransactionResponse{
		Transactions: aggResult[0].Transactions,
		TotalCount:   0,
		TotalInflow:  0,
		TotalOutflow: 0,
	}
	if len(aggResult[0].Totals) > 0 {
		res.TotalCount = aggResult[0].Totals[0].TotalCount
		res.TotalInflow = aggResult[0].Totals[0].TotalInflow
		res.TotalOutflow = aggResult[0].Totals[0].TotalOutflow
	}

	m.logger.Info(" Fetched transactions successfully",
		zap.Int("total_count", res.TotalCount),
		zap.Float64("total_inflow", res.TotalInflow),
		zap.Float64("total_outflow", res.TotalOutflow),
	)

	return res, nil
}
