package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SebbieMzingKe/customer-order-api/internal/handlers"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
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
			name:           "missing authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "missing token",
		},
		{
			name:           "missing bearer prefix",
			authHeader:     "token123",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid token format",
		},
		{
			name:           "valid token",
			authHeader:     "Bearer " + generateTestToken("test@example.com", secret, true),
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "missing token",
		},
		{
			name:           "Invalid token",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				assert.Contains(t, w.Body.String(), tt.expectedError)
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
		req, _ := http.NewRequest("OPTIONS", "test", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
	})

	t.Run("GET request with CORS headers", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "test", nil)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestAuthMiddlewareContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := []byte("test-secret")
	email := "test@example.com"
	token := generateTestToken(email, secret, false)

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/test", func(ctx *gin.Context) {
		claims, exists := ctx.Get("claims")
		assert.True(t, exists)

		userEmail, exists := ctx.Get("user_email")
		assert.True(t, exists)
		assert.Equal(t, email, userEmail)

		userSub, exists := ctx.Get("user_sub")
		assert.True(t, exists)
		assert.Equal(t, email, userSub)

		ctx.JSON(http.StatusOK, gin.H{
			"message": "success",
			"email":   claims.(*handlers.Claims).Email,
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), email)
}
