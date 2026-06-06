package repository

import (
	"context"
	"fmt"
	"time"

	"contact-management/config"
	"contact-management/models"

	"github.com/google/uuid"
)

// CheckDuplicate checks if an active contact exists with the same email or mobile number
func CheckDuplicate(ctx context.Context, email, mobileNumber string, excludeID *uuid.UUID) (bool, error) {
	if email == "" && mobileNumber == "" {
		return false, nil
	}

	var query string
	var args []interface{}


	// used for the update case 
	if excludeID != nil {
		query = `
			SELECT count() 
			FROM contacts FINAL 
			WHERE is_deleted = 0 
			  AND id != ? 
			  AND (
			    (email != '' AND email = ?) 
			    OR (mobile_number != '' AND mobile_number = ?)
			  )
		`
		args = append(args, *excludeID, email, mobileNumber)
	} else {
		// used for the create the contact of the user 
		query = `
			SELECT count() 
			FROM contacts FINAL 
			WHERE is_deleted = 0 
			  AND (
			    (email != '' AND email = ?) 
			    OR (mobile_number != '' AND mobile_number = ?)
			  )
		`
		args = append(args, email, mobileNumber)
	}

	var count uint64
	err := config.DB.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("duplicate check query failed: %w", err)
	}

	return count > 0, nil
}

