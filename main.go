package main

import (
	"context"
	"fmt"
	"github.com/aremxyplug-be/config"
	"github.com/aremxyplug-be/db/mongo"
	"github.com/aremxyplug-be/lib/emailclient/postmark"
	zapLogger "github.com/aremxyplug-be/lib/logger"
	httpSrv "github.com/aremxyplug-be/server/http"
	"go.uber.org/zap"
	"net/http"
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

	httpRouter := httpSrv.MountServer(logger, store, secrets, emailClient)
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
