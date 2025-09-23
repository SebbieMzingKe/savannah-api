package models

import (
	"time"

	"gorm.io/gorm"
)

// Customer - customer in the system
type Customer struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"not null" binding:"required"`
	Code      string         `json:"code" gorm:"uniqueIndex;not null" binding:"required"`
	Phone     string         `json:"phone" gorm:"not null" binding:"required"`
	Email     string         `json:"email" gorm:"uniqueIndex"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	Orders    []Order        `json:"orders,omitempty" gorm:"foreignKey:CustomerID"`
}

type Order struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	Item       string         `json:"item" gorm:"not null" binding:"required"`
	Amount     float64        `json:"amount" gorm:"not null" binding:"required,min=0"`
	Time       time.Time      `json:"time" gorm:"not null"`
	CustomerID uint           `json:"customer_id" gorm:"not null" binding:"required"`
	Customer   Customer       `json:"customer,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

type CreateCustomerRequest struct {
	Name  string `json:"name" binding:"required"`
	Code  string `json:"code" binding:"required"`
	Phone string `json:"phone" binding:"required"`
	Email string `json:"email" binding:"email"`
}

type UpdateCustomerRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email" binding:"omitempty,email"`
}

type CreateOrderRequest struct {
	Item       string    `json:"item" binding:"required"`
	Amount     float64   `json:"amount" binding:"required,min=0"`
	Time       time.Time `json:"time" binding:"required"`
	CustomerID uint      `json:"customer_id" binding:"required"`
}

type UpdateOrderRequest struct {
	Item   string    `json:"item"`
	Amount float64   `json:"amount" binding:"omitempty,min=0"`
	Time   time.Time `json:"time" binding:"omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
