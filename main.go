package main

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"golang.org/x/exp/slog"

	"github.com/ryoyoshida-sh/mydone/middleware"
	"github.com/ryoyoshida-sh/mydone/types"
)

// Middleware: リクエストIDをセットし、ロギング機能を提供
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Set("request_id", requestID)

		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		logger = logger.With("request_id", requestID)
		c.Set("logger", logger)

		logger.Info("Request started", "method", c.Request.Method, "path", c.Request.URL.Path)

		c.Next()

		status := c.Writer.Status()
		logger.Info("Request completed", "status", status)
	}
}

func getLogger(c *gin.Context) *slog.Logger {
	if logger, exists := c.Get("logger"); exists {
		return logger.(*slog.Logger)
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

func loginHandler(c *gin.Context) {
	logger := getLogger(c)

	var inputUser types.User
	if err := c.BindJSON(&inputUser); err != nil {
		logger.Error("Failed to bind JSON", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if inputUser.Username != middleware.ValidUser.Username || inputUser.Password != middleware.ValidUser.Password {
		logger.Warn("Invalid login attempt", "username", inputUser.Username)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": inputUser.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(middleware.SECRET_KEY))
	if err != nil {
		logger.Error("Error signing token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error signing token"})
		return
	}

	logger.Info("Login successful", "username", inputUser.Username)

	c.Header("Authorization", tokenString)
	c.JSON(http.StatusOK, gin.H{"message": "login success"})
}

func main() {
	r := gin.Default()

	r.Use(RequestLogger()) // ロガーミドルウェアを追加

	r.POST("/login", loginHandler)

	authGroup := r.Group("/auth")
	authGroup.Use(middleware.Auth)
	authGroup.GET("/", func(c *gin.Context) {
		logger := getLogger(c)
		logger.Info("Authorized access to /auth")

		c.JSON(http.StatusOK, gin.H{"message": "you are authorized"})
	})

	r.Run(":8080")
}
