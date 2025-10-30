package metric

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewServer создает новый сервер метрик
func NewServer() *echo.Echo {
	e := echo.New()

	e.HideBanner = true

	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	return e
}
