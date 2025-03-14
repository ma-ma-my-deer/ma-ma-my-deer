package handlers

import (
	"errors"
	"net/http"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/lib/pq"
	"github.com/my-deer/mydeer/internal/db"
	apperrors "github.com/my-deer/mydeer/internal/errors"
	"github.com/my-deer/mydeer/middleware"
	"github.com/my-deer/mydeer/utils"
	"golang.org/x/crypto/bcrypt"
)

// RegisterValidators registers custom validators for the application
func RegisterValidators(v *validator.Validate) {
	v.RegisterValidation("complexpassword", validateComplexPassword)
}

// validateComplexPassword checks if password has uppercase, lowercase, digit, and symbol
func validateComplexPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	var hasUpper, hasLower, hasDigit, hasSymbol bool

	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSymbol = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSymbol
}

// LoginInput はログイン用の入力構造体です。
type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// SignupInput はアカウント作成用の入力構造体です。
// カスタムバリデーション("validpassword")はここでは使用せず、後でローカル検証します。
type SignupInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=12,max=72,complexpassword"`
	Name     string `json:"name" binding:"required"`
}

// LoginHandler is, receive Email and Password and login, issue JWT token and set cookie.
func LoginHandler(c *gin.Context) {
	logger := utils.GetLogger(c)
	mydb := c.MustGet("mydb").(*db.DB)

	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Error("login: failed to bind json", "error", err.Error())
		c.Error(apperrors.ErrInvalidInput)
		return
	}

	user, err := mydb.GetUserByEmail(c, input.Email)
	if err != nil {
		logger.Warn("login: user lookup failed", "email", input.Email, "error", err)
		// Always return generic error for authentication attempts to prevent user enumeration
		c.Error(apperrors.ErrInvalidCredentials)
		return
	}

	// パスワード比較（passwordはログに出力しない）
	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		logger.Warn("login: invalid credentials", "email", input.Email)
		c.Error(apperrors.ErrInvalidCredentials)
		return
	}

	// JWTトークン作成
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": input.Email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(middleware.SECRET_KEY))
	if err != nil {
		logger.Error("login: error signing token", "email", input.Email, "error", err.Error())
		c.Error(apperrors.Wrap(err, apperrors.ErrInternal, "Failed to generate authentication token", http.StatusInternalServerError))
		return
	}

	// Set-CookieヘッダーにJWTトークンを設定（24時間有効, HttpOnly）
	c.SetCookie("token", tokenString, int(24*time.Hour.Seconds()), "/", "", false, true)

	logger.Info("login: success", "email", input.Email)
	c.Header("Authorization", tokenString) // 必要ならヘッダーにもセット
	c.JSON(http.StatusOK, gin.H{"message": "login_success"})
}

// SignupHandler は、受け取ったEmail, Password, Nameを検証後、bcryptでハッシュ化しDBに保存します。
func SignupHandler(c *gin.Context) {
	logger := utils.GetLogger(c)
	mydb := c.MustGet("mydb").(*db.DB)

	var input SignupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Warn("signup: validation error", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "validation_failed",
			"code":  "400",
		})
		return
	}

	// bcryptでパスワードをハッシュ化（passwordはログに出さない）
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("signup: failed to hash password", "email", input.Email, "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_server_error", "code": "500"})
		return
	}

	// ユーザー登録（DB側でUUID自動生成前提）
	_, err = mydb.CreateUser(c, db.CreateUserParams{
		Email:    input.Email,
		Password: string(hashedPassword),
		Name:     input.Name,
	})
	if err != nil {
		// 重複エラーの場合、Postgresのエラーコード23505（unique violation）をチェック
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			logger.Warn("signup: duplicate email", "email", input.Email)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "email_already_exists",
				"code":  "400",
			})
			return
		}
		logger.Error("signup: failed to create user", "email", input.Email, "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_server_error", "code": "500"})
		return
	}

	logger.Info("signup: user created", "email", input.Email)
	c.JSON(http.StatusOK, gin.H{"message": "user_created"})
}
