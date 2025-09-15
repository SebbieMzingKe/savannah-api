package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"savannah-api/internal/auth"
	"savannah-api/internal/handlers"
	"savannah-api/internal/middleware"
	"savannah-api/internal/models"
	"savannah-api/internal/services"
)

var db *gorm.DB

func init() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Connect to database
	var err error
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=savannah password=savannah dbname=savannah port=5432 sslmode=disable"
	}

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&models.Customer{}, &models.Order{})
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}

func main() {
	// Initialize services
	smsService := services.NewSMSService(
		os.Getenv("AFRICASTALKING_USERNAME"),
		os.Getenv("AFRICASTALKING_API_KEY"),
		os.Getenv("AFRICASTALKING_SENDER_ID"),
	)

	// Initialize handlers
	customerHandler := handlers.NewCustomerHandler(db)
	orderHandler := handlers.NewOrderHandler(db, smsService)
	authHandler := handlers.NewAuthHandler()

	// Setup router
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth routes
	auth := r.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/callback", authHandler.Callback)
		auth.GET("/userinfo", middleware.AuthMiddleware(), authHandler.UserInfo)
	}

	// API routes with authentication
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware())
	{
		// Customer routes
		customers := api.Group("/customers")
		{
			customers.POST("", customerHandler.CreateCustomer)
			customers.GET("", customerHandler.GetCustomers)
			customers.GET("/:id", customerHandler.GetCustomer)
			customers.PUT("/:id", customerHandler.UpdateCustomer)
			customers.DELETE("/:id", customerHandler.DeleteCustomer)
		}

		// Order routes
		orders := api.Group("/orders")
		{
			orders.POST("", orderHandler.CreateOrder)
			orders.GET("", orderHandler.GetOrders)
			orders.GET("/:id", orderHandler.GetOrder)
			orders.PUT("/:id", orderHandler.UpdateOrder)
			orders.DELETE("/:id", orderHandler.DeleteOrder)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}