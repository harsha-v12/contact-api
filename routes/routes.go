package routes

import (
	"contact-management/handlers"
	"contact-management/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes sets up the routing configuration for the API
func RegisterRoutes(r *gin.Engine){
	v1 := r.Group("/api/v1")
	// for authentiction purpose we can use API key authentication middleware
	v1.Use(middleware.APIkeyAuth())
	{
		contacts := v1.Group("/contacts")
		{
			// Core contact CRUD endpoints
			contacts.GET("", handlers.ListContactsHandler)              
			contacts.POST("", handlers.CreateContactHandler)
			contacts.GET("/:id", handlers.GetContactProfileHandler)
			contacts.PUT("/:id", handlers.UpdateContactHandler)
			contacts.DELETE("/:id", handlers.DeleteContactHandler)
			contacts.POST("/:id/restore", handlers.RestoreContactHandler)

			// Tags endpoint
			contacts.PATCH("/:id/tags", handlers.UpdateTagsHandler)

			// Notes endpoints
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
