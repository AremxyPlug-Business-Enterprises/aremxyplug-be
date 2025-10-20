package mongo

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
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
	inflowCollections := []string{depositColl, pointRedeemColl}
	outflowCollections := []string{airColl, dataColl, transferColl, eduColl, tvColl, electricColl}
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
	projectStage := bson.D{
		{Key: "$project", Value: bson.D{
			{Key: "product", Value: "$transaction_product"},
			{Key: "description", Value: "$transaction_description"},
			{Key: "order_id", Value: "$order_id"},
			{Key: "created_at", Value: "$created_at"},
			{Key: "status", Value: "$status"},
			{Key: "amount", Value: "$amount"}, // keep raw here
		}},
	}

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
		// unionStages := bson.A{
		// 	bson.D{{Key: "$match", Value: matchConditions}},
		// 	projectStage,
		// 	bson.D{{Key: "$addFields", Value: bson.M{"flowType": flowType}}},
		// }
		var unionStages bson.A
		if coll == pointRedeemColl {
			// Special case for pointRedeemColl
			unionStages = bson.A{
				bson.D{{Key: "$match", Value: matchConditions}},
				bson.D{{Key: "$project", Value: bson.D{
					{Key: "product", Value: "$transaction_product"},
					{Key: "description", Value: "$transaction_description"},
					{Key: "order_id", Value: "$order_id"},
					{Key: "created_at", Value: "$created_at"},
					{Key: "status", Value: "$status"},
					{Key: "amount", Value: "$amount_redeemed"}, // map amount_redeemed → amount
				}}},
				bson.D{{Key: "$addFields", Value: bson.M{"flowType": flowType}}},
			}
		} else {
			// Normal collections
			unionStages = bson.A{
				bson.D{{Key: "$match", Value: matchConditions}},
				projectStage,
				bson.D{{Key: "$addFields", Value: bson.M{"flowType": flowType}}},
			}
		}

		pipeline = append(pipeline, bson.D{{
			Key:   "$unionWith",
			Value: bson.M{"coll": coll, "pipeline": unionStages},
		}})
	}

	// Safe amount conversion
	pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
		"amountDecimal": bson.M{
			"$convert": bson.M{
				"input":   "$amount",
				"to":      "double",
				"onError": 0,
				"onNull":  0,
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

	// New: Status-based aggregation pipeline - ensure all statuses are returned
	statusPipeline := append(pipeline,
		bson.D{{Key: "$group", Value: bson.M{
			"_id":    "$status",
			"value":  bson.M{"$sum": "$amountDecimal"},
			"volume": bson.M{"$sum": 1},
		}}},
		bson.D{{Key: "$project", Value: bson.M{
			"status": "$_id",
			"value":  1,
			"volume": 1,
			"_id":    0,
		}}},
	)

	statusCursor, err := m.col(baseCollection).Aggregate(ctx, statusPipeline)
	if err != nil {
		m.logger.Error("Status aggregation failed", zap.Error(err))
		return models.TransactionResponse{}, err
	}
	defer statusCursor.Close(ctx)

	var statusResults []struct {
		Status string  `bson:"status"`
		Value  float64 `bson:"value"`
		Volume int     `bson:"volume"`
	}
	if err := statusCursor.All(ctx, &statusResults); err != nil {
		return models.TransactionResponse{}, err
	}

	// Prepare status metrics - ensure all statuses are present with zero values if missing
	expectedStatuses := []string{"success", "failed", "pending", "refunded"}
	statusMetrics := make(map[string]models.StatusMetrics)

	// Initialize all expected statuses with zero values
	for _, status := range expectedStatuses {
		statusMetrics[status] = models.StatusMetrics{
			Value:  0,
			Volume: 0,
		}
	}

	// Update with actual data from aggregation
	for _, result := range statusResults {
		statusMetrics[result.Status] = models.StatusMetrics{
			Value:  result.Value,
			Volume: result.Volume,
		}
	}

	// Totals pipeline: use all filters except status, and match status in ["success", "pending"]
	totalsMatch := bson.D{}
	for _, cond := range matchConditions {
		if cond.Key != "status" {
			totalsMatch = append(totalsMatch, cond)
		}
	}
	/*
		totalsMatch = append(totalsMatch,
			bson.E{Key: "status", Value: bson.M{"$in": bson.A{"success", "pending"}}},
		)
	*/
	totalsPipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: totalsMatch}},
		projectStage,
		bson.D{{Key: "$addFields", Value: bson.M{"flowType": baseFlowType}}},
	}

	for _, coll := range remainingCollections {
		flowType := "outflow"
		if slices.Contains(inflowCollections, coll) {
			flowType = "inflow"
		}
		// unionStages := mongo.Pipeline{
		// 	bson.D{{Key: "$match", Value: totalsMatch}},
		// 	projectStage,
		// 	bson.D{{Key: "$addFields", Value: bson.M{"flowType": flowType}}},
		// }
		var unionStages mongo.Pipeline
		if coll == pointRedeemColl {
			unionStages = mongo.Pipeline{
				bson.D{{Key: "$match", Value: totalsMatch}},
				bson.D{{Key: "$project", Value: bson.D{
					{Key: "product", Value: "$transaction_product"},
					{Key: "description", Value: "$transaction_description"},
					{Key: "order_id", Value: "$order_id"},
					{Key: "created_at", Value: "$created_at"},
					{Key: "status", Value: "$status"},
					{Key: "amount", Value: "$amount_redeemed"}, // remap for totals too
				}}},
				bson.D{{Key: "$addFields", Value: bson.M{"flowType": flowType}}},
			}
		} else {
			unionStages = mongo.Pipeline{
				bson.D{{Key: "$match", Value: totalsMatch}},
				projectStage,
				bson.D{{Key: "$addFields", Value: bson.M{"flowType": flowType}}},
			}
		}

		totalsPipeline = append(totalsPipeline,
			bson.D{{Key: "$unionWith", Value: bson.M{"coll": coll, "pipeline": unionStages}}},
		)
	}

	totalsPipeline = append(totalsPipeline,
		bson.D{{Key: "$addFields", Value: bson.M{
			"amountDecimal": bson.M{
				"$convert": bson.M{
					"input":   "$amount",
					"to":      "double",
					"onError": 0,
					"onNull":  0,
				},
			},
		}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":        nil,
			"totalCount": bson.M{"$sum": 1},
			"totalInflow": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$and": bson.A{
					bson.M{"$eq": bson.A{"$flowType", "inflow"}},
					bson.M{"$in": bson.A{"$status", bson.A{"success", "pending"}}},
				}},
				"$amountDecimal", 0,
			}}},
			"totalOutflow": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$and": bson.A{
					bson.M{"$eq": bson.A{"$flowType", "outflow"}},
					bson.M{"$in": bson.A{"$status", bson.A{"success", "pending"}}},
				}},
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
		Transactions:  transactions,
		StatusMetrics: statusMetrics,
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
		zap.Any("statusMetrics", statusMetrics),
	)

	return res, nil
}

