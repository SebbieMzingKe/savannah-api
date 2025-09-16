package main

import (
	"log"
	"net/http"
	"os"

	"github.com/SebbieMzingKe/customer-order-api/internal/handlers"
	"github.com/SebbieMzingKe/customer-order-api/internal/middleware"
	"github.com/SebbieMzingKe/customer-order-api/internal/models"
	"github.com/SebbieMzingKe/customer-order-api/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func init()  {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found")
	}

	var err error

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=savannah password=savannah dbname=savannah port=5432 sslmode=disable"
	}

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database", err)
	}

	err = db.AutoMigrate(&models.Customer{}, &models.Order{})
	if err != nil {
		log.Fatal("failed to migrae database", err)
	}
}

func main() {

	smsService := services.NewSMSService(
		os.Getenv("AFRICAS_TALKING_USERNAME"),
		os.Getenv("AFRICAS_TALKING_API_KEY"),
		os.Getenv("AFRICAS_TALKING_SENDER_ID"),
	)


	customerHandler := handlers.NewCustomerHandler(db)
	orderHandler := handlers.NewOrderHandler(db, smsService)
	authHandler := handlers.NewAuthHandler()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	auth := r.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/callback", authHandler.Callback)
		auth.GET("/userinfo",)
	}

	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware())
	{
		customers := api.Group("/customers")
		{
			customers.POST("", customerHandler.CreateCustomer)
			customers.GET("", customerHandler.GetCustomers)
			customers.GET("/:id", customerHandler.GetCustomer)
			customers.PUT("/:id", customerHandler.UpdateCustomer)
			customers.DELETE("/:id", customerHandler.DeleteCustomer)
		}
		
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

	log.Printf("server is starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}