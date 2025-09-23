package main

import (
	"log"
	"os"

	"github.com/SebbieMzingKe/customer-order-api/api"
)

func main() {
	r, err := api.SetupRouter()
	if err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server running on port %s", port)
	log.Fatal(r.Run(":" + port))
}
