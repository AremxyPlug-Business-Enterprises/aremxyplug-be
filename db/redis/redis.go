package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	redisAddr     = os.Getenv("REDIS_ADDRESS")
	redisUsername = os.Getenv("REDIS_USERNAME")
	redisPassword = os.Getenv("REDIS_PASSWORD")
	redisDB       = os.Getenv("REDIS_DB")
)

type RedisConn struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisConn(logger *zap.Logger) *RedisConn {
	db, _ := strconv.Atoi(redisDB)

	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Username: redisUsername,
		Password: redisPassword,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	} else {
		logger.Info("Redis connected successfully")
	}

	return &RedisConn{
		client: client,
		logger: logger,
	}
}

// Set key without TTL (keeps existing behaviour)
func (r *RedisConn) Set(key string, value interface{}) error {
	ctx := context.Background()
	var v interface{} = value
	// if not string or []byte, marshal to json
	switch value.(type) {
	case string, []byte:
		// keep as is
	default:
		b, err := json.Marshal(value)
		if err == nil {
			v = b
		}
	}
	return r.client.Set(ctx, key, v, 0).Err()
}

// SetWithTTL sets key with TTL (keeps existing behaviour)
func (r *RedisConn) SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	ctx := context.Background()
	var v interface{} = value
	switch value.(type) {
	case string, []byte:
	default:
		b, err := json.Marshal(value)
		if err == nil {
			v = b
		}
	}
	return r.client.Set(ctx, key, v, ttl).Err()
}

// Get key - returns (nil, nil) when key missing (preserves original behaviour)
func (r *RedisConn) Get(key string) (interface{}, error) {
	ctx := context.Background()
	r.logger.Info("Getting key from Redis", zap.String("key", key))
	val, err := r.client.Get(ctx, key).Result()
	r.logger.Info("Key retrieved from Redis", zap.String("key", key), zap.String("value", val))
	if err != nil {
		if errors.Is(err, redis.Nil) {
			r.logger.Info("Key does not exist in Redis", zap.String("key", key))
			return nil, nil
		}
		return nil, err
	}
	// Return the value (string) as interface{}
	r.logger.Info("Returning value from Redis", zap.String("key", key), zap.String("value", val))
	return val, nil
}

// IncrWithTTL increments and sets TTL the first time (same behaviour)
func (r *RedisConn) IncrWithTTL(key string, ttl time.Duration) (int64, error) {
	ctx := context.Background()
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		r.client.Expire(ctx, key, ttl)
	}
	return count, nil
}

// Del key
func (r *RedisConn) Del(key string) error {
	ctx := context.Background()
	r.logger.Info("Deleting key from Redis", zap.String("key", key))
	return r.client.Del(ctx, key).Err()
}

