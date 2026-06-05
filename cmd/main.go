package main

import (
	"log"
	"net/http"
	"os"

	"contact-management/config"
	"contact-management/routes"
	"contact-management/worker"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// 1. Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found or error loading it. Using default environment variables.")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 2. Initialize infrastructure connections
	config.ConnectDB()
	config.CreateTables()
	config.ConnectRedis()
	config.ConnectKafka()
	defer config.CloseKafka()

	// 3. Start background Kafka consumer for CSV imports
	go worker.StartImportConsumer()

	// 4. Setup Echo router
	e := echo.New()

	// CORS Middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
	}))

	// Basic logging and recovery
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Register API Routes
	routes.RegisterRoutes(e)
	
	log.Printf("Starting HTTP server on port %s... ✅", port)
	e.Logger.Fatal(e.Start(":" + port))
}
