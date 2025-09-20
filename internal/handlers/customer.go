package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/SebbieMzingKe/customer-order-api/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CustomerHandler struct {
	db *gorm.DB
}

func NewCustomerHandler(db *gorm.DB) *CustomerHandler {
	return &CustomerHandler{db: db}
}

// CreateCustomer creates new customer
func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var req models.CreateCustomerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	var existingCustomer models.Customer
	if err := h.db.Where("code = ?", req.Code).First(&existingCustomer).Error; err == nil {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error:   "customer_exists",
			Message: "customer with this code already exists",
			Code:    http.StatusConflict,
		})
		return
	}

	customer := models.Customer{
		Name:  req.Name,
		Code:  req.Code,
		Phone: req.Phone,
		Email: req.Email,
	}

	if err := h.db.Create(&customer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database error",
			Message: "failed to create customer",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusCreated, customer)
}

func (h *CustomerHandler) GetCustomers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var customers []models.Customer
	var total int64

	h.db.Model(&models.Customer{}).Count(&total)

	if err := h.db.Preload("Orders").Offset(offset).Limit(limit).Find(&customers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database error",
			Message: "failed to retrieve customers",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"customers": customers,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

func (h *CustomerHandler) GetCustomer(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 32)

	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid id",
			Message: "invalid customer id",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var customer models.Customer

	if err := h.db.Preload("Orders").First(&customer, id).Error; err != nil {
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
			Message: "failed to retrieve customer",
			Code:    http.StatusInternalServerError,
		})
		return
	}
	c.JSON(http.StatusOK, customer)
}

func (h *CustomerHandler) UpdateCustomer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid id",
			Message: "invalid customer id",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var req models.UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	var customer models.Customer
	if err := h.db.First(&customer, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "customer not found",
				Message: "customer not found",
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database error",
			Message: "failed to retrieve customer",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Apply updates
	if req.Name != "" {
		customer.Name = req.Name
	}
	if req.Phone != "" {
		customer.Phone = req.Phone
	}
	if req.Email != "" {
		var existingCustomer models.Customer
		if err := h.db.Where("email = ? AND id != ?", req.Email, id).First(&existingCustomer).Error; err == nil {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error:   "email already in use",
				Message: "email already in use",
				Code:    http.StatusConflict,
			})
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "database error",
				Message: "failed to check email",
				Code:    http.StatusInternalServerError,
			})
			return
		}
		customer.Email = req.Email
	}

	if err := h.db.Save(&customer).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database error",
			Message: "failed to update customer",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, customer)
}

func (h *CustomerHandler) DeleteCustomer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid id",
			Message: "invalid customer id",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var customer models.Customer
	if err := h.db.First(&customer, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "customer not found",
				Message: "customer not found",
				Code:    http.StatusNotFound,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database error",
			Message: "failed to retrieve customer",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	if err := h.db.Delete(&models.Customer{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "database error",
			Message: "failed to delete customer",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "customer deleted successfully"})
}
