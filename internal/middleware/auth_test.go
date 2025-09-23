package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/SebbieMzingKe/customer-order-api/internal/handlers"
	"github.com/SebbieMzingKe/customer-order-api/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func generateTestToken(email string, secret []byte, expired bool) string {
	expirationTime := time.Now().Add(24 * time.Hour)
	if expired {
		expirationTime = time.Now().Add(-24 * time.Hour)
	}

	claims := &handlers.Claims{
		Email: email,
		Sub:   email,
		Name:  "test user",
		Iss:   "customer-order-api",
		Aud:   "customer-order-api",
		Exp:   expirationTime.Unix(),
		Iat:   time.Now().Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "customer-order-api",
			Subject:   email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(secret)
	return tokenString
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := []byte("test-secret")

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "missing token",
		},
		{
			name:           "invalid authorization header format",
			authHeader:     "invalidformat token123",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid token format",
		},
		{
			name:           "missing bearer prefix",
			authHeader:     "token123",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid token format",
		},
		{
			name:           "valid token",
			authHeader:     "Bearer " + generateTestToken("test@example.com", secret, false),
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "expired token",
			authHeader:     "Bearer " + generateTestToken("test@example.com", secret, true),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid token",
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("JWT_SECRET", "test-secret")
			defer os.Unsetenv("JWT_SECRET")

			router := gin.New()
			router.Use(AuthMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse models.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
				assert.NoError(t, err)
				assert.Contains(t, errorResponse.Error, tt.expectedError)
			} else {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "success", response["message"])
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	t.Run("OPTIONS request", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("OPTIONS", "/test", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	})

	t.Run("GET request with CORS headers", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestAuthMiddlewareContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := []byte("test-secret")
	email := "test@example.com"
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")

	token := generateTestToken(email, secret, false)

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		claims, exists := c.Get("claims")
		assert.True(t, exists)

		userEmail, exists := c.Get("user_email")
		assert.True(t, exists)
		assert.Equal(t, email, userEmail)

		userSub, exists := c.Get("user_sub")
		assert.True(t, exists)
		assert.Equal(t, email, userSub)

		c.JSON(http.StatusOK, gin.H{
			"message": "success",
			"email":   claims.(*models.Claims).Email,
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, email, response["email"])
}

func TestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    models.LoginRequest
		oidcEnabled    bool
		jwtSecret      string
		expectedStatus int
		expectedError  string
		checkRedirect  bool
	}{
		{
			name: "valid non-OIDC login",
			requestBody: models.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			oidcEnabled:    false,
			jwtSecret:      "test-secret",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name: "OIDC redirect",
			requestBody: models.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			oidcEnabled:    true,
			jwtSecret:      "test-secret",
			expectedStatus: http.StatusFound,
			checkRedirect:  true,
		},
		{
			name: "invalid request body",
			requestBody: models.LoginRequest{
				Email:    "",
				Password: "password123",
			},
			oidcEnabled:    false,
			jwtSecret:      "test-secret",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request",
		},
		{
			name: "missing password",
			requestBody: models.LoginRequest{
				Email:    "test@example.com",
				Password: "",
			},
			oidcEnabled:    false,
			jwtSecret:      "test-secret",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request",
		},
		{
			name: "token generation failure",
			requestBody: models.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			oidcEnabled:    false,
			jwtSecret:      "",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "token generation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("JWT_SECRET")
			os.Unsetenv("OIDC_PROVIDER_URL")
			os.Unsetenv("OIDC_CLIENT_ID")
			os.Unsetenv("OIDC_CLIENT_SECRET")
			os.Unsetenv("OIDC_REDIRECT_URI")

			if tt.jwtSecret != "" {
				os.Setenv("JWT_SECRET", tt.jwtSecret)
			}
			defer os.Unsetenv("JWT_SECRET")

			var handler *AuthHandler

			if tt.oidcEnabled {
				httpmock.Activate()
				defer httpmock.DeactivateAndReset()

				httpmock.RegisterResponder("GET", "https://example.com/.well-known/openid-configuration",
					httpmock.NewStringResponder(http.StatusOK, `{
						"issuer": "https://example.com",
						"authorization_endpoint": "https://example.com/auth",
						"token_endpoint": "https://example.com/token",
						"userinfo_endpoint": "https://example.com/userinfo",
						"jwks_uri": "https://example.com/jwks"
					}`))

				os.Setenv("OIDC_PROVIDER_URL", "https://example.com")
				os.Setenv("OIDC_CLIENT_ID", "test-client")
				os.Setenv("OIDC_CLIENT_SECRET", "test-secret")
				os.Setenv("OIDC_REDIRECT_URI", "https://app.example.com/callback")

				handler = NewAuthHandler()

				defer func() {
					os.Unsetenv("OIDC_PROVIDER_URL")
					os.Unsetenv("OIDC_CLIENT_ID")
					os.Unsetenv("OIDC_CLIENT_SECRET")
					os.Unsetenv("OIDC_REDIRECT_URI")
				}()
			} else {
				handler = NewAuthHandler()
			}

			w := httptest.NewRecorder()
			_, router := gin.CreateTestContext(w)

			router.POST("/login", handler.Login)

			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse models.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
				assert.NoError(t, err)
				assert.Contains(t, errorResponse.Error, tt.expectedError)
			} else if tt.checkRedirect {
				redirectURL := w.Header().Get("Location")
				assert.NotEmpty(t, redirectURL, "Location header should not be empty for redirect")
				assert.Contains(t, redirectURL, "https://example.com")
			} else {
				var authResponse models.AuthResponse
				err := json.Unmarshal(w.Body.Bytes(), &authResponse)
				assert.NoError(t, err)
				assert.NotEmpty(t, authResponse.AccessToken)
				assert.Equal(t, "Bearer", authResponse.TokenType)
				assert.Equal(t, int64(86400), authResponse.ExpiresIn)

				claims, err := handler.ValidateToken(authResponse.AccessToken)
				assert.NoError(t, err)
				assert.Equal(t, tt.requestBody.Email, claims.Email)
				assert.Equal(t, "Seb", claims.Name)
				assert.Equal(t, "customer-order-api", claims.Iss)
			}
		})
	}
}

func TestCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		queryParams    string
		oidcEnabled    bool
		jwtSecret      string
		expectedStatus int
		expectedError  string
		setupMocks     func()
	}{
		{
			name:           "OIDC not configured",
			queryParams:    "code=authcode123&state=state-123",
			oidcEnabled:    false,
			jwtSecret:      "test-secret",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "oidc_not_configured",
			setupMocks:     func() {},
		},
		{
			name:           "missing code",
			queryParams:    "state=state-123",
			oidcEnabled:    true,
			jwtSecret:      "test-secret",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "missing code",
			setupMocks: func() {
				httpmock.RegisterResponder("GET", "https://example.com/.well-known/openid-configuration",
					httpmock.NewStringResponder(http.StatusOK, `{
						"issuer": "https://example.com",
						"authorization_endpoint": "https://example.com/authorize",
						"token_endpoint": "https://example.com/token",
						"userinfo_endpoint": "https://example.com/userinfo",
						"jwks_uri": "https://example.com/jwks"
					}`))
			},
		},
		{
			name:           "token exchange failure",
			queryParams:    "code=authcode123&state=state-123",
			oidcEnabled:    true,
			jwtSecret:      "test-secret",
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "token_exchange_failed",
			setupMocks: func() {
				httpmock.RegisterResponder("GET", "https://example.com/.well-known/openid-configuration",
					httpmock.NewStringResponder(http.StatusOK, `{
						"issuer": "https://example.com",
						"authorization_endpoint": "https://example.com/authorize",
						"token_endpoint": "https://example.com/token",
						"userinfo_endpoint": "https://example.com/userinfo",
						"jwks_uri": "https://example.com/jwks"
					}`))
				httpmock.RegisterResponder("POST", "https://example.com/token",
					httpmock.NewStringResponder(http.StatusBadRequest, `{"error": "invalid_grant"}`))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			os.Setenv("JWT_SECRET", tt.jwtSecret)
			defer os.Unsetenv("JWT_SECRET")

			if tt.oidcEnabled {
				os.Setenv("OIDC_PROVIDER_URL", "https://example.com")
				os.Setenv("OIDC_CLIENT_ID", "test-client")
				os.Setenv("OIDC_CLIENT_SECRET", "test-secret")
				os.Setenv("OIDC_REDIRECT_URI", "https://app.example.com/callback")
				defer func() {
					os.Unsetenv("OIDC_PROVIDER_URL")
					os.Unsetenv("OIDC_CLIENT_ID")
					os.Unsetenv("OIDC_CLIENT_SECRET")
					os.Unsetenv("OIDC_REDIRECT_URI")
				}()
			}

			tt.setupMocks()

			handler := NewAuthHandler()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("GET", "/callback?"+tt.queryParams, nil)
			c.Request = req

			handler.Callback(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse models.ErrorResponse
				err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
				assert.NoError(t, err)
				assert.Contains(t, errorResponse.Error, tt.expectedError)
			}
		})
	}
}
