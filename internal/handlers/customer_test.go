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

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	err = db.AutoMigrate(&models.Customer{}, &models.Order{})
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

func TestCreateCustomer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    models.CreateCustomerRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid customer creation",
			requestBody: models.CreateCustomerRequest{
				Name:  "Sebbie Mzing",
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
				Email: "different@gmail.com",
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "customer_exists",
		},
		{
			name: "missing required fields",
			requestBody: models.CreateCustomerRequest{
				Name:  "Sebbie Mzing",
				Code:  "",
				Phone: "+254740827150",
				Email: "sebbievilar2@gmail.com",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			handler := NewCustomerHandler(db)

			if tt.name == "duplicate customer code" {
				customer := models.Customer{
					Name:  "Sebbie Mzing",
					Code:  "CUST001",
					Phone: "+254740827150",
					Email: "sebbievilar2@gmail.com",
				}
				if err := db.Create(&customer).Error; err != nil {
					t.Fatalf("failed to create customer: %v", err)
				}
			}

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
			} else {
				var customer models.Customer
				json.Unmarshal(w.Body.Bytes(), &customer)
				assert.Equal(t, tt.requestBody.Name, customer.Name)
				assert.Equal(t, tt.requestBody.Code, customer.Code)
				assert.Equal(t, tt.requestBody.Phone, customer.Phone)
				assert.Equal(t, tt.requestBody.Email, customer.Email)
			}
		})
	}
}

func TestGetCustomer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		customerID     string
		setupCustomer  bool
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid customer ID",
			customerID:     "1",
			setupCustomer:  true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid customer ID",
			customerID:     "invalid",
			setupCustomer:  false,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid id",
		},
		{
			name:           "non-existent customer",
			customerID:     "999",
			setupCustomer:  false,
			expectedStatus: http.StatusNotFound,
			expectedError:  "customer not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			handler := NewCustomerHandler(db)

			if tt.setupCustomer {
				customer := models.Customer{
					Name:  "Sebbie Chanzu",
					Code:  "CUST001",
					Phone: "+254740827150",
					Email: "sebbievilar2@gmail.com",
				}
				if err := db.Create(&customer).Error; err != nil {
					t.Fatalf("failed to create customer: %v", err)
				}
			}

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
			} else {
				var customer models.Customer
				json.Unmarshal(w.Body.Bytes(), &customer)
				assert.Equal(t, "Sebbie Chanzu", customer.Name)
				assert.Equal(t, "CUST001", customer.Code)
				assert.Equal(t, "+254740827150", customer.Phone)
				assert.Equal(t, "sebbievilar2@gmail.com", customer.Email)
			}
		})
	}
}

func TestGetCustomers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	handler := NewCustomerHandler(db)

	customers := []models.Customer{
		{
			Name:  "Sebbie Chanzu",
			Code:  "CUST001",
			Phone: "+254740827150",
			Email: "sebbievilar2@gmail.com",
		},
		{
			Name:  "John Doe",
			Code:  "CUST002",
			Phone: "+254740827151",
			Email: "john.doe@gmail.com",
		},
	}
	for _, customer := range customers {
		if err := db.Create(&customer).Error; err != nil {
			t.Fatalf("failed to create customer: %v", err)
		}
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
	assert.Equal(t, float64(2), response["total"])

	customerList, ok := response["customers"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, customerList, 2)
}