func (m *mongoStore) GetSalesSummary(category string, filter map[string]interface{}, page int) (models.SalesSummary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Build match conditions
	matchConditions := bson.D{
		bson.E{Key: "status", Value: "success"},
	}
	if userID, ok := filter["user_id"].(string); ok && userID != "" {
		matchConditions = append(matchConditions, bson.E{Key: "user_id", Value: userID})
	}
	if start, ok := filter["start_date"].(time.Time); ok {
		matchConditions = append(matchConditions, bson.E{Key: "created_at", Value: bson.M{"$gte": start}})
	}
	if end, ok := filter["end_date"].(time.Time); ok {
		matchConditions = append(matchConditions, bson.E{Key: "created_at", Value: bson.M{"$lte": end}})
	}

	// Define collections based on category
	collection := ""
	switch category {
	case "airtime":
		collection = airColl
	case "data":
		collection = dataColl
	case "edu":
		collection = eduColl
	case "tv":
		collection = tvColl
	case "electric":
		collection = electricColl
	default:
		return models.SalesSummary{}, errors.New("invalid category")
	}

	pipeline := mongo.Pipeline{}
	if len(matchConditions) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: matchConditions}})
	}

	// Common stages
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
		"product": "$transaction_description", // Use transaction_description for all
	}}})

	// Category-specific quantity handling
	switch {
	case category == "data":
		// Split the transaction_description into words
		pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
			"words": bson.M{"$split": bson.A{"$transaction_description", " "}},
		}}})

		// Filter out words that contain numbers followed by GB/MB
		pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
			"filteredWords": bson.M{
				"$filter": bson.M{
					"input": "$words",
					"as":    "word",
					"cond": bson.M{
						"$not": bson.M{
							"$regexMatch": bson.M{
								"input":   "$$word",
								"regex":   "^[0-9.]+(GB|MB|gb|mb)?$",
								"options": "i",
							},
						},
					},
				},
			},
		}}})

		// Reconstruct the product type from filtered words
		pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
			"productType": bson.M{
				"$reduce": bson.M{
					"input":        "$filteredWords",
					"initialValue": "",
					"in": bson.M{
						"$cond": bson.A{
							bson.M{"$eq": bson.A{"$$value", ""}},
							"$$this",
							bson.M{"$concat": bson.A{"$$value", " ", "$$this"}},
						},
					},
				},
			},
		}}})

		// Extract numeric data value from product string
		pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
			"dataInfo": bson.M{
				"$regexFind": bson.M{
					"input":   "$transaction_description",
					"regex":   "([0-9.]+)\\s*(GB|MB|gb|mb)",
					"options": "i",
				},
			},
		}}})

		// Extract value and unit with error handling
		pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
			"dataValue": bson.M{
				"$cond": bson.A{
					bson.M{"$ne": bson.A{"$dataInfo", nil}},
					bson.M{"$toDouble": bson.M{
						"$arrayElemAt": bson.A{"$dataInfo.captures", 0},
					}},
					0, // Default value if no match
				},
			},
			"dataUnit": bson.M{
				"$cond": bson.A{
					bson.M{"$ne": bson.A{"$dataInfo", nil}},
					bson.M{"$toLower": bson.M{
						"$arrayElemAt": bson.A{"$dataInfo.captures", 1},
					}},
					"gb", // Default unit if no match
				},
			},
		}}})

		// Convert all quantities to GB
		pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
			"quantityInGB": bson.M{
				"$cond": bson.A{
					bson.M{"$eq": bson.A{"$dataUnit", "mb"}},
					bson.M{"$divide": bson.A{"$dataValue", 1000}},
					"$dataValue",
				},
			},
		}}})

		// Group by the extracted product type
		groupStage := bson.D{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"product_type": "$productType",
			},
			"quantity":        bson.M{"$sum": "$quantityInGB"},
			"totalAmount":     bson.M{"$sum": "$amountDecimal"},
			"lastTransaction": bson.M{"$max": "$created_at"},
			"count":           bson.M{"$sum": 1},
		}}}
		pipeline = append(pipeline, groupStage)

		// Project to final format
		projectStage := bson.D{{Key: "$project", Value: bson.M{
			"product":     "$_id.product_type",
			"quantity":    1,
			"totalAmount": 1,
			"created_at":  "$lastTransaction",
			"count":       1,
			"_id":         0,
		}}}
		pipeline = append(pipeline, projectStage)

		// Clean up temporary fields
		pipeline = append(pipeline, bson.D{{Key: "$project", Value: bson.M{
			"words":         0,
			"filteredWords": 0,
		}}})
	case collection == "edu":
		// Use existing quantity field for education
		pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
			"quantity": "$quantity",
		}}})

	default:
		// Default to document count as quantity
		pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{
			"quantity": 1,
		}}})
	}

	// Only apply generic grouping for non-data categories
	if category != "data" {
		// Group by product
		groupStage := bson.D{{Key: "$group", Value: bson.M{
			"_id":             "$product",
			"quantity":        bson.M{"$sum": "$quantity"},
			"totalAmount":     bson.M{"$sum": "$amountDecimal"},
			"lastTransaction": bson.M{"$max": "$created_at"},
		}}}
		pipeline = append(pipeline, groupStage)

		// Project to final format
		projectStage := bson.D{{Key: "$project", Value: bson.M{
			"product":     "$_id",
			"quantity":    1,
			"totalAmount": 1,
			"created_at":  "$lastTransaction",
			"_id":         0,
		}}}
		pipeline = append(pipeline, projectStage)
	}

	// Execute aggregation
	cursor, err := m.col(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return models.SalesSummary{}, fmt.Errorf("aggregation error in %s: %w", collection, err)
	}
	defer cursor.Close(ctx)

	var results []models.SalesSummaryItem
	if err := cursor.All(ctx, &results); err != nil {

		return models.SalesSummary{}, err
	}

	// Use aggregation results directly
	finalSummary := results

	totalProduct := len(finalSummary)
	//totalQuantity := 0.0
	totalAmount := 0.0
	totalOutflow := 0.0
	for _, item := range finalSummary {
		//totalQuantity += item.Quantity
		totalAmount += item.TotalAmount
		totalOutflow += item.TotalAmount
	}
	// Sort by created_at DESC (newest first)
	sort.Slice(finalSummary, func(i, j int) bool {
		return finalSummary[i].CreatedAt.After(finalSummary[j].CreatedAt)
	})

	// Paginate results
	totalItems := len(finalSummary)

	totalQuantity, err := m.col(collection).CountDocuments(ctx, matchConditions)
	if err != nil {
		return models.SalesSummary{}, fmt.Errorf("count error in %s: %w", collection, err)
	}

	return models.SalesSummary{
		Summary:       finalSummary,
		TotalCount:    totalItems,             // Total distinct products
		TotalProduct:  totalProduct,           // <-- Add this field to your model
		TotalQuantity: float64(totalQuantity), // <-- Add this field to your model
		TotalAmount:   totalAmount,            // <-- Add this field to your model
	}, nil

}

