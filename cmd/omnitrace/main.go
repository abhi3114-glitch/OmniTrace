package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/omnitrace/omnitrace/backend/dashboard"
	"github.com/omnitrace/omnitrace/backend/ingestion"
	"github.com/omnitrace/omnitrace/backend/storage"
	"github.com/omnitrace/omnitrace/internal/config"
)

func main() {
	// Load configuration
	cfg := config.LoadFromEnv()

	// Initialize storage
	spanStore := storage.NewSpanStore(cfg.Storage.MaxSpans, cfg.Storage.SpanTTL)
	metricStore := storage.NewMetricStore(cfg.Storage.MaxMetrics, cfg.Storage.MetricTTL)

	// Initialize ingestion
	processor := ingestion.NewProcessor(spanStore, metricStore)
	ingestionServer := ingestion.NewServer(processor)

	// Initialize dashboard
	// Assuming static files are in ./backend/dashboard/static
	dashboardServer := dashboard.NewServer(spanStore, metricStore, "./backend/dashboard/static")

	// Setup HTTP server
	mux := http.NewServeMux()

	// Register routes
	ingestionServer.RegisterRoutes(mux)
	dashboardServer.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         cfg.GetServerAddr(),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server
	go func() {
		log.Printf("OmniTrace server starting on %s", cfg.GetServerAddr())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")
	server.Close()
}
