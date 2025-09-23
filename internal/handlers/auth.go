package handlers

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/SebbieMzingKe/customer-order-api/internal/models"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/oauth2"
)

type AuthHandler struct {
	jwtSecret    []byte
	provider     *oidc.Provider
	Verifier     *oidc.IDTokenVerifier
	oauth2Config *oauth2.Config
	oidcEnabled  bool
	redirectURI  string
}

type Claims struct {
	Email string `json:"email"`
	Sub   string `json:"sub"`
	Name  string `json:"name"`
	Iss   string `json:"iss"`
	Aud   string `json:"aud"`
	Exp   int64  `json:"exp"`
	Iat   int64  `json:"iat"`
	jwt.RegisteredClaims
}

func NewAuthHandler() *AuthHandler {
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))

	h := &AuthHandler{
		jwtSecret:   jwtSecret,
		oidcEnabled: false,
	}

	providerURL := os.Getenv("OIDC_PROVIDER_URL")
	clientID := os.Getenv("OIDC_CLIENT_ID")
	clientSecret := os.Getenv("OIDC_CLIENT_SECRET")
	redirectURI := os.Getenv("OIDC_REDIRECT_URI")

	if providerURL != "" && clientID != "" && clientSecret != "" && redirectURI != "" {
		ctx := context.Background()
		provider, err := oidc.NewProvider(ctx, providerURL)
		if err == nil {
			verifier := provider.Verifier(&oidc.Config{ClientID: clientID})
			oauth2Config := &oauth2.Config{
				ClientID:     clientID,
				ClientSecret: clientSecret,
				Endpoint:     provider.Endpoint(),
				Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
				RedirectURL:  redirectURI,
			}
			h.provider = provider
			h.Verifier = verifier
			h.oauth2Config = oauth2Config
			h.oidcEnabled = true
			h.redirectURI = redirectURI
		}
	}

	return h
}

func (h *AuthHandler) Login(c *gin.Context) {
	if h.oidcEnabled {
		state := "state-" + time.Now().Format("20060102150405")
		authURL := h.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
		c.Redirect(http.StatusFound, authURL)
		return
	}

	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request",
			Message: "invalid request",
			Code:    http.StatusBadRequest,
		})
		return
	}

	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request",
			Message: "invalid request",
			Code:    http.StatusBadRequest,
		})
		return
	}

	if len(h.jwtSecret) == 0 {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "token generation failed",
			Message: "token generation failed",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
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
	tokenString, err := token.SignedString(h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "token generation failed",
			Message: "token generation failed",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	response := models.AuthResponse{
		AccessToken: tokenString,
		ExpiresIn:   int64(24 * time.Hour / time.Second),
		TokenType:   "Bearer",
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) Callback(c *gin.Context) {
	if !h.oidcEnabled {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "oidc_not_configured",
			Message: "OIDC provider not configured",
			Code:    http.StatusBadRequest,
		})
		return
	}

	ctx := context.Background()
	code := c.Query("code")
	state := c.Query("state")
	if code == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "missing code",
			Message: "authorization code is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	token, err := h.oauth2Config.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "token_exchange_failed",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "id_token_missing",
			Message: "no id_token in token response",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Verify ID Token
	idToken, err := h.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid_id_token",
			Message: err.Error(),
			Code:    http.StatusUnauthorized,
		})
		return
	}

	var oidcClaims struct {
		Email string `json:"email"`
		Sub   string `json:"sub"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&oidcClaims); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "claims_parse_error",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Email: oidcClaims.Email,
		Sub:   oidcClaims.Sub,
		Name:  oidcClaims.Name,
		Iss:   "customer-order-api",
		Aud:   "customer-order-api",
		Exp:   expirationTime.Unix(),
		Iat:   time.Now().Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			Issuer:    "customer-order-api",
			Subject:   oidcClaims.Sub,
		},
	}
	localToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	localTokenString, err := localToken.SignedString(h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "token_generation_failed",
			Message: "could not generate access token",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	response := models.AuthResponse{
		AccessToken: localTokenString,
		ExpiresIn:   86400,
		TokenType:   "Bearer",
	}

	// Return minimal response - redirect to frontend with token as fragment if neccessary/desired)
	c.JSON(http.StatusOK, gin.H{
		"auth":  response,
		"state": state,
	})
}

func (h *AuthHandler) UserInfo(c *gin.Context) {
	claimsI, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "no user info available",
			Code:    http.StatusUnauthorized,
		})
		return
	}
	userClaims := claimsI.(*models.Claims)
	c.JSON(http.StatusOK, gin.H{
		"sub":   userClaims.Sub,
		"email": userClaims.Email,
		"name":  userClaims.Name,
		"iss":   userClaims.Iss,
		"aud":   userClaims.Aud,
		"exp":   userClaims.Exp,
		"iat":   userClaims.Iat,
	})
}

func (h *AuthHandler) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return h.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}
