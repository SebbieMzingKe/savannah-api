package handlers

import (
	"net/http"

	"github.com/SebbieMzingKe/customer-order-api/internal/models"
	"github.com/SebbieMzingKe/customer-order-api/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderHandler struct {
	db         *gorm.DB
	smsService *services.SMSService
}

func NewOrderHandler(db *gorm.DB, smsService *services.SMSService) *OrderHandler {
	return &OrderHandler{
		db:         db,
		smsService: smsService,
	}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req models.CreateOrderRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		
	}
}
