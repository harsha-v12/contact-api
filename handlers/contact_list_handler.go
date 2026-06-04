package handlers

import (
	"math"
	"net/http"

	"contact-management/models"
	"contact-management/repository"

	"github.com/gin-gonic/gin"
)

// ListContactsHandler handles GET /api/v1/contacts
// Supports: pagination, sorting, full-text search, and multiple filters.
//
// Query Parameters:
//
//	page          int      (default: 1)
//	limit         int      (default: 20, max: 200)
//	sort_by       string   (first_name | last_name | email | created_at | last_activity_at)
//	sort_order    string   (asc | desc, default: desc)
//	search        string   (partial, case-insensitive match on name / email / mobile)
//	tags          []string (repeated: ?tags=VIP&tags=Prospect — contact must have at least one)
//	country       string
//	state         string
//	city          string
//	created_from  string   (YYYY-MM-DD)
//	created_to    string   (YYYY-MM-DD)
//	activity_from string   (YYYY-MM-DD)
//	activity_to   string   (YYYY-MM-DD)
func ListContactsHandler(c *gin.Context) {
	var filter models.ContactListFilter

	// Bind all query params at once (gin handles repeated keys for slices)
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters: " + err.Error()})
		return
	}

	ctx := c.Request.Context()

	contacts, total, err := repository.ListContacts(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Apply defaults here as well to return consistent meta
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 || limit > 200 {
		limit = 20
	}

	totalPages := 1
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	c.JSON(http.StatusOK, models.ContactListResponse{
		Data:       contacts,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}
