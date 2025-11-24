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
	"github.com/shopspring/decimal"
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

func NewRedisConn(logger *zap.Logger) (*RedisConn, error) {
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
		logger.Error("Failed to connect to Redis", zap.Error(err))
		return nil, err
	} else {
		logger.Info("Redis connected successfully")
	}

	return &RedisConn{
		client: client,
		logger: logger,
	}, nil
}

// Client returns the underlying *redis.Client.
func (r *RedisConn) Client() *redis.Client {
	return r.client
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
	r.logger.Info("Key retrieved from Redis", zap.String("key", key))
	if err != nil {
		if errors.Is(err, redis.Nil) {
			r.logger.Info("Key does not exist in Redis", zap.String("key", key))
			return nil, nil
		}
		return nil, err
	}
	// Return the value (string) as interface{}
	r.logger.Info("Returning value from Redis", zap.String("key", key))
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
	if not amt then
		return -2
	end
	if bal < amt then
		return -1
	end
	redis.call("INCRBYFLOAT", KEYS[1], -amt)
	redis.call("SET", KEYS[2], ARGV[1], "EX", ARGV[2])
	return tonumber(redis.call("GET", KEYS[1]))
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
	redis.call("INCRBYFLOAT", KEYS[1], val)
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

// Lua script to sum all holds for a given user
var sumHoldsScript = redis.NewScript(`
    local cursor = "0"
    local total = 0
    repeat
        local result = redis.call("SCAN", cursor, "MATCH", KEYS[1], "COUNT", 100)
        cursor = result[1]
        local keys = result[2]
        for i, key in ipairs(keys) do
            local val = redis.call("GET", key)
            if val then
                total = total + tonumber(val)
            end
        end
    until cursor == "0"
    return tostring(total) -- return as string to avoid precision loss
`)

func (r *RedisConn) Close() error {
	return r.client.Close()
}

// InitializeBalance sets the user's balance in Redis (lowest unit)
func (r *RedisConn) InitializeBalance(userID string, amount decimal.Decimal) error {
	ctx := context.Background()
	key := fmt.Sprintf("balance:%s", userID)
	return r.client.Set(ctx, key, amount.String(), time.Hour*1).Err()
}

// GetBalance returns integer balance (available balance; holds are separate)
func (r *RedisConn) GetBalance(userID string) (decimal.Decimal, error) {
	ctx := context.Background()
	key := fmt.Sprintf("balance:%s", userID)
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return decimal.Zero, err
	}
	return decimal.NewFromString(val)
}

func (r *RedisConn) HoldFunds(userID, txID string, currentBalance decimal.Decimal, amount decimal.Decimal, ttl time.Duration) (decimal.Decimal, error) {
	ctx := context.Background()
	balanceKey := fmt.Sprintf("balance:%s", userID)
	holdKey := fmt.Sprintf("hold:%s:%s", userID, txID)

	// Check if balance exists first
	exists, err := r.client.Exists(ctx, balanceKey).Result()
	if err != nil {
		r.logger.Error("Failed to check balance existence in Redis", zap.Error(err))
		return decimal.Zero, err
	}

	// Initialize with provided balance if the key doesn't exist
	if exists == 0 {
		r.logger.Warn("Balance key does not exist, initializing to balance", zap.String("userID", userID))
		if err := r.InitializeBalance(userID, currentBalance); err != nil {
			return decimal.Zero, err
		}
	}

	// Run LUA script
	res, err := holdScript.Run(
		ctx,
		r.client,
		[]string{balanceKey, holdKey},
		amount.String(),
		int(ttl.Seconds()),
	).Result()

	if err != nil {
		r.logger.Error("Failed to hold funds in Redis", zap.Error(err))
		return decimal.Zero, err
	}

	switch v := res.(type) {

	case float64:
		iv := int64(v)
		if iv < 0 {
			switch iv {
			case -1:
				return decimal.Zero, errors.New("insufficient funds")
			case -2:
				return decimal.Zero, errors.New("invalid amount")
			default:
				return decimal.Zero, fmt.Errorf("script returned error code: %d", iv)
			}
		}
		return decimal.NewFromFloat(v), nil

	case int64:
		if v < 0 {
			switch v {
			case -1:
				return decimal.Zero, errors.New("insufficient funds")
			case -2:
				return decimal.Zero, errors.New("invalid amount")
			default:
				return decimal.Zero, fmt.Errorf("script returned error code: %d", v)
			}
		}
		return decimal.NewFromInt(v), nil

	case string:
		// Script might return a string if Redis coerces it
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return decimal.NewFromFloat(f), nil
		}
		// If parsing fails, fallback to reading the balance directly
		valStr, err := r.client.Get(ctx, balanceKey).Result()
		if err != nil {
			return decimal.Zero, err
		}
		return decimal.NewFromString(valStr)

	case nil:
		return decimal.Zero, errors.New("script returned nil")

	default:
		return decimal.Zero, fmt.Errorf("invalid return type from script: %T", v)
	}
}

