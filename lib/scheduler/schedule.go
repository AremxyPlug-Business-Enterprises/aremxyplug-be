package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"github.com/aremxyplug-be/db"
	"github.com/aremxyplug-be/db/models"
	"github.com/aremxyplug-be/db/redis"
)

var (
	api    = os.Getenv("ANCHOR_API")
	apikey = os.Getenv("ANCHORAPI_PROD")
)

type Scheduler struct {
	redis  *redis.RedisConn
	store  db.DataStore
	logger *zap.Logger
}

func NewScheduler(redis *redis.RedisConn, store db.DataStore, logger *zap.Logger) *Scheduler {
	return &Scheduler{
		redis:  redis,
		store:  store,
		logger: logger,
	}
}

// HarmonizeHoldsWithDB scans Redis hold keys, checks for near-expiry, and updates MongoDB if needed.
// mongoUpdateFunc should be a function that takes (userID, txID string) and returns (wasHarmonized bool, err error)
func (s *Scheduler) HarmonizeHoldsWithDB(threshold time.Duration) {
	ctx := context.Background()
	pattern := "hold:*:*"
	var cursor uint64
	for {
		keys, nextCursor, err := s.redis.Client().Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			s.logger.Error("Failed to scan Redis for hold keys", zap.Error(err))
			return // Exit on scan error
		}
		for _, key := range keys {
			ttl, err := s.redis.Client().TTL(ctx, key).Result()
			if err != nil {
				s.logger.Warn("Failed to get TTL for hold key", zap.String("key", key), zap.Error(err))
				continue
			}
			if ttl > 0 && ttl <= threshold {
				// Parse userID and txID from key: hold:{userID}:{txID}
				var userID, txID string
				n, _ := fmt.Sscanf(key, "hold:%[^:]:%s", &userID, &txID)
				if n == 2 {
					err := s.store.UpdateRecieptByTxID(txID)
					if err != nil {
						s.logger.Error("Failed to harmonize transaction in DB", zap.String("userID", userID), zap.String("txID", txID), zap.Error(err))
					} else {
						// Release the hold in Redis after harmonization
						_, relErr := s.redis.ReleaseHold(userID, txID)
						if relErr != nil {
							s.logger.Warn("Failed to release hold after harmonization", zap.String("userID", userID), zap.String("txID", txID), zap.Error(relErr))
						}
						s.logger.Info("Harmonized pending transaction to failed and updated balance", zap.String("userID", userID), zap.String("txID", txID))
					}
				} else {
					s.logger.Warn("Could not parse hold key for harmonization", zap.String("key", key))
				}
			}
		}
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}
}

func (s *Scheduler) StartBankListScheduler(interval time.Duration) {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C

		// Random jitter between 0 and 10 minutes
		jitter := time.Duration(rand.Intn(10*60)) * time.Second
		s.logger.Info("Scheduled bank list sync will run after jitter", zap.Duration("delay", jitter))

		time.Sleep(jitter)

		s.logger.Info("Starting scheduled bank list sync...")
		if err := s.ListBanks(); err != nil {
			s.logger.Error("Scheduled bank list sync failed", zap.Error(err))
		} else {
			s.logger.Info("Scheduled bank list sync completed successfully")
		}
	}
}

type bankLists struct {
	BanksData []bankData `json:"data"`
}

type bankAttributes struct {
	NIPCode string `json:"nipCode"`
	Name    string `json:"name"`
}

type bankData struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Atrributes bankAttributes `json:"attributes"`
}

// this endpoint should auto automatically initialize
func (s *Scheduler) ListBanks() error {
	url := fmt.Sprintf("%s/%s", api, "banks")
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("accept", "application/json")
	req.Header.Add("x-anchor-key", apikey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}

	apiResponse := bankLists{}
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		s.logger.Error(err.Error())
		return err
	}

	// Track all NIP codes from API
	apiNIPCodes := make(map[string]bool)

	for _, bank := range apiResponse.BanksData {
		bankData := models.BankDetails{
			Name:    bank.Atrributes.Name,
			NIPCode: bank.Atrributes.NIPCode,
		}
		apiNIPCodes[bankData.NIPCode] = true

		// Use Upsert to prevent duplicate key issues and race conditions
		if err := s.store.UpsertBankByNIPCode(bankData); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				s.logger.Warn("Bank already exists (race condition), skipping insert",
					zap.String("nip_code", bankData.NIPCode))
				continue
			}
			return err
		}
	}

	// Remove banks not present in API anymore
	dbBanks, err := s.store.GetAllBanks()
	if err != nil {
		return err
	}

	for _, bank := range dbBanks {
		if !apiNIPCodes[bank.NIPCode] {
			if err := s.store.DeleteBankByNIPCode(bank.NIPCode); err != nil {
				return err
			}
			s.logger.Info("Removed outdated bank", zap.String("nip_code", bank.NIPCode))
		}
	}

	s.logger.Info("Bank list full sync completed.")
	return nil
}