func (m *mongoStore) GetSalesOverview(filter map[string]interface{}) (models.SalesOverview, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	categories := map[string]string{
		"airtime":      airColl,
		"data":         dataColl,
		"edu":          eduColl,
		"tv-sub":       tvColl,
		"electric-sub": electricColl,
	}

	overview := models.SalesOverview{
		Categories: make([]models.SalesOverviewCategory, 0),
	}

	var overallTotalAmount float64
	var overallTotalProduct int
	var overallTotalQuantity int

	for cat, coll := range categories {
		// Build match conditions
		matchConditions := bson.D{
			bson.E{Key: "status", Value: "success"},
		}
		if userID, ok := filter["user_id"].(string); ok && userID != "" {
			matchConditions = append(matchConditions, bson.E{Key: "user_id", Value: userID})
		}
		if start, ok := filter["start_date"].(time.Time); ok {
			matchConditions = append(matchConditions, bson.E{Key: "created_at", Value: bson.M{"$gte": start}})
		}
		if end, ok := filter["end_date"].(time.Time); ok {
			matchConditions = append(matchConditions, bson.E{Key: "created_at", Value: bson.M{"$lte": end}})
		}

		// Pipeline to group by product and sum amounts
		pipeline := mongo.Pipeline{
			bson.D{{Key: "$match", Value: matchConditions}},
			bson.D{{Key: "$addFields", Value: bson.M{
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
				"product": "$transaction_description",
			}}},
			bson.D{{Key: "$group", Value: bson.M{
				"_id":         "$product",
				"totalAmount": bson.M{"$sum": "$amountDecimal"},
			}}},
		}

		cursor, err := m.col(coll).Aggregate(ctx, pipeline)
		if err != nil {
			return models.SalesOverview{}, fmt.Errorf("aggregation error in %s: %w", coll, err)
		}
		var groupResults []struct {
			Product     string  `bson:"_id"`
			TotalAmount float64 `bson:"totalAmount"`
		}
		if err := cursor.All(ctx, &groupResults); err != nil {
			return models.SalesOverview{}, err
		}
		cursor.Close(ctx)

		// Count total documents for this category
		totalQuantity, err := m.col(coll).CountDocuments(ctx, matchConditions)
		if err != nil {
			return models.SalesOverview{}, fmt.Errorf("count error in %s: %w", coll, err)
		}

		// Calculate totals for this category
		categoryTotalAmount := 0.0
		for _, g := range groupResults {
			categoryTotalAmount += g.TotalAmount
		}
		categoryTotalProduct := len(groupResults)

		overview.Categories = append(overview.Categories, models.SalesOverviewCategory{
			Category: cat,
			Amount:   categoryTotalAmount,
			Product:  categoryTotalProduct,
			Quantity: int(totalQuantity),
		})

		overallTotalAmount += categoryTotalAmount
		overallTotalProduct += categoryTotalProduct
		overallTotalQuantity += int(totalQuantity)
	}

	overview.TotalAmount = overallTotalAmount
	overview.TotalProduct = overallTotalProduct
	overview.TotalQuantity = overallTotalQuantity

	return overview, nil
}

