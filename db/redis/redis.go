package redis

import (
	"context"
	"errors"
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

// Set key without TTL
func (r *RedisConn) Set(key string, value interface{}) error {
	ctx := context.Background()
	return r.client.Set(ctx, key, value, 0).Err()
}

// Set key with TTL
func (r *RedisConn) SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	ctx := context.Background()
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Get key
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
	// Return the value as an interface{}
	r.logger.Info("Returning value from Redis", zap.String("key", key), zap.String("value", val))
	return val, nil
}

// Increment with TTL if first time
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

// Delete key
func (r *RedisConn) Del(key string) error {
	ctx := context.Background()
	r.logger.Info("Deleting key from Redis", zap.String("key", key))
	return r.client.Del(ctx, key).Err()
}

// Check if key exists
func (r *RedisConn) Exists(key string) (bool, error) {
	ctx := context.Background()
	n, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
