package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/mamadbah2/farmer/internal/server/handlers"
)

// New wires the Gin engine with required routes and middlewares.
func New(handler *handlers.WebhookHandler, logger *zap.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(zapLoggerMiddleware(logger))

	r.GET("/webhook", handler.Verify)
	r.POST("/webhook", handler.Receive)
	r.POST("/send-message", handler.SendMessage)
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	if logger != nil {
		logger.Info("router initialized")
	}

	return r
}

func zapLoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		logger.Info("request completed",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", time.Since(start)),
			zap.String("client_ip", c.ClientIP()))
	}
}
