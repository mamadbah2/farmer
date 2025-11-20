package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/mamadbah2/farmer/internal/domain/models"
	service "github.com/mamadbah2/farmer/internal/service/whatsapp"
)

// WebhookHandler handles inbound and outbound WhatsApp HTTP events.
type WebhookHandler struct {
	svc    service.MessagingService
	logger *zap.Logger
}

// NewWebhookHandler constructs the HTTP handler adapter.
func NewWebhookHandler(svc service.MessagingService, logger *zap.Logger) *WebhookHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &WebhookHandler{svc: svc, logger: logger}
}

// Verify responds to Meta's webhook verification challenge.
func (h *WebhookHandler) Verify(c *gin.Context) {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	resp, err := h.svc.VerifyWebhookToken(mode, token, challenge)
	if err != nil {
		h.logger.Warn("webhook verification failed", zap.Error(err))
		c.String(http.StatusForbidden, "verification failed")
		return
	}

	c.String(http.StatusOK, resp)
}

// Receive ingests webhook POST callbacks from Meta.
func (h *WebhookHandler) Receive(c *gin.Context) {
	var payload models.WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.logger.Warn("invalid webhook payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	if err := h.svc.HandleWebhook(c.Request.Context(), payload); err != nil {
		h.logger.Error("failed processing webhook", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process webhook"})
		return
	}

	c.Status(http.StatusOK)
}

// SendMessage allows sending outbound automation or manual responses.
func (h *WebhookHandler) SendMessage(c *gin.Context) {
	var req models.OutboundMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid outbound payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.svc.SendOutbound(c.Request.Context(), req); err != nil {
		h.logger.Error("failed sending outbound", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "unable to send message"})
		return
	}

	c.Status(http.StatusAccepted)
}
