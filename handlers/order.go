package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/SebbieMzingKe/customer-order-api/internal/models"
	"github.com/SebbieMzingKe/customer-order-api/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderHandler struct {
	db         *gorm.DB
	smsService services.SMSServiceInterface
}

func NewOrderHandler(db *gorm.DB, smsService services.SMSServiceInterface) *OrderHandler {
	return &OrderHandler{
		db:         db,
		smsService: smsService,
	}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req models.CreateOrderRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	if req.Item == "" || req.Amount <= 0 || req.CustomerID == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request",
			Message: "missing or invalid fields",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var customer models.Customer

	if err := h.db.First(&customer, req.CustomerID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "customer not found",
				Message: "customer not found",
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database error",
			Message: "failed to verify customer",
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
			Error:   "database error",
			Message: "failed to create order",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	order.Customer = customer

	go h.sendOrderNotification(customer, order)

	c.JSON(http.StatusCreated, order)
}


func (h *OrderHandler) GetOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	customerID := c.Query("customer_id")
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	var orders []models.Order
	var total int64
	query := h.db.Model(&models.Order{})

	if customerID != "" {
		query = query.Where("customer_id = ?", customerID)
	}

	query.Count(&total)

	if err := query.Preload("Customer").Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database error",
			Message: "failed to retrieve orders",
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

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)

	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid id",
			Message: "invalid order id",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var order models.Order
	if err := h.db.Preload("Customer").First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "order not found",
				Message: "order not found",
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database error",
			Message: "failed to retrieve order",
			Code:    http.StatusInternalServerError,
		})
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) UpdateOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid id",
			Message: "invalid order id",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var req models.UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Println("JSON bind error:", err) 
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	if req.Amount < 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request",
			Message: "amount cannot be negative",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var order models.Order
	if err := h.db.First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "order not found",
				Message: "order not found",
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database error",
			Message: "failed to retrieve order",
			Code:    http.StatusInternalServerError,
		})
		return
	}

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
			Error:   "database error",
			Message: "failed to update order",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	h.db.Preload("Customer").First(&order, order.ID)
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) DeleteOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid id",
			Message: "invalid order id",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var order models.Order
	if err := h.db.First(&order, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "order not found",
				Message: "order not found",
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database error",
			Message: "failed to retrieve order",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	if err := h.db.Delete(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database error",
			Message: "failed to delete order",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order deleted successfully"})
}

func (h *OrderHandler) sendOrderNotification(customer models.Customer, order models.Order) {
	message := fmt.Sprintf("hello %s, your order for %s (amount: ksh %.2f) has been received. order time: %s. thank you for your business",
		customer.Name, order.Item, order.Amount, order.Time.Format("2006-01-02 15:04:05"))

	if err := h.smsService.SendSMS(customer.Phone, message); err != nil {
		log.Printf("failed to send sms to customer %s: %v", customer.Name, err)
		return
	}

	log.Printf("sms sent successfully to customer %s", customer.Name)
}
