// Package daemon implements the background daemon service.
package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/forge-platform/forge/internal/core/services"
)

// HTTPServer provides HTTP endpoints for Cloud Run and Kubernetes health checks.
type HTTPServer struct {
	server    *http.Server
	healthSvc *services.HealthService
	version   string
	startTime time.Time
}

// NewHTTPServer creates a new HTTP server for health checks.
func NewHTTPServer(port string, healthSvc *services.HealthService, version string) *HTTPServer {
	if port == "" {
		// Use PORT environment variable (Cloud Run)
		port = os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
	}

	h := &HTTPServer{
		healthSvc: healthSvc,
		version:   version,
		startTime: time.Now(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleRoot)
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/health/liveness", h.handleLiveness)
	mux.HandleFunc("/health/readiness", h.handleReadiness)
	mux.HandleFunc("/metrics", h.handleMetrics)

	h.server = &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	return h
}

// Start starts the HTTP server.
func (h *HTTPServer) Start() error {
	return h.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (h *HTTPServer) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

// Addr returns the server address.
func (h *HTTPServer) Addr() string {
	return h.server.Addr
}

// handleRoot handles the root endpoint.
func (h *HTTPServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"name":    "Forge Platform",
		"version": h.version,
		"uptime":  time.Since(h.startTime).String(),
		"status":  "running",
	})
}

// handleHealth handles full health check.
func (h *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.healthSvc == nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"version": h.version,
			"uptime":  time.Since(h.startTime).String(),
		})
		return
	}

	health := h.healthSvc.Check(r.Context())
	if health.Status == services.HealthStatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     string(health.Status),
		"version":    health.Version,
		"uptime":     health.Uptime.String(),
		"checked_at": health.CheckedAt.Format(time.RFC3339),
		"components": h.componentsToSlice(health.Components),
	})
}

// handleLiveness handles Kubernetes/Cloud Run liveness probe.
func (h *HTTPServer) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	alive := true
	if h.healthSvc != nil {
		alive = h.healthSvc.CheckLiveness(r.Context())
	}
	if !alive {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"alive": alive})
}

// handleReadiness handles Kubernetes/Cloud Run readiness probe.
func (h *HTTPServer) handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ready := true
	if h.healthSvc != nil {
		ready = h.healthSvc.CheckReadiness(r.Context())
	}
	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ready": ready})
}

// handleMetrics handles Prometheus-style metrics endpoint.
func (h *HTTPServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	uptime := time.Since(h.startTime).Seconds()
	fmt.Fprintf(w, "# HELP forge_uptime_seconds Uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE forge_uptime_seconds gauge\n")
	fmt.Fprintf(w, "forge_uptime_seconds %.2f\n", uptime)
}

// componentsToSlice converts component health to a slice for JSON.
func (h *HTTPServer) componentsToSlice(components []services.ComponentHealth) []map[string]interface{} {
	result := make([]map[string]interface{}, len(components))
	for i, c := range components {
		result[i] = map[string]interface{}{
			"name":    c.Name,
			"status":  string(c.Status),
			"message": c.Message,
			"latency": c.Latency.String(),
		}
	}
	return result
}

