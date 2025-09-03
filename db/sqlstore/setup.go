package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

var (
	dbUser     = os.Getenv("SQL_DB_USER")
	dbPassword = os.Getenv("SQL_DB_PASSWORD")
	dbHost     = os.Getenv("SQL_DB_HOST")
	dbName     = os.Getenv("SQL_DB_NAME")
)

type SqlStore struct {
	db         *sql.DB
	logger     *zap.Logger
	sshFactory func() (*ssh.Client, error)
	mu         sync.Mutex
}

func NewSQLConn(sshFactory func() (*ssh.Client, error), logger *zap.Logger) (*SqlStore, error) {
	store := &SqlStore{
		logger:     logger,
		sshFactory: sshFactory,
	}

	if err := store.connect(); err != nil {
		return nil, err
	}

	// start one keepalive goroutine
	go store.keepAlive()

	log.Println("Connected to MySQL via SSH tunnel")
	return store, nil
}

func (s *SqlStore) connect() error {
	sshClient, err := s.sshFactory()
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}

	mysql.RegisterDialContext("tcp+ssh", func(_ context.Context, addr string) (net.Conn, error) {
		return sshClient.Dial("tcp", dbHost)
	})

	dsn := fmt.Sprintf("%s:%s@tcp+ssh(%s)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("mysql open: %w", err)
	}

	db.SetConnMaxLifetime(2 * time.Minute)
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("mysql ping: %w", err)
	}

	s.db = db
	return nil
}

func (s *SqlStore) keepAlive() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := s.db.Ping(); err != nil {
			s.logger.Warn("MySQL keep-alive ping failed, reconnecting...", zap.Error(err))
			if err := s.reconnect(); err != nil {
				s.logger.Error("Reconnection failed", zap.Error(err))
			}
		}
	}
}

func (s *SqlStore) reconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		_ = s.db.Close()
	}

	return s.connect()
}
