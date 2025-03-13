package handlers

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/lib/pq"
	"github.com/my-deer/mydeer/internal/db"
	apperrors "github.com/my-deer/mydeer/internal/errors"
	"github.com/my-deer/mydeer/middleware"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slog"
)

// LoginInput はログイン用の入力構造体です。
type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// SignupInput はアカウント作成用の入力構造体です。
// カスタムバリデーション("validpassword")はここでは使用せず、後でローカル検証します。
type SignupInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=12,max=72"`
	Name     string `json:"name" binding:"required"`
}

// validPassword はパスワードが大文字・小文字・数字・記号を各1文字以上含むかをチェックします。
func validPassword(password string) bool {
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

// getLogger はコンテキストからloggerを取得します。
func getLogger(c *gin.Context) *slog.Logger {
	if logger, exists := c.Get("logger"); exists {
		return logger.(*slog.Logger)
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

// LoginHandler is, receive Email and Password and login, issue JWT token and set cookie.
func LoginHandler(c *gin.Context) {
	logger := getLogger(c)
	mydb := c.MustGet("mydb").(*db.DB)

	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		var validationErrors gin.H
		if ve, ok := err.(validator.ValidationErrors); ok {
			validationErrors = buildValidationErrorMap(ve)
		}
		logger.Error("login: failed to bind json", "error", err.Error())
		// Use the errors.ErrInvalidInput error with details
		appErr := apperrors.ErrInvalidInput.WithDetails(validationErrors)
		c.Error(appErr) // Will be caught by the error middleware
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

// Helper function to build validation error map
func buildValidationErrorMap(ve validator.ValidationErrors) gin.H {
	errorDetails := gin.H{}
	for _, fieldErr := range ve {
		fieldName := strings.ToLower(fieldErr.Field())
		var errorMsg string
		switch fieldErr.Tag() {
		case "required":
			errorMsg = fieldName + "_required"
		case "email":
			errorMsg = "invalid_email_format"
		case "min":
			errorMsg = fieldName + "_too_short"
		case "max":
			errorMsg = fieldName + "_too_long"
		default:
			errorMsg = "invalid_" + fieldName
		}
		errorDetails[fieldName] = errorMsg
	}
	return errorDetails
}

// SignupHandler は、受け取ったEmail, Password, Nameを検証後、bcryptでハッシュ化しDBに保存します。
// バリデーションエラー時は、どのフィールドがどの理由で不正かを明示的なエラーコードで返します。
func SignupHandler(c *gin.Context) {
	logger := getLogger(c)
	mydb := c.MustGet("mydb").(*db.DB)

	var input SignupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		// バリデーションエラーを詳細に返す
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			errorDetails := make(map[string]string)
			for _, fieldErr := range ve {
				fieldName := strings.ToLower(fieldErr.Field())
				var errorMsg string
				switch fieldErr.Field() {
				case "Email":
					if fieldErr.Tag() == "email" {
						errorMsg = "invalid_email_format"
					} else {
						errorMsg = "invalid_email"
					}
				case "Password":
					if fieldErr.Tag() == "min" {
						errorMsg = "password_too_short"
					} else if fieldErr.Tag() == "max" {
						errorMsg = "password_too_long"
					} else {
						errorMsg = "invalid_password"
					}
				case "Name":
					errorMsg = "invalid_name"
				default:
					errorMsg = "invalid_" + fieldName
				}
				errorDetails[fieldName] = errorMsg
			}
			logger.Warn("signup: validation error", "errors", errorDetails)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "validation_failed",
				"details": errorDetails,
				"code":    "400",
			})
			return
		}
		logger.Error("signup: failed to bind json", "error", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request_payload", "code": "400"})
		return
	}

	// ローカルでパスワードの複雑性をチェック（大文字・小文字・数字・記号を各1文字以上含む）
	if !validPassword(input.Password) {
		logger.Warn("signup: password complexity insufficient", "email", input.Email)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "password_complexity_insufficient",
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
