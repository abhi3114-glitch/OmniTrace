package dashboard

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/omnitrace/omnitrace/backend/storage"
	"github.com/omnitrace/omnitrace/internal/models"
)

// Server serves the dashboard UI and API
type Server struct {
	spanStore   *storage.SpanStore
	metricStore *storage.MetricStore
	staticDir   string
}

// NewServer creates a new dashboard server
func NewServer(spanStore *storage.SpanStore, metricStore *storage.MetricStore, staticDir string) *Server {
	return &Server{
		spanStore:   spanStore,
		metricStore: metricStore,
		staticDir:   staticDir,
	}
}

// RegisterRoutes registers the dashboard routes
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// API routes
	mux.HandleFunc("/api/traces", s.handleTraces)
	mux.HandleFunc("/api/traces/", s.handleTraceDetail) // Matches /api/traces/{id}
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/api/services", s.handleServices)

	// Static files
	fs := http.FileServer(http.Dir(s.staticDir))
	mux.Handle("/", fs)
}

func (s *Server) handleTraces(w http.ResponseWriter, r *http.Request) {
	query := models.TraceQuery{
		Limit:  50,
		Offset: 0,
	}

	// Parse query params
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			query.Limit = l
		}
	}
	if service := r.URL.Query().Get("service"); service != "" {
		query.Service = service
	}
	if operation := r.URL.Query().Get("operation"); operation != "" {
		query.Operation = operation
	}
	if hasError := r.URL.Query().Get("error"); hasError != "" {
		val := hasError == "true"
		query.HasError = &val
	}
	// Time range params parsing omitted for brevity

	summaries, err := s.spanStore.QueryTraces(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}

func (s *Server) handleTraceDetail(w http.ResponseWriter, r *http.Request) {
	traceID := filepath.Base(r.URL.Path)
	if traceID == "" || traceID == "traces" {
		http.Error(w, "Missing trace ID", http.StatusBadRequest)
		return
	}

	trace, err := s.spanStore.GetTrace(traceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if trace == nil {
		http.Error(w, "Trace not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trace)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing metric name", http.StatusBadRequest)
		return
	}

	query := models.MetricQuery{
		Name:      name,
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
		Step:      time.Minute,
	}

	metrics, err := s.metricStore.QueryMetrics(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (s *Server) handleServices(w http.ResponseWriter, r *http.Request) {
	// In a real implementation this would aggregate from storage
	// For now returns a stub or simple list
	services := []string{"demo-service", "auth-service", "db-service"}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}
