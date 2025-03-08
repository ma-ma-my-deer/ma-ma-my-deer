package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

const SECRET_KEY = "SECRET"

func Auth(c *gin.Context) {
	// Cookie "token" からJWTトークンを取得
	tokenString, err := c.Cookie("token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token not found in cookie"})
		c.Abort()
		return
	}

	// JWTトークンの解析と署名方式の検証
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// HMAC方式のみ許容
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(SECRET_KEY), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "details": err.Error()})
		c.Abort()
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
		c.Abort()
		return
	}

	// トークンの有効期限チェック
	exp, ok := claims["exp"].(float64)
	if !ok || int64(exp) < time.Now().Unix() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
		c.Abort()
		return
	}

	// 必要に応じてclaimsをコンテキストにセット
	c.Set("claims", claims)
	c.Next()
}
