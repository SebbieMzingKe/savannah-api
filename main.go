package main

import (
	"log"
	"net/http"
	"os"

	"github.com/SebbieMzingKe/customer-order-api/internal/handlers"
	"github.com/SebbieMzingKe/customer-order-api/internal/models"

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

func init() {
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

		log.Fatal("failed to connect to database", err)
	}

	err = db.AutoMigrate(&models.Customer{}, &models.Order{})
	if err != nil {
		log.Fatal("failed to migrae database", err)

		log.Fatal("Failed to connect to database:", err)
	}

	err = db.AutoMigrate()
	if err != nil {
		log.Fatal("Failed to migrate database:", err)

	}
}

func main() {

	customerHandler := handlers.NewCustomerHandler(db)
	// orderHandler := handlers.NewOrderHandler(db, smsService)
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
	api.Use()
	{
		customers := api.Group("/customers")
		{
			customers.POST("", customerHandler.CreateCustomer)
		}
	}
}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
