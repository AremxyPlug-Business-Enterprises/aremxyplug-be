package mongo

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/aremxyplug-be/db/models"
	"github.com/hashicorp/go-multierror"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// dayStart normalizes a time to the start of its day in UTC+1 (00:00:00.0).
func dayStart(t time.Time) time.Time {
	// Convert to UTC+1 timezone
	utcPlus1 := time.FixedZone("UTC+1", 1*60*60)
	t = t.In(utcPlus1)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, utcPlus1)
}

// dateStringGroupID returns a BSON grouping expression for $dateToString with the given format.
func dateStringGroupID(format string) bson.D {
	return bson.D{
		{Key: "$dateToString", Value: bson.D{
			{Key: "format", Value: format},
			{Key: "date", Value: "$created_at"},
		}},
	}
}

func (m *mongoStore) GetChart(filter map[string]interface{}, rangeType string) (models.StatsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build baseFilter (Mongo query) and determine grouping strategy
	var baseFilter bson.D
	var groupID interface{}

	start, hasStart := filter["start_date"].(time.Time)
	end, hasEnd := filter["end_date"].(time.Time)
	useExplicitDates := hasStart || hasEnd

	now := time.Now().UTC()
	var s, e time.Time

	if useExplicitDates {
		// Explicit date filters provided: these override default range and rangeType
		// Normalize dates to day boundaries (UTC+1) and derive grouping from span
		if hasStart {
			s = dayStart(start)
		} else {
			// only end given: use that day as both start and end
			s = dayStart(end)
		}
		if hasEnd {
			e = dayStart(end).Add(24 * time.Hour)
		} else {
			e = s.Add(24 * time.Hour)
		}

		baseFilter = append(baseFilter, bson.E{Key: "created_at", Value: bson.M{"$gte": s, "$lt": e}})

		// Derive grouping from the span
		span := e.Sub(s)
		switch {
		case span <= 24*time.Hour:
			groupID = bson.D{{Key: "$hour", Value: "$created_at"}} // hourly
		case span <= 31*24*time.Hour:
			groupID = dateStringGroupID("%Y-%m-%d") // daily
		case span <= 120*24*time.Hour:
			groupID = dateStringGroupID("%Y-%U") // weekly
		default:
			groupID = dateStringGroupID("%Y-%m") // monthly
		}
	} else {
		// No explicit filters: default to TODAY (start of day UTC+1 to current time), unless rangeType overrides
		s = dayStart(now)
		e = now  // Current time, not full 24 hours
		groupID = bson.D{{Key: "$hour", Value: "$created_at"}}

		// Allow rangeType to override default TODAY range if explicitly provided
		if rangeType != "" && rangeType != "TODAY" {
			switch rangeType {
			case "WEEKLY":
				s = now.AddDate(0, 0, -7)
				e = now
				groupID = dateStringGroupID("%Y-%U")
			case "MONTHLY":
				s = now.AddDate(0, -1, 0)
				e = now
				groupID = dateStringGroupID("%Y-%m")
			case "ALL-TIME":
				// No date restriction; group by month
				groupID = dateStringGroupID("%Y-%m")
				// For ALL-TIME, skip adding date filter
				goto skipDateFilter
			case "DAILY":
				// Same as default TODAY: from midnight UTC+1 to current time
				s = dayStart(now)
				e = now
				groupID = bson.D{{Key: "$hour", Value: "$created_at"}}
			}
		}

		// Add created_at filter for ranged queries (not for ALL-TIME)
		baseFilter = append(baseFilter, bson.E{Key: "created_at", Value: bson.M{"$gte": s, "$lt": e}})

		skipDateFilter:
	}

	// Optional user filter
	if userID, ok := filter["user_id"].(string); ok && userID != "" {
		baseFilter = append(baseFilter, bson.E{Key: "user_id", Value: userID})
	}

	// Chart metrics should consider only successful and pending transactions.
	baseFilter = append(baseFilter, bson.E{Key: "status", Value: bson.M{"$in": bson.A{"success", "pending"}}})

	// for the inflow collection, point-redeem is a part

	// Process inflow (deposit + point-redeem)
	inflowCollections := []string{depositColl, pointRedeemColl}
	inflowResult := models.CollectionResult{TimeMap: make(map[string]models.TimeStat)}
	for _, collName := range inflowCollections {
		res, err := m.processCollection(ctx, collName, groupID, baseFilter)
		if err != nil {
			return models.StatsResponse{}, fmt.Errorf("%s inflow processing failed: %w", collName, err)
		}
		mergeCollectionResult(&inflowResult, res)
	}

	// Process outflow in parallel with cancellation
	outflowCtx, outflowCancel := context.WithCancel(ctx)
	defer outflowCancel()

	collections := []string{airColl, dataColl, eduColl, electricColl, tvColl, transferColl}
	results := make(chan models.CollectionResult, len(collections))
	errChan := make(chan error, len(collections))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // Limit concurrency

	for _, collName := range collections {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-outflowCtx.Done():
				return
			}
			res, err := m.processCollection(outflowCtx, name, groupID, baseFilter)
			if err != nil {
				select {
				case errChan <- fmt.Errorf("%s processing failed: %w", name, err):
					outflowCancel()
				case <-outflowCtx.Done():
				}
				return
			}
			select {
			case results <- res:
			case <-outflowCtx.Done():
			}
		}(collName)
	}

	go func() {
		wg.Wait()
		close(results)
		close(errChan)
	}()

	outflowResult := models.CollectionResult{TimeMap: make(map[string]models.TimeStat)}
	var merr *multierror.Error
	for err := range errChan {
		merr = multierror.Append(merr, err)
	}
	if merr != nil {
		return models.StatsResponse{}, merr.ErrorOrNil()
	}

	for res := range results {
		mergeCollectionResult(&outflowResult, res)
	}

	inflow := convertTimeMapToSlice(inflowResult.TimeMap)
	outflow := convertTimeMapToSlice(outflowResult.TimeMap)

	return models.StatsResponse{
		TotalInflowCount:    inflowResult.TotalCount,
		TotalInflowAmount:   inflowResult.TotalAmount,
		TotalOutflowCount:   outflowResult.TotalCount,
		TotalOutflowAmount:  outflowResult.TotalAmount,
		Inflow:              inflow,
		Outflow:             outflow,
		InflowTransactions:  inflowResult.Transactions,
		OutflowTransactions: outflowResult.Transactions,
	}, nil
}

