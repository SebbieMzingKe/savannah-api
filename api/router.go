package api

import (
	"net/http"
	"os"

	"github.com/SebbieMzingKe/customer-order-api/handlers"
	"github.com/SebbieMzingKe/customer-order-api/middleware"
	"github.com/SebbieMzingKe/customer-order-api/models"
	"github.com/SebbieMzingKe/customer-order-api/services"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func SetupRouter() (*gin.Engine, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, ErrMissingDB
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.Customer{}, &models.Order{}); err != nil {
		return nil, err
	}

	smsService := services.NewSMSService(
		os.Getenv("AFRICASTALKING_USERNAME"),
		os.Getenv("AFRICASTALKING_API_KEY"),
		os.Getenv("AFRICASTALKING_SENDER_ID"),
	)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	authHandler := handlers.NewAuthHandler()
	auth := r.Group("/auth")
	{
		auth.GET("/login", authHandler.Login)
		auth.GET("/callback", authHandler.Callback)
		auth.GET("/userinfo", middleware.AuthMiddleware(), authHandler.UserInfo)
	}

	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware())
	{
		customers := api.Group("/customers")
		{
			customerHandler := handlers.NewCustomerHandler(db)
			customers.POST("", customerHandler.CreateCustomer)
			customers.GET("", customerHandler.GetCustomers)
			customers.GET("/:id", customerHandler.GetCustomer)
			customers.PUT("/:id", customerHandler.UpdateCustomer)
			customers.DELETE("/:id", customerHandler.DeleteCustomer)
		}

		orders := api.Group("/orders")
		{
			orderHandler := handlers.NewOrderHandler(db, smsService)
			orders.POST("", orderHandler.CreateOrder)
			orders.GET("", orderHandler.GetOrders)
			orders.GET("/:id", orderHandler.GetOrder)
			orders.PUT("/:id", orderHandler.UpdateOrder)
			orders.DELETE("/:id", orderHandler.DeleteOrder)
		}
	}

	return r, nil
}

var ErrMissingDB = &CustomError{"DATABASE_URL not set"}

type CustomError struct{ msg string }

func (e *CustomError) Error() string { return e.msg }
