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
	customerHandler := handlers.NewCustomerHandler(db)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
}