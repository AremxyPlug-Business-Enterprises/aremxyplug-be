package sqlstore

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

var (
	dbUser     = os.Getenv("SQL_DB_USER")
	dbPassword = os.Getenv("SQL_DB_PASSWORD")
	dbHost     = os.Getenv("SQL_DB_HOST")
	dbPort     = os.Getenv("SQL_DB_PORT")
	dbName     = os.Getenv("SQL_DB_NAME")
	dbSSLMode  = os.Getenv("SQL_DB_SSLMODE")
)

type SqlStore struct {
	db     *sql.DB
	logger *zap.Logger
	mu     sync.Mutex
}

func NewSQLConn(logger *zap.Logger) (*SqlStore, error) {
	store := &SqlStore{
		logger: logger,
	}

	if err := store.connect(); err != nil {
		return nil, err
	}

	// start one keepalive goroutine
	go store.keepAlive()

	log.Println("Connected to PostgreSQL")
	return store, nil
}

func (s *SqlStore) connect() error {
	// Set default port if not provided
	port := dbPort
	if port == "" {
		port = "5432"
	}

	sslMode := dbSSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, port, dbName, sslMode)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("postgres open: %w", err)
	}

	db.SetConnMaxLifetime(2 * time.Minute)
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return fmt.Errorf("postgres ping: %w", err)
	}

	s.mu.Lock()
	s.db = db
	s.mu.Unlock()

	return nil
}

func (s *SqlStore) keepAlive() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		db := s.db
		s.mu.Unlock()

		if db == nil {
			continue
		}

		if err := db.Ping(); err != nil {
			s.logger.Warn("PostgreSQL keep-alive ping failed, reconnecting...", zap.Error(err))
			if err := s.reconnect(); err != nil {
				s.logger.Error("Reconnection failed", zap.Error(err))
			}
		}
	}
}

func (s *SqlStore) reconnect() error {
	s.mu.Lock()
	if s.db != nil {
		_ = s.db.Close()
		s.db = nil
	}
	s.mu.Unlock()

	return s.connect()
}
