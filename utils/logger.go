package utils

import (
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// GetLogger retrieves the logger from the gin context
// If no logger exists in the context, it returns a default JSON logger
func GetLogger(c *gin.Context) *slog.Logger {
	if logger, exists := c.Get("logger"); exists {
		return logger.(*slog.Logger)
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
