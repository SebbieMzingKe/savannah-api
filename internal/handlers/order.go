package handlers

import "gorm.io/gorm"

type OrderHandler struct {
	db *gorm.DB
	// smsService *services.SMSService
}