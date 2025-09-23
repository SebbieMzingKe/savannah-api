package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SebbieMzingKe/customer-order-api/models"
	"github.com/SebbieMzingKe/customer-order-api/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestCreateOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	mockSMSService := services.NewMockSMSService()
	handler := NewOrderHandler(db, mockSMSService)

	customer := models.Customer{
		Name:  "Sebbie Chanzu",
		Code:  "CUST001",
		Phone: "+254740827150",
		Email: "sebbievilar2@gmail.com",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("failed to create customer: %v", err)
	}

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
				CustomerID: uint(customer.ID),
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
				CustomerID: uint(customer.ID),
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
				CustomerID: uint(customer.ID),
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSMSService.SentMessages = nil
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
				assert.Len(t, mockSMSService.SentMessages, 0)

				if len(mockSMSService.SentMessages) > 0 {
					lastMessage := mockSMSService.SentMessages[len(mockSMSService.SentMessages)-1]
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
	db := setupTestDB(t)
	mockSMSService := services.NewMockSMSService()
	handler := NewOrderHandler(db, mockSMSService)

	customer := models.Customer{
		Name:  "Sebbie Chanzu",
		Code:  "CUST001",
		Phone: "+254740827150",
		Email: "sebbievilar2@gmail.com",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("failed to create customer: %v", err)
	}

	order := models.Order{
		Item:       "laptop",
		Amount:     1500.00,
		Time:       time.Now(),
		CustomerID: customer.ID,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	tests := []struct {
		name           string
		orderID        string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid order id",
			orderID:        "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid order id",
			orderID:        "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid id",
		},
		{
			name:           "non-existent order",
			orderID:        "999",
			expectedStatus: http.StatusNotFound,
			expectedError:  "order not found",
		},
	}

	for _, tt := range tests {
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
	db := setupTestDB(t)
	mockSMSService := services.NewMockSMSService()
	handler := NewOrderHandler(db, mockSMSService)

	customer := models.Customer{
		Name:  "Sebbie Chanzu",
		Code:  "CUST001",
		Phone: "+254740827150",
		Email: "sebbievilar2@gmail.com",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("failed to create customer: %v", err)
	}

	orders := []models.Order{
		{Item: "laptop", Amount: 1500.00, Time: time.Now(), CustomerID: customer.ID},
		{Item: "phone", Amount: 800.00, Time: time.Now(), CustomerID: customer.ID},
		{Item: "tablet", Amount: 600.00, Time: time.Now(), CustomerID: customer.ID},
	}

	for _, order := range orders {
		if err := db.Create(&order).Error; err != nil {
			t.Fatalf("failed to create order: %v", err)
		}
	}

	tests := []struct {
		name           string
		query          string
		expectedTotal  int
		expectedStatus int
	}{
		{
			name:           "get all orders",
			query:          "",
			expectedTotal:  3,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "filter orders by customer",
			query:          "customer_id=1",
			expectedTotal:  3,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("GET", "/orders?"+tt.query, nil)
			c.Request = req

			handler.GetOrders(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			assert.Contains(t, response, "orders")
			assert.Contains(t, response, "total")
			assert.Equal(t, float64(tt.expectedTotal), response["total"])
		})
	}
}

func TestUpdateOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	mockSMSService := services.NewMockSMSService()
	handler := NewOrderHandler(db, mockSMSService)

	customer := models.Customer{
		Name:  "Sebbie Chanzu",
		Code:  "CUST001",
		Phone: "+254740827150",
		Email: "sebbievilar2@gmail.com",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("failed to create customer: %v", err)
	}

	order := models.Order{
		Item:       "laptop",
		Amount:     1500.00,
		Time:       time.Now(),
		CustomerID: customer.ID,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	tests := []struct {
		name           string
		orderID        string
		requestBody    models.UpdateOrderRequest
		expectedStatus int
		expectedError  string
		expectedItem   string
		expectedAmount float64
		expectedTime   time.Time
	}{
		{
			name:    "valid full update",
			orderID: "1",
			requestBody: models.UpdateOrderRequest{
				Item:   "phone",
				Amount: 800.00,
				Time:   time.Now().Add(1 * time.Hour),
			},
			expectedStatus: http.StatusOK,
			expectedItem:   "phone",
			expectedAmount: 800.00,
			expectedTime:   time.Now().Add(1 * time.Hour).Truncate(time.Second),
		},
		{
			name:    "valid partial update",
			orderID: "1",
			requestBody: models.UpdateOrderRequest{
				Item:   "tablet",
				Amount: 0,
				Time:   time.Time{},
			},
			expectedStatus: http.StatusOK,
			expectedItem:   "tablet",
			expectedAmount: 800.00,
			expectedTime:   time.Now().Add(1 * time.Hour).Truncate(time.Second),
		},
		{
			name:           "invalid order id",
			orderID:        "invalid",
			requestBody:    models.UpdateOrderRequest{Item: "phone"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid id",
		},
		{
			name:           "non-existent order",
			orderID:        "999",
			requestBody:    models.UpdateOrderRequest{Item: "phone"},
			expectedStatus: http.StatusNotFound,
			expectedError:  "order not found",
		},
		{
			name:           "invalid request body",
			orderID:        "1",
			requestBody:    models.UpdateOrderRequest{Amount: -100.00},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/orders/"+tt.orderID, bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req
			c.Params = []gin.Param{{Key: "id", Value: tt.orderID}}

			handler.UpdateOrder(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse models.ErrorResponse
				json.Unmarshal(w.Body.Bytes(), &errorResponse)
				assert.Equal(t, tt.expectedError, errorResponse.Error)
			} else {
				var updatedOrder models.Order
				json.Unmarshal(w.Body.Bytes(), &updatedOrder)
				assert.Equal(t, tt.expectedItem, updatedOrder.Item)
				assert.Equal(t, tt.expectedAmount, updatedOrder.Amount)
				assert.WithinDuration(t, tt.expectedTime, updatedOrder.Time, time.Second)

				var dbOrder models.Order
				err := db.First(&dbOrder, tt.orderID).Error
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedItem, dbOrder.Item)
				assert.Equal(t, tt.expectedAmount, dbOrder.Amount)
				assert.WithinDuration(t, tt.expectedTime, dbOrder.Time, time.Second)
			}
		})
	}
}

func TestDeleteOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	mockSMSService := services.NewMockSMSService()
	handler := NewOrderHandler(db, mockSMSService)

	customer := models.Customer{
		Name:  "Sebbie Chanzu",
		Code:  "CUST001",
		Phone: "+254740827150",
		Email: "sebbievilar2@gmail.com",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("failed to create customer: %v", err)
	}

	order := models.Order{
		Item:       "laptop",
		Amount:     1500.00,
		Time:       time.Now(),
		CustomerID: customer.ID,
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	tests := []struct {
		name           string
		orderID        string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid order deletion",
			orderID:        "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid order id",
			orderID:        "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid id",
		},
		{
			name:           "non-existent order",
			orderID:        "999",
			expectedStatus: http.StatusNotFound,
			expectedError:  "order not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("DELETE", "/orders/"+tt.orderID, nil)
			c.Request = req
			c.Params = []gin.Param{{Key: "id", Value: tt.orderID}}

			handler.DeleteOrder(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse models.ErrorResponse
				json.Unmarshal(w.Body.Bytes(), &errorResponse)
				assert.Equal(t, tt.expectedError, errorResponse.Error)
			} else {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				assert.Equal(t, "order deleted successfully", response["message"])

				var dbOrder models.Order
				err := db.First(&dbOrder, tt.orderID).Error
				assert.Error(t, err)
				assert.Equal(t, gorm.ErrRecordNotFound, err)
			}
		})
	}
}
