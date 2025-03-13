package errors

import (
	"fmt"
	"net/http"

	"github.com/cockroachdb/errors"
)

// Application error codes
const (
	// Database error codes
	ErrDBNotFound    = "DB_NOT_FOUND"
	ErrDBDuplicate   = "DB_DUPLICATE"
	ErrDBConnection  = "DB_CONNECTION"
	ErrDBExecution   = "DB_EXECUTION"
	ErrDBTransaction = "DB_TRANSACTION"

	// Authentication error codes
	ErrAuthInvalid  = "AUTH_INVALID"
	ErrAuthExpired  = "AUTH_EXPIRED"
	ErrAuthRequired = "AUTH_REQUIRED"

	// Validation error codes
	ErrValidation = "VALIDATION_ERROR"

	// General error codes
	ErrInternal   = "INTERNAL_ERROR"
	ErrBadRequest = "BAD_REQUEST"
)

// AppError represents an application error with code, message, and HTTP status
type AppError struct {
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Details    interface{} `json:"details,omitempty"`
	HTTPStatus int         `json:"-"`
	err        error       `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.err
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(details interface{}) *AppError {
	e.Details = details
	return e
}

// New creates a new AppError with the given code and message
func New(code string, message string, status int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: status,
		err:        errors.New(message),
	}
}

// Wrap wraps an existing error with an AppError
func Wrap(err error, code string, message string, status int) *AppError {
	if err == nil {
		return nil
	}

	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: status,
		err:        errors.Wrap(err, message),
	}
}

// Common application errors
var (
	ErrNotFound           = New(ErrDBNotFound, "Resource not found", http.StatusNotFound)
	ErrInvalidCredentials = New(ErrAuthInvalid, "Invalid credentials", http.StatusUnauthorized)
	ErrInternalServer     = New(ErrInternal, "Internal server error", http.StatusInternalServerError)
	ErrDuplicateEntry     = New(ErrDBDuplicate, "Resource already exists", http.StatusConflict)
	ErrInvalidInput       = New(ErrValidation, "Invalid input parameters", http.StatusBadRequest)
)

// IsNotFound checks if the error is a not found error
func IsNotFound(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == ErrDBNotFound
	}
	return false
}

// IsDuplicate checks if the error is a duplicate entry error
func IsDuplicate(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == ErrDBDuplicate
	}
	return false
}

// GetHTTPStatus returns the HTTP status code for an error
func GetHTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.HTTPStatus
	}
	return http.StatusInternalServerError
}

// FormatError returns an AppError from any error
// If the error is already an AppError, it is returned as is
// Otherwise, it is wrapped with ErrInternalServer
func FormatError(err error) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	return &AppError{
		Code:       ErrInternal,
		Message:    "An unexpected error occurred",
		HTTPStatus: http.StatusInternalServerError,
		err:        errors.Wrap(err, "unexpected error"),
	}
}
