package mongo

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"sync"
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
	inflowCollections := []string{depositColl}
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

func (m *mongoStore) GetSalesSummary(category string, filter map[string]interface{}, page int) (models.SalesSummary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Validate pagination
	if page < 1 {
		page = 1
	}

	pageSize := 50

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
	var collections []string
	switch category {
	case "airtime":
		collections = []string{"airtime"}
	case "data":
		collections = []string{"data"}
	case "bills":
		collections = []string{"edu", "tv-sub", "electric-sub"}
	default:
		return models.SalesSummary{}, errors.New("invalid category")
	}

	// Create channels for parallel processing
	summaryChan := make(chan []models.SalesSummaryItem, len(collections))
	errChan := make(chan error, len(collections))
	var wg sync.WaitGroup

	// Process each collection in parallel
	for _, coll := range collections {
		wg.Add(1)
		go func(collection string) {
			defer wg.Done()

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
				errChan <- fmt.Errorf("%s aggregation failed: %w", collection, err)
				return
			}
			defer cursor.Close(ctx)

			var results []models.SalesSummaryItem
			if err := cursor.All(ctx, &results); err != nil {
				errChan <- err
				return
			}

			summaryChan <- results
		}(coll)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(summaryChan)
		close(errChan)
	}()

	// Collect results
	allSummaries := []models.SalesSummaryItem{}
	totalOutflow := 0.0

	for i := 0; i < len(collections); i++ {
		select {
		case err := <-errChan:
			return models.SalesSummary{}, err
		case summaries := <-summaryChan:
			allSummaries = append(allSummaries, summaries...)
		}
	}

	// Merge duplicate products
	summaryMap := make(map[string]models.SalesSummaryItem)
	for _, item := range allSummaries {
		if existing, exists := summaryMap[item.Product]; exists {
			existing.Quantity += item.Quantity
			existing.TotalAmount += item.TotalAmount
			// Keep earliest created_at
			if item.CreatedAt.Before(existing.CreatedAt) {
				existing.CreatedAt = item.CreatedAt
			}
			summaryMap[item.Product] = existing
		} else {
			summaryMap[item.Product] = item
		}
	}

	// Convert map to slice
	finalSummary := make([]models.SalesSummaryItem, 0, len(summaryMap))
	for _, item := range summaryMap {
		finalSummary = append(finalSummary, item)
		totalOutflow += item.TotalAmount
	}

	// Sort by created_at DESC (newest first)
	sort.Slice(finalSummary, func(i, j int) bool {
		// Newest first (descending order)
		return finalSummary[i].CreatedAt.After(finalSummary[j].CreatedAt)
	})

	// Paginate results
	totalItems := len(finalSummary)
	start := (page - 1) * pageSize
	if start > totalItems || totalItems == 0 {
		return models.SalesSummary{
			Summary:      []models.SalesSummaryItem{},
			TotalCount:   totalItems,
			TotalOutflow: totalOutflow,
		}, nil
	}
	end := start + pageSize
	if end > totalItems {
		end = totalItems
	}

	// paginatedSummary := finalSummary[start:end]

	return models.SalesSummary{
		Summary:      finalSummary,
		TotalCount:   totalItems, // Total distinct products
		TotalInflow:  0,          // Always 0 for these categories
		TotalOutflow: totalOutflow,
	}, nil

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
		{{Key: "$match", Value: matchConditions}},
		amountConversionStage,
		{{Key: "$addFields", Value: bson.M{"flowType": "inflow"}}},
		projectStage,
	}

	// Transfer union pipeline (transfer = outflow)
	transferUnionPipeline := bson.A{
		bson.D{{Key: "$match", Value: matchConditions}},
		amountConversionStage,
		bson.D{{Key: "$addFields", Value: bson.M{"flowType": "outflow"}}},
		projectStage,
	}

	// Data pipeline (with union + pagination)
	dataPipeline := append(basePipeline,
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

	// Totals pipeline
	totalsPipeline := append(basePipeline,
		bson.D{{Key: "$unionWith", Value: bson.M{"coll": transferColl, "pipeline": transferUnionPipeline}}},
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
