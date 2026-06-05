package routes

import (
	"contact-management/handlers"
	"contact-management/middleware"
	"net/http"

	"github.com/labstack/echo/v4"
)

// RegisterRoutes sets up all the API routes
func RegisterRoutes(e *echo.Echo) {
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
		return c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
		})
	})

	// API v1 Group
	v1 := e.Group("/api/v1")
	{
		// Apply Auth Middleware to all v1 routes
		v1.Use(middleware.APIKeyAuthMiddleware())

		// Contacts Group
		contacts := v1.Group("/contacts")
		{
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
		}
	}
}
