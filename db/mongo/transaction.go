package mongo

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aremxyplug-be/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

var (
	ErrInvalidCategory = errors.New("invalid category filter")
	ErrInvalidFlow     = errors.New("invalid flow filter")
	ErrInvalidSubcat   = errors.New("invalid subcategory filter")
)

func (m *mongoStore) GetTransactions(ctx context.Context, filter map[string]interface{}, page, pageSize int) (models.TransactionResponse, error) {
	ctx = m.ensureCtx(ctx)
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 500 {
		pageSize = 50
	}

	matchConditions := bson.D{}

	// Filter by user_id
	if userID, ok := filter["user_id"].(string); ok && userID != "" {
		matchConditions = append(matchConditions, bson.E{Key: "user_id", Value: userID})
	}

	if start, ok := filter["start_date"].(time.Time); ok {
		start = start.UTC()
		var endPtr *time.Time
		if end, hasEnd := filter["end_date"].(time.Time); hasEnd {
			endUTC := end.UTC()
			endPtr = &endUTC
		}
		s, e := normalizeDayRange(start, endPtr)
		// use half-open interval [s, e) to include full end day safely
		matchConditions = append(matchConditions, bson.E{
			Key:   "created_at",
			Value: bson.M{"$gte": s, "$lt": e},
		})
	} else if end, ok := filter["end_date"].(time.Time); ok {
		// only end_date provided -> include that full day
		endUTC := end.UTC()
		_, e := normalizeDayRange(endUTC, &endUTC)
		matchConditions = append(matchConditions, bson.E{
			Key:   "created_at",
			Value: bson.M{"$lt": e},
		})
	} else {
		// No date filters → return all time (no restriction)
		// ❌ Don't append anything for created_at here
	}

	if status, ok := filter["status"].(string); ok && status != "" {
		matchConditions = append(matchConditions, bson.E{Key: "status", Value: status})
	}

	// Collection setup
	inflowCollections := []string{depositColl, pointRedeemColl}
	outflowCollections := []string{airColl, dataColl, transferColl, eduColl, tvColl, electricColl}
	collectionsToQuery := append(outflowCollections, inflowCollections...)

	// Normalize filter strings
	getLower := func(key string) string {
		if v, ok := filter[key].(string); ok {
			return strings.ToLower(strings.TrimSpace(v))
		}
		return ""
	}
	flowVal := getLower("flow")
	categoryVal := getLower("category")
	subcategoryVal := getLower("subcategory")

	// Flow filter
	if flowVal != "" {
		switch flowVal {
		case "inflow":
			collectionsToQuery = inflowCollections
		case "outflow":
			collectionsToQuery = outflowCollections
		default:
			return models.TransactionResponse{}, ErrInvalidFlow
		}
	}

	// Subcategory has highest precedence: if valid subcategory provided, apply and skip category logic
	appliedSubcategory := false
	if subcategoryVal != "" {
		switch subcategoryVal {
		case "airtime":
			collectionsToQuery = []string{airColl}
			appliedSubcategory = true
		case "data":
			collectionsToQuery = []string{dataColl}
			appliedSubcategory = true
		case "edu":
			collectionsToQuery = []string{eduColl}
			appliedSubcategory = true
		case "tv-sub":
			collectionsToQuery = []string{tvColl}
			appliedSubcategory = true
		case "elect", "electric", "electricity":
			collectionsToQuery = []string{electricColl}
			appliedSubcategory = true
		case "points", "point", "redeem":
			collectionsToQuery = []string{pointRedeemColl}
			appliedSubcategory = true
		case "virtual accounts":
			// Virtual accounts is a specific filter on deposit collection
			collectionsToQuery = []string{depositColl}
			appliedSubcategory = true
			matchConditions = append(matchConditions, bson.E{Key: "transaction_product", Value: "Virtual Account"})
		case "wallet transfer":
			collectionsToQuery = []string{depositColl, transferColl}
			matchConditions = append(matchConditions, bson.E{
				Key:   "transaction_product",
				Value: bson.M{"$in": bson.A{"Internal Deposit", "Internal Transfer"}},
			})
			appliedSubcategory = true

		default:
			// If subcategory provided but doesn't match known values, return error
			return models.TransactionResponse{}, ErrInvalidSubcat
		}
	}

	// Category logic (only if subcategory not applied)
	if !appliedSubcategory && categoryVal != "" {
		switch categoryVal {
		case "telecoms":
			// telecoms excludes deposit, transfer, points
			collectionsToQuery = []string{airColl, dataColl, eduColl, tvColl, electricColl}
		case "payments":
			collectionsToQuery = []string{depositColl, transferColl, pointRedeemColl}
		default:
			// If category provided but doesn't match known values, return error
			return models.TransactionResponse{}, ErrInvalidCategory
		}
	}

	baseCollection := collectionsToQuery[0]
	remainingCollections := collectionsToQuery[1:]

	// Validate collections
	for _, coll := range collectionsToQuery {
		if m.col(coll) == nil {
			return models.TransactionResponse{},
				errors.New("collection " + coll + " does not exist")
		}
	}

	projectStageForColl := func(coll string) bson.D {
		amountField := "$amount"
		if coll == airColl {
			amountField = "$discount_amount"
		}
		if coll == pointRedeemColl {
			amountField = "$amount_redeemed"
		}
		return bson.D{
			{Key: "$project", Value: bson.D{
				{Key: "product", Value: "$transaction_product"},
				{Key: "description", Value: "$transaction_description"},
				{Key: "order_id", Value: "$order_id"},
				{Key: "created_at", Value: "$created_at"},
				{Key: "status", Value: "$status"},
				{Key: "amount", Value: amountField},
			}},
		}
	}

	// Projection
	projectStage := projectStageForColl(baseCollection)

	// Base flow type
	baseFlowType := "outflow"
	if slices.Contains(inflowCollections, baseCollection) {
		baseFlowType = "inflow"
	}

	// Base pipeline
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: matchConditions}},
		projectStage,
		bson.D{{Key: "$addFields", Value: bson.M{"flowType": baseFlowType}}},
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
				projectStageForColl(coll),
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
	totalValue := 0.0
	for _, result := range statusResults {
		statusMetrics[result.Status] = models.StatusMetrics{
			Value:  result.Value,
			Volume: result.Volume,
		}
		totalValue += result.Value
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
				projectStageForColl(coll),
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
		res.TotalValue = totalValue
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

func (m *mongoStore) GetSalesSummary(ctx context.Context, category string, filter map[string]interface{}, page int) (models.SalesSummary, error) {
	ctx = m.ensureCtx(ctx)

	// Build match conditions
	matchConditions := bson.D{
		bson.E{Key: "status", Value: "success"},
	}
	if userID, ok := filter["user_id"].(string); ok && userID != "" {
		matchConditions = append(matchConditions, bson.E{Key: "user_id", Value: userID})
	}

	// Determine start and end times
	if start, ok := filter["start_date"].(time.Time); ok {
		start = start.UTC()
		var endPtr *time.Time
		if end, hasEnd := filter["end_date"].(time.Time); hasEnd {
			endUTC := end.UTC()
			endPtr = &endUTC
		}
		s, e := normalizeDayRange(start, endPtr)
		// use half-open interval [s, e) to include full end day safely
		matchConditions = append(matchConditions, bson.E{
			Key:   "created_at",
			Value: bson.M{"$gte": s, "$lt": e},
		})
	} else if end, ok := filter["end_date"].(time.Time); ok {
		// only end_date provided -> include that full day
		endUTC := end.UTC()
		_, e := normalizeDayRange(endUTC, &endUTC)
		matchConditions = append(matchConditions, bson.E{
			Key:   "created_at",
			Value: bson.M{"$lt": e},
		})
	} else {
		// No date filters → use current day (UTC)
		currentDay := time.Now().UTC()
		s, e := normalizeDayRange(currentDay, &currentDay)
		matchConditions = append(matchConditions, bson.E{
			Key:   "created_at",
			Value: bson.M{"$gte": s, "$lt": e},
		})
	}

	// Define collections based on category
	var collections []string
	switch category {
	case "airtime":
		collections = []string{airColl}
	case "data":
		collections = []string{dataColl}
	case "bills":
		collections = []string{eduColl, tvColl, electricColl}
	default:
		return models.SalesSummary{}, errors.New("invalid category")
	}

	// channels & sync
	summaryCh := make(chan []models.SalesSummaryItem, len(collections))
	errCh := make(chan error, len(collections))
	var wg sync.WaitGroup

	// common addFields stage for amountDecimal + product field
	commonAddForColl := func(collection string) bson.D {
		amountField := "$amount"
		if collection == airColl {
			amountField = "$discount_amount"
		}
		return bson.D{{Key: "$addFields", Value: bson.M{
			"amountDecimal": bson.M{
				"$cond": bson.A{
					bson.M{"$eq": bson.A{bson.M{"$type": amountField}, "double"}},
					amountField,
					bson.M{"$convert": bson.M{
						"input": amountField, "to": "double",
						"onError": 0, "onNull": 0,
					}},
				},
			},
			"product": "$transaction_description",
		}}}
	}

	// per-collection worker
	for _, coll := range collections {
		wg.Add(1)
		go func(collection string) {
			defer wg.Done()

			pipeline := mongo.Pipeline{}
			if len(matchConditions) > 0 {
				pipeline = append(pipeline, bson.D{{Key: "$match", Value: matchConditions}})
			}
			// add common fields
			pipeline = append(pipeline, commonAddForColl(collection))

			// category-specific handling
			if category == "data" {
				// data parsing -> produce productType, quantityInGB, group by productType
				pipeline = append(pipeline,
					bson.D{{Key: "$addFields", Value: bson.M{"words": bson.M{"$split": bson.A{"$transaction_description", " "}}}}},
					bson.D{{Key: "$addFields", Value: bson.M{"filteredWords": bson.M{
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
					}}}},
					bson.D{{Key: "$addFields", Value: bson.M{"productType": bson.M{
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
					}}}},
					bson.D{{Key: "$addFields", Value: bson.M{"dataInfo": bson.M{
						"$regexFind": bson.M{
							"input":   "$transaction_description",
							"regex":   "([0-9.]+)\\s*(GB|MB|gb|mb)",
							"options": "i",
						},
					}}}},
					bson.D{{Key: "$addFields", Value: bson.M{
						"dataValue": bson.M{
							"$cond": bson.A{
								bson.M{"$ne": bson.A{"$dataInfo", nil}},
								bson.M{"$toDouble": bson.M{"$arrayElemAt": bson.A{"$dataInfo.captures", 0}}},
								0,
							},
						},
						"dataUnit": bson.M{
							"$cond": bson.A{
								bson.M{"$ne": bson.A{"$dataInfo", nil}},
								bson.M{"$toLower": bson.M{"$arrayElemAt": bson.A{"$dataInfo.captures", 1}}},
								"gb",
							},
						},
					}}},

					bson.D{{Key: "$addFields", Value: bson.M{
						"quantityInGB": bson.M{
							"$cond": bson.A{
								bson.M{"$eq": bson.A{"$dataUnit", "mb"}},
								bson.M{"$divide": bson.A{"$dataValue", 1000}},
								"$dataValue",
							},
						},
					}}},

					// group by productType
					bson.D{{Key: "$group", Value: bson.M{
						"_id":             bson.M{"product": "$productType"},
						"quantity":        bson.M{"$sum": "$quantityInGB"},
						"totalAmount":     bson.M{"$sum": "$amountDecimal"},
						"lastTransaction": bson.M{"$max": "$created_at"},
					}}},
					bson.D{{Key: "$project", Value: bson.M{
						"product":     "$_id.product",
						"quantity":    1,
						"totalAmount": 1,
						"created_at":  "$lastTransaction",
						"_id":         0,
					}}},
				)
			} else {
				// non-data: use quantity field for edu, default quantity=1 otherwise
				if collection == eduColl {
					pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{"quantity": "$quantity"}}})
				} else {
					pipeline = append(pipeline, bson.D{{Key: "$addFields", Value: bson.M{"quantity": 1}}})
				}

				pipeline = append(pipeline,
					// group by product
					bson.D{{Key: "$group", Value: bson.M{
						"_id":             bson.M{"product": "$product"},
						"quantity":        bson.M{"$sum": "$quantity"},
						"totalAmount":     bson.M{"$sum": "$amountDecimal"},
						"lastTransaction": bson.M{"$max": "$created_at"},
					}}},
					bson.D{{Key: "$project", Value: bson.M{
						"product":     "$_id.product",
						"quantity":    1,
						"totalAmount": 1,
						"created_at":  "$lastTransaction",
						"_id":         0,
					}}},
				)
			}

			cur, err := m.col(collection).Aggregate(ctx, pipeline)
			if err != nil {
				errCh <- fmt.Errorf("aggregation error in %s: %w", collection, err)
				return
			}
			defer cur.Close(ctx)

			var items []models.SalesSummaryItem
			if err := cur.All(ctx, &items); err != nil {
				errCh <- fmt.Errorf("decode error in %s: %w", collection, err)
				return
			}
			summaryCh <- items
		}(coll)
	}

	// wait and close channels
	go func() {
		wg.Wait()
		close(summaryCh)
		close(errCh)
	}()

	// collect errors if any
	if len(errCh) > 0 {
		// drain errors to return first one (non-blocking)
		select {
		case e := <-errCh:
			return models.SalesSummary{}, e
		default:
			// continue
		}
	}

	// merge results
	mergedMap := make(map[string]models.SalesSummaryItem)
	for res := range summaryCh {
		for _, it := range res {
			existing, ok := mergedMap[it.Product]
			if !ok {
				mergedMap[it.Product] = it
			} else {
				// sum quantities and amounts, pick latest created_at
				existing.Quantity += it.Quantity
				existing.TotalAmount += it.TotalAmount
				if it.CreatedAt.After(existing.CreatedAt) {
					existing.CreatedAt = it.CreatedAt
				}
				mergedMap[it.Product] = existing
			}
		}
	}

	// build final slice
	finalSummary := make([]models.SalesSummaryItem, 0, len(mergedMap))
	for _, v := range mergedMap {
		finalSummary = append(finalSummary, v)
	}

	// sort by created_at DESC
	sort.Slice(finalSummary, func(i, j int) bool {
		return finalSummary[i].CreatedAt.After(finalSummary[j].CreatedAt)
	})

	// pagination (page param present in signature; apply simple pagination)
	pageSize := 50
	if page < 1 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start > len(finalSummary) {
		start = len(finalSummary)
	}
	end := start + pageSize
	if end > len(finalSummary) {
		end = len(finalSummary)
	}
	paged := finalSummary[start:end]

	// totals
	totalProduct := len(mergedMap)
	totalAmount := 0.0
	for _, it := range finalSummary {
		totalAmount += it.TotalAmount
	}

	// totalQuantity (transaction count across the selected collections / match conditions)
	totalQuantity := 0.0
	for _, coll := range collections {
		count, err := m.col(coll).CountDocuments(ctx, matchConditions)
		if err != nil {
			// log and continue; prefer returning error if desired
			m.logger.Error("count documents failed", zap.String("coll", coll), zap.Error(err))
			continue
		}
		totalQuantity += float64(count)
	}

	return models.SalesSummary{
		Summary:       paged,
		TotalCount:    len(finalSummary),
		TotalProduct:  totalProduct,
		TotalQuantity: totalQuantity,
		TotalAmount:   totalAmount,
	}, nil
}

