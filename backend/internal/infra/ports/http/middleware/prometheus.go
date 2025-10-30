package middleware

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/qrave1/RoomSpeak/internal/application/metric"
)

// PrometheusMiddleware создает middleware для сбора метрик HTTP запросов
func PrometheusMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Получаем информацию о запросе
			method := c.Request().Method
			endpoint := c.Path() // Полный путь эндпоинта

			// Обрабатываем запрос
			err := next(c)

			// Вычисляем длительность
			duration := time.Since(start)

			// Определяем статус код
			statusCode := c.Response().Status
			if statusCode == 0 {
				statusCode = 200
			}

			// Если произошла ошибка, но статус не установлен, устанавливаем 500
			if err != nil && statusCode < 400 {
				statusCode = 500
			}

			// Записываем метрики
			metric.RecordHTTPMetrics(method, endpoint, statusCode, duration)

			return err
		}
	}
}
