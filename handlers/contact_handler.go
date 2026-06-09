package handlers

import (
	"net/http"
	"time"

	"contact-management/models"
	"contact-management/repository"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

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
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, existing)
}

func DeleteContactHandler(c echo.Context) error {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid contact UUID"})
	}

	ctx := c.Request().Context()

	_, err = repository.GetContactByID(ctx, contactID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "contact not found"})
	}

	now := time.Now()
	if err := repository.SoftDeleteContact(ctx, contactID, now); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "contact Deleted successfully"})
}

func RestoreContactHandler(c echo.Context) error {
	contactIDStr := c.Param("id")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid contact UUID"})
	}

	ctx := c.Request().Context()

	_, err = repository.GetContactByID(ctx, contactID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{"error": "contact not found"})
	}

	now := time.Now()
	if err := repository.RestoreContact(ctx, contactID, now); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"message": "contact restored successfully"})
}

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
