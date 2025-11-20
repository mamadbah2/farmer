package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/mamadbah2/farmer/internal/config"
	"github.com/mamadbah2/farmer/internal/domain/models"
	client "github.com/mamadbah2/farmer/pkg/clients/whatsapp"
)

// MessagingService describes the operations the HTTP layer can perform.
type MessagingService interface {
	VerifyWebhookToken(mode, verifyToken, challenge string) (string, error)
	HandleWebhook(ctx context.Context, payload models.WebhookPayload) error
	SendOutbound(ctx context.Context, req models.OutboundMessageRequest) error
}

// MetaWhatsAppService is the production implementation backed by WhatsApp Cloud API.
type MetaWhatsAppService struct {
	cfg    config.WhatsAppConfig
	client client.Client
	logger *zap.Logger
}

// NewMetaWhatsAppService wires a new service instance.
func NewMetaWhatsAppService(cfg config.WhatsAppConfig, client client.Client, logger *zap.Logger) *MetaWhatsAppService {
	svc := &MetaWhatsAppService{
		cfg:    cfg,
		client: client,
		logger: logger,
	}
	if svc.logger == nil {
		svc.logger = zap.NewNop()
	}
	return svc
}

var commandReplies = map[models.CommandType]models.AutomationReply{
	models.CommandEggs: {
		Title:   "Egg Collection",
		Message: "Please provide today's egg count in trays and cracked count, e.g. /eggs 120 trays 3 cracked.",
	},
	models.CommandFeed: {
		Title:   "Feed Usage",
		Message: "Share feed consumption with remaining inventory, e.g. /feed 6 bags remaining 20 bags.",
	},
	models.CommandMortality: {
		Title:   "Mortality Update",
		Message: "Report mortality and suspected causes, e.g. /mortality 3 heat stress.",
	},
	models.CommandSales: {
		Title:   "Sales Report",
		Message: "Capture livestock or egg sales, e.g. /sales 10 crates 250000.",
	},
	models.CommandExpenses: {
		Title:   "Expense Logging",
		Message: "Record expenses with supplier name, e.g. /expenses medication 55000 vet-shop.",
	},
	models.CommandUnknown: {
		Title:   "Command Help",
		Message: "Unknown command. Supported: /eggs, /feed, /mortality, /sales, /expenses.",
	},
}

// VerifyWebhookToken validates the callback verification token.
func (s *MetaWhatsAppService) VerifyWebhookToken(mode, verifyToken, challenge string) (string, error) {
	if mode == "" || verifyToken == "" {
		return "", errors.New("missing mode or verify token")
	}

	if !strings.EqualFold(mode, "subscribe") {
		return "", fmt.Errorf("unsupported hub.mode %s", mode)
	}

	if verifyToken != s.cfg.VerifyToken {
		return "", errors.New("invalid verify token")
	}

	return challenge, nil
}

// HandleWebhook processes inbound webhook payloads.
func (s *MetaWhatsAppService) HandleWebhook(ctx context.Context, payload models.WebhookPayload) error {
	if len(payload.Entry) == 0 {
		return nil
	}

	var firstErr error

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if len(change.Value.Messages) == 0 {
				continue
			}

			for _, msg := range change.Value.Messages {
				if err := s.handleInboundMessage(ctx, msg); err != nil {
					s.logger.Error("failed to handle inbound message", zap.Error(err), zap.String("message_id", msg.ID))
					if firstErr == nil {
						firstErr = err
					}
				}
			}
		}
	}

	return firstErr
}

func (s *MetaWhatsAppService) handleInboundMessage(ctx context.Context, msg models.InboundMessage) error {
	text := extractMessageText(msg)
	if text == "" {
		return errors.New("empty message body")
	}

	cmd := models.ParseCommand(text)
	reply := commandReplies[cmd.Type]
	if reply.Message == "" {
		reply = commandReplies[models.CommandUnknown]
	}

	outbound := fmt.Sprintf("%s\n%s", reply.Title, reply.Message)

	s.logger.Info("parsed inbound command",
		zap.String("from", msg.From),
		zap.String("command", string(cmd.Type)),
		zap.Any("args", cmd.Args))

	// TODO: persist parsed command for reporting when storage module is ready.

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := s.client.SendTextMessage(ctxWithTimeout, client.SendTextMessageRequest{
		To:         msg.From,
		Body:       outbound,
		PreviewURL: false,
	})
	return err
}

// SendOutbound lets internal operators push quick notifications via HTTP.
func (s *MetaWhatsAppService) SendOutbound(ctx context.Context, req models.OutboundMessageRequest) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := s.client.SendTextMessage(ctxWithTimeout, client.SendTextMessageRequest{
		To:         req.To,
		Body:       req.Message,
		PreviewURL: req.PreviewURL,
	})
	return err
}

func extractMessageText(msg models.InboundMessage) string {
	if msg.Text != nil {
		return msg.Text.Body
	}

	if msg.Interactive != nil {
		if msg.Interactive.ButtonReply != nil {
			return msg.Interactive.ButtonReply.ID
		}
		if msg.Interactive.ListReply != nil {
			return msg.Interactive.ListReply.ID
		}
	}

	// TODO: support template replies or future message types as needed.
	return ""
}