func (m *mongoStore) processCollection(ctx context.Context, collName string, groupID interface{}, filter bson.D) (models.CollectionResult, error) {
	coll := m.col(collName)
	result := models.CollectionResult{
		TimeMap: make(map[string]models.TimeStat),
	}

	amountField := "$amount"
	if collName == pointRedeemColl {
		amountField = "$amount_redeemed"
	}
	if collName == airColl {
		amountField = "$discount_amount"
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$addFields", Value: bson.D{
			{Key: "convertedAmount", Value: bson.D{
				{Key: "$convert", Value: bson.D{
					{Key: "input", Value: amountField},
					{Key: "to", Value: "double"},
					{Key: "onError", Value: 0.0},
					{Key: "onNull", Value: 0.0},
				}},
			}},
		}}},
		{{Key: "$facet", Value: bson.D{
			{Key: "timeData", Value: mongo.Pipeline{
				{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: groupID},
					{Key: "totalAmount", Value: bson.D{{Key: "$sum", Value: "$convertedAmount"}}},
					{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
				}}},
			}},
			{Key: "summaryData", Value: mongo.Pipeline{
				{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: nil},
					{Key: "totalAmount", Value: bson.D{{Key: "$sum", Value: "$convertedAmount"}}},
					{Key: "totalCount", Value: bson.D{{Key: "$sum", Value: 1}}},
				}}},
			}},
			{Key: "transactionSamples", Value: mongo.Pipeline{
				{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}}}},
				{{Key: "$limit", Value: 1000}},
				{{Key: "$project", Value: bson.D{
					{Key: "_id", Value: 0},
					{Key: "created_at", Value: 1},
					{Key: "amount", Value: "$convertedAmount"},
				}}},
			}}},
		}}}

	cursor, err := coll.Aggregate(ctx, pipeline, options.Aggregate().SetAllowDiskUse(true))
	if err != nil {
		return result, err
	}
	defer cursor.Close(ctx)

	var aggResult struct {
		TimeData []struct {
			ID          interface{} `bson:"_id"`
			TotalAmount float64     `bson:"totalAmount"`
			Count       int         `bson:"count"`
		} `bson:"timeData"`
		SummaryData []struct {
			TotalAmount float64 `bson:"totalAmount"`
			TotalCount  int     `bson:"totalCount"`
		} `bson:"summaryData"`
		TransactionSamples []struct {
			CreatedAt time.Time `bson:"created_at"`
			Amount    float64   `bson:"amount"`
		} `bson:"transactionSamples"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&aggResult); err != nil {
			return result, err
		}
	}

	for _, stat := range aggResult.TimeData {
		label := formatLabel(stat.ID)
		result.TimeMap[label] = models.TimeStat{
			Label:  label,
			Amount: stat.TotalAmount,
			Count:  stat.Count,
		}
	}

	if len(aggResult.SummaryData) > 0 {
		result.TotalAmount = aggResult.SummaryData[0].TotalAmount
		result.TotalCount = aggResult.SummaryData[0].TotalCount
	}

	for _, tx := range aggResult.TransactionSamples {
		result.Transactions = append(result.Transactions, models.TransactionPoint{
			CreatedAt: tx.CreatedAt,
			Amount:    tx.Amount,
		})
	}

	return result, nil
}

// Helper to format label based on range type
func formatLabel(id interface{}) string {
	switch v := id.(type) {
	case string:
		return v
	case int32, int:
		return fmt.Sprintf("%02d:00", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Helper to convert time map to sorted slice
func convertTimeMapToSlice(timeMap map[string]models.TimeStat) []models.TimeStat {
	slice := make([]models.TimeStat, 0, len(timeMap))
	for _, stat := range timeMap {
		slice = append(slice, stat)
	}
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].Label < slice[j].Label
	})
	return slice
}

func mergeCollectionResult(target *models.CollectionResult, res models.CollectionResult) {
	target.TotalAmount += res.TotalAmount
	target.TotalCount += res.TotalCount

	for timeKey, stat := range res.TimeMap {
		existing := target.TimeMap[timeKey]
		if existing.Label == "" {
			existing.Label = timeKey
		}
		existing.Amount += stat.Amount
		existing.Count += stat.Count
		target.TimeMap[timeKey] = existing
	}

	if len(target.Transactions) < 1000 {
		remaining := 1000 - len(target.Transactions)
		if len(res.Transactions) > remaining {
			target.Transactions = append(target.Transactions, res.Transactions[:remaining]...)
		} else {
			target.Transactions = append(target.Transactions, res.Transactions...)
		}
	}
}
