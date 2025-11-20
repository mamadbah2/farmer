package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.uber.org/zap"

	"github.com/mamadbah2/farmer/internal/config"
	"github.com/mamadbah2/farmer/internal/server/handlers"
	"github.com/mamadbah2/farmer/internal/server/router"
	whatsappsvc "github.com/mamadbah2/farmer/internal/service/whatsapp"
	whatsappclient "github.com/mamadbah2/farmer/pkg/clients/whatsapp"
	"github.com/mamadbah2/farmer/pkg/logger"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		panic(err)
	}

	baseLogger := logger.Must(logger.New())
	defer func() { _ = baseLogger.Sync() }()

	zap.ReplaceGlobals(baseLogger)

	whatsClient := whatsappclient.NewClient(cfg.WhatsApp)
	messagingSvc := whatsappsvc.NewMetaWhatsAppService(cfg.WhatsApp, whatsClient, baseLogger.Named("svc.whatsapp"))
	webhookHandler := handlers.NewWebhookHandler(messagingSvc, baseLogger.Named("handlers.whatsapp"))
	engine := router.New(webhookHandler, baseLogger.Named("router"))

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		baseLogger.Info("server starting", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			baseLogger.Fatal("http server crashed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	baseLogger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		baseLogger.Error("graceful shutdown failed", zap.Error(err))
	}
}
