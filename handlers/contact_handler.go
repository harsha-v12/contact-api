package handlers

import (
	"net/http"
	"time"

	"contact-management/models"
	"contact-management/repository"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// @Summary Create a Contact
// @Description Creates a new contact with strict validation
// @Tags contacts
// @Accept json
// @Produce json
// @Param request body models.ContactCreateRequest true "Contact data"
// @Success 201 {object} models.Contact
// @Failure 400 {object} map[string]string "Validation error"
// @Failure 409 {object} map[string]string "Conflict (duplicate)"
// @Security ApiKeyAuth
// @Router /contacts [post]
func CreateContactHandler(c echo.Context) error {
	var req models.ContactCreateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	if err := req.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	ctx := c.Request().Context()

	isDuplicate, err := repository.CheckDuplicate(ctx, req.Email, req.MobileNumber, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}
	if isDuplicate {
		return c.JSON(http.StatusConflict, map[string]interface{}{"error": "contact with this email or mobile number already exists"})
	}

	var dob time.Time
	if req.DateOfBirth != "" {
		dob, _ = time.Parse("2006-01-02", req.DateOfBirth)
	}

	now := time.Now()
	contactID := uuid.New()

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

	if err := repository.CreateContact(ctx, contact); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

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

		_ = repository.AddContactActivity(ctx, &models.ContactActivity{
			ID:           uuid.New(),
			ContactID:    contactID,
			ActivityType: "note_added",
			Details:      "Initial contact note created: " + req.Notes,
			CreatedAt:    now,
		})
	}

	_ = repository.AddContactActivity(ctx, &models.ContactActivity{
		ID:           uuid.New(),
		ContactID:    contactID,
		ActivityType: "contact_created",
		Details:      "Contact record created",
		CreatedAt:    now,
	})

	return c.JSON(http.StatusCreated, contact)
}

// @Summary Get Contact Profile
// @Description View complete contact details, notes, and activity timeline.
// @Tags contacts
// @Produce json
// @Param id path string true "Contact UUID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string "Contact not found"
// @Security ApiKeyAuth
// @Router /contacts/{id} [get]
func GetContactProfileHandler(c echo.Context) error {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid contact UUID"})
	}

	ctx := c.Request().Context()

	contact, err := repository.GetContactByID(ctx, contactID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "contact not found"})
	}

	if contact.IsDeleted == 1 {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "contact is deleted"})
	}

	notes, _ := repository.GetContactNotes(ctx, contactID)
	summary, _ := repository.GetContactActivitySummary(ctx, contactID)
	activities, _ := repository.GetContactActivities(ctx, contactID)

	if notes == nil {
		notes = make([]models.ContactNote, 0)
	}
	if activities == nil {
		activities = make([]models.ContactActivity, 0)
	}

	name := contact.FirstName
	if contact.LastName != "" {
		name += " " + contact.LastName
	}

	type LocationStruct struct {
		City    string `json:"City"`
		State   string `json:"State"`
		Country string `json:"Country"`
	}

	type BasicInfoStruct struct {
		Name         string         `json:"Name"`
		Email        string         `json:"Email"`
		MobileNumber string         `json:"Mobile Number"`
		DateOfBirth  time.Time      `json:"Date of Birth"`
		Gender       string         `json:"Gender"`
		Location     LocationStruct `json:"Location"`
	}

	type ActivitySummaryStruct struct {
		TotalEmailsSent           int `json:"Total Emails Sent"`
		TotalEventsAttended       int `json:"Total Events Attended"`
		TotalMeetingsAttended     int `json:"Total Meetings Attended"`
		TotalVideosWatched        int `json:"Total Videos Watched"`
		TotalWhatsAppMessagesSent int `json:"Total WhatsApp Messages Sent"`
	}

	type ProfileResponse struct {
		BasicInformation BasicInfoStruct          `json:"Basic Information"`
		Notes            []models.ContactNote     `json:"Notes"`
		ActivitySummary  ActivitySummaryStruct    `json:"Activity Summary"`
		ActivityTimeline []models.ContactActivity `json:"Activity Timeline"`
	}

	response := ProfileResponse{
		BasicInformation: BasicInfoStruct{
			Name:         name,
			Email:        contact.Email,
			MobileNumber: contact.MobileNumber,
			DateOfBirth:  contact.DateOfBirth,
			Gender:       contact.Gender,
			Location: LocationStruct{
				City:    contact.City,
				State:   contact.State,
				Country: contact.Country,
			},
		},
		Notes: notes,
		ActivitySummary: ActivitySummaryStruct{
			TotalEmailsSent:           summary["email_sent"],
			TotalEventsAttended:       summary["event_attended"],
			TotalMeetingsAttended:     summary["meeting_attended"],
			TotalVideosWatched:        summary["video_watched"],
			TotalWhatsAppMessagesSent: summary["whatsapp_sent"],
		},
		ActivityTimeline: activities,
	}

	return c.JSON(http.StatusOK, response)
}

