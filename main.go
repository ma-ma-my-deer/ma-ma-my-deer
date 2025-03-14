package main

import (
	"database/sql"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/my-deer/mydeer/handlers"
	"github.com/my-deer/mydeer/internal/db"
	"github.com/my-deer/mydeer/middleware"
	"golang.org/x/exp/slog"
	"github.com/go-playground/validator/v10"
)

func main() {
	// DB接続設定 (docker-composeで起動中のPostgreSQLへ接続)
	dsn := "postgres://myuser:mypassword@localhost:5432/mydb?sslmode=disable"
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		slog.Error("main: failed to connect to database", "error", err.Error())
		os.Exit(1)
	}
	defer conn.Close()

	mydb := db.New(conn)
	r := gin.Default()

	// Register custom validators
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		handlers.RegisterValidators(v)
	}

	// sqlcのクエリインスタンスをコンテキストにセット
	r.Use(func(c *gin.Context) {
		c.Set("mydb", mydb)
		c.Next()
	})

	// ミドルウェア設定
	r.Use(middleware.RequestLogger())
	r.Use(middleware.ErrorHandler()) // Add this line
	r.Use(gin.Recovery())

	// エンドポイント設定
	r.POST("/login", handlers.LoginHandler)
	r.POST("/signup", handlers.SignupHandler)
	r.GET("/auth", middleware.Auth) // Cookie検証ミドルウェア等を適用するならこちらに追加

	// サーバー起動 (ポート:8080)
	r.Run(":8080")
}
