package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"contact-management/config"
	"contact-management/models"
)

// allowedSortFields maps safe external field names to actual ClickHouse column names
var allowedSortFields = map[string]string{
	"first_name":       "first_name",
	"last_name":        "last_name",
	"email":            "email",
	"created_at":       "created_at",
	"last_activity_at": "last_activity_at",
}

// ListContacts returns a paginated, filtered, searchable list of active contacts
func ListContacts(ctx context.Context, f models.ContactListFilter) ([]models.ContactListItem, int, error) {
	
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 200 {
		f.Limit = 20
	}

	sortCol := "created_at"
	if col, ok := allowedSortFields[f.SortBy]; ok {
		sortCol = col
	}

	sortDir := "DESC"
	if strings.ToUpper(f.SortOrder) == "ASC" {
		sortDir = "ASC"
	}

	//  Build WHERE clauses 
	where := []string{"is_deleted = 0"}
	args := []interface{}{}

	// Partial, case-insensitive search across name / email / mobile
	if f.Search != "" {
		searchTerm := "%" + strings.ToLower(f.Search) + "%"
		where = append(where,
			"(lowerUTF8(first_name) LIKE ? OR lowerUTF8(last_name) LIKE ? OR lowerUTF8(email) LIKE ? OR mobile_number LIKE ?)",
		)
		args = append(args, searchTerm, searchTerm, searchTerm, "%"+f.Search+"%")
	}

	// Location filters
	if f.Country != "" {
		where = append(where, "lowerUTF8(country) = lowerUTF8(?)")
		args = append(args, f.Country)
	}
	if f.State != "" {
		where = append(where, "lowerUTF8(state) = lowerUTF8(?)")
		args = append(args, f.State)
	}
	if f.City != "" {
		where = append(where, "lowerUTF8(city) = lowerUTF8(?)")
		args = append(args, f.City)
	}

	// Tag filter — contact must have at least one matching tag
	if len(f.Tags) > 0 {
		tagPlaceholders := make([]string, len(f.Tags))
		for i, tag := range f.Tags {
			tagPlaceholders[i] = "?"
			args = append(args, tag)
		}
		where = append(where, fmt.Sprintf("hasAny(tags, [%s])", strings.Join(tagPlaceholders, ", ")))
	}

	// Date range filters
	if f.CreatedFrom != "" {
		t, err := time.Parse("2006-01-02", f.CreatedFrom)
		if err == nil {
			where = append(where, "created_at >= ?")
			args = append(args, t)
		}
	}
	if f.CreatedTo != "" {
		t, err := time.Parse("2006-01-02", f.CreatedTo)
		if err == nil {
			// Include the whole end-day
			where = append(where, "created_at <= ?")
			args = append(args, t.Add(24*time.Hour-time.Second))
		}
	}
	if f.ActivityFrom != "" {
		t, err := time.Parse("2006-01-02", f.ActivityFrom)
		if err == nil {
			where = append(where, "last_activity_at >= ?")
			args = append(args, t)
		}
	}
	if f.ActivityTo != "" {
		t, err := time.Parse("2006-01-02", f.ActivityTo)
		if err == nil {
			where = append(where, "last_activity_at <= ?")
			args = append(args, t.Add(24*time.Hour-time.Second))
		}
	}

	whereClause := strings.Join(where, " AND ")

	// ── Count total matching rows ─────────────────────────────────────────────
	countQuery := fmt.Sprintf(`SELECT count() FROM contacts FINAL WHERE %s`, whereClause)
	var total uint64
	if err := config.DB.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("list contacts count query failed: %w", err)
	}

	
	offset := (f.Page - 1) * f.Limit

	dataQuery := fmt.Sprintf(`
		SELECT id, first_name, last_name, email, mobile_number, tags, created_at, last_activity_at
		FROM contacts FINAL
		WHERE %s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, whereClause, sortCol, sortDir)

	dataArgs := append(args, f.Limit, offset)

	rows, err := config.DB.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list contacts data query failed: %w", err)
	}
	defer rows.Close()

	var contacts []models.ContactListItem
	for rows.Next() {
		var c models.ContactListItem
		var tags []string
		var createdAt, lastActivityAt time.Time

		err := rows.Scan(&c.ID, &c.FirstName, &c.LastName, &c.Email, &c.MobileNumber, &tags, &createdAt, &lastActivityAt)
		if err != nil {
			return nil, 0, fmt.Errorf("list contacts scan failed: %w", err)
		}
		c.Tags = tags
		c.CreatedAt = createdAt.Format(time.RFC3339)
		c.LastActivityAt = lastActivityAt.Format(time.RFC3339)
		contacts = append(contacts, c)
	}

	if contacts == nil {
		contacts = []models.ContactListItem{}
	}

	return contacts, int(total), nil
}
