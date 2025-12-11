package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	Server  ServerConfig
	Storage StorageConfig
	SDK     SDKConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// StorageConfig holds storage-related configuration
type StorageConfig struct {
	SpanTTL         time.Duration
	MetricTTL       time.Duration
	MaxSpans        int
	MaxMetrics      int
	CleanupInterval time.Duration
}

// SDKConfig holds SDK-related configuration
type SDKConfig struct {
	ServiceName   string
	CollectorURL  string
	BatchSize     int
	FlushInterval time.Duration
	SampleRate    float64
	EnableTracing bool
	EnableMetrics bool
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         10001,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Storage: StorageConfig{
			SpanTTL:         24 * time.Hour,
			MetricTTL:       7 * 24 * time.Hour,
			MaxSpans:        1000000,
			MaxMetrics:      10000000,
			CleanupInterval: 5 * time.Minute,
		},
		SDK: SDKConfig{
			ServiceName:   "unknown-service",
			CollectorURL:  "http://localhost:8081",
			BatchSize:     100,
			FlushInterval: 5 * time.Second,
			SampleRate:    1.0,
			EnableTracing: true,
			EnableMetrics: true,
		},
	}
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() *Config {
	cfg := DefaultConfig()

	// Server config
	if host := os.Getenv("OMNITRACE_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if port := os.Getenv("OMNITRACE_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}

	// Storage config
	if ttl := os.Getenv("OMNITRACE_SPAN_TTL"); ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			cfg.Storage.SpanTTL = d
		}
	}
	if maxSpans := os.Getenv("OMNITRACE_MAX_SPANS"); maxSpans != "" {
		if m, err := strconv.Atoi(maxSpans); err == nil {
			cfg.Storage.MaxSpans = m
		}
	}

	// SDK config
	if service := os.Getenv("OMNITRACE_SERVICE_NAME"); service != "" {
		cfg.SDK.ServiceName = service
	}
	if url := os.Getenv("OMNITRACE_COLLECTOR_URL"); url != "" {
		cfg.SDK.CollectorURL = url
	}
	if batch := os.Getenv("OMNITRACE_BATCH_SIZE"); batch != "" {
		if b, err := strconv.Atoi(batch); err == nil {
			cfg.SDK.BatchSize = b
		}
	}
	if rate := os.Getenv("OMNITRACE_SAMPLE_RATE"); rate != "" {
		if r, err := strconv.ParseFloat(rate, 64); err == nil {
			cfg.SDK.SampleRate = r
		}
	}

	return cfg
}

// GetServerAddr returns the server address string
func (c *Config) GetServerAddr() string {
	return c.Server.Host + ":" + strconv.Itoa(c.Server.Port)
}