// ...existing code...
func (m *mongoStore) GetSalesOverview(ctx context.Context, filter map[string]interface{}) (models.SalesSummary, error) {
	ctx = m.ensureCtx(ctx)

	// categories => collection name
	categories := map[string]string{
		"airtime":      airColl,
		"data":         dataColl,
		"edu":          eduColl,
		"tv-sub":       tvColl,
		"electric-sub": electricColl,
	}

	// Build match conditions; default start = today 00:00:00 UTC when no start_date provided
	matchConditions := bson.D{
		bson.E{Key: "status", Value: "success"},
	}
	if userID, ok := filter["user_id"].(string); ok && userID != "" {
		matchConditions = append(matchConditions, bson.E{Key: "user_id", Value: userID})
	}

	// Determine start and end times
	if start, ok := filter["start_date"].(time.Time); ok {
		start = start.UTC()
		var endPtr *time.Time
		if end, hasEnd := filter["end_date"].(time.Time); hasEnd {
			endUTC := end.UTC()
			endPtr = &endUTC
		}
		s, e := normalizeDayRange(start, endPtr)
		// use half-open interval [s, e) to include full end day safely
		matchConditions = append(matchConditions, bson.E{
			Key:   "created_at",
			Value: bson.M{"$gte": s, "$lt": e},
		})
	} else if end, ok := filter["end_date"].(time.Time); ok {
		// only end_date provided -> include that full day
		endUTC := end.UTC()
		_, e := normalizeDayRange(endUTC, &endUTC)
		matchConditions = append(matchConditions, bson.E{
			Key:   "created_at",
			Value: bson.M{"$lt": e},
		})
	} else {
		currentDay := time.Now().UTC()
		s, e := normalizeDayRange(currentDay, &currentDay)
		matchConditions = append(matchConditions, bson.E{
			Key:   "created_at",
			Value: bson.M{"$gte": s, "$lt": e},
		})
	}

	// helper: amount conversion stage (cond type-check then convert)
	amountConversion := bson.D{{Key: "$addFields", Value: bson.M{
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
	}}}

	// Build per-collection pipelines that emit documents of shape:
	// { product, quantity, totalAmount, created_at, txnCount }
	var baseColl string
	first := true
	var combinedPipeline mongo.Pipeline

	for cat, coll := range categories {
		per := mongo.Pipeline{}
		per = append(per, bson.D{{Key: "$match", Value: matchConditions}})

		amountField := "$amount"
		if coll == airColl {
			amountField = "$discount_amount"
		}

		per = append(per, bson.D{{Key: "$addFields", Value: bson.M{
			"product":                "$transaction_description",
			"transaction_created_at": "$created_at",
			"amount":                 amountField,
		}}})

		switch cat {
		case "data":
			per = append(per, bson.D{{Key: "$addFields", Value: bson.M{
				"words": bson.M{"$split": bson.A{"$transaction_description", " "}},
			}}})
			per = append(per, bson.D{{Key: "$addFields", Value: bson.M{
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
			per = append(per, bson.D{{Key: "$addFields", Value: bson.M{
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
			per = append(per, bson.D{{Key: "$addFields", Value: bson.M{
				"dataInfo": bson.M{
					"$regexFind": bson.M{
						"input":   "$transaction_description",
						"regex":   "([0-9.]+)\\s*(GB|MB|gb|mb)",
						"options": "i",
					},
				},
			}}})
			per = append(per, bson.D{{Key: "$addFields", Value: bson.M{
				"dataValue": bson.M{
					"$cond": bson.A{
						bson.M{"$ne": bson.A{"$dataInfo", nil}},
						bson.M{"$toDouble": bson.M{
							"$arrayElemAt": bson.A{"$dataInfo.captures", 0},
						}},
						0,
					},
				},
				"dataUnit": bson.M{
					"$cond": bson.A{
						bson.M{"$ne": bson.A{"$dataInfo", nil}},
						bson.M{"$toLower": bson.M{
							"$arrayElemAt": bson.A{"$dataInfo.captures", 1},
						}},
						"gb",
					},
				},
			}}})
			per = append(per, bson.D{{Key: "$addFields", Value: bson.M{
				"quantityInGB": bson.M{
					"$cond": bson.A{
						bson.M{"$eq": bson.A{"$dataUnit", "mb"}},
						bson.M{"$divide": bson.A{"$dataValue", 1000}},
						"$dataValue",
					},
				},
				"product":    "$productType",
				"amount":     amountField,
				"created_at": "$transaction_created_at",
			}}})

			per = append(per, amountConversion)

			// include txnCount in group
			per = append(per, bson.D{{Key: "$group", Value: bson.M{
				"_id":             "$product",
				"quantity":        bson.M{"$sum": "$quantityInGB"},
				"totalAmount":     bson.M{"$sum": "$amountDecimal"},
				"lastTransaction": bson.M{"$max": "$created_at"},
				"txnCount":        bson.M{"$sum": 1},
			}}})

			per = append(per, bson.D{{Key: "$project", Value: bson.M{
				"product":     "$_id",
				"quantity":    1,
				"totalAmount": 1,
				"created_at":  "$lastTransaction",
				"txnCount":    1,
				"_id":         0,
			}}})

		case "edu":
			per = append(per, bson.D{{Key: "$addFields", Value: bson.M{
				"quantity":   "$quantity",
				"created_at": "$transaction_created_at",
			}}})
			per = append(per, amountConversion)

			per = append(per, bson.D{{Key: "$group", Value: bson.M{
				"_id":             "$product",
				"quantity":        bson.M{"$sum": "$quantity"},
				"totalAmount":     bson.M{"$sum": "$amountDecimal"},
				"lastTransaction": bson.M{"$max": "$created_at"},
				"txnCount":        bson.M{"$sum": 1},
			}}})

			per = append(per, bson.D{{Key: "$project", Value: bson.M{
				"product":     "$_id",
				"quantity":    1,
				"totalAmount": 1,
				"created_at":  "$lastTransaction",
				"txnCount":    1,
				"_id":         0,
			}}})

		default:
			per = append(per, bson.D{{Key: "$addFields", Value: bson.M{
				"quantity":   1,
				"created_at": "$transaction_created_at",
			}}})
			per = append(per, amountConversion)

			per = append(per, bson.D{{Key: "$group", Value: bson.M{
				"_id":             "$product",
				"quantity":        bson.M{"$sum": "$quantity"},
				"totalAmount":     bson.M{"$sum": "$amountDecimal"},
				"lastTransaction": bson.M{"$max": "$created_at"},
				"txnCount":        bson.M{"$sum": 1},
			}}})

			per = append(per, bson.D{{Key: "$project", Value: bson.M{
				"product":     "$_id",
				"quantity":    1,
				"totalAmount": 1,
				"created_at":  "$lastTransaction",
				"txnCount":    1,
				"_id":         0,
			}}})
		}

		if first {
			combinedPipeline = per
			baseColl = coll
			first = false
		} else {
			unionWithStage := bson.D{{Key: "$unionWith", Value: bson.M{"coll": coll, "pipeline": per}}}
			combinedPipeline = append(combinedPipeline, unionWithStage)
		}
	}

	// Merge same products across collections and sum txnCount
	combinedPipeline = append(combinedPipeline,
		bson.D{{Key: "$group", Value: bson.M{
			"_id":             "$product",
			"quantity":        bson.M{"$sum": "$quantity"},
			"totalAmount":     bson.M{"$sum": "$totalAmount"},
			"lastTransaction": bson.M{"$max": "$created_at"},
			"txnCount":        bson.M{"$sum": "$txnCount"},
		}}},

		bson.D{{Key: "$project", Value: bson.M{
			"product":     "$_id",
			"quantity":    1,
			"totalAmount": 1,
			"created_at":  "$lastTransaction",
			"txnCount":    1,
			"_id":         0,
		}}},

		bson.D{{Key: "$sort", Value: bson.M{"created_at": -1}}},
	)

	// Execute aggregation on baseColl
	cursor, err := m.col(baseColl).Aggregate(ctx, combinedPipeline)
	if err != nil {
		return models.SalesSummary{}, fmt.Errorf("sales overview aggregation failed: %w", err)
	}
	defer cursor.Close(ctx)

	// decode into local struct that includes txnCount
	type mergedItem struct {
		Product     string    `bson:"product"`
		Quantity    float64   `bson:"quantity"`
		TotalAmount float64   `bson:"totalAmount"`
		CreatedAt   time.Time `bson:"created_at"`
		TxnCount    float64   `bson:"txnCount"`
	}
	var items []mergedItem
	if err := cursor.All(ctx, &items); err != nil {
		return models.SalesSummary{}, err
	}

	// map to models.SalesSummaryItem and compute totals.
	var merged []models.SalesSummaryItem
	totalAmount := 0.0
	totalTxnCount := 0.0
	for _, it := range items {
		merged = append(merged, models.SalesSummaryItem{
			Product:     it.Product,
			Quantity:    it.Quantity,
			TotalAmount: it.TotalAmount,
			CreatedAt:   it.CreatedAt,
		})
		totalAmount += it.TotalAmount
		totalTxnCount += it.TxnCount
	}

	totalProduct := len(merged)

	// Return as SalesSummary (merged product level).
	// TotalQuantity now represents transaction count across the merged results.
	return models.SalesSummary{
		Summary:       merged,
		TotalCount:    totalProduct,
		TotalProduct:  totalProduct,
		TotalQuantity: totalTxnCount,
		TotalAmount:   totalAmount,
	}, nil
}

func (m *mongoStore) GetWalletSummary(ctx context.Context, filter map[string]interface{}, page int) (models.TransactionResponse, error) {
	ctx = m.ensureCtx(ctx)
	// Validate pagination
	if page < 1 {
		page = 1
	}
	pageSize := 50

	matchConditions := bson.D{}

	// Apply filters
	if userID, ok := filter["user_id"].(string); ok && userID != "" {
		matchConditions = append(matchConditions, bson.E{Key: "user_id", Value: userID})
	}

	// Handle date range
	if start, ok := filter["start_date"].(time.Time); ok {
		start = start.UTC()
		var endPtr *time.Time
		if end, hasEnd := filter["end_date"].(time.Time); hasEnd {
			endUTC := end.UTC()
			endPtr = &endUTC
		}
		s, e := normalizeDayRange(start, endPtr)
		// use half-open interval [s, e) to include full end day safely
		matchConditions = append(matchConditions, bson.E{
			Key:   "created_at",
			Value: bson.M{"$gte": s, "$lt": e},
		})
	} else if end, ok := filter["end_date"].(time.Time); ok {
		// only end_date provided -> include that full day
		endUTC := end.UTC()
		_, e := normalizeDayRange(endUTC, &endUTC)
		matchConditions = append(matchConditions, bson.E{
			Key:   "created_at",
			Value: bson.M{"$lt": e},
		})
	} else {
		// No date filters → return all time (no restriction)
		// ❌ Don't append anything for created_at here
	}

	if status, ok := filter["status"].(string); ok && status != "" {
		matchConditions = append(matchConditions, bson.E{Key: "status", Value: status})
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

	// allow filtering by record type: "inflow" | "outflow" (deposit/pointRedeem => inflow, transfer => outflow)
	record := ""
	if r, ok := filter["record"].(string); ok && r != "" {
		record = r
	}

	category := ""
	if c, ok := filter["category"].(string); ok && c != "" {
		category = c
	}

	// build dataPipeline and pick which collection to run Aggregate on (base)
	dataBaseColl := depositColl
	var dataPipeline mongo.Pipeline

	if category != "" {
		switch category {
		case "virtual":
			// Only Virtual Account product from deposit collection (inflow)
			dataPipeline = append(basePipeline,
				bson.D{{Key: "$match", Value: bson.M{"product": "Virtual Account"}}},
				bson.D{{Key: "$sort", Value: bson.M{"created_at": -1}}},
				bson.D{{Key: "$skip", Value: int64((page - 1) * pageSize)}},
				bson.D{{Key: "$limit", Value: int64(pageSize)}},
			)
			dataBaseColl = depositColl

		case "wallet":
			// Internal wallet movements across deposit and transfer
			p := append(basePipeline,
				bson.D{{Key: "$unionWith", Value: bson.M{"coll": transferColl, "pipeline": transferUnionPipeline}}},
			)
			p = append(p,
				bson.D{{Key: "$match", Value: bson.M{"product": bson.M{"$in": []string{"Internal Deposit", "Internal Transfer"}}}}},
				bson.D{{Key: "$sort", Value: bson.M{"created_at": -1}}},
				bson.D{{Key: "$skip", Value: int64((page - 1) * pageSize)}},
				bson.D{{Key: "$limit", Value: int64(pageSize)}},
			)
			dataPipeline = p
			dataBaseColl = depositColl

		case "point":
			// Only point redeem collection (inflow)
			dataPipeline = mongo.Pipeline{
				bson.D{{Key: "$match", Value: matchConditions}},
			}
			dataPipeline = append(dataPipeline,
				bson.D{{Key: "$unionWith", Value: bson.M{"coll": pointRedeemColl, "pipeline": pointRedeemUnionPipeline}}},
				bson.D{{Key: "$sort", Value: bson.M{"created_at": -1}}},
				bson.D{{Key: "$skip", Value: int64((page - 1) * pageSize)}},
				bson.D{{Key: "$limit", Value: int64(pageSize)}},
			)
			dataBaseColl = depositColl
		default:
			// Fallback: all collections, no category-specific filter
			dataPipeline = append(basePipeline,
				bson.D{{Key: "$unionWith", Value: bson.M{"coll": pointRedeemColl, "pipeline": pointRedeemUnionPipeline}}},
				bson.D{{Key: "$unionWith", Value: bson.M{"coll": transferColl, "pipeline": transferUnionPipeline}}},
				bson.D{{Key: "$sort", Value: bson.M{"created_at": -1}}},
				bson.D{{Key: "$skip", Value: int64((page - 1) * pageSize)}},
				bson.D{{Key: "$limit", Value: int64(pageSize)}},
			)
		}
	} else {
		// Standard 'record' based filtering (existing logic)
		switch record {
		case "deposit":
			// only deposit + pointRedeem (both inflow)
			dataPipeline = append(basePipeline,
				bson.D{{Key: "$unionWith", Value: bson.M{"coll": pointRedeemColl, "pipeline": pointRedeemUnionPipeline}}},
				bson.D{{Key: "$sort", Value: bson.M{"created_at": -1}}},
				bson.D{{Key: "$skip", Value: int64((page - 1) * pageSize)}},
				bson.D{{Key: "$limit", Value: int64(pageSize)}},
			)
			dataBaseColl = depositColl

		case "transfer":
			// only transfer collection (outflow)
			dataPipeline = mongo.Pipeline{
				bson.D{{Key: "$match", Value: matchConditions}},
				amountConversionStage,
				bson.D{{Key: "$addFields", Value: bson.M{"flowType": "outflow"}}},
				projectStage,
				bson.D{{Key: "$sort", Value: bson.M{"created_at": -1}}},
				bson.D{{Key: "$skip", Value: int64((page - 1) * pageSize)}},
				bson.D{{Key: "$limit", Value: int64(pageSize)}},
			}
			dataBaseColl = transferColl

		default:
			// both inflow and outflow (original behaviour)
			dataPipeline = append(basePipeline,
				bson.D{{Key: "$unionWith", Value: bson.M{"coll": pointRedeemColl, "pipeline": pointRedeemUnionPipeline}}},
				bson.D{{Key: "$unionWith", Value: bson.M{"coll": transferColl, "pipeline": transferUnionPipeline}}},
				bson.D{{Key: "$sort", Value: bson.M{"created_at": -1}}},
				bson.D{{Key: "$skip", Value: int64((page - 1) * pageSize)}},
				bson.D{{Key: "$limit", Value: int64(pageSize)}},
			)
		}
	}

	cursor, err := m.col(dataBaseColl).Aggregate(ctx, dataPipeline)
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
		bson.D{{Key: "$match", Value: totalsMatch}},
		amountConversionStage,
		bson.D{{Key: "$addFields", Value: bson.M{"flowType": "inflow"}}},
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

func normalizeDayRange(start time.Time, end *time.Time) (time.Time, time.Time) {
	s := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	if end == nil {
		return s, s.Add(24 * time.Hour)
	}
	e := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
	return s, e
}