// Exists checks if key exists
func (r *RedisConn) Exists(key string) (bool, error) {
	ctx := context.Background()
	n, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

var (
	// hold script:
	// KEYS[1] = balance:{user}
	// KEYS[2] = hold:{user}:{txid}
	// ARGV[1] = amount (integer)
	// ARGV[2] = holdTTLSeconds
	holdScript = redis.NewScript(`
	local bal = tonumber(redis.call("GET", KEYS[1]) or "0")
	local amt = tonumber(ARGV[1])
	if bal < amt then
		return -1
	end
	redis.call("DECRBY", KEYS[1], amt)
	redis.call("SET", KEYS[2], ARGV[1], "EX", ARGV[2])
	return bal - amt
`)

	// release script:
	// KEYS[1] = balance:{user}
	// KEYS[2] = hold:{user}:{txid}
	// returns 1 if released, 0 if no hold exists
	releaseScript = redis.NewScript(`
	local val = redis.call("GET", KEYS[2])
	if not val then
		return 0
	end
	redis.call("INCRBY", KEYS[1], val)
	redis.call("DEL", KEYS[2])
	return 1
`)

	// confirm script:
	// KEYS[1] = hold:{user}:{txid}
	// returns 1 deleted, 0 not present
	confirmScript = redis.NewScript(`
	local val = redis.call("GET", KEYS[1])
	if not val then
		return 0
	end
	redis.call("DEL", KEYS[1])
	return 1
`)
)

func (r *RedisConn) Close() error {
	return r.client.Close()
}

// SetInitialBalance sets the user's balance in Redis (lowest unit)
func (r *RedisConn) SetInitialBalance(userID string, amount float64) error {
	ctx := context.Background()
	key := fmt.Sprintf("balance:%s", userID)
	return r.client.Set(ctx, key, amount, 0).Err()
}

// GetBalance returns integer balance (available balance; holds are separate)
func (r *RedisConn) GetBalance(userID string) (float64, error) {
	ctx := context.Background()
	key := fmt.Sprintf("balance:%s", userID)
	return r.client.Get(ctx, key).Float64()
}

// HoldFunds atomically decrements balance and creates a hold key. Returns new available balance or error if insufficient funds.
func (r *RedisConn) HoldFunds(userID, txID string, amount float64, ttl time.Duration) (float64, error) {
	ctx := context.Background()
	balanceKey := fmt.Sprintf("balance:%s", userID)
	holdKey := fmt.Sprintf("hold:%s:%s", userID, txID)
	res, err := holdScript.Run(ctx, r.client, []string{balanceKey, holdKey}, amount, int(ttl.Seconds())).Result()
	if err != nil {
		return 0, err
	}
	switch v := res.(type) {
	case float64:
		if v < 0 {
			return 0, errors.New("insufficient funds")
		}
		return v, nil
	case string:
		// in some redis clients returns string
		// try to read balance directly
		return r.client.Get(ctx, balanceKey).Float64()
	default:
		return 0, nil
	}
}

// ReleaseHold increments balance by hold amount and deletes hold. Returns true if hold existed.
func (r *RedisConn) ReleaseHold(userID, txID string) (bool, error) {
	ctx := context.Background()
	balanceKey := fmt.Sprintf("balance:%s", userID)
	holdKey := fmt.Sprintf("hold:%s:%s", userID, txID)
	res, err := releaseScript.Run(ctx, r.client, []string{balanceKey, holdKey}).Result()
	if err != nil {
		return false, err
	}
	if n, ok := res.(int64); ok && n == 1 {
		return true, nil
	}
	return false, nil
}

// ConfirmHold deletes the hold without changing balance (finalize deduction)
func (r *RedisConn) ConfirmHold(userID, txID string) (bool, error) {
	ctx := context.Background()
	holdKey := fmt.Sprintf("hold:%s:%s", userID, txID)
	res, err := confirmScript.Run(ctx, r.client, []string{holdKey}).Result()
	if err != nil {
		return false, err
	}
	if n, ok := res.(int64); ok && n == 1 {
		return true, nil
	}
	return false, nil
}

// PutJob pushes a job JSON to a queue (left push)
func (r *RedisConn) PutJob(queue string, payload interface{}) error {
	ctx := context.Background()
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return r.client.LPush(ctx, queue, b).Err()
}

// Blocking pop (right pop) returns the raw job JSON
func (r *RedisConn) PopJobBlocking(queue string, timeout time.Duration) (string, error) {
	ctx := context.Background()
	// If timeout==0 use 0 -> block indefinitely
	res, err := r.client.BRPop(ctx, timeout, queue).Result()
	if err != nil {
		return "", err
	}
	// BRPop returns [queue, val]
	if len(res) < 2 {
		return "", nil
	}
	return res[1], nil
}

// Helper: set external mapping transfer:ext:{apiRef} -> txID
func (r *RedisConn) SetExternalMapping(apiRef, txID string, ttl time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("transfer:ext:%s", apiRef)
	return r.client.Set(ctx, key, txID, ttl).Err()
}

func (r *RedisConn) GetTxIDByExternalRef(apiRef string) (string, error) {
	ctx := context.Background()
	key := fmt.Sprintf("transfer:ext:%s", apiRef)
	return r.client.Get(ctx, key).Result()
}

// SetMeta stores transfer metadata (user_id, amount) as JSON at transfer:meta:{txID}
func (r *RedisConn) SetMeta(txID string, meta interface{}, ttl time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("transfer:meta:%s", txID)
	b, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, b, ttl).Err()
}

// GetMeta retrieves transfer meta and unmarshals into dest (pass pointer). returns (found, error)
func (r *RedisConn) GetMeta(txID string, dest interface{}) (bool, error) {
	ctx := context.Background()
	key := fmt.Sprintf("transfer:meta:%s", txID)
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, err
	}
	return true, nil
}
