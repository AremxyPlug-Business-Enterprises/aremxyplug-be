package termii

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	rdb "github.com/aremxyplug-be/db/redis"

	"go.uber.org/zap"
)

var (
	whatsAppBaseURL     = os.Getenv("TERMII_WHATSAPP_BASE_URL")
	whatsAppSender      = os.Getenv("TERMII_WHATSAPP_SENDER")
	whatsAppChannel     = os.Getenv("TERMII_WHATSAPP_CHANNEL")
	whatsAppType        = os.Getenv("TERMII_WHATSAPP_TYPE")
	kudiSMSBaseURL      = os.Getenv("KUDISMS_WHATSAPP_BASE_URL")
	kudiSMSAPIKey       = os.Getenv("KUDISMS_API_KEY")
	kudiSMSTemplateCode = os.Getenv("KUDISMS_WHATSAPP_TEMPLATE_CODE")
)

type WhatsAppClient struct {
	logger *zap.Logger
	redis  *rdb.RedisConn
}

func NewWhatsAppClient(redisConn *rdb.RedisConn, logger *zap.Logger) *WhatsAppClient {
	w := &WhatsAppClient{logger: logger, redis: redisConn}

	// Read tuning env vars (optional) with sensible defaults
	workers := 3
	if v := os.Getenv("KUDISMS_FALLBACK_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			workers = n
		}
	}

	queueSize := 200
	if v := os.Getenv("KUDISMS_FALLBACK_QUEUE_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			queueSize = n
		}
	}

	maxAttempts := 5
	if v := os.Getenv("KUDISMS_FALLBACK_MAX_ATTEMPTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxAttempts = n
		}
	}

	baseBackoffMs := 1000
	if v := os.Getenv("KUDISMS_FALLBACK_BASE_BACKOFF_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			baseBackoffMs = n
		}
	}

	jobTTL := 600
	if v := os.Getenv("KUDISMS_FALLBACK_JOB_TTL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			jobTTL = n
		}
	}

	// set job TTL package var before starting
	fallbackJobTTLSeconds = jobTTL
	startFallbackProcessor(w, logger, workers, queueSize, maxAttempts, time.Duration(baseBackoffMs)*time.Millisecond)

	return w
}

type fallbackJob struct {
	OTP        string
	Phone      string
	Attempts   int
	CreatedAt  time.Time `json:"created_at"`
	TTLSeconds int       `json:"ttl_seconds"`
}

var (
	fallbackQueue     chan fallbackJob
	fallbackOnce      sync.Once
	fallbackEnqueued  uint64
	fallbackDropped   uint64
	fallbackSucceeded uint64
	fallbackFailed    uint64
	// tunable parameters (defaults set in startFallbackProcessor)
	fallbackMaxAttempts      int
	fallbackBaseBackoff      time.Duration
	fallbackJobTTLSeconds    int
	kudismsFallbackQueueName = "kudisms:fallback:queue"
)

func startFallbackProcessor(client *WhatsAppClient, logger *zap.Logger, workers int, queueSize int, maxAttempts int, baseBackoff time.Duration) {
	fallbackOnce.Do(func() {
		fallbackQueue = make(chan fallbackJob, queueSize)
		// set package-level tuning variables
		fallbackMaxAttempts = maxAttempts
		fallbackBaseBackoff = baseBackoff

		// If Redis connection is provided, start Redis-backed workers
		if client != nil && client.redis != nil {
			for i := 0; i < workers; i++ {
				go redisFallbackWorker(client, logger)
			}
			logger.Info("started redis-backed fallback workers", zap.Int("workers", workers), zap.String("queue", kudismsFallbackQueueName))
			return
		}

		// otherwise start in-memory workers
		for i := 0; i < workers; i++ {
			go fallbackWorker(client, logger)
		}
	})
}

