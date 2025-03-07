package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/ryoyoshida-sh/mydone/types"
)

var ValidUser = types.User{
	Username: "test",
	Password: "testpass",
}

const SECRET_KEY = "SECRET"

func Auth(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(SECRET_KEY), nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		c.Abort()
		return
	}

	if token.Claims.(jwt.MapClaims)["exp"].(float64) < float64(time.Now().Unix()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
		c.Abort()
		return
	}

	if token.Claims.(jwt.MapClaims)["username"] != ValidUser.Username {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		c.Abort()
		return
	}

	c.Next()
}
