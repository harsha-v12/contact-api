package middleware

import (
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

// APIKeyAuthMiddleware ensures the correct X-API-Key is passed in headers
func APIKeyAuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := os.Getenv("API_KEY")
			
			// If API_KEY is not set in .env, just let the request through (disabled security)
			// if apiKey == "" {
			// 	return next(c)
			// }

			if apiKey == "" {
				return echo.NewHTTPError(
					http.StatusInternalServerError,
					"API_KEY is not configured",
				)
			}

			clientKey := c.Request().Header.Get("X-API-Key")
			
			if clientKey != apiKey {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing or invalid X-API-Key header")
			}

			return next(c)
		}
	}
}