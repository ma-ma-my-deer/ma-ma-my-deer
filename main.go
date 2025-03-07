package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/ryoyoshida-sh/mydone/middleware"
	"github.com/ryoyoshida-sh/mydone/types"
)

func loginHandler(c *gin.Context) {
	var inputUser types.User

	if err := c.BindJSON(&inputUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if inputUser.Username != middleware.ValidUser.Username || inputUser.Password != middleware.ValidUser.Password {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": inputUser.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(middleware.SECRET_KEY))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error signing token"})
		return
	}

	// ヘッダーにトークンをセット
	c.Header("Authorization", tokenString)
	c.JSON(http.StatusOK, gin.H{"message": "login success"})
}

func main() {
	r := gin.Default()

	r.POST("/login", loginHandler)

	authGroup := r.Group("/auth")
	authGroup.Use(middleware.Auth)
	authGroup.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "you are authorized"})
	})

	r.Run(":8080")
}
