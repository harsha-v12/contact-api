package handlers

import (
	"math"
	"net/http"

	"contact-management/models"
	"contact-management/repository"

	"github.com/labstack/echo/v4"
)

// ListContactsHandler handles GET /api/v1/contacts with support of pagination all the things 
// @Summary List all contacts
// @Description Get a paginated list of contacts with optional filtering, sorting, and search.
// @Tags contacts
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param search query string false "Search across name, email, mobile"
// @Param tags query []string false "Filter by tags (comma separated)"
// @Success 200 {object} models.ContactListResponse
// @Failure 400 {object} map[string]string "Invalid query parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security ApiKeyAuth
// @Router /contacts [get]
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
