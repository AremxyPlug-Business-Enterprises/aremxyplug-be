package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aremxyplug-be/config"
	"github.com/aremxyplug-be/db/mongo"
	elect "github.com/aremxyplug-be/lib/bills/electricity"
	"github.com/aremxyplug-be/lib/bills/tvsub"
	"github.com/aremxyplug-be/lib/emailclient/postmark"
	zapLogger "github.com/aremxyplug-be/lib/logger"
	otpgen "github.com/aremxyplug-be/lib/otp_gen"
	vtu "github.com/aremxyplug-be/lib/telcom/airtime"
	"github.com/aremxyplug-be/lib/telcom/data"
	"github.com/aremxyplug-be/lib/telcom/edu"
	httpSrv "github.com/aremxyplug-be/server/http"
	"go.uber.org/zap"
)

func main() {
	logger := zapLogger.New()
	secrets := config.GetSecrets()

	// Get data store
	store, client, err := mongo.New(secrets.MongdbUrl, secrets.DbName, logger)
	if err != nil {
		logger.Fatal("failed to open mongodb", zap.Error(err))
	}

	// setup email client
	emailClient := postmark.New(secrets)
	otp := otpgen.NewOTP(store)
	data := data.NewData(store, logger)
	edu := edu.NewEdu(store, logger)
	vtu := vtu.NewAirtimeConn(logger, store)
	tvSub := tvsub.NewTvConn(store, logger)
	electSub := elect.NewElectricConn(store, logger)

	config := httpSrv.ServerConfig{
		Store:       store,
		EmailClient: emailClient,
		Logger:      logger,
		Secrets:     secrets,
		DataClient:  data,
		EduClient:   edu,
		Vtu:         vtu,
		TvSub:       tvSub,
		ElectSub:    electSub,
		Otp:         otp,
	}

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

// corsMiddleware handles the CORS middleware
