package middlewares

import (
	"fmt"
	"mkk_basis/rest_api/internal/config"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type metricKey struct {
	Method string
	Path   string
	Status int
}

type httpMetrics struct {
	mu              sync.Mutex
	requestsTotal   map[metricKey]uint64
	errorsTotal     map[metricKey]uint64
	durationSeconds map[metricKey]float64
}

var defaultHTTPMetrics = &httpMetrics{
	requestsTotal:   map[metricKey]uint64{},
	errorsTotal:     map[metricKey]uint64{},
	durationSeconds: map[metricKey]float64{},
}

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := config.CurrentConfig.Metrics
		if cfg == nil || !cfg.Enabled {
			c.Next()
			return
		}
		metricsPath := cfg.Path
		if metricsPath == "" {
			metricsPath = "/metrics"
		}
		if c.Request.URL.Path == metricsPath {
			c.Next()
			return
		}

		startedAt := time.Now()
		c.Next()
		defaultHTTPMetrics.observe(c, time.Since(startedAt).Seconds())
	}
}

func MetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := config.CurrentConfig.Metrics
		if cfg == nil || !cfg.Enabled {
			c.Status(http.StatusNotFound)
			return
		}

		c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(defaultHTTPMetrics.render()))
	}
}

func (m *httpMetrics) observe(c *gin.Context, durationSeconds float64) {
	key := metricKey{
		Method: c.Request.Method,
		Path:   routePath(c),
		Status: c.Writer.Status(),
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestsTotal[key]++
	m.durationSeconds[key] += durationSeconds
	if key.Status >= http.StatusInternalServerError || len(c.Errors) > 0 {
		m.errorsTotal[key]++
	}
}

func (m *httpMetrics) render() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	keys := m.keys()
	var b strings.Builder
	b.WriteString("# HELP http_requests_total Total number of HTTP requests.\n")
	b.WriteString("# TYPE http_requests_total counter\n")
	for _, key := range keys {
		fmt.Fprintf(&b, "http_requests_total%s %d\n", key.promLabels(), m.requestsTotal[key])
	}

	b.WriteString("# HELP http_errors_total Total number of failed HTTP requests.\n")
	b.WriteString("# TYPE http_errors_total counter\n")
	for _, key := range keys {
		if value := m.errorsTotal[key]; value > 0 {
			fmt.Fprintf(&b, "http_errors_total%s %d\n", key.promLabels(), value)
		}
	}

	b.WriteString("# HELP http_response_duration_seconds_sum Total HTTP response duration in seconds.\n")
	b.WriteString("# TYPE http_response_duration_seconds_sum counter\n")
	for _, key := range keys {
		fmt.Fprintf(&b, "http_response_duration_seconds_sum%s %.6f\n", key.promLabels(), m.durationSeconds[key])
	}

	b.WriteString("# HELP http_response_duration_seconds_count Count of observed HTTP response durations.\n")
	b.WriteString("# TYPE http_response_duration_seconds_count counter\n")
	for _, key := range keys {
		fmt.Fprintf(&b, "http_response_duration_seconds_count%s %d\n", key.promLabels(), m.requestsTotal[key])
	}

	return b.String()
}

func (m *httpMetrics) keys() []metricKey {
	seen := make(map[metricKey]struct{}, len(m.requestsTotal))
	keys := make([]metricKey, 0, len(m.requestsTotal))
	for key := range m.requestsTotal {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Path != keys[j].Path {
			return keys[i].Path < keys[j].Path
		}
		if keys[i].Method != keys[j].Method {
			return keys[i].Method < keys[j].Method
		}
		return keys[i].Status < keys[j].Status
	})

	return keys
}

func (k metricKey) promLabels() string {
	return fmt.Sprintf("{method=%q,path=%q,status=%q}", escapePromLabel(k.Method), escapePromLabel(k.Path), fmt.Sprintf("%d", k.Status))
}

func routePath(c *gin.Context) string {
	if path := c.FullPath(); path != "" {
		return path
	}
	return c.Request.URL.Path
}

func escapePromLabel(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	return strings.ReplaceAll(value, "\"", "\\\"")
}
