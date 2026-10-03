package vision

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAnalyzeImage_BackwardCompatibility(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"content": "{\"title\": \"Mona Lisa\", \"medium\": \"Oil on Poplar\", \"description\": \"Portrait with enigmatic expression\", \"theme\": \"Renaissance\"}"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	svc := NewService(
		WithEndpoint(server.URL),
		WithHTTPClient(server.Client()),
	)

	desc, theme, err := svc.AnalyzeImage(context.Background(), []byte("image-bytes"))
	if err != nil {
		t.Fatalf("expected no error from AnalyzeImage wrapper, got: %v", err)
	}
	if desc != "Portrait with enigmatic expression" {
		t.Errorf("expected description 'Portrait with enigmatic expression', got %q", desc)
	}
	if theme != "Renaissance" {
		t.Errorf("expected theme 'Renaissance', got %q", theme)
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

func TestAnalyze_EmptyImagePayload(t *testing.T) {
	svc := NewService()

	// Empty slice
	res, err := svc.Analyze(context.Background(), []byte{})
	if err == nil {
		t.Fatal("expected error for empty byte slice, got nil")
	}
	if res != nil {
		t.Errorf("expected nil result, got %+v", res)
	}
	if !strings.Contains(err.Error(), "image payload cannot be empty") {
		t.Errorf("expected 'image payload cannot be empty' error, got %v", err)
	}

	// Nil slice
	res, err = svc.Analyze(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil byte slice, got nil")
	}
	if !strings.Contains(err.Error(), "image payload cannot be empty") {
		t.Errorf("expected 'image payload cannot be empty' error, got %v", err)
	}
}

func TestAnalyze_Success(t *testing.T) {
	var requestBody map[string]interface{}
	var requestHeader http.Header
	var requestPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		requestHeader = r.Header.Clone()
		bodyBytes, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bodyBytes, &requestBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"content": "{\"title\": \"Starry Night\", \"medium\": \"Oil on Canvas\", \"description\": \"A swirling night sky over a quiet town.\", \"theme\": \"Post-Impressionism\"}"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	svc := NewService(
		WithEndpoint(server.URL),
		WithModel("test-vision-model"),
		WithHTTPClient(server.Client()),
	)

	// Valid PNG bytes: standard 8-byte PNG signature
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00}
	res, err := svc.Analyze(context.Background(), pngBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.Title != "Starry Night" {
		t.Errorf("expected title 'Starry Night', got %q", res.Title)
	}
	if res.Medium != "Oil on Canvas" {
		t.Errorf("expected medium 'Oil on Canvas', got %q", res.Medium)
	}
	if res.Description != "A swirling night sky over a quiet town." {
		t.Errorf("expected description 'A swirling night sky over a quiet town.', got %q", res.Description)
	}
	if res.Theme != "Post-Impressionism" {
		t.Errorf("expected theme 'Post-Impressionism', got %q", res.Theme)
	}

	// Verify request headers
	if ct := requestHeader.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	// Verify request path
	if !strings.HasSuffix(requestPath, "/chat/completions") {
		t.Errorf("expected path ending in /chat/completions, got %q", requestPath)
	}

	// Verify payload structure
	if requestBody["model"] != "test-vision-model" {
		t.Errorf("expected model 'test-vision-model', got %v", requestBody["model"])
	}
	respFormat, ok := requestBody["response_format"].(map[string]interface{})
	if !ok || respFormat["type"] != "json_object" {
		t.Errorf("expected response_format type json_object, got %v", requestBody["response_format"])
	}

	messages, ok := requestBody["messages"].([]interface{})
	if !ok || len(messages) != 1 {
		t.Fatalf("expected 1 user message, got %v", requestBody["messages"])
	}
	msgMap, ok := messages[0].(map[string]interface{})
	if !ok || msgMap["role"] != "user" {
		t.Fatalf("expected user role, got %v", messages[0])
	}
	contents, ok := msgMap["content"].([]interface{})
	if !ok || len(contents) < 2 {
		t.Fatalf("expected at least 2 content parts, got %v", msgMap["content"])
	}

	// Check text part
	part0 := contents[0].(map[string]interface{})
	if part0["type"] != "text" || !strings.Contains(part0["text"].(string), "title") {
		t.Errorf("expected text prompt mentioning title, got %v", part0)
	}

	// Check image part
	part1 := contents[1].(map[string]interface{})
	if part1["type"] != "image_url" {
		t.Errorf("expected image_url type, got %v", part1)
	}
	imgURLMap := part1["image_url"].(map[string]interface{})
	dataURI := imgURLMap["url"].(string)
	if !strings.HasPrefix(dataURI, "data:image/png;base64,") {
		t.Errorf("expected data:image/png;base64, prefix, got %q", dataURI)
	}
}

func TestAnalyze_MarkdownCodeFence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"content": "\n` + "```json" + `\n{\n  \"title\": \"Water Lilies\",\n  \"medium\": \"Oil on Canvas\",\n  \"description\": \"Monet's water garden at Giverny.\",\n  \"theme\": \"Impressionism\"\n}\n` + "```" + `\n"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	svc := NewService(
		WithEndpoint(server.URL),
		WithHTTPClient(server.Client()),
	)

	res, err := svc.Analyze(context.Background(), []byte("unrecognized-image-content-type"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.Title != "Water Lilies" || res.Medium != "Oil on Canvas" || res.Description != "Monet's water garden at Giverny." || res.Theme != "Impressionism" {
		t.Errorf("unexpected parsed result: %+v", res)
	}
}

func TestAnalyze_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name        string
		jsonContent string
	}{
		{
			name:        "missing medium",
			jsonContent: `{"title": "Art", "description": "Desc", "theme": "Theme"}`,
		},
		{
			name:        "empty theme",
			jsonContent: `{"title": "Art", "medium": "Oil", "description": "Desc", "theme": ""}`,
		},
		{
			name:        "missing title",
			jsonContent: `{"medium": "Oil", "description": "Desc", "theme": "Theme"}`,
		},
		{
			name:        "whitespace only description",
			jsonContent: `{"title": "Art", "medium": "Oil", "description": "   ", "theme": "Theme"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"choices": []map[string]interface{}{
						{
							"message": map[string]string{
								"content": tt.jsonContent,
							},
						},
					},
				})
			}))
			defer server.Close()

			svc := NewService(
				WithEndpoint(server.URL),
				WithHTTPClient(server.Client()),
			)

			res, err := svc.Analyze(context.Background(), []byte("some-image-data"))
			if err == nil {
				t.Fatalf("expected error for %s, got nil result: %+v", tt.name, res)
			}
			if res != nil {
				t.Errorf("expected nil result on validation failure, got %+v", res)
			}
		})
	}
}

func TestAnalyze_TransientRetryRecovery(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`Service Unavailable`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"content": "{\"title\": \"Sunflower\", \"medium\": \"Oil\", \"description\": \"Bright flowers\", \"theme\": \"Nature\"}"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	svc := NewService(
		WithEndpoint(server.URL),
		WithHTTPClient(server.Client()),
		WithMaxRetries(3),
		WithInitialBackoff(5*time.Millisecond),
	)

	res, err := svc.Analyze(context.Background(), []byte("image-data"))
	if err != nil {
		t.Fatalf("expected successful recovery after retries, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 server attempts, got %d", attempts)
	}
	if res.Title != "Sunflower" || res.Theme != "Nature" {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestAnalyze_RetriesExhausted(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Internal Server Error`))
	}))
	defer server.Close()

	svc := NewService(
		WithEndpoint(server.URL),
		WithHTTPClient(server.Client()),
		WithMaxRetries(2),
		WithInitialBackoff(5*time.Millisecond),
	)

	res, err := svc.Analyze(context.Background(), []byte("image-data"))
	if err == nil {
		t.Fatal("expected error when retries exhausted, got nil")
	}
	if res != nil {
		t.Errorf("expected nil result, got %+v", res)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts (1 initial + 2 retries), got %d", attempts)
	}
}

