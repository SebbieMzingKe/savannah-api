package middleware

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"

	"savannah-api/internal/handlers"
	"savannah-api/internal/models"
)

// AuthMiddleware validates JWT tokens and sets user context
func AuthMiddleware() gin.HandlerFunc {
	authHandler := handlers.NewAuthHandler()
	
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "missing_token",
				Message: "Authorization header is required",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_token_format",
				Message: "Authorization header must be in format 'Bearer <token>'",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := authHandler.ValidateToken(tokenString)
		if err != nil {
			var errorMsg string
			if ve, ok := err.(*jwt.ValidationError); ok {
				if ve.Errors&jwt.ValidationErrorMalformed != 0 {
					errorMsg = "Malformed token"
				} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
					errorMsg = "Token expired or not active yet"
				} else {
					errorMsg = "Invalid token"
				}
			} else {
				errorMsg = "Invalid token"
			}

			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_token",
				Message: errorMsg,
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Set claims in context for use in handlers
		c.Set("claims", claims)
		c.Set("user_email", claims.Email)
		c.Set("user_sub", claims.Sub)

		c.Next()
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// LoggingMiddleware provides detailed request/response logging
func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format("02/Jan/2006:15:04:05 -0700"),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}

// SecurityHeadersMiddleware adds security headers to responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		
		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Control referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'")
		
		// Prevent information disclosure
		c.Header("X-Powered-By", "")
		c.Header("Server", "")
		
		// HSTS (only if using HTTPS)
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// RateLimitMiddleware implements basic rate limiting
func RateLimitMiddleware() gin.HandlerFunc {
	// Simple in-memory rate limiter (for production, use Redis)
	type client struct {
		requests []time.Time
		limit    int
		window   time.Duration
	}

	clients := make(map[string]*client)
	defaultLimit := 100 // requests per minute
	defaultWindow := time.Minute

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		// Initialize client if not exists
		if clients[clientIP] == nil {
			clients[clientIP] = &client{
				requests: make([]time.Time, 0),
				limit:    defaultLimit,
				window:   defaultWindow,
			}
		}

		client := clients[clientIP]

		// Remove old requests outside the window
		var validRequests []time.Time
		for _, req := range client.requests {
			if now.Sub(req) < client.window {
				validRequests = append(validRequests, req)
			}
		}
		client.requests = validRequests

		// Check if limit exceeded
		if len(client.requests) >= client.limit {
			c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
				Error:   "rate_limit_exceeded",
				Message: "Too many requests, please try again later",
				Code:    http.StatusTooManyRequests,
			})
			c.Abort()
			return
		}

		// Add current request
		client.requests = append(client.requests, now)

		// Add rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", client.limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", client.limit-len(client.requests)))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(client.window).Unix()))

		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		
		c.Next()
	}
}

// TimeoutMiddleware sets a timeout for requests
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		
		done := make(chan struct{})
		go func() {
			c.Next()
			done <- struct{}{}
		}()

		select {
		case <-done:
			return
		case <-ctx.Done():
			c.JSON(http.StatusRequestTimeout, models.ErrorResponse{
				Error:   "request_timeout",
				Message: "Request timeout",
				Code:    http.StatusRequestTimeout,
			})
			c.Abort()
		}
	}
}

// AdminMiddleware checks if user has admin privileges
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "Authentication required",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		userClaims := claims.(*handlers.Claims)
		
		// Check if user has admin role (customize based on your auth system)
		if userClaims.Email != "admin@savannahinformatics.com" {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error:   "forbidden",
				Message: "Admin privileges required",
				Code:    http.StatusForbidden,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ValidationMiddleware provides additional input validation
func ValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add custom validation logic here
		// For example, check content type for POST/PUT requests
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			contentType := c.GetHeader("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				c.JSON(http.StatusUnsupportedMediaType, models.ErrorResponse{
					Error:   "unsupported_media_type",
					Message: "Content-Type must be application/json",
					Code:    http.StatusUnsupportedMediaType,
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// Helper functions

// generateRequestID creates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomString(8))
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}