// ReleaseHold increments balance by hold amount and deletes hold. Returns true if hold existed.
func (r *RedisConn) ReleaseHold(userID, txnID string) (bool, error) {
	ctx := context.Background()
	balanceKey := fmt.Sprintf("balance:%s", userID)
	holdKey := fmt.Sprintf("hold:%s:%s", userID, txnID)
	res, err := releaseScript.Run(ctx, r.client, []string{balanceKey, holdKey}).Result()
	if err != nil {
		r.logger.Error("Failed to release hold in Redis", zap.Error(err))
		return false, err
	}
	if n, ok := res.(int64); ok && n == 1 {
		r.logger.Info("Hold released in Redis", zap.String("userID", userID), zap.String("txnID", txnID))
		return true, nil
	}
	r.logger.Warn("Failed to release hold in Redis: not found", zap.String("userID", userID), zap.String("txnID", txnID))
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

func (r *RedisConn) GetHeldFundsLua(userID string) (float64, error) {
	ctx := context.Background()
	pattern := fmt.Sprintf("hold:%s:*", userID)

	res, err := sumHoldsScript.Run(ctx, r.client, []string{pattern}).Result()
	if err != nil {
		return 0, err
	}

	switch v := res.(type) {
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, err
		}
		return f, nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, nil
	}
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

func (r *RedisConn) AddToBalance(userID string, amount decimal.Decimal) error {
	ctx := context.Background()
	key := fmt.Sprintf("balance:%s", userID)
	f, exact := amount.Float64()
	if !exact {
		// log warning if loss may occur
		r.logger.Warn("Converting decimal to float64 may lose precision", zap.String("amount", amount.String()))
	}
	return r.client.IncrByFloat(ctx, key, f).Err()
}

func (r *RedisConn) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int64, error) {
	// increment counter
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	if count == 1 {
		r.client.Expire(ctx, key, window)
	}

	allowed := count <= int64(limit)
	return allowed, count, nil
}

// PublishUserEvent publishes a JSON event to a user-specific Redis channel.
func (r *RedisConn) PublishUserEvent(ctx context.Context, userID string, event interface{}) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	channel := fmt.Sprintf("transfer:events:%s", userID)
	// short timeout so publish doesn't block
	ctxPub, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return r.client.Publish(ctxPub, channel, b).Err()
}

// SubscribeUserEvents subscribes to a user's event channel and returns a channel of JSON payloads.
// Caller cancels ctx to stop the subscription; the returned channel will be closed.
func (r *RedisConn) SubscribeUserEvents(ctx context.Context, userID string) (<-chan string, error) {
	out := make(chan string)
	channel := fmt.Sprintf("transfer:events:%s", userID)
	pubsub := r.client.Subscribe(ctx, channel)

	// ensure subscription established
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, err
	}

	ch := pubsub.Channel()

	go func() {
		defer close(out)
		defer pubsub.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case m, ok := <-ch:
				if !ok {
					return
				}
				out <- m.Payload
			}
		}
	}()

	return out, nil
}
