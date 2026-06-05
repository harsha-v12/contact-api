package models

// ContactListFilter holds all query parameters for listing, searching, and filtering contacts
type ContactListFilter struct {
	// Pagination:
	Page  int `form:"page"`
	Limit int `form:"limit"`

	// Sorting — field name and direction (asc | desc)
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order"`

	// Full-text search (name, email, mobile_number — partial, case-insensitive)
	Search string `form:"search"`

	// Filters
	Tags         []string `form:"tags"`          // filter contacts that have ANY of these tags
	Country      string   `form:"country"`
	State        string   `form:"state"`
	City         string   `form:"city"`          
	CreatedFrom  string   `form:"created_from"`  // YYYY-MM-DD
	CreatedTo    string   `form:"created_to"`    // YYYY-MM-DD
	ActivityFrom string   `form:"activity_from"` // last_activity_at range start
	ActivityTo   string   `form:"activity_to"`   // last_activity_at range end
}

// ContactListItem is the lightweight row returned in a contact listing
type ContactListItem struct {
	ID             string   `json:"id"`
	FirstName      string   `json:"first_name"`
	LastName       string   `json:"last_name"`
	Email          string   `json:"email"`
	MobileNumber   string   `json:"mobile_number"`
	Tags           []string `json:"tags"`
	CreatedAt      string   `json:"created_at"`
	LastActivityAt string   `json:"last_activity_at"`
}

// ContactListResponse wraps the paginated list result
type ContactListResponse struct {
	Data       []ContactListItem `json:"data"`
	Total      int               `json:"total"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"total_pages"`
}