func redisFallbackWorker(client *WhatsAppClient, logger *zap.Logger) {
	if client == nil || client.redis == nil {
		logger.Warn("redisFallbackWorker started without redis client")
		return
	}
	for {
		// blocking pop from Redis list
		jobJSON, err := client.redis.PopJobBlocking(context.Background(), kudismsFallbackQueueName, 0)
		if err != nil {
			logger.Error("redis pop job failed", zap.Error(err))
			time.Sleep(2 * time.Second)
			continue
		}
		if jobJSON == "" {
			continue
		}
		var job fallbackJob
		if err := json.Unmarshal([]byte(jobJSON), &job); err != nil {
			logger.Error("failed to unmarshal redis fallback job", zap.Error(err))
			continue
		}

		// drop job if expired
		if job.TTLSeconds > 0 && time.Since(job.CreatedAt) > time.Duration(job.TTLSeconds)*time.Second {
			atomic.AddUint64(&fallbackDropped, 1)
			logger.Warn("redis fallback job expired; dropping", zap.String("phone", job.Phone))
			continue
		}

		attempt := job.Attempts
		for attempt < fallbackMaxAttempts {
			logger.Info("redis fallback worker attempt", zap.Int("attempt", attempt), zap.String("phone", job.Phone))
			err := client.SendWhatsAppToken(context.Background(), job.OTP, job.Phone)
			if err == nil {
				atomic.AddUint64(&fallbackSucceeded, 1)
				logger.Info("redis fallback worker succeeded", zap.String("phone", job.Phone))
				break
			}
			attempt++
			sleep := fallbackBaseBackoff * time.Duration(1<<attempt)
			logger.Warn("redis fallback attempt failed; backing off", zap.Int("attempt", attempt), zap.Duration("sleep", sleep), zap.Error(err))
			time.Sleep(sleep)
		}
		if attempt >= fallbackMaxAttempts {
			atomic.AddUint64(&fallbackFailed, 1)
			logger.Error("redis fallback worker exhausted max attempts", zap.String("phone", job.Phone))
		}
	}
}

func enqueueFallback(client *WhatsAppClient, otp, phone string) {
	logger := client.logger

	// If Redis is configured, persist job to Redis queue
	if client != nil && client.redis != nil {
		job := fallbackJob{OTP: otp, Phone: phone, Attempts: 0, CreatedAt: time.Now().UTC(), TTLSeconds: fallbackJobTTLSeconds}
		if err := client.redis.PutJob(context.Background(), kudismsFallbackQueueName, job); err != nil {
			atomic.AddUint64(&fallbackDropped, 1)
			logger.Error("failed to put fallback job to redis; dropping", zap.Error(err), zap.String("phone", phone))
			return
		}
		atomic.AddUint64(&fallbackEnqueued, 1)
		logger.Info("enqueued fallback job to redis", zap.String("phone", phone))
		return
	}

	// fall back to in-memory queue
	if fallbackQueue == nil {
		logger.Warn("fallback queue not initialized; attempting immediate fallback")
		_ = (&WhatsAppClient{logger: logger}).SendWhatsAppToken(context.Background(), otp, phone)
		return
	}

	select {
	case fallbackQueue <- fallbackJob{OTP: otp, Phone: phone, Attempts: 0, CreatedAt: time.Now().UTC(), TTLSeconds: fallbackJobTTLSeconds}:
		atomic.AddUint64(&fallbackEnqueued, 1)
		logger.Info("enqueued fallback job for Termii sender", zap.String("phone", phone))
	default:
		atomic.AddUint64(&fallbackDropped, 1)
		logger.Warn("fallback queue full; dropping fallback job", zap.String("phone", phone))
	}
}

func fallbackWorker(client *WhatsAppClient, logger *zap.Logger) {
	for job := range fallbackQueue {
		// drop job if expired
		if job.TTLSeconds > 0 && time.Since(job.CreatedAt) > time.Duration(job.TTLSeconds)*time.Second {
			atomic.AddUint64(&fallbackDropped, 1)
			logger.Warn("in-memory fallback job expired; dropping", zap.String("phone", job.Phone))
			continue
		}

		attempt := job.Attempts
		for attempt < fallbackMaxAttempts {
			logger.Info("fallback worker attempt", zap.Int("attempt", attempt), zap.String("phone", job.Phone))
			err := client.SendWhatsAppToken(context.Background(), job.OTP, job.Phone)
			if err == nil {
				atomic.AddUint64(&fallbackSucceeded, 1)
				logger.Info("fallback worker succeeded", zap.String("phone", job.Phone))
				break
			}

			attempt++
			sleep := fallbackBaseBackoff * time.Duration(1<<attempt)
			logger.Warn("fallback attempt failed; backing off", zap.Int("attempt", attempt), zap.Duration("sleep", sleep), zap.Error(err))
			time.Sleep(sleep)
		}
		if attempt >= fallbackMaxAttempts {
			atomic.AddUint64(&fallbackFailed, 1)
			logger.Error("fallback worker exhausted max attempts", zap.String("phone", job.Phone))
		}
	}
}

type FallbackMetrics struct {
	Enqueued  uint64 `json:"enqueued"`
	Dropped   uint64 `json:"dropped"`
	Succeeded uint64 `json:"succeeded"`
	Failed    uint64 `json:"failed"`
}

// GetFallbackMetrics returns current fallback queue metrics.
func GetFallbackMetrics() FallbackMetrics {
	return FallbackMetrics{
		Enqueued:  atomic.LoadUint64(&fallbackEnqueued),
		Dropped:   atomic.LoadUint64(&fallbackDropped),
		Succeeded: atomic.LoadUint64(&fallbackSucceeded),
		Failed:    atomic.LoadUint64(&fallbackFailed),
	}
}

