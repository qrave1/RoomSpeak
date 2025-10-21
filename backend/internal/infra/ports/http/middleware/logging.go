package middleware

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Кастомный логгер через slog
func SlogLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(
		middleware.RequestLoggerConfig{
			LogStatus: true,
			LogURI:    true,
			LogMethod: true,
			LogError:  true,

			LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
				level := slog.LevelInfo
				if v.Error != nil || v.Status >= http.StatusInternalServerError {
					level = slog.LevelError
				} else if v.Status >= http.StatusBadRequest {
					level = slog.LevelWarn
				}

				slog.LogAttrs(
					c.Request().Context(),
					level,
					"HTTP request",
					slog.Int("status", v.Status),
					slog.String("uri", v.URI),
					slog.String("method", v.Method),
				)

				return nil
			},
		},
	)
}