// @Summary Update Contact
// @Description Update contact basic information
// @Tags contacts
// @Accept json
// @Produce json
// @Param id path string true "Contact UUID"
// @Param request body models.ContactUpdateRequest true "Update data"
// @Success 200 {object} models.Contact
// @Failure 400 {object} map[string]string "Validation error"
// @Failure 404 {object} map[string]string "Contact not found"
// @Security ApiKeyAuth
// @Router /contacts/{id} [put]
func UpdateContactHandler(c echo.Context) error {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid contact UUID"})
	}

	var req models.ContactUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	if err := req.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	ctx := c.Request().Context()

	existing, err := repository.GetContactByID(ctx, contactID)
	if err != nil || existing.IsDeleted == 1 {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "contact not found"})
	}

	isDuplicate, err := repository.CheckDuplicate(ctx, req.Email, req.MobileNumber, &contactID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}
	if isDuplicate {
		return c.JSON(http.StatusConflict, map[string]interface{}{"error": "another contact already uses this email or mobile number"})
	}

	var dob time.Time
	if req.DateOfBirth != "" {
		dob, _ = time.Parse("2006-01-02", req.DateOfBirth)
	}

	now := time.Now()

	if req.FirstName != "" { existing.FirstName = req.FirstName }
	if req.LastName != "" { existing.LastName = req.LastName }
	if req.Email != "" { existing.Email = req.Email }
	if req.MobileNumber != "" { existing.MobileNumber = req.MobileNumber }
	if req.Gender != "" { existing.Gender = req.Gender }
	if req.DateOfBirth != "" { existing.DateOfBirth = dob }
	if req.City != "" { existing.City = req.City }
	if req.State != "" { existing.State = req.State }
	if req.Country != "" { existing.Country = req.Country }
	if req.Tags != nil { existing.Tags = req.Tags }
	existing.LastActivityAt = now
	existing.Version = now

	if err := repository.UpdateContact(ctx, existing); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, existing)
}

// @Summary Soft Delete Contact
// @Description Soft deletes a contact, keeping it in the database but removing it from active lists.
// @Tags contacts
// @Produce json
// @Param id path string true "Contact UUID"
// @Success 200 {object} map[string]string "message: contact Deleted successfully"
// @Failure 400 {object} map[string]string "Already deleted"
// @Failure 404 {object} map[string]string "Contact not found"
// @Security ApiKeyAuth
// @Router /contacts/{id} [delete]
func DeleteContactHandler(c echo.Context) error {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid contact UUID"})
	}

	ctx := c.Request().Context()

	contact, err := repository.GetContactByID(ctx, contactID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "contact not found"})
	}

	if contact.IsDeleted == 1 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "contact is already deleted"})
	}

	now := time.Now()
	if err := repository.SoftDeleteContact(ctx, contactID, now); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "contact Deleted successfully"})
}

// @Summary Restore Contact
// @Description Restores a soft-deleted contact.
// @Tags contacts
// @Produce json
// @Param id path string true "Contact UUID"
// @Success 200 {object} map[string]string "message: contact restored successfully"
// @Failure 400 {object} map[string]string "Already active"
// @Failure 404 {object} map[string]string "Contact not found"
// @Security ApiKeyAuth
// @Router /contacts/{id}/restore [post]
func RestoreContactHandler(c echo.Context) error {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid contact UUID"})
	}

	ctx := c.Request().Context()

	contact, err := repository.GetContactByID(ctx, contactID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "contact not found"})
	}

	if contact.IsDeleted == 0{
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Contact is already active "})
	}

	now := time.Now()
	if err := repository.RestoreContact(ctx, contactID, now); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "contact restored successfully"})
}

// @Summary Get Notes
// @Description View all notes for a specific contact.
// @Tags notes
// @Produce json
// @Param id path string true "Contact UUID"
// @Success 200 {array} models.ContactNote
// @Failure 404 {object} map[string]string "Contact not found"
// @Security ApiKeyAuth
// @Router /contacts/{id}/notes [get]
func GetNotesHandler(c echo.Context) error {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid contact UUID"})
	}

	ctx := c.Request().Context()

	notes, err := repository.GetContactNotes(ctx, contactID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, notes)
}

