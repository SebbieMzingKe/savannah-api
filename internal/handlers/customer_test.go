package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SebbieMzingKe/customer-order-api/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database")
	}

	db.AutoMigrate(&models.Customer{}, &models.Order{})
	return db
}

func TestCreateCustomer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB()
	handler := NewCustomerHandler(db)

	tests := []struct {
		name           string
		requestBody    models.CreateCustomerRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid customer creation",
			requestBody: models.CreateCustomerRequest{
				Name:  "Sebbie Chanzu",
				Code:  "CUST001",
				Phone: "+254740827150",
				Email: "sebbievilar2@gmail.com",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "duplicate customer code",
			requestBody: models.CreateCustomerRequest{
				Name:  "Sebbie Mzing",
				Code:  "CUST001",
				Phone: "+254740827150",
				Email: "sebbievilar2@gmail.com",
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "customer_exists",
		},

		{
			name: "missing required fields",
			requestBody: models.CreateCustomerRequest{
				Name: "Sebbie Chanzu",
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
			req, _ := http.NewRequest("POST", "/customers", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			handler.CreateCustomer(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse models.ErrorResponse
				json.Unmarshal(w.Body.Bytes(), &errorResponse)
				assert.Equal(t, tt.expectedError, errorResponse.Error)
			}
		})
	}
}

func TestGetCustomer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB()
	handler := NewCustomerHandler(db)

	customer := models.Customer{
		Name:  "Sebbie Mzing",
		Code:  "CUST001",
		Phone: "+254740827150",
		Email: "sebbievilar2@gmail.com",
	}
	db.Create(&customer)

	tests := []struct {
		name           string
		customerID     string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid customer ID",
			customerID:     "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid customer ID",
			customerID:     "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid id",
		},
		{
			name:           "non-existent customer",
			customerID:     "999",
			expectedStatus: http.StatusNotFound,
			expectedError:  "customer not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("GET", "/customers/"+tt.customerID, nil)
			c.Request = req
			c.Params = []gin.Param{{Key: "id", Value: tt.customerID}}

			handler.GetCustomer(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse models.ErrorResponse
				json.Unmarshal(w.Body.Bytes(), &errorResponse)
				assert.Equal(t, tt.expectedError, errorResponse.Error)
			}
		})
	}
}

func TestGetCustomers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB()
	handler := NewCustomerHandler(db)

	customers := []models.Customer{
		{Name: "Sebbie Chanzu", Code: "CUST001", Phone: "+254740827150", Email: "sebbievilar2@gmail.com"},
		{Name: "Sebbie Mzing", Code: "CUST002", Phone: "+254111768132", Email: "sebbievayo2@gmail.com"},
		{Name: "Sebbie Evayo", Code: "CUST003", Phone: "+254740834150", Email: "sebbiemzing2@gmail.com"},
	}

	for _, customer := range customers {
		db.Create(&customer)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/customers", nil)
	c.Request = req

	handler.GetCustomers(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Contains(t, response, "customers")
	assert.Contains(t, response, "total")
	assert.Equal(t, float64(3), response["total"])
}
