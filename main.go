package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aremxyplug-be/config"
	"github.com/aremxyplug-be/db/mongo"
	"github.com/aremxyplug-be/db/redis"
	"github.com/aremxyplug-be/db/sqlstore"
	"github.com/aremxyplug-be/lib/auth"
	auth_pin "github.com/aremxyplug-be/lib/auth/pin"
	bankacc "github.com/aremxyplug-be/lib/bank/bank_acc"
	"github.com/aremxyplug-be/lib/bank/deposit"
	"github.com/aremxyplug-be/lib/bank/transactions"
	"github.com/aremxyplug-be/lib/bank/transfer"
	elect "github.com/aremxyplug-be/lib/bills/electricity"
	"github.com/aremxyplug-be/lib/bills/tvsub"
	"github.com/aremxyplug-be/lib/emailclient/postmark"
	"github.com/aremxyplug-be/lib/events"
	zapLogger "github.com/aremxyplug-be/lib/logger"
	otpgen "github.com/aremxyplug-be/lib/otp_gen"
	pointredeem "github.com/aremxyplug-be/lib/point-redeem"
	"github.com/aremxyplug-be/lib/scheduler"
	"github.com/aremxyplug-be/lib/smsclient/termii"
	"github.com/aremxyplug-be/lib/tasks"
	vtu "github.com/aremxyplug-be/lib/telcom/airtime"
	"github.com/aremxyplug-be/lib/telcom/data"
	"github.com/aremxyplug-be/lib/telcom/edu"
	"github.com/aremxyplug-be/lib/verification"
	"github.com/aremxyplug-be/lib/verification/bvn"
	"github.com/aremxyplug-be/lib/verification/nin"
	httpSrv "github.com/aremxyplug-be/server/http"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

var (
	sshUser     = os.Getenv("SSH_USER")
	sshHost     = os.Getenv("SSH_HOST")
	sshPassword = os.Getenv("SSH_PASSWORD")
)

func main() {
	logger := zapLogger.New()
	secrets := config.GetSecrets()
	bvnConfig := bvn.NewBvnConfig(logger)
	ninConfig := nin.NewNINConfig(logger)

	// Get data store
	store, client, err := mongo.New(secrets.MongdbUrl, secrets.DbName, logger)
	if err != nil {
		logger.Fatal("failed to open mongodb", zap.Error(err))
	}

	redisClient, err := redis.NewRedisConn(logger)
	if err != nil {
		logger.Fatal("failed to connect to redis", zap.Error(err))
	}

	sshFactory := func() (*ssh.Client, error) {
		return createSSHClient()
	}

	sqlStore, err := sqlstore.NewSQLConn(sshFactory, logger)
	if err != nil {
		logger.Fatal("failed to create SQL connection", zap.Error(err))
	}

	// setup email client
	emailClient := postmark.New(secrets)
	verifyClient := verification.NewVerificationClient(bvnConfig, ninConfig)
	otp := otpgen.NewOTP(store, logger)
	data := data.NewData(store, logger)
	edu := edu.NewEdu(store, logger)
	vtu := vtu.NewAirtimeConn(store, logger)
	tvSub := tvsub.NewTvConn(store, logger)
	electSub := elect.NewElectricConn(store, logger)
	virtualAcc := bankacc.NewBankConfig(store, logger)
	bankTransc := transactions.NewTransaction(store)
	bankTrf := transfer.NewConfig(store, logger, redisClient)
	point := pointredeem.NewPointConfig(store)
	pin := auth_pin.NewPinConfig(logger, store)
	sms := termii.NewSMSConn(store, logger)
	auth := auth.NewAuthConn(secrets, redisClient)
	scheduler := scheduler.NewScheduler(redisClient, store, logger)
	taskService := tasks.NewService(store)
	processor := events.NewProcessor(redisClient, taskService, logger)
	bankDep := deposit.NewDepositConfig(store, logger, processor)

	config := httpSrv.ServerConfig{
		Store:        store,
		SqlStore:     sqlStore,
		EmailClient:  emailClient,
		Logger:       logger,
		Secrets:      secrets,
		DataClient:   data,
		EduClient:    edu,
		Vtu:          vtu,
		TvSub:        tvSub,
		ElectSub:     electSub,
		Otp:          otp,
		Auth:         auth,
		VirtualAcc:   virtualAcc,
		BankTranc:    bankTransc,
		BankTrf:      bankTrf,
		BankDep:      bankDep,
		Point:        point,
		Pin:          pin,
		SmsClient:    sms,
		VerifyClient: verifyClient,
		RedisClient:  redisClient,
		Processor:    processor,
	}

	// Initialize bank list if needed
	if err := scheduler.ListBanks(); err != nil {
		logger.Fatal("failed to initialize bank list", zap.Error(err))
	}

	go scheduler.StartBankListScheduler(24 * time.Hour)
	go scheduler.HarmonizeHoldsWithDB(10 * time.Minute)

	httpRouter := httpSrv.MountServer(config)
	// Start HTTP server
	httpAddr := fmt.Sprintf(":%s", secrets.AppPort)
	logger.Info(fmt.Sprintf("HTTP service running on %v.", httpAddr))
	if err := http.ListenAndServe(httpAddr, httpRouter); err != nil {
		logger.With(zap.Error(err)).Fatal("start http server")
	}
	logger.Info("closing application...")
	if err := client.Disconnect(context.Background()); err != nil {
		logger.Fatal("failed to disconnect from database", zap.Error(err))
	}
}

func createSSHClient() (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(sshPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	sshClient, err := ssh.Dial("tcp", sshHost, config)
	if err != nil {
		return nil, fmt.Errorf("SSH connection failed: %w", err)
	}

	// Start SSH keep-alive goroutine
	go func(client *ssh.Client) {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				fmt.Printf("SSH keepalive failed: %v\n", err)
				return // tunnel is dead → SqlStore.reconnect() will build a new one
			}
		}
	}(sshClient)

	return sshClient, nil
}
