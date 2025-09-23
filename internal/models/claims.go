package models

import "github.com/golang-jwt/jwt/v4"

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
