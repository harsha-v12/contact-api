package handlers

import (
	"net/http"
	"time"

	"contact-management/models"
	"contact-management/repository"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
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


func LogActivityHandler(c echo.Context) error {
	// Parse contact ID
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid contact UUID"})
	}

	// Parse request body
	var req struct {
		ActivityType string `json:"activity_type" binding:"required"`
		Details      string `json:"details"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "activity_type is required"})
	}

	// Validate activity type
	if !validActivityTypes[req.ActivityType] {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
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
	}

	ctx := c.Request().Context()

	// Verify contact exists and is not deleted
	contact, err := repository.GetContactByID(ctx, contactID)
	if err != nil || contact.IsDeleted == 1 {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "contact not found"})
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
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "failed to log activity: " + err.Error()})
	}

	// Update contact's last_activity_at
	contact.LastActivityAt = now
	contact.Version = now
	_ = repository.UpdateContact(ctx, contact)

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message":  "activity logged successfully",
		"activity": activity,
	})
}

// GetActivitiesHandler handles GET /api/v1/contacts/:id/activities
// Returns the full chronological activity timeline for a contact
func GetActivitiesHandler(c echo.Context) error {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid contact UUID"})
	}

	ctx := c.Request().Context()

	// Verify contact exists
	contact, err := repository.GetContactByID(ctx, contactID)
	if err != nil || contact.IsDeleted == 1 {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "contact not found"})
	}

	// Get timeline
	activities, err := repository.GetContactActivities(ctx, contactID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	// Get summary counts
	summary, err := repository.GetContactActivitySummary(ctx, contactID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"contact_id":       contactID,
		"activity_summary": summary,
		"activities":       activities,
	})
}
