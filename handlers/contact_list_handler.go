package handlers

import (
	"math"
	"net/http"

	"contact-management/models"
	"contact-management/repository"

	"github.com/labstack/echo/v4"
)

// ListContactsHandler handles GET /api/v1/contacts with support of pagination all the things 
func ListContactsHandler(c echo.Context) error {
	var filter models.ContactListFilter

	// Bind all query params at once
	if err := c.Bind(&filter); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "invalid query parameters: " + err.Error()})
	}

	ctx := c.Request().Context()

	contacts, total, err := repository.ListContacts(ctx, filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
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

	return c.JSON(http.StatusOK, models.ContactListResponse{
		Data:       contacts,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}
