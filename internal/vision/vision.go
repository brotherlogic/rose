package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
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

type chatMessageContentPart struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *chatImageURL `json:"image_url,omitempty"`
}

type chatImageURL struct {
	URL string `json:"url"`
}

type chatMessage struct {
	Role    string                   `json:"role"`
	Content []chatMessageContentPart `json:"content"`
}

type chatCompletionRequest struct {
	Model          string            `json:"model"`
	Messages       []chatMessage     `json:"messages"`
	ResponseFormat map[string]string `json:"response_format"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func cleanJSONResponse(content string) string {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```")
		if strings.HasPrefix(trimmed, "json\n") || strings.HasPrefix(trimmed, "json\r\n") {
			trimmed = strings.TrimPrefix(trimmed, "json")
		} else if idx := strings.Index(trimmed, "\n"); idx != -1 && !strings.Contains(trimmed[:idx], "{") {
			trimmed = trimmed[idx+1:]
		}
		if idx := strings.LastIndex(trimmed, "```"); idx != -1 {
			trimmed = trimmed[:idx]
		}
	}
	return strings.TrimSpace(trimmed)
}

// Analyze performs multimodal vision analysis on the provided image payload with exponential backoff retries.
func (s *Service) Analyze(ctx context.Context, img []byte) (*AnalysisResult, error) {
	if len(img) == 0 {
		return nil, errors.New("image payload cannot be empty")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	mimeType := http.DetectContentType(img)
	if !strings.HasPrefix(mimeType, "image/") {
		mimeType = "image/jpeg"
	}

	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(img))

	textPrompt := "Analyze the artwork image. Provide a valid JSON response with keys: 'title', 'medium', 'description', and 'theme'."

	reqPayload := chatCompletionRequest{
		Model: s.cfg.Model,
		Messages: []chatMessage{
			{
				Role: "user",
				Content: []chatMessageContentPart{
					{
						Type: "text",
						Text: textPrompt,
					},
					{
						Type: "image_url",
						ImageURL: &chatImageURL{
							URL: dataURI,
						},
					},
				},
			},
		},
		ResponseFormat: map[string]string{
			"type": "json_object",
		},
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chat completion request: %w", err)
	}

	endpoint := s.cfg.Endpoint
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint = strings.TrimRight(endpoint, "/") + "/chat/completions"
	}

	var lastErr error
	for attempt := 0; attempt <= s.cfg.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		attemptCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
		req, err := http.NewRequestWithContext(attemptCtx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create http request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.cfg.HTTPClient.Do(req)
		if err != nil {
			cancel()
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			lastErr = fmt.Errorf("vision inference request failed: %w", err)
			if attempt < s.cfg.MaxRetries {
				delay := s.cfg.InitialBackoff * (1 << attempt)
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()

		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			lastErr = fmt.Errorf("failed to read vision inference response body: %w", err)
			if attempt < s.cfg.MaxRetries {
				delay := s.cfg.InitialBackoff * (1 << attempt)
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			continue
		}

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return nil, fmt.Errorf("vision inference request failed with status %d: %s", resp.StatusCode, string(respBody))
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("vision inference request failed with status %d: %s", resp.StatusCode, string(respBody))
			if attempt < s.cfg.MaxRetries {
				delay := s.cfg.InitialBackoff * (1 << attempt)
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			continue
		}

		var chatResp chatCompletionResponse
		if err := json.Unmarshal(respBody, &chatResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal chat completion response: %w", err)
		}

		if len(chatResp.Choices) == 0 {
			return nil, errors.New("no completion choices returned by model")
		}

		cleanedContent := cleanJSONResponse(chatResp.Choices[0].Message.Content)

		var result AnalysisResult
		if err := json.Unmarshal([]byte(cleanedContent), &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal analysis result json: %w", err)
		}

		if strings.TrimSpace(result.Title) == "" {
			return nil, errors.New("analysis result missing required field: title")
		}
		if strings.TrimSpace(result.Medium) == "" {
			return nil, errors.New("analysis result missing required field: medium")
		}
		if strings.TrimSpace(result.Description) == "" {
			return nil, errors.New("analysis result missing required field: description")
		}
		if strings.TrimSpace(result.Theme) == "" {
			return nil, errors.New("analysis result missing required field: theme")
		}

		return &result, nil
	}

	return nil, fmt.Errorf("vision inference retries exhausted: %w", lastErr)
}

// AnalyzeImage delegates to Analyze and returns (description, theme, nil) for backward compatibility.
func (s *Service) AnalyzeImage(ctx context.Context, img []byte) (string, string, error) {
	res, err := s.Analyze(ctx, img)
	if err != nil {
		return "", "", err
	}
	return res.Description, res.Theme, nil
}

