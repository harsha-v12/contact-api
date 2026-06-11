package models

import (
	"regexp"
	"time"

	"github.com/google/uuid"
)

var (
	EmailRegex  = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	MobileRegex = regexp.MustCompile(`^(?:\+91)?[6-9]\d{9}$`) 
)

// Contact represents the ClickHouse contact schema structure
type Contact struct {
	ID             uuid.UUID `json:"id"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	Email          string    `json:"email"`
	MobileNumber   string    `json:"mobile_number"`
	Gender         string    `json:"gender"`
	DateOfBirth    time.Time `json:"date_of_birth"`
	City           string    `json:"city"`
	State          string    `json:"state"`
	Country        string    `json:"country"`
	Tags           []string  `json:"tags"`
	Notes          string    `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
	LastActivityAt time.Time `json:"last_activity_at"`
	IsDeleted      uint8     `json:"-"`
	Version        time.Time `json:"version"`
}


type ContactNote struct {
	ID        uuid.UUID `json:"id"`
	ContactID uuid.UUID `json:"contact_id"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsDeleted uint8     `json:"-"`
	Version   time.Time `json:"version"`
}

// ContactActivity represents an activity timeline entry
type ContactActivity struct {
	ID           uuid.UUID `json:"id"`
	ContactID    uuid.UUID `json:"contact_id"`
	ActivityType string    `json:"activity_type"` // contact_created, note_added, email_sent, whatsapp_sent, etc.
	Details      string    `json:"details"`
	CreatedAt    time.Time `json:"created_at"`
}

// ContactCreateRequest is the payload for creating a contact
type ContactCreateRequest struct {
	FirstName    string   `json:"first_name" binding:"required"`
	LastName     string   `json:"last_name"`
	Email        string   `json:"email"`
	MobileNumber string   `json:"mobile_number"`
	Gender       string   `json:"gender"`
	DateOfBirth  string   `json:"date_of_birth"` // Expected format: YYYY-MM-DD
	City         string   `json:"city"`
	State        string   `json:"state"`
	Country      string   `json:"country"`
	Tags         []string `json:"tags"`
	Notes        string   `json:"notes"`
}

// Validate checks creation requirements and formats and returns all errors
func (r *ContactCreateRequest) Validate() map[string]string {
	errors := make(map[string]string)

	if r.FirstName == "" {
		errors["first_name"] = "first_name is mandatory"
	}

	if r.Email == "" && r.MobileNumber == "" {
		errors["contact_info"] = "either email or mobile_number must be provided"
	}

	if r.Email != "" && !EmailRegex.MatchString(r.Email) {
		errors["email"] = "email format is invalid"
	}

	if r.MobileNumber != "" && !MobileRegex.MatchString(r.MobileNumber) {
		errors["mobile_number"] = "mobile_number format is invalid (must be a valid 10-digit Indian number, optionally starting with +91)"
	}

	if r.DateOfBirth != "" {
		_, err := time.Parse("2006-01-02", r.DateOfBirth)
		if err != nil {
			errors["date_of_birth"] = "date_of_birth format is invalid (expected YYYY-MM-DD)"
		}
	}

	return errors
}

// ContactUpdateRequest is the payload for updating a contact
type ContactUpdateRequest struct {
	FirstName    string   `json:"first_name" binding:"required"`
	LastName     string   `json:"last_name"`
	Email        string   `json:"email"`
	MobileNumber string   `json:"mobile_number"`
	Gender       string   `json:"gender"`
	DateOfBirth  string   `json:"date_of_birth"` // Expected format: YYYY-MM-DD
	City         string   `json:"city"`
	State        string   `json:"state"`
	Country      string   `json:"country"`
	Tags         []string `json:"tags"`
}

// Validate checks update requirements and returns all errors
func (r *ContactUpdateRequest) Validate() map[string]string {
	// All fields are optional for partial updates. We only validate formats if they are provided.
	errors := make(map[string]string)

	if r.Email != "" && !EmailRegex.MatchString(r.Email) {
		errors["email"] = "email format is invalid"
	}

	if r.MobileNumber != "" && !MobileRegex.MatchString(r.MobileNumber) {
		errors["mobile_number"] = "mobile_number format is invalid (must be a valid 10-digit Indian number, optionally starting with +91)"
	}

	if r.DateOfBirth != "" {
		_, err := time.Parse("2006-01-02", r.DateOfBirth)
		if err != nil {
			errors["date_of_birth"] = "date_of_birth format is invalid (expected YYYY-MM-DD)"
		}
	}

	return errors
}
