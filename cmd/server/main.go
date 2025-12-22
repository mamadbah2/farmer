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
	"github.com/mamadbah2/farmer/internal/repository/mongodb"
	"github.com/mamadbah2/farmer/internal/repository/sheets"
	"github.com/mamadbah2/farmer/internal/scheduler"
	"github.com/mamadbah2/farmer/internal/server/handlers"
	"github.com/mamadbah2/farmer/internal/server/router"
	commandsvc "github.com/mamadbah2/farmer/internal/service/commands"
	reportingsvc "github.com/mamadbah2/farmer/internal/service/reporting"
	whatsappsvc "github.com/mamadbah2/farmer/internal/service/whatsapp"
	"github.com/mamadbah2/farmer/pkg/clients/anthropic"
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

	sheetsRepo, err := sheets.NewGoogleSheetRepository(context.Background(), cfg.Sheets, baseLogger.Named("repo.sheets"))
	if err != nil {
		baseLogger.Fatal("failed to init sheets repository", zap.Error(err))
	}

	mongoRepo, err := mongodb.NewMongoDBRepository(context.Background(), cfg.MongoDB.URI, cfg.MongoDB.DBName)
	if err != nil {
		baseLogger.Fatal("failed to init mongodb repository", zap.Error(err))
	}
	defer func() {
		if err := mongoRepo.Close(context.Background()); err != nil {
			baseLogger.Error("failed to close mongodb connection", zap.Error(err))
		}
	}()

	reportingSvc := reportingsvc.NewService(sheetsRepo, mongoRepo, baseLogger.Named("svc.reporting"))
	commandDispatcher := commandsvc.NewService(sheetsRepo, mongoRepo, reportingSvc, baseLogger.Named("svc.commands"))

	// Initialize AI Client
	var aiClient anthropic.Client
	if cfg.AI.AnthropicKey != "" {
		aiClient = anthropic.NewClient(cfg.AI.AnthropicKey)
		baseLogger.Info("anthropic ai client enabled")
	} else {
		baseLogger.Warn("anthropic api key missing, natural language processing disabled")
	}

	whatsClient := whatsappclient.NewClient(cfg.WhatsApp)
	messagingSvc := whatsappsvc.NewMetaWhatsAppService(cfg.WhatsApp, whatsClient, aiClient, commandDispatcher, baseLogger.Named("svc.whatsapp"))
	webhookHandler := handlers.NewWebhookHandler(messagingSvc, baseLogger.Named("handlers.whatsapp"))
	engine := router.New(webhookHandler, baseLogger.Named("router"))

	// Initialize Scheduler
	sched := scheduler.NewScheduler(*cfg, reportingSvc, messagingSvc, baseLogger.Named("scheduler"))
	sched.Start()
	defer sched.Stop()

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
