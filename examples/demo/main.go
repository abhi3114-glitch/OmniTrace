package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/omnitrace/omnitrace/sdk"
)

func main() {
	collectorURL := os.Getenv("OMNITRACE_COLLECTOR_URL")
	if collectorURL == "" {
		collectorURL = "http://localhost:10001"
	}

	// Initialize Tracer
	exporter := sdk.NewExporter(sdk.ExporterConfig{
		CollectorURL:  collectorURL,
		BatchSize:     10,
		FlushInterval: 1 * time.Second,
		OnError:       func(err error) { log.Printf("Exporter error: %v", err) },
	})

	sdk.InitGlobalTracer("demo-service", sdk.WithExporter(exporter))

	tracer := sdk.GlobalTracer()
	middleware := sdk.NewMiddleware(tracer)
	httpClient := sdk.InstrumentedClient(tracer, 5*time.Second)

	// Define handlers
	http.HandleFunc("/api/process", middleware.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Simulate some work with a child span
		span, ctx := sdk.StartSpanFromContext(ctx, "processing_logic")
		time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
		span.SetTag("item_count", "42")
		span.Finish()

		// Call external service (simulated)
		callExternalService(ctx, httpClient)

		// Randomly error
		if rand.Float32() < 0.1 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Random failure"))
			return
		}

		w.Write([]byte("Processed"))
	}))

	port := 9003
	log.Printf("Demo service running on http://localhost:%d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func callExternalService(ctx context.Context, client *http.Client) {
	// In a real app this would call another service.
	// We'll just fake it with a span for now to show the concept,
	// or properly call a mock endpoint if we had one.
	// Since we don't have another service running, let's just create a span to represent it.

	span, _ := sdk.StartSpanFromContext(ctx, "external_api_call")
	defer span.Finish()

	span.SetTag("peer.service", "logistics-service")
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
}