func (m *mongoStore) GetWalletSummary(filter map[string]interface{}, page int) (models.TransactionResponse, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	pageSize := 50

	ctx := context.Background()
	matchConditions := bson.D{}

	// Apply filters
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

	// Conversion for amountDecimal
	amountConversionStage := bson.D{{Key: "$addFields", Value: bson.M{
		"amountDecimal": bson.M{
			"$convert": bson.M{
				"input":   "$amount",
				"to":      "double",
				"onError": 0,
				"onNull":  0,
			},
		},
	}}}

	// Projection (after conversion!)
	projectStage := bson.D{
		{Key: "$project", Value: bson.D{
			{Key: "product", Value: "$transaction_product"},
			{Key: "description", Value: "$transaction_description"},
			{Key: "order_id", Value: "$order_id"},
			{Key: "created_at", Value: "$created_at"},
			{Key: "status", Value: "$status"},
			{Key: "amountDecimal", Value: 1},
			{Key: "flowType", Value: 1},
		}},
	}

	// Base pipeline (deposit = inflow)
	basePipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: matchConditions}},
		amountConversionStage,
		bson.D{{Key: "$addFields", Value: bson.M{"flowType": "inflow"}}},
		projectStage,
	}

	// Add the point redeem collection as part of the inflow
	// ...existing code...
	pointRedeemUnionPipeline := bson.A{
		bson.D{{Key: "$match", Value: matchConditions}},
		// map amount_redeemed -> amount so the convert stage can see it
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "product", Value: "$transaction_product"},
			{Key: "description", Value: "$transaction_description"},
			{Key: "order_id", Value: "$order_id"},
			{Key: "created_at", Value: "$created_at"},
			{Key: "status", Value: "$status"},
			{Key: "amount", Value: "$amount_redeemed"},
		}}},
		// convert the mapped amount to a numeric field
		amountConversionStage,
		// mark flowType
		bson.D{{Key: "$addFields", Value: bson.M{"flowType": "inflow"}}},
		// keep expected output fields (amountDecimal produced by conversion)
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "product", Value: 1},
			{Key: "description", Value: 1},
			{Key: "order_id", Value: 1},
			{Key: "created_at", Value: 1},
			{Key: "status", Value: 1},
			{Key: "amountDecimal", Value: 1},
			{Key: "flowType", Value: 1},
		}}},
	}
	// ...existing code...
	// Transfer union pipeline (transfer = outflow)
	transferUnionPipeline := bson.A{
		bson.D{{Key: "$match", Value: matchConditions}},
		amountConversionStage,
		bson.D{{Key: "$addFields", Value: bson.M{"flowType": "outflow"}}},
		projectStage,
	}

	// Data pipeline (with union + pagination)
	dataPipeline := append(basePipeline,
		bson.D{{Key: "$unionWith", Value: bson.M{"coll": pointRedeemColl, "pipeline": pointRedeemUnionPipeline}}},
		bson.D{{Key: "$unionWith", Value: bson.M{"coll": transferColl, "pipeline": transferUnionPipeline}}},
		bson.D{{Key: "$sort", Value: bson.M{"created_at": -1}}},
		bson.D{{Key: "$skip", Value: int64((page - 1) * pageSize)}},
		bson.D{{Key: "$limit", Value: int64(pageSize)}},
	)

	cursor, err := m.col(depositColl).Aggregate(ctx, dataPipeline)
	if err != nil {
		m.logger.Error("Wallet inflow+outflow data aggregation failed", zap.Error(err))
		return models.TransactionResponse{}, err
	}
	defer cursor.Close(ctx)

	var transactions []models.TransactionItem
	if err := cursor.All(ctx, &transactions); err != nil {
		return models.TransactionResponse{}, err
	}

	// Copy matchConditions and add status filter for totals only
	totalsMatch := append(bson.D{}, matchConditions...)
	totalsMatch = append(totalsMatch,
		bson.E{Key: "status", Value: bson.M{"$in": bson.A{"success", "pending"}}},
	)

	// Base pipeline for totals (deposit = inflow)
	totalsBasePipeline := mongo.Pipeline{
		{{Key: "$match", Value: totalsMatch}},
		amountConversionStage,
		{{Key: "$addFields", Value: bson.M{"flowType": "inflow"}}},
		projectStage,
	}

	// Transfer union pipeline (transfer = outflow)
	totalsTransferUnionPipeline := bson.A{
		bson.D{{Key: "$match", Value: totalsMatch}},
		amountConversionStage,
		bson.D{{Key: "$addFields", Value: bson.M{"flowType": "outflow"}}},
		projectStage,
	}

	// Totals pipeline
	totalsPipeline := append(totalsBasePipeline,
		bson.D{{Key: "$unionWith", Value: bson.M{"coll": pointRedeemColl, "pipeline": pointRedeemUnionPipeline}}},
		bson.D{{Key: "$unionWith", Value: bson.M{"coll": transferColl, "pipeline": totalsTransferUnionPipeline}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":        nil,
			"totalCount": bson.M{"$sum": 1},
			"totalInflow": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$eq": bson.A{"$flowType", "inflow"}}, "$amountDecimal", 0,
			}}},
			"totalOutflow": bson.M{"$sum": bson.M{"$cond": bson.A{
				bson.M{"$eq": bson.A{"$flowType", "outflow"}}, "$amountDecimal", 0,
			}}},
		}}},
	)

	totalsCursor, err := m.col(depositColl).Aggregate(ctx, totalsPipeline)
	if err != nil {
		m.logger.Error("Wallet totals aggregation failed", zap.Error(err))
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

	m.logger.Info("Wallet summary fetched",
		zap.Int("count", len(transactions)),
		zap.Int("total", res.TotalCount),
		zap.Float64("totalInflow", res.TotalInflow),
		zap.Float64("totalOutflow", res.TotalOutflow),
		zap.Any("filter", filter),
	)

	return res, nil
}
