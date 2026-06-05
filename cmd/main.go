package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"contact-management/config"
	"contact-management/routes"
	"contact-management/worker"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	// 2. Initialize infrastructure connections
	config.ConnectDB()
	config.CreateTables()
	config.ConnectRedis()
	config.ConnectKafka()
	defer config.CloseKafka()

	// 3. Start background Kafka consumer for CSV imports
	go worker.StartImportConsumer()

	// 4. Setup Gin router
	r := gin.Default()

	// CORS Middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Register API Routes
	routes.RegisterRoutes(r)

	// 4. Default Root Endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Contact Management API is running! 🚀",
			"version": "1.0",
		})
	})

	// 5. Health Check Endpoint
	r.GET("/health", func(c *gin.Context) {
		healthStatus := gin.H{
			"status":      "UP",
			"timestamp":   time.Now().Format(time.RFC3339),
			"clickhouse":  "UP",
			"redis":       "UP",
			"kafka":       "UP",
		}

		// Check ClickHouse
		if err := config.DB.Ping(c.Request.Context()); err != nil {
			healthStatus["clickhouse"] = fmt.Sprintf("DOWN: %v", err)
			healthStatus["status"] = "DEGRADED"
		}

		// Check Redis
		if err := config.RedisClient.Ping(c.Request.Context()).Err(); err != nil {
			healthStatus["redis"] = fmt.Sprintf("DOWN: %v", err)
			healthStatus["status"] = "DEGRADED"
		}

		// Check Kafka — actually dial the broker TCP to verify real connectivity
		kafkaBroker := os.Getenv("KAFKA_BROKER")
		if kafkaBroker == "" {
			kafkaBroker = "localhost:9092"
		}
		conn, kafkaErr := net.DialTimeout("tcp", kafkaBroker, 3*time.Second)
		if kafkaErr != nil {
			healthStatus["kafka"] = fmt.Sprintf("DOWN: %v", kafkaErr)
			healthStatus["status"] = "DEGRADED"
		} else {
			conn.Close()
		}

		if healthStatus["status"] == "UP" {
			c.JSON(http.StatusOK, healthStatus)
		} else {
			c.JSON(http.StatusServiceUnavailable, healthStatus)
		}
	})

	// 5. Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Starting HTTP server on port %s... ✅", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run HTTP server: %v", err)
	}
}
