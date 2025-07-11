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

func (m *mongoStore) GetChart(userID string, rangeType string, fromTime time.Time, toTime time.Time) (models.StatsResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	baseFilter := bson.D{
		{Key: "created_at", Value: bson.D{
			{Key: "$gte", Value: fromTime},
			{Key: "$lt", Value: toTime},
		}},
		// {Key: "status", Value: "success"},
	}

	if userID != "" {
		baseFilter = append(baseFilter, bson.E{Key: "user_id", Value: userID})
	}

	// Process inflow (deposits)
	inflowResult, err := m.processCollection(ctx, depositColl, rangeType, baseFilter, true)
	if err != nil {
		return models.StatsResponse{}, fmt.Errorf("inflow processing failed: %w", err)
	}

	// Process outflow collections in parallel with context cancellation
	outflowCtx, outflowCancel := context.WithCancel(ctx)
	defer outflowCancel()

	collections := []string{airColl, dataColl, eduColl, electricColl, tvColl, transferColl}
	results := make(chan models.CollectionResult, len(collections))
	errChan := make(chan error, len(collections))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // Limit to 5 concurrent DB queries

	for _, collName := range collections {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-outflowCtx.Done():
				return
			}

			res, err := m.processCollection(outflowCtx, name, rangeType, baseFilter, false)
			if err != nil {
				select {
				case errChan <- fmt.Errorf("%s processing failed: %w", name, err):
					outflowCancel() // Cancel other operations on error
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

	// Process outflow results
	outflowResult := models.CollectionResult{
		TimeMap: make(map[string]models.TimeStat),
	}
	var merr *multierror.Error

	// Collect errors first
	for err := range errChan {
		merr = multierror.Append(merr, err)
	}

	if merr != nil {
		return models.StatsResponse{}, merr.ErrorOrNil()
	}

	// Process results
	for res := range results {
		outflowResult.TotalAmount += res.TotalAmount
		outflowResult.TotalCount += res.TotalCount

		// Merge time stats with proper label initialization
		for timeKey, stat := range res.TimeMap {
			existing, exists := outflowResult.TimeMap[timeKey]
			if !exists {
				existing = models.TimeStat{Label: timeKey}
			}
			existing.Amount += stat.Amount
			existing.Count += stat.Count
			outflowResult.TimeMap[timeKey] = existing
		}

		// Limit total transactions to 1000
		if len(outflowResult.Transactions) < 1000 {
			remaining := 1000 - len(outflowResult.Transactions)
			if len(res.Transactions) > remaining {
				outflowResult.Transactions = append(outflowResult.Transactions, res.Transactions[:remaining]...)
			} else {
				outflowResult.Transactions = append(outflowResult.Transactions, res.Transactions...)
			}
		}
	}

	// Convert hourly maps to sorted slices
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

func (m *mongoStore) processCollection(ctx context.Context, collName, rangeType string, filter bson.D, isInflow bool) (models.CollectionResult, error) {
	coll := m.col(collName)
	result := models.CollectionResult{
		TimeMap: make(map[string]models.TimeStat),
	}

	// Determine grouping key based on rangeType
	var groupID interface{}
	switch rangeType {
	case "TODAY":
		groupID = bson.D{{Key: "$hour", Value: "$created_at"}}
	case "WEEKLY":
		groupID = bson.D{{Key: "$dateToString", Value: bson.D{
			{Key: "format", Value: "%Y-%U"},
			{Key: "date", Value: "$created_at"},
		}}}
	case "MONTHLY", "ALL_TIME":
		groupID = bson.D{{Key: "$dateToString", Value: bson.D{
			{Key: "format", Value: "%Y-%m"},
			{Key: "date", Value: "$created_at"},
		}}}
	default: // DAILY
		groupID = bson.D{{Key: "$dateToString", Value: bson.D{
			{Key: "format", Value: "%Y-%m-%d"},
			{Key: "date", Value: "$created_at"},
		}}}
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$addFields", Value: bson.D{
			{Key: "convertedAmount", Value: bson.D{
				{Key: "$convert", Value: bson.D{
					{Key: "input", Value: "$amount"},
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
