package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aremxyplug-be/config"
	"github.com/aremxyplug-be/db/mongo"
	"github.com/aremxyplug-be/lib/emailclient/postmark"
	zapLogger "github.com/aremxyplug-be/lib/logger"
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
	data := data.NewData(store, logger)
	edu := edu.NewEdu(store, logger)

	httpRouter := httpSrv.MountServer(logger, store, secrets, emailClient, data, edu)
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
