package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var validUser = User{
	Username: "test",
	Password: "testpass",
}

const SECRET_KEY = "SECRET"

func authMiddleware(c *gin.Context) {
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

	if token.Claims.(jwt.MapClaims)["username"] != validUser.Username {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		c.Abort()
		return
	}

	c.Next()
}

func loginHandler(c *gin.Context) {
	var inputUser User

	if err := c.BindJSON(&inputUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if inputUser.Username != validUser.Username || inputUser.Password != validUser.Password {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": inputUser.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(SECRET_KEY))
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
	authGroup.Use(authMiddleware)
	authGroup.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "you are authorized"})
	})

	r.Run(":8080")
}
