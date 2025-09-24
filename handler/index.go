package handler

import (
	"fmt"
	"net/http"
	"os"

	"github.com/SebbieMzingKe/customer-order-api/internal/handlers"
	"github.com/SebbieMzingKe/customer-order-api/internal/middleware"
	"github.com/SebbieMzingKe/customer-order-api/internal/models"
	"github.com/SebbieMzingKe/customer-order-api/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var router *gin.Engine

func init() {

	dsn := os.Getenv("DATABASE_URL")
	jwt := os.Getenv("JWT_SECRET")
	fmt.Println("jwt:", jwt)
	fmt.Println("DATABASE_URL:", dsn)

	var err error

	dsn = os.Getenv("DATABASE_URL")
	if dsn == "" {
		panic("database url ennvironment variable is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {

		panic("failed to connect to database: " + err.Error())
	}

	if err := db.AutoMigrate(&models.Customer{}, &models.Order{}); err != nil {
		panic("failed to migrate database: " + err.Error())
	}

	smsService := services.NewSMSService(
		os.Getenv("AFRICASTALKING_USERNAME"),
		os.Getenv("AFRICASTALKING_API_KEY"),
		os.Getenv("AFRICASTALKING_SENDER_ID"),
	)

	router = gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "welcome to customer order api"})
	})

	authHandler := handlers.NewAuthHandler()
	auth := router.Group("/auth")
	{
		auth.GET("/login", authHandler.Login)
		auth.GET("/callback", authHandler.Callback)
		auth.GET("/userinfo", middleware.AuthMiddleware(), authHandler.UserInfo)
	}

	api := router.Group("/api/v1")
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
}

func Handler(w http.ResponseWriter, r *http.Request) {
	router.ServeHTTP(w, r)
}