func TestUpdateCustomer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		customerID     string
		requestBody    models.UpdateCustomerRequest
		setupCustomer  bool
		expectedStatus int
		expectedError  string
		expectedName   string
		expectedPhone  string
		expectedEmail  string
	}{
		{
			name:       "valid full update",
			customerID: "1",
			requestBody: models.UpdateCustomerRequest{
				Name:  "Sebbie Chanzu Updated",
				Phone: "+254111768132",
				Email: "sebbievayo2@gmail.com",
			},
			setupCustomer:  true,
			expectedStatus: http.StatusOK,
			expectedName:   "Sebbie Chanzu Updated",
			expectedPhone:  "+254111768132",
			expectedEmail:  "sebbievayo2@gmail.com",
		},
		{
			name:       "valid partial update",
			customerID: "1",
			requestBody: models.UpdateCustomerRequest{
				Phone: "+254111768132",
			},
			setupCustomer:  true,
			expectedStatus: http.StatusOK,
			expectedName:   "Sebbie Chanzu",
			expectedPhone:  "+254111768132",
			expectedEmail:  "sebbievilar2@gmail.com",
		},
		{
			name:           "invalid customer ID",
			customerID:     "invalid",
			requestBody:    models.UpdateCustomerRequest{Name: "Updated"},
			setupCustomer:  false,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid id",
		},
		{
			name:           "non-existent customer",
			customerID:     "999",
			requestBody:    models.UpdateCustomerRequest{Name: "Updated"},
			setupCustomer:  false,
			expectedStatus: http.StatusNotFound,
			expectedError:  "customer not found",
		},
		{
			name:       "email conflict on update",
			customerID: "1",
			requestBody: models.UpdateCustomerRequest{
				// Attempt to use the email that belongs to the conflictCustomer
				Email: "conflict@example.com",
			},
			setupCustomer:  true,
			expectedStatus: http.StatusConflict,
			expectedError:  "email already in use",      
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			handler := NewCustomerHandler(db)

			if tt.setupCustomer {
				// Create the primary customer to be updated
				customer := models.Customer{
					ID:    1,
					Name:  "Sebbie Chanzu",
					Code:  "CUST001",
					Phone: "+254740827150",
					Email: "sebbievilar2@gmail.com",
				}
				if err := db.Create(&customer).Error; err != nil {
					t.Fatalf("failed to create primary customer: %v", err)
				}

				if tt.name == "email conflict on update" {
					conflictCustomer := models.Customer{
						ID:    2,
						Name:  "John Doe",
						Code:  "CUST002",
						Phone: "+254740827151",
						Email: "conflict@example.com",
					}
					if err := db.Create(&conflictCustomer).Error; err != nil {
						t.Fatalf("failed to create conflict customer: %v", err)
					}
				}
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonBody, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPut, "/customers/"+tt.customerID, bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req
			c.Params = gin.Params{{Key: "id", Value: tt.customerID}}

			handler.UpdateCustomer(c)

			assert.Equal(t, tt.expectedStatus, w.Code, "status code mismatch")

			if tt.expectedError != "" {
				var errorResponse models.ErrorResponse
				json.Unmarshal(w.Body.Bytes(), &errorResponse)
				assert.Equal(t, tt.expectedError, errorResponse.Error, "error key mismatch")
			} else {
				var updatedCustomer models.Customer
				db.First(&updatedCustomer, tt.customerID)
				assert.Equal(t, tt.expectedName, updatedCustomer.Name, "name mismatch")
				assert.Equal(t, tt.expectedPhone, updatedCustomer.Phone, "phone mismatch")
				assert.Equal(t, tt.expectedEmail, updatedCustomer.Email, "email mismatch")
			}
		})
	}
}

func TestDeleteCustomer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		customerID     string
		setupCustomer  bool
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid customer deletion",
			customerID:     "1",
			setupCustomer:  true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid customer ID",
			customerID:     "invalid",
			setupCustomer:  false,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid id",
		},
		{
			name:           "non-existent customer",
			customerID:     "999",
			setupCustomer:  false,
			expectedStatus: http.StatusNotFound,
			expectedError:  "customer not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			handler := NewCustomerHandler(db)

			if tt.setupCustomer {
				customer := models.Customer{
					Name:  "Sebbie Chanzu",
					Code:  "CUST001",
					Phone: "+254740827150",
					Email: "sebbievilar2@gmail.com",
				}
				if err := db.Create(&customer).Error; err != nil {
					t.Fatalf("failed to create customer: %v", err)
				}
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("DELETE", "/customers/"+tt.customerID, nil)
			c.Request = req
			c.Params = []gin.Param{{Key: "id", Value: tt.customerID}}

			handler.DeleteCustomer(c)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var errorResponse models.ErrorResponse
				json.Unmarshal(w.Body.Bytes(), &errorResponse)
				assert.Equal(t, tt.expectedError, errorResponse.Error)
			} else {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				assert.Equal(t, "customer deleted successfully", response["message"])

				var dbCustomer models.Customer
				err := db.First(&dbCustomer, tt.customerID).Error
				assert.Error(t, err)
				assert.Equal(t, gorm.ErrRecordNotFound, err)
			}
		})
	}
}
