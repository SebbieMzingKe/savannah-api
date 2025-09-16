package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"savannah-api/internal/models"
	"savannah-api/internal/services"
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

// CreateOrder creates a new order and sends SMS notification
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req models.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Check if customer exists
	var customer models.Customer
	if err := h.db.First(&customer, req.CustomerID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "customer_not_found",
				Message: "Customer not found",
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to verify customer",
			Code:    http.StatusInternalServerError,
		})
		return
	}

order := models.Order{
	Item:       req.Item,
	Amount:     req.Amount,
	Time:       req.Time,
	CustomerID: req.CustomerID,
}

	if err := h.db.Create(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to create order",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Load the customer relationship
	h.db.Preload("Customer").First(&order, order.ID)

	// Send SMS notification to customer
	go h.sendOrderNotification(customer, order)

	c.JSON(http.StatusCreated, order)
}

// GetOrders retrieves all orders with pagination
func (h *OrderHandler) GetOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	customerID := c.Query("customer_id")
	offset := (page - 1) * limit

	var orders []models.Order
	var total int64
	query := h.db.Model(&models.Order{})

	// Filter by customer ID if provided
	if customerID != "" {
		query = query.Where("customer_id = ?", customerID)
	}

	// Get total count
	query.Count(&total)

	// Get paginated results with preloaded customer
	if err := query.Preload("Customer").Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve orders",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// GetOrder retrieves a specific order by ID
func (h *OrderHandler) GetOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid order ID",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var order models.Order
	if err := h.db.Preload("Customer").First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "order_not_found",
				Message: "Order not found",
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve order",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, order)
}

// UpdateOrder updates an existing order
func (h *OrderHandler) UpdateOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid order ID",
			Code:    http.StatusBadRequest,
		})
		return
	}
}

	var req models.UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	var order models.Order
	if err := h.db.First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "order_not_found",
				Message: "Order not found",
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to retrieve order",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Update fields if provided
	if req.Item != "" {
		order.Item = req.Item
	}
	if req.Amount > 0 {
		order.Amount = req.Amount
	}
	if !req.Time.IsZero() {
		order.Time = req.Time
	}

	if err := h.db.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to update order",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Load customer relationship
	h.db.Preload("Customer").First(&order, order.ID)

	c.JSON(http.StatusOK, order)
}

// DeleteOrder deletes an order
func (h *OrderHandler) DeleteOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid order ID",
			Code:    http.StatusBadRequest,
		})
		return
	}

	if err := h.db.Delete(&models.Order{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database_error",
			Message: "Failed to delete order",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order deleted successfully"})
}

// sendOrderNotification sends an SMS notification to the customer about their order
func (h *OrderHandler) sendOrderNotification(customer models.Customer, order models.Order) {
	message := fmt.Sprintf("Hi %s, your order for %s (Amount: KSH %.2f) has been received. Order Time: %s. Thank you for your business!",
		customer.Name, order.Item, order.Amount, order.Time.Format("2006-01-02 15:04:05"))

	if err := h.smsService.SendSMS(customer.Phone, message); err != nil {
		log.Printf("Failed to send SMS to customer %s: %v", customer.Name, err)
		return
	}

	log.Printf("SMS sent successfully to customer %s", customer.Name)
}