func TestAnalyze_NonRetryable400(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`Bad Request: invalid prompt`))
	}))
	defer server.Close()

	svc := NewService(
		WithEndpoint(server.URL),
		WithHTTPClient(server.Client()),
		WithMaxRetries(3),
		WithInitialBackoff(5*time.Millisecond),
	)

	res, err := svc.Analyze(context.Background(), []byte("image-data"))
	if err == nil {
		t.Fatal("expected error on HTTP 400, got nil")
	}
	if res != nil {
		t.Errorf("expected nil result, got %+v", res)
	}
	if attempts != 1 {
		t.Errorf("expected exactly 1 attempt with no retries on HTTP 400, got %d", attempts)
	}
}

func TestAnalyze_ContextCancellation(t *testing.T) {
	// Case 1: Context already cancelled before call
	ctxCancelled, cancel := context.WithCancel(context.Background())
	cancel()

	svc := NewService(
		WithEndpoint("http://unused"),
		WithMaxRetries(3),
	)
	_, err := svc.Analyze(ctxCancelled, []byte("image-data"))
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	// Case 2: Cancellation during retry backoff
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelTimeout()

	svc2 := NewService(
		WithEndpoint(server.URL),
		WithHTTPClient(server.Client()),
		WithMaxRetries(5),
		WithInitialBackoff(100*time.Millisecond),
	)

	_, err2 := svc2.Analyze(ctxTimeout, []byte("image-data"))
	if err2 == nil {
		t.Fatal("expected error when context is cancelled during backoff, got nil")
	}
	if !errors.Is(err2, context.DeadlineExceeded) && !errors.Is(err2, context.Canceled) {
		t.Errorf("expected context cancellation or deadline error, got %v", err2)
	}
}

func TestAnalyze_PerAttemptTimeout(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	svc := NewService(
		WithEndpoint(server.URL),
		WithHTTPClient(server.Client()),
		WithTimeout(25*time.Millisecond),
		WithMaxRetries(1),
		WithInitialBackoff(5*time.Millisecond),
	)

	_, err := svc.Analyze(context.Background(), []byte("image-data"))
	if err == nil {
		t.Fatal("expected error due to per-attempt timeout, got nil")
	}
	if attempts < 2 {
		t.Errorf("expected at least 2 attempts (initial + retry) under timeout, got %d", attempts)
	}
}

