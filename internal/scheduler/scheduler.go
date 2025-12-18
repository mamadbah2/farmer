package scheduler

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/mamadbah2/farmer/internal/config"
	"github.com/mamadbah2/farmer/internal/domain/models"
	"github.com/mamadbah2/farmer/internal/service/reporting"
	"github.com/mamadbah2/farmer/internal/service/whatsapp"
)

// Scheduler manages scheduled tasks.
type Scheduler struct {
	cron         *cron.Cron
	reportingSvc *reporting.Service
	messagingSvc whatsapp.MessagingService
	cfg          config.Config
	logger       *zap.Logger
}

// NewScheduler creates a new scheduler instance.
func NewScheduler(cfg config.Config, reportingSvc *reporting.Service, messagingSvc whatsapp.MessagingService, logger *zap.Logger) *Scheduler {
	if logger == nil {
		logger = zap.NewNop()
	}

	// Create a cron instance with a custom location if needed, or use default (Local)
	// Here we use the standard parser which supports seconds if configured, but standard cron is minute-based.
	// robfig/cron/v3 default parser is standard cron (5 fields: min, hour, dom, month, dow).
	c := cron.New()

	return &Scheduler{
		cron:         c,
		reportingSvc: reportingSvc,
		messagingSvc: messagingSvc,
		cfg:          cfg,
		logger:       logger,
	}
}

// Start starts the scheduler.
func (s *Scheduler) Start() {
	s.logger.Info("starting scheduler")

	// Schedule weekly report for Friday at 20:00
	// Cron expression: "0 20 * * 5" (At 20:00 on Friday)
	_, err := s.cron.AddFunc("0 20 * * 5", s.sendWeeklyReport)
	if err != nil {
		s.logger.Error("failed to schedule weekly report", zap.Error(err))
	}

	s.cron.Start()
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.logger.Info("stopping scheduler")
	s.cron.Stop()
}

func (s *Scheduler) sendWeeklyReport() {
	s.logger.Info("generating weekly report")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	report, err := s.reportingSvc.GenerateWeeklyReport(ctx, time.Now())
	if err != nil {
		s.logger.Error("failed to generate weekly report", zap.Error(err))
		return
	}

	req := models.OutboundMessageRequest{
		To:      s.cfg.WhatsApp.ExpenseManagerID,
		Message: report,
	}

	if err := s.messagingSvc.SendOutbound(ctx, req); err != nil {
		s.logger.Error("failed to send weekly report", zap.Error(err))
	} else {
		s.logger.Info("weekly report sent successfully")
	}
}
