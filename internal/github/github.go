package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// FailureReport holds diagnostic details for a syncer failure run.
type FailureReport struct {
	Stage            string
	Timestamp        time.Time
	Error            error
	PhotosFetched    int
	PhotosAttempted  int
	PhotosSuccessful int
	LogSummary       string
}

// IssueReporter defines the interface for querying and reporting issues.
type IssueReporter interface {
	HasActiveFailureIssue(ctx context.Context) (bool, error)
	CreateFailureIssue(ctx context.Context, report FailureReport) error
}

// Client implements IssueReporter against the GitHub REST API.
type Client struct {
	Token      string
	BaseURL    string
	Repository string
	HTTPClient *http.Client
}

// Option configures Client instances.
type Option func(*Client)

// WithBaseURL overrides the default GitHub API base URL.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.BaseURL = baseURL
	}
}

// WithRepository overrides the target GitHub repository (default: brotherlogic/rose).
func WithRepository(repo string) Option {
	return func(c *Client) {
		c.Repository = repo
	}
}

// WithHTTPClient overrides the HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.HTTPClient = httpClient
	}
}

// NewClient initializes a new GitHub client with the given token and options.
func NewClient(token string, opts ...Option) *Client {
	c := &Client{
		Token:      token,
		BaseURL:    "https://api.github.com",
		Repository: "brotherlogic/rose",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) getBaseURL() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return "https://api.github.com"
}

func (c *Client) getRepository() string {
	if c.Repository != "" {
		return strings.Trim(c.Repository, "/")
	}
	return "brotherlogic/rose"
}

func (c *Client) getHTTPClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{
		Timeout: 15 * time.Second,
	}
}

func (c *Client) setHeaders(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

// HasActiveFailureIssue checks if an open issue with title "Syncer failed" exists.
func (c *Client) HasActiveFailureIssue(ctx context.Context) (bool, error) {
	reqCtx := ctx
	cancel := func() {}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		reqCtx, cancel = context.WithTimeout(ctx, 15*time.Second)
	}
	defer cancel()

	endpoint := fmt.Sprintf("%s/repos/%s/issues?state=open&labels=seraphine-bug", c.getBaseURL(), c.getRepository())
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.getHTTPClient().Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("github api error (%d): %s", resp.StatusCode, string(body))
	}

	var issues []struct {
		Title string `json:"title"`
		State string `json:"state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return false, fmt.Errorf("failed to parse issues response: %w", err)
	}

	for _, issue := range issues {
		if issue.Title == "Syncer failed" {
			return true, nil
		}
	}

	return false, nil
}

type createIssuePayload struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Labels    []string `json:"labels"`
	Assignees []string `json:"assignees"`
}

// CreateFailureIssue creates a new issue in GitHub labeled seraphine-bug and assigned to brotherlogic-automation.
func (c *Client) CreateFailureIssue(ctx context.Context, report FailureReport) error {
	reqCtx := ctx
	cancel := func() {}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		reqCtx, cancel = context.WithTimeout(ctx, 15*time.Second)
	}
	defer cancel()

	endpoint := fmt.Sprintf("%s/repos/%s/issues", c.getBaseURL(), c.getRepository())

	errStr := "None"
	if report.Error != nil {
		errStr = report.Error.Error()
	}

	bodyContent := fmt.Sprintf("### Syncer Failure Report\n\n"+
		"- **Stage**: %s\n"+
		"- **Timestamp**: %s\n"+
		"- **Error**: %s\n"+
		"- **Photos Fetched**: %d\n"+
		"- **Photos Attempted**: %d\n"+
		"- **Photos Successful**: %d\n\n"+
		"#### Log Summary\n\n```\n%s\n```",
		report.Stage,
		report.Timestamp.UTC().Format(time.RFC3339),
		errStr,
		report.PhotosFetched,
		report.PhotosAttempted,
		report.PhotosSuccessful,
		report.LogSummary,
	)

	payload := createIssuePayload{
		Title:     "Syncer failed",
		Body:      bodyContent,
		Labels:    []string{"seraphine-bug"},
		Assignees: []string{"brotherlogic-automation"},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal issue payload: %w", err)
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.getHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github api error (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}
