package ingestion

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/omnitrace/omnitrace/internal/models"
)

// Server handles HTTP ingestion of spans and metrics
type Server struct {
	processor *Processor
}

// NewServer creates a new ingestion server
func NewServer(processor *Processor) *Server {
	return &Server{
		processor: processor,
	}
}

// HandleSpans handles interactions for span ingestion
func (s *Server) HandleSpans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var batch models.SpanBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Received batch of %d spans", len(batch.Spans))

	// Process spans asynchronously
	go s.processor.ProcessSpans(batch.Spans)

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"accepted"}`))
}

// HandleMetrics handles interactions for metric ingestion
func (s *Server) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var batch models.MetricBatch
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Process metrics asynchronously
	go s.processor.ProcessMetrics(batch.Metrics)

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"accepted"}`))
}

// RegisterRoutes registers the ingestion routes
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/spans", s.HandleSpans)
	mux.HandleFunc("/api/v1/metrics", s.HandleMetrics)
}