func (w *WhatsAppClient) SendWhatsAppToken(ctx context.Context, otp, phone string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	w.logger.Info("Sending WhatsApp OTP with Termii", zap.String("phone", phone))

	payload := whatsAppRequest{
		APIKey:  apiKey,
		To:      phone,
		From:    whatsAppSender,
		SMS:     otp,
		Channel: whatsAppChannel,
		Type:    whatsAppType,
	}

	requestBody, err := json.Marshal(payload)
	if err != nil {
		w.logger.Error("Failed to marshal WhatsApp payload", zap.Error(err))
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, whatsAppBaseURL, bytes.NewReader(requestBody))
	if err != nil {
		w.logger.Error("Failed to create WhatsApp request", zap.Error(err))
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		w.logger.Error("Failed to send WhatsApp request", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		w.logger.Error("Failed to read WhatsApp response body", zap.Error(err))
		return err
	}

	w.logger.Info("WhatsApp response received", zap.Int("status_code", resp.StatusCode), zap.String("body", string(bodyBytes)))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("termii whatsapp send failed with status %d", resp.StatusCode)
	}

	apiResponse := whatsAppResponse{}
	if err := json.NewDecoder(bytes.NewBuffer(bodyBytes)).Decode(&apiResponse); err != nil {
		w.logger.Error("Failed to decode WhatsApp response body", zap.Error(err))
		return err
	}

	if apiResponse.Code != "" && apiResponse.Code != "ok" {
		return fmt.Errorf("termii whatsapp send failed: %s", apiResponse.Message)
	}

	w.logger.Info("WhatsApp OTP sent successfully", zap.String("phone", phone), zap.String("message_id", apiResponse.MessageID))
	return nil
}

func (w *WhatsAppClient) SendWhatsAppTokenKudiSMS(ctx context.Context, otp, phone string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	baseURL := strings.TrimSpace(kudiSMSBaseURL)
	if baseURL == "" {
		baseURL = "https://my.kudisms.net/api/whatsapp"
	}

	token := strings.TrimSpace(kudiSMSAPIKey)
	if token == "" {
		return errors.New("kudisms api key is not configured")
	}

	templateCode := strings.TrimSpace(kudiSMSTemplateCode)
	if templateCode == "" {
		return errors.New("kudisms whatsapp template code is not configured")
	}

	w.logger.Info("Sending WhatsApp OTP with KudiSMS", zap.String("phone", phone), zap.String("template_code", templateCode))

	form := url.Values{}
	form.Set("token", token)
	form.Set("recipient", phone)
	form.Set("template_code", templateCode)
	form.Set("parameters", otp)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, strings.NewReader(form.Encode()))
	if err != nil {
		w.logger.Error("Failed to create KudiSMS WhatsApp request", zap.Error(err))
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		w.logger.Error("Failed to send KudiSMS WhatsApp request", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		w.logger.Error("Failed to read KudiSMS WhatsApp response body", zap.Error(err))
		return err
	}

	w.logger.Info("KudiSMS WhatsApp response received", zap.Int("status_code", resp.StatusCode), zap.String("body", string(bodyBytes)))

	apiResponse := kudiSMSWhatsAppResponse{}
	if err := json.NewDecoder(bytes.NewBuffer(bodyBytes)).Decode(&apiResponse); err != nil {
		w.logger.Error("Failed to decode KudiSMS WhatsApp response body", zap.Error(err))
		return err
	}

	if resp.StatusCode != http.StatusOK {
		if strings.TrimSpace(apiResponse.Msg) != "" {
			return fmt.Errorf("kudisms whatsapp send failed with status %d: %s", resp.StatusCode, apiResponse.Msg)
		}
		return fmt.Errorf("kudisms whatsapp send failed with status %d", resp.StatusCode)
	}

	if !strings.EqualFold(strings.TrimSpace(apiResponse.Status), "success") || strings.TrimSpace(apiResponse.ErrorCode) != "000" {
		// If KudiSMS returned authorization error (401), enqueue fallback to Termii sender
		if strings.TrimSpace(apiResponse.ErrorCode) == "401" {
			w.logger.Warn("KudiSMS returned 401; enqueuing fallback to Termii sender", zap.String("phone", phone))
			enqueueFallback(w, otp, phone)
			return nil
		}

		if strings.TrimSpace(apiResponse.Msg) != "" {
			return fmt.Errorf("kudisms whatsapp send failed: %s", apiResponse.Msg)
		}
		return fmt.Errorf("kudisms whatsapp send failed with status %s and error code %s", apiResponse.Status, apiResponse.ErrorCode)
	}

	w.logger.Info("KudiSMS WhatsApp OTP sent successfully", zap.String("phone", phone), zap.String("data", apiResponse.Data))
	return nil
}
