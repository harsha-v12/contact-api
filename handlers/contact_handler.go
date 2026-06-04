package handlers

import (
	"net/http"
	"time"

	"contact-management/models"
	"contact-management/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateContactHandler handles POST /api/v1/contacts
func CreateContactHandler(c *gin.Context) {
	var req models.ContactCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate request fields
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Check duplicates
	isDuplicate, err := repository.CheckDuplicate(ctx, req.Email, req.MobileNumber, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if isDuplicate {
		c.JSON(http.StatusConflict, gin.H{"error": "contact with this email or mobile number already exists"})
		return
	}

	// Parse DateOfBirth
	var dob time.Time
	if req.DateOfBirth != "" {
		dob, _ = time.Parse("2006-01-02", req.DateOfBirth)
	}

	now := time.Now()
	contactID := uuid.New()

	// Build Contact Model
	contact := &models.Contact{
		ID:             contactID,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Email:          req.Email,
		MobileNumber:   req.MobileNumber,
		Gender:         req.Gender,
		DateOfBirth:    dob,
		City:           req.City,
		State:          req.State,
		Country:        req.Country,
		Tags:           req.Tags,
		Notes:          req.Notes,
		CreatedAt:      now,
		LastActivityAt: now,
		IsDeleted:      0,
		Version:        now,
	}

	// Save to DB
	if err := repository.CreateContact(ctx, contact); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Save Initial Note if provided
	if req.Notes != "" {
		note := &models.ContactNote{
			ID:        uuid.New(),
			ContactID: contactID,
			Note:      req.Notes,
			CreatedAt: now,
			UpdatedAt: now,
			IsDeleted: 0,
			Version:   now,
		}
		_ = repository.AddContactNote(ctx, note)

		// Log Note Added Activity
		_ = repository.AddContactActivity(ctx, &models.ContactActivity{
			ID:           uuid.New(),
			ContactID:    contactID,
			ActivityType: "note_added",
			Details:      "Initial contact note created: " + req.Notes,
			CreatedAt:    now,
		})
	}

	// Log Created Activity
	_ = repository.AddContactActivity(ctx, &models.ContactActivity{
		ID:           uuid.New(),
		ContactID:    contactID,
		ActivityType: "contact_created",
		Details:      "Contact record created",
		CreatedAt:    now,
	})

	c.JSON(http.StatusCreated, contact)
}

// GetContactProfileHandler handles GET /api/v1/contacts/:id
func GetContactProfileHandler(c *gin.Context) {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contact UUID"})
		return
	}

	ctx := c.Request.Context()

	// Get basic info
	contact, err := repository.GetContactByID(ctx, contactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
		return
	}

	// Check if soft-deleted
	if contact.IsDeleted == 1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contact is deleted"})
		return
	}

	// Get Notes
	notes, _ := repository.GetContactNotes(ctx, contactID)

	// Get Activity Summary
	summary, _ := repository.GetContactActivitySummary(ctx, contactID)

	// Get Activity Timeline
	activities, _ := repository.GetContactActivities(ctx, contactID)

	c.JSON(http.StatusOK, gin.H{
		"contact":          contact,
		"notes":            notes,
		"activity_summary": summary,
		"activities":      activities,
	})
}

// UpdateContactHandler handles PUT /api/v1/contacts/:id
func UpdateContactHandler(c *gin.Context) {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contact UUID"})
		return
	}

	var req models.ContactUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Fetch existing
	existing, err := repository.GetContactByID(ctx, contactID)
	if err != nil || existing.IsDeleted == 1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
		return
	}

	// Check duplicate email/mobile on other contacts
	isDuplicate, err := repository.CheckDuplicate(ctx, req.Email, req.MobileNumber, &contactID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if isDuplicate {
		c.JSON(http.StatusConflict, gin.H{"error": "another contact already uses this email or mobile number"})
		return
	}

	// Parse DOB
	var dob time.Time
	if req.DateOfBirth != "" {
		dob, _ = time.Parse("2006-01-02", req.DateOfBirth)
	}

	now := time.Now()

	// Update fields
	existing.FirstName = req.FirstName
	existing.LastName = req.LastName
	existing.Email = req.Email
	existing.MobileNumber = req.MobileNumber
	existing.Gender = req.Gender
	existing.DateOfBirth = dob
	existing.City = req.City
	existing.State = req.State
	existing.Country = req.Country
	existing.Tags = req.Tags
	existing.LastActivityAt = now
	existing.Version = now 

	if err := repository.UpdateContact(ctx, existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// DeleteContactHandler handles DELETE /api/v1/contacts/:id
func DeleteContactHandler(c *gin.Context) {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contact UUID"})
		return
	}

	ctx := c.Request.Context()

	// Retrieve to verify existence
	_, err = repository.GetContactByID(ctx, contactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
		return
	}

	now := time.Now()
	if err := repository.SoftDeleteContact(ctx, contactID, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "contact soft-deleted successfully"})
}

