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
	sessions   *SessionManager
	logger     *zap.Logger
}

// NewMetaWhatsAppService wires a new service instance.
func NewMetaWhatsAppService(cfg config.WhatsAppConfig, client client.Client, aiClient anthropic.Client, dispatcher commandsvc.Dispatcher, logger *zap.Logger) *MetaWhatsAppService {
	svc := &MetaWhatsAppService{
		cfg:        cfg,
		client:     client,
		aiClient:   aiClient,
		dispatcher: dispatcher,
		sessions:   NewSessionManager(),
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

	// 1. Check if it's a direct command (starts with /)
	if strings.HasPrefix(text, "/") {
		cmd := models.ParseCommand(text)
		return s.executeCommand(ctx, cmd, msg.From)
	}

	// 2. If AI is enabled, use the conversational flow
	if s.aiClient != nil {
		return s.handleConversation(ctx, msg.From, text)
	}

	// 3. Fallback to legacy command parsing for non-AI mode
	cmd := models.ParseCommand(text)
	return s.executeCommand(ctx, cmd, msg.From)
}

func (s *MetaWhatsAppService) handleConversation(ctx context.Context, userID, input string) error {
	// Get current session state
	currentState := s.sessions.GetSession(userID)

	// Process with AI
	newState, reply, err := s.aiClient.ProcessConversation(ctx, currentState, input)
	if err != nil {
		s.logger.Error("ai conversation failed", zap.Error(err))
		return s.sendReply(ctx, userID, "Désolé, une erreur technique est survenue. Veuillez réessayer.")
	}

	// MERGE LOGIC: Ensure we don't lose data if AI returns nulls but we had data before
	// This is a safety net in case the AI fails to copy the state correctly
	if currentState.EggsBand1 != nil && newState.EggsBand1 == nil {
		newState.EggsBand1 = currentState.EggsBand1
	}
	if currentState.EggsBand2 != nil && newState.EggsBand2 == nil {
		newState.EggsBand2 = currentState.EggsBand2
	}
	if currentState.EggsBand3 != nil && newState.EggsBand3 == nil {
		newState.EggsBand3 = currentState.EggsBand3
	}
	if currentState.MortalityQty != nil && newState.MortalityQty == nil {
		newState.MortalityQty = currentState.MortalityQty
	}
	if currentState.MortalityBand != "" && newState.MortalityBand == "" {
		newState.MortalityBand = currentState.MortalityBand
	}
	if currentState.FeedReceived != nil && newState.FeedReceived == nil {
		newState.FeedReceived = currentState.FeedReceived
	}
	if currentState.FeedQty != nil && newState.FeedQty == nil {
		newState.FeedQty = currentState.FeedQty
	}
	// Notes are cumulative or replaced, let's assume replacement is fine, or we could append.

	// Update session
	s.sessions.UpdateSession(userID, newState)

	// Check if conversation is complete
	if newState.Step == "COMPLETED" {
		// Save all data
		if err := s.saveDailyReport(ctx, newState); err != nil {
			s.logger.Error("failed to save daily report", zap.Error(err))
			return s.sendReply(ctx, userID, "Merci, mais j'ai eu un problème pour sauvegarder les données. Veuillez contacter l'admin.")
		}

		// Clear session and confirm
		s.sessions.ClearSession(userID)
		return s.sendReply(ctx, userID, "✅ Rapport journalier enregistré avec succès ! À demain.")
	}

	// Otherwise, send the AI's follow-up question
	return s.sendReply(ctx, userID, reply)
}

func (s *MetaWhatsAppService) saveDailyReport(ctx context.Context, state anthropic.ConversationState) error {
	s.logger.Info("attempting to save daily report",
		zap.Any("eggs_b1", state.EggsBand1),
		zap.Any("eggs_b2", state.EggsBand2),
		zap.Any("eggs_b3", state.EggsBand3),
		zap.Any("sales", state.SalesQty),
		zap.Any("mortality", state.MortalityQty),
		zap.Any("feed_qty", state.FeedQty),
		zap.String("notes", state.Notes),
	)

	if s.dispatcher == nil {
		return errors.New("dispatcher not configured")
	}

	// Save Eggs
	if state.EggsBand1 != nil || state.EggsBand2 != nil || state.EggsBand3 != nil {
		b1, b2, b3 := 0, 0, 0
		if state.EggsBand1 != nil {
			b1 = *state.EggsBand1
		}
		if state.EggsBand2 != nil {
			b2 = *state.EggsBand2
		}
		if state.EggsBand3 != nil {
			b3 = *state.EggsBand3
		}

		err := s.dispatcher.SaveEggsRecord(ctx, models.EggRecord{
			Date:     time.Now(),
			Band1:    b1,
			Band2:    b2,
			Band3:    b3,
			Quantity: b1 + b2 + b3,
			Notes:    state.Notes,
		})
		if err != nil {
			return fmt.Errorf("saving eggs: %w", err)
		}
	}

	// Save Sales
	// REMOVED: Farmer does not track sales quantity anymore.
	/*
		if state.SalesQty != nil && *state.SalesQty > 0 {
			err := s.dispatcher.SaveSaleRecord(ctx, models.SaleRecord{
				Date:         time.Now(),
				Client:       "Daily Report",
				Quantity:     *state.SalesQty,
				PricePerUnit: 0, // Price not captured in this flow yet
				Paid:         0,
			})
			if err != nil {
				return fmt.Errorf("saving sales: %w", err)
			}
		}
	*/

	// Save Mortality
	if state.MortalityQty != nil && *state.MortalityQty >= 0 {
		qty := *state.MortalityQty
		reason := state.MortalityBand

		// Si quantité est 0 et pas de raison, on écrit "RAS" ou "Aucune"
		if qty == 0 && (reason == "" || reason == "0") {
			reason = "RAS"
		}

		err := s.dispatcher.SaveMortalityRecord(ctx, models.MortalityRecord{
			Date:     time.Now(),
			Quantity: qty,
			Reason:   reason,
		})
		if err != nil {
			return fmt.Errorf("saving mortality: %w", err)
		}
	}

	// Save Feed (Reception)
	if state.FeedReceived != nil && *state.FeedReceived {
		feedKg := 0.0
		if state.FeedQty != nil {
			feedKg = *state.FeedQty
		}
		// Log feed reception (assuming 0 population for this record type, or just tracking event)
		err := s.dispatcher.SaveFeedRecord(ctx, models.FeedRecord{
			Date:       time.Now(),
			FeedKg:     feedKg,
			Population: 0,
		})
		if err != nil {
			return fmt.Errorf("saving feed reception: %w", err)
		}
	}

	return nil
}

func (s *MetaWhatsAppService) executeCommand(ctx context.Context, cmd models.Command, sender string) error {
	if s.dispatcher == nil {
		s.logger.Warn("command dispatcher not configured")
		reply := commandReplies[cmd.Type]
		outbound := fmt.Sprintf("%s\n%s", reply.Title, reply.Message)
		return s.sendReply(ctx, sender, outbound)
	}

	response, err := s.dispatcher.HandleCommand(ctx, cmd, sender)
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

		return s.sendReply(ctx, sender, outbound)
	}

	if response == "" {
		reply := commandReplies[cmd.Type]
		if reply.Title != "" {
			response = fmt.Sprintf("%s update logged.", reply.Title)
		} else {
			response = "Update stored successfully."
		}
	}

	return s.sendReply(ctx, sender, response)
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
