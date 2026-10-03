package vision

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestAnalyzeImage_Success(t *testing.T) {
	service := NewService()
	desc, theme, err := service.AnalyzeImage(context.Background(), []byte("fake-image"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if desc == "" || theme == "" {
		t.Errorf("expected description and theme, got empty strings")
	}
}

func TestNewService_DefaultConfig(t *testing.T) {
	t.Setenv("VISION_ENDPOINT", "")
	t.Setenv("VISION_MODEL", "")
	t.Setenv("VISION_TIMEOUT", "")
	t.Setenv("VISION_MAX_RETRIES", "")

	svc := NewService()
	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	cfg := svc.Config()
	if cfg.Endpoint != "http://localhost:11434/v1" {
		t.Errorf("expected default endpoint 'http://localhost:11434/v1', got %q", cfg.Endpoint)
	}
	if cfg.Model != "llama3.2-vision:90b" {
		t.Errorf("expected default model 'llama3.2-vision:90b', got %q", cfg.Model)
	}
	if cfg.Timeout != 120*time.Second {
		t.Errorf("expected default timeout 120s, got %v", cfg.Timeout)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("expected default max retries 3, got %d", cfg.MaxRetries)
	}
	if cfg.InitialBackoff != 500*time.Millisecond {
		t.Errorf("expected default initial backoff 500ms, got %v", cfg.InitialBackoff)
	}
	if cfg.HTTPClient == nil {
		t.Error("expected non-nil HTTPClient")
	}
}

func TestNewService_EnvOverrides(t *testing.T) {
	t.Setenv("VISION_ENDPOINT", "http://env-host:8000/v1")
	t.Setenv("VISION_MODEL", "custom-vision:11b")
	t.Setenv("VISION_TIMEOUT", "45s")
	t.Setenv("VISION_MAX_RETRIES", "5")

	svc := NewService()
	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	cfg := svc.Config()
	if cfg.Endpoint != "http://env-host:8000/v1" {
		t.Errorf("expected endpoint from env, got %q", cfg.Endpoint)
	}
	if cfg.Model != "custom-vision:11b" {
		t.Errorf("expected model from env, got %q", cfg.Model)
	}
	if cfg.Timeout != 45*time.Second {
		t.Errorf("expected timeout 45s from env, got %v", cfg.Timeout)
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("expected max retries 5 from env, got %d", cfg.MaxRetries)
	}
}

func TestNewService_FunctionalOptions(t *testing.T) {
	t.Setenv("VISION_ENDPOINT", "http://env-host:8000/v1")
	t.Setenv("VISION_MODEL", "env-model")

	customClient := &http.Client{Timeout: 10 * time.Second}
	svc := NewService(
		WithEndpoint("http://override-host:9000/v1"),
		WithModel("override-model:latest"),
		WithTimeout(30*time.Second),
		WithMaxRetries(2),
		WithInitialBackoff(1*time.Second),
		WithHTTPClient(customClient),
	)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}

	cfg := svc.Config()
	if cfg.Endpoint != "http://override-host:9000/v1" {
		t.Errorf("expected option override endpoint, got %q", cfg.Endpoint)
	}
	if cfg.Model != "override-model:latest" {
		t.Errorf("expected option override model, got %q", cfg.Model)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected option override timeout 30s, got %v", cfg.Timeout)
	}
	if cfg.MaxRetries != 2 {
		t.Errorf("expected option override max retries 2, got %d", cfg.MaxRetries)
	}
	if cfg.InitialBackoff != 1*time.Second {
		t.Errorf("expected option override backoff 1s, got %v", cfg.InitialBackoff)
	}
	if cfg.HTTPClient != customClient {
		t.Errorf("expected custom HTTP client, got %v", cfg.HTTPClient)
	}
}

func TestNewServiceWithConfig(t *testing.T) {
	// Case 1: With explicit HTTP client
	client := &http.Client{Timeout: 5 * time.Second}
	cfg := Config{
		Endpoint:       "http://direct-host:11434/v1",
		Model:          "direct-model",
		Timeout:        15 * time.Second,
		MaxRetries:     4,
		InitialBackoff: 250 * time.Millisecond,
		HTTPClient:     client,
	}

	svc := NewServiceWithConfig(cfg)
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.Config().Endpoint != cfg.Endpoint || svc.Config().Model != cfg.Model || svc.Config().HTTPClient != client {
		t.Errorf("expected injected config, got %+v", svc.Config())
	}

	// Case 2: Nil HTTP client defaults to http.DefaultClient
	cfgNoClient := Config{
		Endpoint: "http://direct-host:11434/v1",
		Model:    "direct-model",
	}
	svcDefaultClient := NewServiceWithConfig(cfgNoClient)
	if svcDefaultClient == nil {
		t.Fatal("expected non-nil service")
	}
	if svcDefaultClient.Config().HTTPClient != http.DefaultClient {
		t.Errorf("expected http.DefaultClient fallback when HTTPClient is nil, got %v", svcDefaultClient.Config().HTTPClient)
	}
}