// RestoreContactHandler handles POST /api/v1/contacts/:id/restore
func RestoreContactHandler(c *gin.Context) {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contact UUID"})
		return
	}

	ctx := c.Request.Context()

	// Verify existence
	_, err = repository.GetContactByID(ctx, contactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
		return
	}

	now := time.Now()
	if err := repository.RestoreContact(ctx, contactID, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "contact restored successfully"})
}

// GetNotesHandler handles GET /api/v1/contacts/:id/notes
func GetNotesHandler(c *gin.Context) {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contact UUID"})
		return
	}

	ctx := c.Request.Context()

	notes, err := repository.GetContactNotes(ctx, contactID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, notes)
}

// AddNoteHandler handles POST /api/v1/contacts/:id/notes
func AddNoteHandler(c *gin.Context) {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contact UUID"})
		return
	}

	var req struct {
		Note string `json:"note" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Verify contact exists
	contact, err := repository.GetContactByID(ctx, contactID)
	if err != nil || contact.IsDeleted == 1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
		return
	}

	now := time.Now()
	note := &models.ContactNote{
		ID:        uuid.New(),
		ContactID: contactID,
		Note:      req.Note,
		CreatedAt: now,
		UpdatedAt: now,
		IsDeleted: 0,
		Version:   now,
	}

	if err := repository.AddContactNote(ctx, note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update contact's LastActivityAt
	contact.LastActivityAt = now
	contact.Version = now
	_ = repository.UpdateContact(ctx, contact)

	// Log activity
	_ = repository.AddContactActivity(ctx, &models.ContactActivity{
		ID:           uuid.New(),
		ContactID:    contactID,
		ActivityType: "note_added",
		Details:      "Note added: " + req.Note,
		CreatedAt:    now,
	})

	c.JSON(http.StatusCreated, note)
}

// UpdateNoteHandler handles PUT /api/v1/contacts/:id/notes/:note_id
func UpdateNoteHandler(c *gin.Context){
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contact UUID"})
		return
	}

	noteIDStr := c.Param("note_id")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid note UUID"})
		return
	}

	var req struct {
		Note string `json:"note" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Fetch note first to verify
	note, err := repository.GetContactNoteByID(ctx, contactID, noteID)
	if err != nil || note.IsDeleted == 1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}

	now := time.Now()
	note.Note = req.Note
	note.UpdatedAt = now
	note.Version = now // new version

	if err := repository.UpdateContactNote(ctx, note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update contact activity
	contact, err := repository.GetContactByID(ctx, contactID)
	if err == nil {
		contact.LastActivityAt = now
		contact.Version = now
		_ = repository.UpdateContact(ctx, contact)
	}

	c.JSON(http.StatusOK, note)
}

// DeleteNoteHandler handles DELETE /api/v1/contacts/:id/notes/:note_id
func DeleteNoteHandler(c *gin.Context) {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contact UUID"})
		return
	}

	noteIDStr := c.Param("note_id")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid note UUID"})
		return
	}

	ctx := c.Request.Context()

	// Fetch note first to verify
	note, err := repository.GetContactNoteByID(ctx, contactID, noteID)
	if err != nil || note.IsDeleted == 1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
		return
	}

	now := time.Now()
	note.IsDeleted = 1
	note.UpdatedAt = now
	note.Version = now

	if err := repository.UpdateContactNote(ctx, note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update contact activity
	contact, err := repository.GetContactByID(ctx, contactID)
	if err == nil {
		contact.LastActivityAt = now
		contact.Version = now
		_ = repository.UpdateContact(ctx, contact)
	}

	c.JSON(http.StatusOK, gin.H{"message": "note deleted successfully"})
}

// UpdateTagsHandler handles PATCH /api/v1/contacts/:id/tags
func UpdateTagsHandler(c *gin.Context) {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid contact UUID"})
		return
	}

	var req struct {
		Tags []string `json:"tags" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Fetch contact
	contact, err := repository.GetContactByID(ctx, contactID)
	if err != nil || contact.IsDeleted == 1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
		return
	}

	now := time.Now()
	contact.Tags = req.Tags
	contact.LastActivityAt = now
	contact.Version = now

	if err := repository.UpdateContact(ctx, contact); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contact)
}
