package mongo

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/aremxyplug-be/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

func (m *mongoStore) GetTransactions(filter map[string]interface{}, page, pageSize int) (models.TransactionResponse, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 50
	}

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

	// Collection setup
	inflowCollections := []string{"deposit-transaction", "point-redeem"}
	outflowCollections := []string{"airtime", "data", "transfer", "edu", "tv-sub", "electric-sub"}
	collectionsToQuery := append(outflowCollections, inflowCollections...)
	baseCollection := collectionsToQuery[0]
	remainingCollections := collectionsToQuery[1:]

	// Validate collections
	for _, coll := range collectionsToQuery {
		if m.col(coll) == nil {
			return models.TransactionResponse{},
				errors.New("collection " + coll + " does not exist")
		}
	}

	// Projection
	projectStage := bson.D{{Key: "$project", Value: bson.M{
		"transaction_product": 1, "transaction_description": 1, "order_id": 1,
		"created_at": 1, "status": 1, "amount": 1,
	}}}

	// Base flow type
	baseFlowType := "outflow"
	if slices.Contains(inflowCollections, baseCollection) {
		baseFlowType = "inflow"
	}

	// Base pipeline
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: matchConditions}},
		projectStage,
		{{Key: "$addFields", Value: bson.M{"flowType": baseFlowType}}},
	}

	// Union with other collections
	for _, coll := range remainingCollections {
		flowType := "outflow"
		if slices.Contains(inflowCollections, coll) {
			flowType = "inflow"
		}
		unionStages := bson.A{
			bson.D{{Key: "$match", Value: matchConditions}},
			projectStage,
			bson.D{{Key: "$addFields", Value: bson.M{"flowType": flowType}}},
		}
		pipeline = append(pipeline, bson.D{{
			Key:   "$unionWith",
			Value: bson.M{"coll": coll, "pipeline": unionStages},
		}})
	}

	// Safe amount conversion
	pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
		"amountDecimal": bson.M{
			"$cond": bson.A{
				bson.M{"$eq": bson.A{bson.M{"$type": "$amount"}, "double"}},
				"$amount",
				bson.M{"$convert": bson.M{
					"input": "$amount", "to": "double",
					"onError": 0, "onNull": 0,
				}},
			},
		},
	}}})

	// Get data
	dataPipeline := append(pipeline,
		bson.D{{Key: "$sort", Value: bson.M{"created_at": -1}}},
		bson.D{{Key: "$skip", Value: int64((page - 1) * pageSize)}},
		bson.D{{Key: "$limit", Value: int64(pageSize)}},
	)

	cursor, err := m.col(baseCollection).Aggregate(ctx, dataPipeline)
	if err != nil {
		m.logger.Error("Data aggregation failed", zap.Error(err))
		return models.TransactionResponse{}, err
	}
	defer cursor.Close(ctx)

	var transactions []models.TransactionItem
	if err := cursor.All(ctx, &transactions); err != nil {
		return models.TransactionResponse{}, err
	}

	// Get totals
	totalsPipeline := append(pipeline,
		bson.D{{Key: "$group", Value: bson.M{
			"_id":        nil,
			"totalCount": bson.M{"$sum": 1},
			"totalInflow": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$eq": bson.A{"$flowType", "inflow"}},
				"$amountDecimal", 0,
			}}},
			"totalOutflow": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$eq": bson.A{"$flowType", "outflow"}},
				"$amountDecimal", 0,
			}}},
		}}},
	)

	totalsCursor, err := m.col(baseCollection).Aggregate(ctx, totalsPipeline)
	if err != nil {
		m.logger.Error("Totals aggregation failed", zap.Error(err))
		return models.TransactionResponse{}, err
	}
	defer totalsCursor.Close(ctx)

	var totals []models.TotalsAggregation
	if err := totalsCursor.All(ctx, &totals); err != nil {
		return models.TransactionResponse{}, err
	}

	// Prepare response
	res := models.TransactionResponse{
		Transactions: transactions,
	}
	if len(totals) > 0 {
		res.TotalCount = totals[0].TotalCount
		res.TotalInflow = totals[0].TotalInflow
		res.TotalOutflow = totals[0].TotalOutflow
	}

	m.logger.Info("Transactions fetched successfully",
		zap.Int("total", res.TotalCount),
		zap.Int("page", page),
		zap.Int("pageSize", pageSize),
		zap.Any("filter", filter),
	)

	return res, nil
}