// CreateContact inserts a contact record
func CreateContact(ctx context.Context, c *models.Contact) error {
	query := `
		INSERT INTO contacts (
			id, first_name, last_name, email, mobile_number, gender, 
			date_of_birth, city, state, country, tags, notes, 
			created_at, last_activity_at, is_deleted, version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	err := config.DB.Exec(ctx, query,
		c.ID, c.FirstName, c.LastName, c.Email, c.MobileNumber, c.Gender,
		c.DateOfBirth, c.City, c.State, c.Country, c.Tags, c.Notes,
		c.CreatedAt, c.LastActivityAt, c.IsDeleted, c.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to insert contact: %w", err)
	}
	return nil
}

// Given a Contact ID
//         ↓
// Fetch that contact from ClickHouse
//         ↓
// Convert DB row → Go struct
//         ↓
// Return Contact object
// GetContactByID retrieves a contact by UUID
func GetContactByID(ctx context.Context, id uuid.UUID) (*models.Contact, error) {
	query := `
		SELECT 
			id, first_name, last_name, email, mobile_number, gender, 
			date_of_birth, city, state, country, tags, notes, 
			created_at, last_activity_at, is_deleted, version
		FROM contacts FINAL 
		WHERE id = ? LIMIT 1
	`
	var c models.Contact
	row := config.DB.QueryRow(ctx, query, id)
	err := row.Scan(
		&c.ID, &c.FirstName, &c.LastName, &c.Email, &c.MobileNumber, &c.Gender,
		&c.DateOfBirth, &c.City, &c.State, &c.Country, &c.Tags, &c.Notes,
		&c.CreatedAt, &c.LastActivityAt, &c.IsDeleted, &c.Version,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan contact: %w", err)
	}
	return &c, nil
}

// UpdateContact updates an existing contact details by inserting a new version
func UpdateContact(ctx context.Context, c *models.Contact) error {
	query := `
		INSERT INTO contacts (
			id, first_name, last_name, email, mobile_number, gender, 
			date_of_birth, city, state, country, tags, notes, 
			created_at, last_activity_at, is_deleted, version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	err := config.DB.Exec(ctx, query,
		c.ID, c.FirstName, c.LastName, c.Email, c.MobileNumber, c.Gender,
		c.DateOfBirth, c.City, c.State, c.Country, c.Tags, c.Notes,
		c.CreatedAt, c.LastActivityAt, c.IsDeleted, c.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}
	return nil
}

// SoftDeleteContact soft-deletes a contact by setting is_deleted=1 with new version
func SoftDeleteContact(ctx context.Context, id uuid.UUID, version time.Time) error {
	// We need to retrieve the latest record to preserve fields (since ClickHouse replacing merge tree replaces the entire row)
	c, err := GetContactByID(ctx, id)
	if err != nil {
		return err
	}

	c.IsDeleted = 1
	c.Version = version

	return UpdateContact(ctx, c)
}

// RestoreContact restores a soft-deleted contact by setting is_deleted=0
func RestoreContact(ctx context.Context, id uuid.UUID, version time.Time) error {
	c, err := GetContactByID(ctx, id)
	if err != nil {
		return err
	}

	c.IsDeleted = 0
	c.Version = version

	return UpdateContact(ctx, c)
}

// AddContactNote inserts a new note record
func AddContactNote(ctx context.Context, n *models.ContactNote) error {
	query := `
		INSERT INTO contact_notes (
			id, contact_id, note, created_at, updated_at, is_deleted, version
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	err := config.DB.Exec(ctx, query,
		n.ID, n.ContactID, n.Note, n.CreatedAt, n.UpdatedAt, n.IsDeleted, n.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to insert contact note: %w", err)
	}
	return nil
}

// GetContactNotes retrieves active notes for a contact
func GetContactNotes(ctx context.Context, contactID uuid.UUID) ([]models.ContactNote, error) {
	query := `
		SELECT id, contact_id, note, created_at, updated_at, is_deleted, version
		FROM contact_notes FINAL
		WHERE contact_id = ? AND is_deleted = 0
		ORDER BY created_at DESC
	`
	rows, err := config.DB.Query(ctx, query, contactID)
	if err != nil {
		return nil, fmt.Errorf("notes fetch query failed: %w", err)
	}
	defer rows.Close()

	var notes []models.ContactNote
	for rows.Next() {
		var n models.ContactNote
		err := rows.Scan(&n.ID, &n.ContactID, &n.Note, &n.CreatedAt, &n.UpdatedAt, &n.IsDeleted, &n.Version)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contact note: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, nil
}

// GetContactNoteByID retrieves a single note by ID
func GetContactNoteByID(ctx context.Context, contactID, noteID uuid.UUID) (*models.ContactNote, error) {
	query := `
		SELECT id, contact_id, note, created_at, updated_at, is_deleted, version
		FROM contact_notes FINAL
		WHERE contact_id = ? AND id = ? LIMIT 1
	`
	var n models.ContactNote
	err := config.DB.QueryRow(ctx, query, contactID, noteID).Scan(
		&n.ID, &n.ContactID, &n.Note, &n.CreatedAt, &n.UpdatedAt, &n.IsDeleted, &n.Version,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch contact note: %w", err)
	}
	return &n, nil
}

// UpdateContactNote updates a note's text by inserting a new version
func UpdateContactNote(ctx context.Context, n *models.ContactNote) error {
	return AddContactNote(ctx, n)
}

// AddContactActivity logs an activity event for a contact
func AddContactActivity(ctx context.Context, a *models.ContactActivity) error {
	query := `
		INSERT INTO contact_activities (
			id, contact_id, activity_type, details, created_at
		) VALUES (?, ?, ?, ?, ?)
	`
	err := config.DB.Exec(ctx, query,
		a.ID, a.ContactID, a.ActivityType, a.Details, a.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to log contact activity: %w", err)
	}
	return nil
}

// GetContactActivities retrieves the chronological activity timeline for a contact
func GetContactActivities(ctx context.Context, contactID uuid.UUID) ([]models.ContactActivity, error) {
	query := `
		SELECT id, contact_id, activity_type, details, created_at
		FROM contact_activities
		WHERE contact_id = ?
		ORDER BY created_at DESC
	`
	rows, err := config.DB.Query(ctx, query, contactID)
	if err != nil {
		return nil, fmt.Errorf("activities query failed: %w", err)
	}
	defer rows.Close()

	var activities []models.ContactActivity
	for rows.Next() {
		var a models.ContactActivity
		err := rows.Scan(&a.ID, &a.ContactID, &a.ActivityType, &a.Details, &a.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity: %w", err)
		}
		activities = append(activities, a)
	}
	return activities, nil
}

// GetContactActivitySummary returns counts of each activity type for a contact
func GetContactActivitySummary(ctx context.Context, contactID uuid.UUID) (map[string]int, error) {
	query := `
		SELECT activity_type, count()
		FROM contact_activities
		WHERE contact_id = ?
		GROUP BY activity_type
	`
	rows, err := config.DB.Query(ctx, query, contactID)
	if err != nil {
		return nil, fmt.Errorf("activity summary query failed: %w", err)
	}
	defer rows.Close()

	summary := map[string]int{
		"contact_created":  0,
		"note_added":       0,
		"email_sent":       0,
		"whatsapp_sent":    0,
		"meeting_attended": 0,
		"video_watched":    0,
		"event_attended":   0,
	}

	for rows.Next() {
		var actType string
		var count uint64
		if err := rows.Scan(&actType,&count); err != nil {
			return nil, fmt.Errorf("failed to scan activity summary row: %w", err)
		}
		summary[actType] = int(count)
	}
	return summary, nil
}