// @Summary Add Note
// @Description Add a new note to a contact.
// @Tags notes
// @Accept json
// @Produce json
// @Param id path string true "Contact UUID"
// @Param request body map[string]string true "Note content"
// @Success 201 {object} models.ContactNote
// @Failure 400 {object} map[string]string "Validation error"
// @Security ApiKeyAuth
// @Router /contacts/{id}/notes [post]
func AddNoteHandler(c echo.Context) error {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid contact UUID"})
	}

	var req struct {
		Note string `json:"note" binding:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	ctx := c.Request().Context()

	contact, err := repository.GetContactByID(ctx, contactID)
	if err != nil || contact.IsDeleted == 1 {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "contact not found"})
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
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	contact.LastActivityAt = now
	contact.Version = now
	_ = repository.UpdateContact(ctx, contact)

	_ = repository.AddContactActivity(ctx, &models.ContactActivity{
		ID:           uuid.New(),
		ContactID:    contactID,
		ActivityType: "note_added",
		Details:      "Note added: " + req.Note,
		CreatedAt:    now,
	})

	return c.JSON(http.StatusCreated, note)
}

// @Summary Update Note
// @Description Edit an existing note.
// @Tags notes
// @Accept json
// @Produce json
// @Param id path string true "Contact UUID"
// @Param note_id path string true "Note UUID"
// @Param request body map[string]string true "Note content"
// @Success 200 {object} models.ContactNote
// @Failure 400 {object} map[string]string "Validation error"
// @Security ApiKeyAuth
// @Router /contacts/{id}/notes/{note_id} [put]
func UpdateNoteHandler(c echo.Context) error {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid contact UUID"})
	}

	noteIDStr := c.Param("note_id")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid note UUID"})
	}

	var req struct {
		Note string `json:"note" binding:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	ctx := c.Request().Context()

	note, err := repository.GetContactNoteByID(ctx, contactID, noteID)
	if err != nil || note.IsDeleted == 1 {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "note not found"})
	}

	now := time.Now()
	note.Note = req.Note
	note.UpdatedAt = now
	note.Version = now // new version

	if err := repository.UpdateContactNote(ctx, note); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	contact, err := repository.GetContactByID(ctx, contactID)
	if err == nil {
		contact.LastActivityAt = now
		contact.Version = now
		_ = repository.UpdateContact(ctx, contact)
	}

	return c.JSON(http.StatusOK, note)
}

// @Summary Delete Note
// @Description Soft delete a note.
// @Tags notes
// @Produce json
// @Param id path string true "Contact UUID"
// @Param note_id path string true "Note UUID"
// @Success 200 {object} map[string]string "message: note deleted successfully"
// @Failure 404 {object} map[string]string "Note not found"
// @Security ApiKeyAuth
// @Router /contacts/{id}/notes/{note_id} [delete]
func DeleteNoteHandler(c echo.Context) error {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid contact UUID"})
	}

	noteIDStr := c.Param("note_id")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid note UUID"})
	}

	ctx := c.Request().Context()

	note, err := repository.GetContactNoteByID(ctx, contactID, noteID)
	if err != nil || note.IsDeleted == 1 {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "note not found"})
	}

	now := time.Now()
	note.IsDeleted = 1
	note.UpdatedAt = now
	note.Version = now

	if err := repository.UpdateContactNote(ctx, note); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	contact, err := repository.GetContactByID(ctx, contactID)
	if err == nil {
		contact.LastActivityAt = now
		contact.Version = now
		_ = repository.UpdateContact(ctx, contact)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "note deleted successfully"})
}

// @Summary Update Tags
// @Description Overwrite tags array for a contact.
// @Tags contacts
// @Accept json
// @Produce json
// @Param id path string true "Contact UUID"
// @Param request body map[string][]string true "Tags array"
// @Success 200 {object} models.Contact
// @Failure 400 {object} map[string]string "Validation error"
// @Security ApiKeyAuth
// @Router /contacts/{id}/tags [patch]
func UpdateTagsHandler(c echo.Context) error {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid contact UUID"})
	}

	var req struct {
		Tags []string `json:"tags" binding:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": err.Error()})
	}

	ctx := c.Request().Context()

	contact, err := repository.GetContactByID(ctx, contactID)
	if err != nil || contact.IsDeleted == 1 {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "contact not found"})
	}

	now := time.Now()
	contact.Tags = req.Tags
	contact.LastActivityAt = now
	contact.Version = now

	if err := repository.UpdateContact(ctx, contact); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, contact)
}
