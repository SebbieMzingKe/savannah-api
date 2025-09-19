package middleware

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/SebbieMzingKe/customer-order-api/internal/handlers"
	"github.com/SebbieMzingKe/customer-order-api/internal/models"
	
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func AuthMiddleware() gin.HandlerFunc {
	authHandler := handlers.NewAuthHandler()

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "missing token",
				Message: "authorization header is required",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid token format",
				Message: "authorization header must be in format 'Bearer <token>",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}
		tokenString := parts[1]

		claims, err := authHandler.ValidateToken(tokenString)
		if err != nil {
			var errorMsg string
			if ve, ok := err.(*jwt.ValidationError); ok {
				if ve.Errors&jwt.ValidationErrorMalformed != 0 {
					errorMsg = "malformed token"
				} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
					errorMsg = "token expired or not active yet"
				} else {
					errorMsg = "invalid token"
				}
			} else {
				errorMsg = "invalid token"
			}

			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid token",
				Message: errorMsg,
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		c.Set("claims", claims)
		c.Set("user_email", claims.Email)
		c.Set("user_sub", claims.Sub)

		c.Next()
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Conent-Type")
		c.Writer.Header().Set("Access-Control-Allow-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func LoggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format("02/Jan/2026:15:04:05 - 0700"),
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

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Header("X-Content-Type-Options", "nosniff")

		c.Header("X-Frame-Options", "DENY")

		c.Header("X-XSS-Protection", "1; mode=block")

		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'")

		c.Header("X-Powered-By", "")
		c.Header("Server", "")

		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

func RateLimitMiddleware() gin.HandlerFunc {

	type Client struct {
		requests []time.Time
		limit    int
		window   time.Duration
	}

	clients := make(map[string]*Client)
	defaultLimit := 100
	defaultWindow := time.Minute

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		if clients[clientIP] == nil {
			clients[clientIP] = &Client{
				requests: make([]time.Time, 0),
				limit:    defaultLimit,
				window:   defaultWindow,
			}
		}

		client := clients[clientIP]

		var validRequests []time.Time

		for _, req := range client.requests {
			if now.Sub(req) < client.window {
				validRequests = append(validRequests, req)
			}
		}
		client.requests = validRequests

		if len(client.requests) >= client.limit {
			c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
				Error:   "rate limit exceeded",
				Message: "too many requests try again later",
				Code:    http.StatusTooManyRequests,
			})
			c.Abort()
			return
		}

		client.requests = append(client.requests, now)

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", client.limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", client.limit-len(client.requests)))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(client.window).Unix()))

		c.Next()
	}
}

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set("request-id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

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
		case <- done:
			return
		case <- ctx.Done():
			c.JSON(http.StatusRequestTimeout, models.ErrorResponse{
				Error:   "request timeout",
				Message: "request timeout",
				Code:    http.StatusRequestTimeout,
			})
			c.Abort()
		}
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "unauthorized",
				Message: "authentiction required",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}
		userClaims := claims.(*handlers.Claims)
		if userClaims.Email != "sebbievilar2@gmail.com" {
			c.JSON(http.StatusForbidden, models.ErrorResponse{
				Error: "forbidden",
				Message: "admin privileges required",
				Code: http.StatusForbidden,
			})
			c.Abort()
			return 
		}
		c.Next()
	}
}

func ValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			contentType := c.GetHeader("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				c.JSON(http.StatusUnsupportedMediaType, models.ErrorResponse{
					Error: "unsupported media type",
					Message: "content type must be application json",
					Code: http.StatusUnsupportedMediaType,
				})
				c.Abort()
				return 
			}
		}
		c.Next()
	}
}

func generateRequestID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomString(8))
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
