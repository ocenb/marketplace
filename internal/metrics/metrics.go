package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	instance *Metrics
	once     sync.Once
)

type Metrics struct {
	RequestsCounter *prometheus.CounterVec
	ResponseTime    *prometheus.HistogramVec
	ListingsCounter prometheus.Counter
}

func NewMetrics(namespace string) *Metrics {
	once.Do(func() {
		instance = &Metrics{
			RequestsCounter: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Name:      "http_requests_total",
					Help:      "Total number of HTTP requests",
				},
				[]string{"method", "endpoint", "status"},
			),
			ResponseTime: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: namespace,
					Name:      "http_response_time_seconds",
					Help:      "HTTP response time in seconds",
					Buckets:   prometheus.DefBuckets,
				},
				[]string{"method", "endpoint"},
			),
			ListingsCounter: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: namespace,
					Name:      "listings_created_total",
					Help:      "Total number of listings created",
				},
			),
		}
	})
	return instance
}
