package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

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
	db     *sql.DB
	logger *zap.Logger
}

func NewSQLConn(sshClient *ssh.Client, logger *zap.Logger) (*SqlStore, error) {
	// Register SSH tunnel dialer for MySQL
	mysql.RegisterDialContext("tcp+ssh", func(_ context.Context, addr string) (net.Conn, error) {
		return sshClient.Dial("tcp", dbHost)
	})

	dsn := fmt.Sprintf("%s:%s@tcp+ssh(%s)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql open: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("mysql ping: %w", err)
	}

	log.Println("Connected to MySQL via SSH tunnel")
	return &SqlStore{
		db:     db,
		logger: logger,
	}, nil
}
