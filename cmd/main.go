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

// @title Contact Management API
// @version 1.0
// @description Enterprise-grade API for managing contacts and async CSV imports.
// @host localhost:8081
// @BasePath /api/v1
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
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

	// 3. Start background Kafka consumer for CSV imports infinitely if any topic exits in kafka then consumer takes the topic and process the data 
	go worker.StartImportConsumer()

	// 4. Setup Echo router
	e := echo.New()

	// CORS Middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "X-API-Key"},
	}))

	// Basic logging and recovery
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true,
		LogURI:    true,
		LogMethod: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Printf("REQUEST: %s %s | STATUS: %d\n", v.Method, v.URI, v.Status)
			return nil
		},
	}))
	e.Use(middleware.Recover())

	// Register API Routes
	routes.RegisterRoutes(e)
	
	log.Printf("Starting HTTP server on port %s... ✅", port)
	e.Logger.Fatal(e.Start(":" + port))
}
