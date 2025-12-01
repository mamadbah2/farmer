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
	commandsvc "github.com/mamadbah2/farmer/internal/service/commands"
	"github.com/mamadbah2/farmer/pkg/clients/anthropic"
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
	cfg        config.WhatsAppConfig
	client     client.Client
	aiClient   anthropic.Client
	dispatcher commandsvc.Dispatcher
	logger     *zap.Logger
}

// NewMetaWhatsAppService wires a new service instance.
func NewMetaWhatsAppService(cfg config.WhatsAppConfig, client client.Client, aiClient anthropic.Client, dispatcher commandsvc.Dispatcher, logger *zap.Logger) *MetaWhatsAppService {
	svc := &MetaWhatsAppService{
		cfg:        cfg,
		client:     client,
		aiClient:   aiClient,
		dispatcher: dispatcher,
		logger:     logger,
	}
	if svc.logger == nil {
		svc.logger = zap.NewNop()
	}
	return svc
}

var commandReplies = map[models.CommandType]models.AutomationReply{
	models.CommandEggs: {
		Title:   "Egg Collection",
		Message: "Please provide egg counts for all 3 bands, e.g. /eggs 120 130 110 (Band1 Band2 Band3).",
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

	// 2. If unknown and AI is configured, try to translate natural language
	if cmd.Type == models.CommandUnknown && s.aiClient != nil {
		s.logger.Info("attempting ai translation", zap.String("input", text))
		translated, err := s.aiClient.TranslateToCommand(ctx, text)

		if err != nil {
			s.logger.Error("ai translation failed", zap.Error(err))
			// Fallthrough to unknown command handling
		} else {
			s.logger.Info("ai translated command", zap.String("original", text), zap.String("translated", translated))
			// If AI returns "unknown", ParseCommand will handle it as unknown anyway
			cmd = models.ParseCommand(translated)
		}
	}

	s.logger.Info("parsed inbound command",
		zap.String("from", msg.From),
		zap.String("command", string(cmd.Type)),
		zap.Any("args", cmd.Args))

	if cmd.Type == models.CommandUnknown {
		reply := commandReplies[models.CommandUnknown]
		return s.sendReply(ctx, msg.From, fmt.Sprintf("%s\n%s", reply.Title, reply.Message))
	}

	if s.dispatcher == nil {
		s.logger.Warn("command dispatcher not configured")
		reply := commandReplies[cmd.Type]
		outbound := fmt.Sprintf("%s\n%s", reply.Title, reply.Message)
		return s.sendReply(ctx, msg.From, outbound)
	}

	response, err := s.dispatcher.HandleCommand(ctx, cmd, msg.From)
	if err != nil {
		s.logger.Warn("dispatcher failed to handle command", zap.Error(err), zap.String("command", string(cmd.Type)))
		reply := commandReplies[cmd.Type]
		if reply.Message == "" {
			reply = commandReplies[models.CommandUnknown]
		}

		var outbound string
		switch {
		case errors.Is(err, commandsvc.ErrInvalidArguments):
			outbound = fmt.Sprintf("Could not parse your %s update.\n%s", string(cmd.Type), reply.Message)
		case errors.Is(err, commandsvc.ErrUnsupportedCommand):
			outbound = fmt.Sprintf("%s\n%s", reply.Title, reply.Message)
		default:
			outbound = "We hit a technical issue storing your update. Please retry shortly."
		}

		return s.sendReply(ctx, msg.From, outbound)
	}

	if response == "" {
		reply := commandReplies[cmd.Type]
		if reply.Title != "" {
			response = fmt.Sprintf("%s update logged.", reply.Title)
		} else {
			response = "Update stored successfully."
		}
	}

	return s.sendReply(ctx, msg.From, response)
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

func (s *MetaWhatsAppService) sendReply(ctx context.Context, to, body string) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := s.client.SendTextMessage(ctxWithTimeout, client.SendTextMessageRequest{
		To:         to,
		Body:       body,
		PreviewURL: false,
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
