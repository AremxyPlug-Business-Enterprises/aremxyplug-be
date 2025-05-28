package mongo

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/aremxyplug-be/db/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (m *mongoStore) GetChart(userID string, fromTime time.Time, toTime time.Time) (models.StatsResponse, error) {
	ctx := context.Background()
	resp := models.StatsResponse{}

	// Common base filter - changed 'timestamp' to 'created_at'
	baseFilter := bson.D{
		{Key: "created_at", Value: bson.D{
			{Key: "$gte", Value: fromTime},
			{Key: "$lt", Value: toTime},
		}},
		{Key: "status", Value: "success"},
	}

	if userID != "" {
		baseFilter = append(baseFilter, bson.E{Key: "user_id", Value: userID})
	}

	// ================= INFLOW (DEPOSITS) =================
	depositColl := m.col(depositColl)

	// Get inflow transactions
	inflowTxs, inflowHourly, totalInAmount, totalInCount, err := m.getTransactionData(ctx, depositColl, baseFilter)
	if err != nil {
		return resp, fmt.Errorf("failed to get inflow data: %w", err)
	}

	// ================= OUTFLOW COLLECTIONS =================
	collections := []string{airColl, dataColl, eduColl, electricColl, tvColl, transferColl}
	var outflowTxs []models.TransactionPoint
	outflowHourly := make(map[int]models.HourlyStat)
	totalOutAmount := 0.0
	totalOutCount := 0

	for _, collName := range collections {
		coll := m.col(collName)
		txs, hourly, amount, count, err := m.getTransactionData(ctx, coll, baseFilter)
		if err != nil {
			return resp, fmt.Errorf("failed to get %s data: %w", collName, err)
		}

		outflowTxs = append(outflowTxs, txs...)
		totalOutAmount += amount
		totalOutCount += count

		// Merge hourly stats
		for hour, stat := range hourly {
			existing := outflowHourly[hour]
			existing.Amount += stat.Amount
			existing.Count += stat.Count
			outflowHourly[hour] = existing
		}
	}

	// Convert merged outflow hourly to slice
	var outflowHourlySlice []models.HourlyStat
	for hour, stat := range outflowHourly {
		outflowHourlySlice = append(outflowHourlySlice, models.HourlyStat{
			Hour:   hour,
			Amount: stat.Amount,
			Count:  stat.Count,
		})
	}

	// ================= RESPONSE ASSEMBLY =================
	return models.StatsResponse{
		TotalInflowCount:    totalInCount,
		TotalInflowAmount:   totalInAmount,
		TotalOutflowCount:   totalOutCount,
		TotalOutflowAmount:  totalOutAmount,
		InflowHourly:        convertHourlyMapToSlice(inflowHourly),
		OutflowHourly:       outflowHourlySlice,
		InflowTransactions:  inflowTxs,
		OutflowTransactions: outflowTxs,
	}, nil
}

// Helper function to process transactions for any collection
func (m *mongoStore) getTransactionData(ctx context.Context, coll *mongo.Collection, filter bson.D) (
	[]models.TransactionPoint,
	map[int]models.HourlyStat,
	float64,
	int,
	error,
) {
	var txs []models.TransactionPoint
	hourlyStats := make(map[int]models.HourlyStat)
	totalAmount := 0.0
	totalCount := 0

	// Get raw transactions
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &txs); err != nil {
		return nil, nil, 0, 0, err
	}

	// Process transactions for hourly stats and totals
	for _, tx := range txs {
		hour := tx.CreatedAt.Hour()
		stat := hourlyStats[hour]
		stat.Amount += tx.Amount
		stat.Count++
		hourlyStats[hour] = stat

		totalAmount += tx.Amount
		totalCount++
	}

	return txs, hourlyStats, totalAmount, totalCount, nil
}

// Helper to convert hourly map to sorted slice
func convertHourlyMapToSlice(hourlyMap map[int]models.HourlyStat) []models.HourlyStat {
	var slice []models.HourlyStat
	for hour, stat := range hourlyMap {
		slice = append(slice, models.HourlyStat{
			Hour:   hour,
			Amount: stat.Amount,
			Count:  stat.Count,
		})
	}
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].Hour < slice[j].Hour
	})
	return slice
}
