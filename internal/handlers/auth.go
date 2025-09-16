package handlers

import (
	"net/http"
	"time"

	"github.com/SebbieMzingKe/customer-order-api/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

type AuthHandler struct {
	jwtSecret []byte
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		jwtSecret: []byte("secret-key"),
	}
}

type Claims struct {
	Email string `json:"email"`
	Sub   string `json:"sub"`
	Name  string `json:"name"`
	Iss   string `json:"iss"`
	Aud   string `json:"aud"`
	Exp   int64  `json:"exp"`
	Iat   int64  `json:"iat"`
	jwt.StandardClaims
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "invalid credentials",
			Message: "invalid email or password",
			Code:    http.StatusUnauthorized,
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
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			Issuer:    "customer-order-api",
			Subject:   req.Email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(h.jwtSecret)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "token generation failed",
			Message: "could not generate access token",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	response := models.AuthResponse{
		AccessToken: tokenString,
		ExpiresIn:   86400,
		TokenType:   "Bearer",
	}

	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) Callback(c *gin.Context) {
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

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Email: "sebbivilar@gmail.com", // This would come from the OIDC provider
		Sub:   "Seb",
		Name:  "Seb",
		Iss:   "customer-order-api",
		Aud:   "customer-order-api",
		Exp:   expirationTime.Unix(),
		Iat:   time.Now().Unix(),
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			Issuer:    "customer-order-api",
			Subject:   "sebbievayo2@gmail.com",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(h.jwtSecret)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "token generation failed",
			Message: "could not generate access token",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	response := models.AuthResponse{
		AccessToken: tokenString,
		ExpiresIn:   86400,
		TokenType:   "Bearer",
	}

	c.JSON(http.StatusOK, gin.H{
		"auth": response,
		"state": state,
	})
}

func(h *AuthHandler) UserInfo(c *gin.Context) {
	claims, exists := c.Get("claims")

	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "no user info available",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	userClaims := claims.(*Claims)
	c.JSON(http.StatusOK, gin.H{
		"sub": userClaims.Sub,
		"email": userClaims.Email,
		"name": userClaims.Name,
		"iss": userClaims.Iss,
		"aud": userClaims.Aud,
		"exp": userClaims.Exp,
		"iat": userClaims.Iat,
	})
}

func (h *AuthHandler) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid{
		return nil, err
	}

	return claims, err
}