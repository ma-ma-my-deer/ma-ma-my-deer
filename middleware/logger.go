package middleware

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/exp/slog"
)

// RequestLogger ミドルウェアはリクエストIDを設定し、リクエスト開始・終了時のログを出力します。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("request_id", requestID)
		c.Set("logger", logger)
		logger.Info("request_started", "method", c.Request.Method, "path", c.Request.URL.Path)
		c.Next()
		status := c.Writer.Status()
		logger.Info("request_completed", "status", status)
	}
}
