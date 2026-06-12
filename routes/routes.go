package routes

import (
	"contact-management/config"
	"contact-management/handlers"
	"net"
	"net/http"
	"os"
	"time"

	_ "contact-management/docs" // Swagger docs

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// RegisterRoutes sets up all the API routes
func RegisterRoutes(e *echo.Echo) {
	// Swagger UI Route
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Root route
	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"service": "contact-api",
			"status":  "running",
			"version": "1.0.0",
		})
	})

	// Health check
	e.GET("/health", func(c echo.Context) error {
		healthStatus := map[string]interface{}{
			"status":      "UP",
			"clickhouse":  "UP",
			"redis":       "UP",
			"kafka":       "UP",
		}

		// Check ClickHouse
		if err := config.DB.Ping(c.Request().Context()); err != nil {
			healthStatus["clickhouse"] = "DOWN"
			healthStatus["status"] = "DEGRADED"
		}

		// Check Redis
		if err := config.RedisClient.Ping(c.Request().Context()).Err(); err != nil {
			healthStatus["redis"] = "DOWN"
			healthStatus["status"] = "DEGRADED"
		}

		// Check Kafka (Real-time TCP Ping)
		kafkaBroker := os.Getenv("KAFKA_BROKER")
		if kafkaBroker == "" {
			kafkaBroker = "localhost:9092"
		}
		conn, kafkaErr := net.DialTimeout("tcp", kafkaBroker, 3*time.Second)
		if kafkaErr != nil {
			healthStatus["kafka"] = "DOWN"
			healthStatus["status"] = "DEGRADED"
		} else {
			conn.Close()
		}

		if healthStatus["status"] == "UP" {
			return c.JSON(http.StatusOK, healthStatus)
		}
		return c.JSON(http.StatusServiceUnavailable, healthStatus)
	})

	// API v1 Group
	v1 := e.Group("/api/v1")
	{
		// Apply Auth Middleware to all v1 routes (DISABLED FOR LOCAL TESTING)
		// v1.Use(middleware.APIKeyAuthMiddleware())

		// Contacts Group
		contacts := v1.Group("/contacts")
		{	
			//contact endpoints with all CRUD operations
			contacts.GET("", handlers.ListContactsHandler)
			contacts.POST("", handlers.CreateContactHandler)
			contacts.GET("/:id", handlers.GetContactProfileHandler)
			contacts.PUT("/:id", handlers.UpdateContactHandler)
			contacts.DELETE("/:id", handlers.DeleteContactHandler)
			contacts.POST("/:id/restore", handlers.RestoreContactHandler)

			// Tags endpoint
			contacts.PATCH("/:id/tags", handlers.UpdateTagsHandler)
			
			// Note endpoints
			contacts.GET("/:id/notes", handlers.GetNotesHandler)
			contacts.POST("/:id/notes", handlers.AddNoteHandler)
			contacts.PUT("/:id/notes/:note_id", handlers.UpdateNoteHandler)
			contacts.DELETE("/:id/notes/:note_id", handlers.DeleteNoteHandler)

			// Activity endpoints
			contacts.POST("/:id/activities", handlers.LogActivityHandler)
			contacts.GET("/:id/activities", handlers.GetActivitiesHandler)
			
			// CSV Import endpoints
			contacts.POST("/import", handlers.UploadImportFileHandler)
			contacts.GET("/import/:import_id", handlers.GetImportStatusHandler)

			// For real time updates for the Preogress Bar route 
			contacts.GET("/ws/import/:id",handlers.ImportProgressWS)
		}
	}
}
