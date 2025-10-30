package metric

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP метрики - количество запросов
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Общее количество HTTP запросов",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTP метрики - время обработки запросов
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Время обработки HTTP запросов в секундах",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTP метрики - количество ошибок
	httpErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Общее количество HTTP ошибок",
		},
		[]string{"method", "endpoint", "status"},
	)

	// WS метрики - количество активных соединений
	wsActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ws_active_connections",
			Help: "Количество активных WebSocket соединений",
		},
	)
)

// RecordHTTPMetrics записывает метрики HTTP запроса
func RecordHTTPMetrics(method, endpoint string, status int, duration time.Duration) {
	strStatus := strconv.Itoa(status)

	httpRequestsTotal.WithLabelValues(method, endpoint, strStatus).Inc()
	httpRequestDuration.WithLabelValues(method, endpoint, strStatus).Observe(duration.Seconds())

	// Записываем ошибки (статус >= 400)
	if status >= 400 {
		httpErrorsTotal.WithLabelValues(method, endpoint, strStatus).Inc()
	}
}

func SetWSActiveConnections(count int) {
	wsActiveConnections.Set(float64(count))
}

func IncrementWSActiveConnections() {
	wsActiveConnections.Inc()
}

func DecrementWSActiveConnections() {
	wsActiveConnections.Dec()
}
