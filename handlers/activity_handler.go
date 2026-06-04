package handlers

import (
	"net/http"
	"time"

	"contact-management/models"
	"contact-management/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// validActivityTypes defines all allowed activity types
var validActivityTypes = map[string]bool{
	"email_sent":       true,
	"whatsapp_sent":    true,
	"meeting_attended": true,
	"video_watched":    true,
	"event_attended":   true,
	"note_added":       true,
	"contact_created":  true,
}

// LogActivityHandler handles POST /api/v1/contacts/:id/activities
//
// Request Body:
//
//	{
//	  "activity_type": "email_sent",        ← required
//	  "details":       "Welcome email sent" ← optional
//	}
//
// Supported activity_type values:
//   - email_sent
//   - whatsapp_sent
//   - meeting_attended
//   - video_watched
//   - event_attended
//   - note_added
//   - contact_created
func LogActivityHandler(c *gin.Context) {
	// Parse contact ID
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contact UUID"})
		return
	}

	// Parse request body
	var req struct {
		ActivityType string `json:"activity_type" binding:"required"`
		Details      string `json:"details"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "activity_type is required"})
		return
	}

	// Validate activity type
	if !validActivityTypes[req.ActivityType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid activity_type",
			"allowed": []string{
				"email_sent",
				"whatsapp_sent",
				"meeting_attended",
				"video_watched",
				"event_attended",
				"note_added",
				"contact_created",
			},
		})
		return
	}

	ctx := c.Request.Context()

	// Verify contact exists and is not deleted
	contact, err := repository.GetContactByID(ctx, contactID)
	if err != nil || contact.IsDeleted == 1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
		return
	}

	now := time.Now()

	// Log the activity
	activity := &models.ContactActivity{
		ID:           uuid.New(),
		ContactID:    contactID,
		ActivityType: req.ActivityType,
		Details:      req.Details,
		CreatedAt:    now,
	}

	if err := repository.AddContactActivity(ctx, activity); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to log activity: " + err.Error()})
		return
	}

	// Update contact's last_activity_at
	contact.LastActivityAt = now
	contact.Version = now
	_ = repository.UpdateContact(ctx, contact)

	c.JSON(http.StatusCreated, gin.H{
		"message":  "activity logged successfully",
		"activity": activity,
	})
}

// GetActivitiesHandler handles GET /api/v1/contacts/:id/activities
// Returns the full chronological activity timeline for a contact
func GetActivitiesHandler(c *gin.Context) {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contact UUID"})
		return
	}

	ctx := c.Request.Context()

	// Verify contact exists
	contact, err := repository.GetContactByID(ctx, contactID)
	if err != nil || contact.IsDeleted == 1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
		return
	}

	// Get timeline
	activities, err := repository.GetContactActivities(ctx, contactID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get summary counts
	summary, err := repository.GetContactActivitySummary(ctx, contactID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"contact_id":       contactID,
		"activity_summary": summary,
		"activities":       activities,
	})
}
