package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/my-deer/mydeer/internal/errors"
	"github.com/my-deer/mydeer/utils"
)

// ErrorHandler middleware catches panics and returns JSON error responses
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the logger
		logger := utils.GetLogger(c)

		// Recover from any panics
		defer func() {
			if r := recover(); r != nil {
				var err error
				switch e := r.(type) {
				case error:
					err = e
				default:
					err = errors.New(errors.ErrInternal, "Internal server error", http.StatusInternalServerError)
				}

				logger.Error("panic recovered", "error", err)
				appErr := errors.FormatError(err)
				c.AbortWithStatusJSON(appErr.HTTPStatus, gin.H{
					"error":   appErr.Code,
					"message": appErr.Message,
					"details": appErr.Details,
				})
			}
		}()

		// Process the request
		c.Next()

		// Handle errors that were set during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			logger.Error("request error", "error", err, "path", c.Request.URL.Path)

			appErr := errors.FormatError(err)
			c.JSON(appErr.HTTPStatus, gin.H{
				"error":   appErr.Code,
				"message": appErr.Message,
				"details": appErr.Details,
			})
		}
	}
}
