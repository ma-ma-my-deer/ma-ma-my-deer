package main

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slog"

	_ "github.com/lib/pq"

	// ※生成されたsqlcパッケージのパスに合わせて変更してください
	db "github.com/mydeer/mydeer/internal/db"
	"github.com/mydeer/mydeer/middleware"
)

// RequestLogger ミドルウェアはリクエストIDを設定し、リクエストの開始と終了をログ出力します。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("request_id", requestID)
		c.Set("logger", logger)
		logger.Info("Request started", "method", c.Request.Method, "path", c.Request.URL.Path)
		c.Next()
		status := c.Writer.Status()
		logger.Info("Request completed", "status", status)
	}
}

// getLogger はコンテキストからloggerを取得します。
func getLogger(c *gin.Context) *slog.Logger {
	if logger, exists := c.Get("logger"); exists {
		return logger.(*slog.Logger)
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

// loginHandler は、受け取ったEmailとPasswordをもとにログインを処理します。
func loginHandler(c *gin.Context) {
	logger := getLogger(c)
	queries := c.MustGet("queries").(*db.Queries)

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BindJSON(&req); err != nil {
		logger.Error("failed to bind JSON", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := queries.GetUserByEmail(c, req.Email)
	if err != nil {
		logger.Warn("user not found", "email", req.Email, "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// bcrypt によるパスワード比較
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		logger.Warn("invalid password", "email", req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// JWT トークン作成
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": req.Email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(middleware.SECRET_KEY))
	if err != nil {
		logger.Error("error signing token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error signing token"})
		return
	}

	logger.Info("login successful", "email", req.Email)
	c.Header("Authorization", tokenString)
	c.JSON(http.StatusOK, gin.H{"message": "login success"})
}

// signupHandler は、受け取ったEmail, Password, Nameをそのまま
// bcrypt でハッシュ化し、DBに保存するアカウント作成エンドポイントです。
func signupHandler(c *gin.Context) {
	logger := getLogger(c)
	queries := c.MustGet("queries").(*db.Queries)

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := c.BindJSON(&req); err != nil {
		logger.Error("failed to bind JSON", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	// パスワードをbcryptでハッシュ化
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("failed to hash password", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	// ユーザー登録 (UUIDはDB側で自動生成する前提)
	user, err := queries.CreateUser(c, db.CreateUserParams{
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
	})
	if err != nil {
		logger.Error("failed to create user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	logger.Info("user created", "email", req.Email)
	c.JSON(http.StatusOK, gin.H{"message": "user created", "user": user})
}

func main() {
	// DB接続設定 (docker-composeで起動中のPostgreSQLへ接続)
	dsn := "postgres://myuser:mypassword@localhost:5432/mydb?sslmode=disable"
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	queries := db.New(conn)
	r := gin.Default()

	// sqlc のクエリインスタンスをコンテキストにセット
	r.Use(func(c *gin.Context) {
		c.Set("queries", queries)
		c.Next()
	})

	// ミドルウェア設定
	r.Use(RequestLogger())
	r.Use(gin.Recovery())

	// エンドポイント設定
	r.POST("/login", loginHandler)
	r.POST("/signup", signupHandler)

	// 認証が必要なエンドポイント例
	authGroup := r.Group("/auth")
	authGroup.Use(middleware.Auth)
	authGroup.GET("/", func(c *gin.Context) {
		logger := getLogger(c)
		logger.Info("authorized access to /auth")
		c.JSON(http.StatusOK, gin.H{"message": "you are authorized"})
	})

	// サーバー起動 (ポート:8080)
	r.Run(":8080")
}
