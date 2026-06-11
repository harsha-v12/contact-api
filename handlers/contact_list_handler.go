package handlers

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"time"

	"contact-management/config"

	"contact-management/models"
	"contact-management/repository"

	"github.com/labstack/echo/v4"
)
//2026/06/11 11:30:28 REQUEST: GET /api/v1/contacts?page=1&limit=5&search=a&status=Male&city=Guntur&state=Andhra+Pradesh&country=India | STATUS: 200

// ListContactsHandler handles GET /api/v1/contacts with support of pagination all the things 
// @Summary List all contacts
// @Description Get a paginated list of contacts with optional filtering, sorting, and search.
// @Tags contacts
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param search query string false "Search across name, email, mobile"
// @Param gender query string false "Filter by gender (e.g., male, female)"
// @Param tags query []string false "Filter by tags (e.g., VIP)"
// @Param city query string false "Filter by city"
// @Param state query string false "Filter by state"
// @Param country query string false "Filter by country"
// @Param created_from query string false "Filter by creation date from (YYYY-MM-DD)"
// @Param created_to query string false "Filter by creation date to (YYYY-MM-DD)"
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


	// 1. SMART REDIS CACHE (Only cache the default view!)
	
	isDefaultLoad := filter.Search == "" && filter.Gender == "" && len(filter.Tags) == 0 &&
		filter.Country == "" && filter.State == "" && filter.City == "" &&
		filter.CreatedFrom == "" && filter.CreatedTo == "" &&
		filter.ActivityFrom == "" && filter.ActivityTo == "" &&
		(filter.Page <= 1)

	cacheKey := "cache:contacts:dashboard_default"

	if isDefaultLoad {
		if cachedData, err := config.RedisClient.Get(ctx, cacheKey).Result(); err == nil && cachedData != "" {
			log.Printf("🚀 [REDIS] CACHE HIT! Returning 200 contacts instantly from memory.")
			c.Response().Header().Set("X-Cache", "HIT")
			return c.JSONBlob(http.StatusOK, []byte(cachedData))
		}
	}

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

	response := models.ContactListResponse{
		Data:       contacts,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}

	
	// 2. SAVE DASHBOARD TO CACHE
	if isDefaultLoad {
		log.Printf("🗄️ [CLICKHOUSE] CACHE MISS! Fetching contacts and saving to Redis...")
		if responseBytes, err := json.Marshal(response); err == nil {
			// Save the default dashboard in Redis for 10 seconds
			config.RedisClient.Set(ctx, cacheKey, responseBytes, 10*time.Second)
		}
	}

	c.Response().Header().Set("X-Cache", "MISS")
	return c.JSON(http.StatusOK, response)
}
