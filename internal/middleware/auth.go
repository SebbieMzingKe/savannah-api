package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/SebbieMzingKe/customer-order-api/internal/handlers"
	"github.com/SebbieMzingKe/customer-order-api/internal/models"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// Claims defines the structure of the JWT claims.
// It embeds jwt.RegisteredClaims to handle standard claims like expiration (exp),
type Claims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Sub   string `json:"sub"`
	Iss   string `json:"iss"`
	jwt.RegisteredClaims
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "missing token", Message: "missing token", Code: http.StatusUnauthorized})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid token format", Message: "invalid token format", Code: http.StatusUnauthorized})
			c.Abort()
			return
		}

		tokenString := parts[1]
		secret := []byte(os.Getenv("JWT_SECRET"))
		if len(secret) == 0 {
			secret = []byte("secret-key")
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return secret, nil
		})

		if err != nil {
			if strings.Contains(err.Error(), "token is malformed") {
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid token", Message: "malformed token", Code: http.StatusUnauthorized})
			} else if strings.Contains(err.Error(), "token is expired") {
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid token", Message: "expired token", Code: http.StatusUnauthorized})
			} else {
				c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid token", Message: err.Error(), Code: http.StatusUnauthorized})
			}
			c.Abort()
			return
		}

		// Additional validation for token validity and expiration
		if !token.Valid {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid token", Message: "invalid token", Code: http.StatusUnauthorized})
			c.Abort()
			return
		}

		// Check if token is expired (additional check)
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid token", Message: "expired token", Code: http.StatusUnauthorized})
			c.Abort()
			return
		}

		c.Set("claims", claims)
		c.Set("user_email", claims.Email)
		c.Set("user_sub", claims.Sub)
		c.Next()
	}
}

type AuthHandler struct {
	provider   *oidc.Provider
	verifier   *oidc.IDTokenVerifier
	oauth2     *oauth2.Config
	oidcConfig bool
}

func NewAuthHandler() *AuthHandler {
	providerURL := os.Getenv("OIDC_PROVIDER_URL")
	clientID := os.Getenv("OIDC_CLIENT_ID")
	clientSecret := os.Getenv("OIDC_CLIENT_SECRET")
	redirectURL := os.Getenv("OIDC_REDIRECT_URI")

	if providerURL == "" || clientID == "" || clientSecret == "" || redirectURL == "" {
		return &AuthHandler{oidcConfig: false}
	}

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, providerURL)
	if err != nil {
		return &AuthHandler{oidcConfig: false}
	}

	oauth2Config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		Endpoint:     provider.Endpoint(),
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	return &AuthHandler{
		provider:   provider,
		verifier:   verifier,
		oauth2:     oauth2Config,
		oidcConfig: true,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request", Message: "invalid request"})
		return
	}

	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request", Message: "invalid request"})
		return
	}

	if h.oidcConfig {
		state := "state-" + time.Now().Format("20060102150405")
		url := h.oauth2.AuthCodeURL(state)
		c.Redirect(http.StatusFound, url)
		return
	}

	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token generation failed", Message: "token generation failed"})
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &handlers.Claims{
		Email: req.Email,
		Sub:   req.Email,
		Name:  "Seb",
		Iss:   "customer-order-api",
		Aud:   "customer-order-api",
		Exp:   expirationTime.Unix(),
		Iat:   time.Now().Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "customer-order-api",
			Subject:   req.Email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token generation failed", Message: "token generation failed"})
		return
	}

	response := models.AuthResponse{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   int64((24 * time.Hour).Seconds()),
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) Callback(c *gin.Context) {
	if !h.oidcConfig {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "oidc_not_configured",
			Message: "OIDC provider not configured",
			Code:    http.StatusBadRequest,
		})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "missing code", Message: "missing code"})
		return
	}

	state := c.Query("state")
	ctx := c.Request.Context()
	oauth2Token, err := h.oauth2.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_exchange_failed", Message: err.Error()})
		return
	}

	idToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "id_token_missing", Message: "id_token missing"})
		return
	}

	token, err := h.verifier.Verify(ctx, idToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid_id_token", Message: err.Error()})
		return
	}

	var claims struct {
		Email string `json:"email"`
		Sub   string `json:"sub"`
		Name  string `json:"name"`
	}
	if err := token.Claims(&claims); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "invalid_id_token", Message: err.Error()})
		return
	}

	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		secret = []byte("secret-key")
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	jwtClaims := &handlers.Claims{
		Email: claims.Email,
		Sub:   claims.Sub,
		Name:  claims.Name,
		Iss:   "customer-order-api",
		Aud:   "customer-order-api",
		Exp:   expirationTime.Unix(),
		Iat:   time.Now().Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "customer-order-api",
			Subject:   claims.Sub,
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	tokenString, err := jwtToken.SignedString(secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "token_generation_failed", Message: err.Error()})
		return
	}

	response := gin.H{
		"auth": models.AuthResponse{
			AccessToken: tokenString,
			TokenType:   "Bearer",
			ExpiresIn:   int64((24 * time.Hour).Seconds()),
		},
		"state": state,
	}

	c.JSON(http.StatusOK, response)
}

// ValidateToken is a helper function to validate a token string.
func (h *AuthHandler) ValidateToken(tokenString string) (*Claims, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		secret = []byte("secret-key")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
