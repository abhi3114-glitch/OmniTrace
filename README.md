# OmniTrace

OmniTrace is a complete observability suite comprising a Go-based SDK for application instrumentation and a backend for data ingestion, storage, and visualization. It provides distributed tracing, metrics aggregation, and a real-time dashboard.

## Features

### SDK
- **Context Propagation**: Automatic Trace ID and Span ID generation compatible with W3C Trace Context.
- **Instrumentation**: Middleware for HTTP requests, instrumented HTTP client, and async context tracking.
- **Metrics**: Support for Counters, Gauges, and Histograms.
- **Exporter**: Batched, asynchronous data export with retry logic.

### Backend
- **Ingestion API**: High-throughput HTTP endpoints for receiving spans and metrics.
- **Storage**: In-memory storage with fast indexing for traces and time-series metrics.
- **Processing**: Real-time data normalization and enrichment.

### Dashboard
- **Trace Visualization**: Waterfall view for analyzing request latency and service dependencies.
- **Metrics**: Real-time charts for request rates, error rates, and duration.
- **Service Graph**: Visual dependency mapping between services.

## Getting Started

### Prerequisites
- Go 1.21 or higher
- Git

### Installation

Clone the repository:

```bash
git clone https://github.com/abhi3114-glitch/OmniTrace.git
cd OmniTrace
```

### Running the Backend

The backend server handles data ingestion and serves the dashboard.

```bash
go build -o omnitrace.exe ./cmd/omnitrace
./omnitrace.exe
```

By default, the server listens on port 10000. You can access the dashboard at http://localhost:10000.

### Running the Demo Application

An example application is provided to demonstrate the SDK's capabilities.

```bash
go build -o demo.exe ./examples/demo
./demo.exe
```

The demo service runs on port 9003 and generates synthetic traffic to the backend.

### Configuration

Configuration is managed via environment variables.

| Variable | Description | Default |
|----------|-------------|---------|
| OMNITRACE_PORT | Server listening port | 10000 |
| OMNITRACE_HOST | Server bind address | 0.0.0.0 |
| OMNITRACE_COLLECTOR_URL | Backend URL for SDK | http://localhost:10000 |

## Architecture

OmniTrace follows a standard observability architecture:

1. **SDK**: integrated into applications to capture telemetry.
2. **Collector/Backend**: receives telemetry, processes it, and stores it.
3. **UI**: queries the backend to visualize data.

## Project Structure

- `sdk/`: Go SDK for instrumentation.
- `backend/`: Server, storage, and ingestion logic.
- `backend/dashboard/`: Web UI static files and handlers.
- `cmd/omnitrace/`: Main entry point for the backend.
- `examples/`: Demo applications.

## License

This project is open source and available under the MIT License.
