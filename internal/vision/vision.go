package vision

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"
)

// AnalysisResult holds structured inference curatorial metadata.
type AnalysisResult struct {
	Title       string `json:"title"`
	Medium      string `json:"medium"`
	Description string `json:"description"`
	Theme       string `json:"theme"`
}

// Config holds configuration parameters for the vision service.
type Config struct {
	Endpoint       string
	Model          string
	Timeout        time.Duration
	MaxRetries     int
	InitialBackoff time.Duration
	HTTPClient     *http.Client
}

// Option configures a vision service instance.
type Option func(*Config)

// WithEndpoint sets the vision inference API endpoint.
func WithEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.Endpoint = endpoint
	}
}

// WithModel sets the model name for inference.
func WithModel(model string) Option {
	return func(c *Config) {
		c.Model = model
	}
}

// WithTimeout sets the per-attempt HTTP timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(maxRetries int) Option {
	return func(c *Config) {
		c.MaxRetries = maxRetries
	}
}

// WithInitialBackoff sets the initial retry delay.
func WithInitialBackoff(backoff time.Duration) Option {
	return func(c *Config) {
		c.InitialBackoff = backoff
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Config) {
		c.HTTPClient = client
	}
}

// Service handles vision analysis inference.
type Service struct {
	cfg Config
}

// Config returns the configuration used by the service.
func (s *Service) Config() Config {
	return s.cfg
}

// NewServiceWithConfig creates a Service directly from a Config struct.
// If cfg.HTTPClient is nil, it defaults to http.DefaultClient.
func NewServiceWithConfig(cfg Config) *Service {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	return &Service{cfg: cfg}
}

// NewService creates a Service with defaults and environment variable overrides,
// and applies any functional options provided.
func NewService(opts ...Option) *Service {
	endpoint := os.Getenv("VISION_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:11434/v1"
	}

	model := os.Getenv("VISION_MODEL")
	if model == "" {
		model = "llama3.2-vision:90b"
	}

	timeout := 120 * time.Second
	if envTimeout := os.Getenv("VISION_TIMEOUT"); envTimeout != "" {
		if d, err := time.ParseDuration(envTimeout); err == nil {
			timeout = d
		} else if s, err := strconv.Atoi(envTimeout); err == nil {
			timeout = time.Duration(s) * time.Second
		}
	}

	maxRetries := 3
	if envRetries := os.Getenv("VISION_MAX_RETRIES"); envRetries != "" {
		if r, err := strconv.Atoi(envRetries); err == nil {
			maxRetries = r
		}
	}

	cfg := Config{
		Endpoint:       endpoint,
		Model:          model,
		Timeout:        timeout,
		MaxRetries:     maxRetries,
		InitialBackoff: 500 * time.Millisecond,
		HTTPClient:     &http.Client{},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}

	return &Service{cfg: cfg}
}

// AnalyzeImage provides mock analysis for backward compatibility.
func (s *Service) AnalyzeImage(ctx context.Context, img []byte) (string, string, error) {
	// Mock implementation
	return "A beautiful landscape", "Nature", nil
}
