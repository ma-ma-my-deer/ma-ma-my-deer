package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	_ "github.com/lib/pq"
	"github.com/my-deer/mydeer/handlers"
	"github.com/my-deer/mydeer/internal/db"
	"github.com/my-deer/mydeer/middleware"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

var (
	testDB     *db.DB
	testRouter *gin.Engine
)

// setupTestServer configures a test server with the same middleware and routes as production
func setupTestServer(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new router
	r := gin.New()

	// Register custom validators
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		handlers.RegisterValidators(v)
	}

	// Set up database connection
	dsn := "postgres://myuser:mypassword@localhost:5432/mydb?sslmode=disable"
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	testDB = db.New(conn)

	// Add middleware
	r.Use(func(c *gin.Context) {
		c.Set("mydb", testDB)
		c.Next()
	})
	r.Use(middleware.RequestLogger())
	r.Use(middleware.ErrorHandler())

	// Register routes
	r.POST("/login", handlers.LoginHandler)
	r.POST("/signup", handlers.SignupHandler)

	testRouter = r
}

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Set up custom logger for testing
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Run tests
	exitVal := m.Run()

	// Clean up, if required

	os.Exit(exitVal)
}

func TestSignupEndpoint(t *testing.T) {
	setupTestServer(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedKey    string
	}{
		{
			name: "Valid User Registration",
			requestBody: map[string]interface{}{
				"email":    "test_user@example.com",
				"password": "Test1234!@#$",
				"name":     "Test User",
			},
			expectedStatus: http.StatusOK,
			expectedKey:    "message",
		},
		{
			name: "Missing Email",
			requestBody: map[string]interface{}{
				"password": "Test1234!@#$",
				"name":     "Test User",
			},
			expectedStatus: http.StatusBadRequest,
			expectedKey:    "error",
		},
		{
			name: "Invalid Password (Too Simple)",
			requestBody: map[string]interface{}{
				"email":    "simple_password@example.com",
				"password": "password",
				"name":     "Test User",
			},
			expectedStatus: http.StatusBadRequest,
			expectedKey:    "error",
		},
		{
			name: "Duplicate Email",
			requestBody: map[string]interface{}{
				"email":    "duplicate@example.com",
				"password": "Test1234!@#$",
				"name":     "Duplicate User",
			},
			expectedStatus: http.StatusOK, // First attempt should succeed
			expectedKey:    "message",
		},
		{
			name: "Duplicate Email Retry",
			requestBody: map[string]interface{}{
				"email":    "duplicate@example.com",
				"password": "Test1234!@#$",
				"name":     "Duplicate User",
			},
			expectedStatus: http.StatusBadRequest, // Second attempt should fail
			expectedKey:    "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Convert request body to JSON
			jsonBody, err := json.Marshal(tc.requestBody)
			assert.NoError(t, err)

			// Create request
			req, err := http.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(jsonBody))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Perform request
			testRouter.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tc.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check if expected key exists in response
			_, exists := response[tc.expectedKey]
			assert.True(t, exists, fmt.Sprintf("Expected key '%s' not found in response", tc.expectedKey))

			if tc.expectedKey == "message" && tc.expectedStatus == http.StatusOK {
				assert.Equal(t, "user_created", response["message"])
			}
		})
	}
}

func TestLoginEndpoint(t *testing.T) {
	setupTestServer(t)

	// First create a test user for login tests
	createTestUser(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedKey    string
	}{
		{
			name: "Valid Login",
			requestBody: map[string]interface{}{
				"email":    "login_test@example.com",
				"password": "Test1234!@#$",
			},
			expectedStatus: http.StatusOK,
			expectedKey:    "message",
		},
		{
			name: "Invalid Email",
			requestBody: map[string]interface{}{
				"email":    "nonexistent@example.com",
				"password": "Test1234!@#$",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedKey:    "error",
		},
		{
			name: "Wrong Password",
			requestBody: map[string]interface{}{
				"email":    "login_test@example.com",
				"password": "WrongPassword123!",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedKey:    "error",
		},
		{
			name: "Missing Fields",
			requestBody: map[string]interface{}{
				"email": "incomplete@example.com",
			},
			expectedStatus: http.StatusBadRequest,
			expectedKey:    "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Convert request body to JSON
			jsonBody, err := json.Marshal(tc.requestBody)
			assert.NoError(t, err)

			// Create request
			req, err := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(jsonBody))
			assert.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Perform request
			testRouter.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tc.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			// Check if expected key exists in response
			_, exists := response[tc.expectedKey]
			assert.True(t, exists, fmt.Sprintf("Expected key '%s' not found in response", tc.expectedKey))

			// For successful login, check for auth token in cookie
			if tc.name == "Valid Login" {
				assert.Equal(t, "login_success", response["message"])
				cookies := w.Result().Cookies()
				var hasToken bool
				for _, cookie := range cookies {
					if cookie.Name == "token" {
						hasToken = true
						break
					}
				}
				assert.True(t, hasToken, "Token cookie not set")
			}
		})
	}
}

// Helper function to create a test user for login tests
func createTestUser(t *testing.T) {
	jsonBody, _ := json.Marshal(map[string]interface{}{
		"email":    "login_test@example.com",
		"password": "Test1234!@#$",
		"name":     "Login Test User",
	})

	req, _ := http.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	// Ensure user was created
	assert.Equal(t, http.StatusOK, w.Code)
}
