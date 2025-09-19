package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SebbieMzingKe/customer-order-api/internal/models"
	"github.com/SebbieMzingKe/customer-order-api/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCreateOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB()
	mockSMSService := services.NewMockSMSService()
	handler := NewOrderHandler(db, mockSMSService)

	customer := models.Customer{
		Name:  "Sebbie Chanzu",
		Code:  "CUST001",
		Phone: "+254740827150",
		Email: "sebbievilar2@gmail.com",
	}
	db.Create(&customer)

	tests := []struct {
		name           string
		requestBody    models.CreateOrderRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid order creation",
			requestBody: models.CreateOrderRequest{
				Item:       "laptop",
				Amount:     1500.00,
				Time:       time.Now(),
				CustomerID: 1,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid customer id",
			requestBody: models.CreateOrderRequest{
				Item:       "phone",
				Amount:     800.00,
				Time:       time.Now(),
				CustomerID: 999,
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "customer not found",
		},
		{
			name: "missing required fields",
			requestBody: models.CreateOrderRequest{
				Time:       time.Now(),
				CustomerID: 1,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request",
		},
		{
			name: "negative amount",
			requestBody: models.CreateOrderRequest{
				Item:       "item",
				Amount:     -100.00,
				Time:       time.Now(),
				CustomerID: 1,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/orders", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			handler.CreateOrder(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse models.ErrorResponse
				json.Unmarshal(w.Body.Bytes(), &errorResponse)
				assert.Equal(t, tt.expectedError, errorResponse.Error)
			} else if tt.expectedStatus == http.StatusCreated {
				assert.Len(t, mockSMSService.SentMessages, len(mockSMSService.SentMessages))

				if len(mockSMSService.SentMessages) > 0 {
					lastMessage := mockSMSService.SentMessages[len(mockSMSService.SentMessages) - 1]
					assert.Equal(t, customer.Phone, lastMessage.To)
					assert.Contains(t, lastMessage.Message, customer.Name)
					assert.Contains(t, lastMessage.Message, tt.requestBody.Item)
				}
			}

		})
	}
}


func TestGetOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB()
	mockSMSService := services.NewMockSMSService()
	handler := NewOrderHandler(db, mockSMSService)

	customer := models.Customer{
		Name: "Sebbie Chanzu",
		Code: "CUST001",
		Phone: "+254740827150",
		Email: "sebbievilar2@gmail",
	}
	db.Create(&customer)

	order := models.Order{
		Item: "laptop",
		Amount: 1500.00,
		Time: time.Now(),
		CustomerID: customer.ID,
	}
	db.Create(&order)

	tests := []struct {
		name string
		orderID string
		expectedStatus int
		expectedError string
	}{
		{
			name: "valid order id",
			orderID: "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid order ID",
			orderID:        "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid_id",
		},
		{
			name:           "Non-existent order",
			orderID:        "999",
			expectedStatus: http.StatusNotFound,
			expectedError:  "order_not_found",
		},
	}

	for _, tt := range tests{
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("GET", "/orders/"+tt.orderID, nil)
			c.Request = req
			c.Params = []gin.Param{{Key: "id", Value: tt.orderID}}

			handler.GetOrder(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse models.ErrorResponse
				json.Unmarshal(w.Body.Bytes(), &errorResponse)
				assert.Equal(t, tt.expectedError, errorResponse.Error)
			}
		})
	}
}

func TestGetOrders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB()
	mockSMSService := services.NewMockSMSService()
	handler := NewOrderHandler(db, mockSMSService)

	customer := models.Customer{
		Name: "Sebbie Chanzu",
		Code: "CUST001",
		Phone: "+254740827150",
		Email: "sebbievilar2@gmail",
	}
	db.Create(&customer)

	orders := []models.Order{
		{Item: "laptop", Amount: 1500.00, Time: time.Now(), CustomerID: customer.ID},
		{Item: "phone", Amount: 800.00, Time: time.Now(), CustomerID: customer.ID},
		{Item: "tablet", Amount: 600.00, Time: time.Now(), CustomerID: customer.ID},
	}

	for _, order := range orders {
		db.Create(&order)
	}

	t.Run("get all orders", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/orders", nil)
		c.Request = req

		handler.GetOrders(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		assert.Contains(t, response, "orders")
		assert.Contains(t, response, "total")
		assert.Equal(t, float64(3), response["total"])
	})

	t.Run("filter orders by customer", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/orders?customer_id=1", nil)
		c.Request = req

		handler.GetOrders(c)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		assert.Contains(t, response, "orders")
		assert.Contains(t, response, "total")
		assert.Equal(t, float64(3), response["total"])
	})
}