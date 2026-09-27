package github_test

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

	"github.com/brotherlogic/rose/internal/github"
)

func TestHasActiveFailureIssue_Found(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/repos/brotherlogic/rose/issues") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or invalid Authorization header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("missing or invalid Accept header: %s", r.Header.Get("Accept"))
		}
		if r.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
			t.Errorf("missing or invalid X-GitHub-Api-Version header: %s", r.Header.Get("X-GitHub-Api-Version"))
		}

		issues := []map[string]interface{}{
			{"title": "Some other bug", "state": "open"},
			{"title": "Syncer failed", "state": "open"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(issues)
	}))
	defer server.Close()

	client := github.NewClient("test-token",
		github.WithBaseURL(server.URL),
		github.WithRepository("brotherlogic/rose"),
		github.WithHTTPClient(server.Client()),
	)

	hasActive, err := client.HasActiveFailureIssue(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasActive {
		t.Errorf("expected true, got false")
	}
}

func TestHasActiveFailureIssue_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		issues := []map[string]interface{}{
			{"title": "Some other issue", "state": "open"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(issues)
	}))
	defer server.Close()

	client := github.NewClient("test-token",
		github.WithBaseURL(server.URL),
		github.WithRepository("brotherlogic/rose"),
		github.WithHTTPClient(server.Client()),
	)

	hasActive, err := client.HasActiveFailureIssue(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasActive {
		t.Errorf("expected false, got true")
	}
}

func TestHasActiveFailureIssue_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message": "internal server error"}`))
	}))
	defer server.Close()

	client := github.NewClient("test-token",
		github.WithBaseURL(server.URL),
		github.WithRepository("brotherlogic/rose"),
		github.WithHTTPClient(server.Client()),
	)

	hasActive, err := client.HasActiveFailureIssue(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if hasActive {
		t.Errorf("expected false on error, got true")
	}
}

func TestCreateFailureIssue_Success(t *testing.T) {
	var capturedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/repos/brotherlogic/rose/issues") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or invalid Authorization header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("missing or invalid Accept header: %s", r.Header.Get("Accept"))
		}
		if r.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
			t.Errorf("missing or invalid X-GitHub-Api-Version header: %s", r.Header.Get("X-GitHub-Api-Version"))
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if err := json.Unmarshal(bodyBytes, &capturedPayload); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": 12345, "title": "Syncer failed"}`))
	}))
	defer server.Close()

	client := github.NewClient("test-token",
		github.WithBaseURL(server.URL),
		github.WithRepository("brotherlogic/rose"),
		github.WithHTTPClient(server.Client()),
	)

	ts := time.Date(2026, 9, 27, 15, 0, 0, 0, time.UTC)
	report := github.FailureReport{
		Stage:            "Photo Ingestion",
		Timestamp:        ts,
		Error:            errors.New("connection reset by peer"),
		PhotosFetched:    10,
		PhotosAttempted:  5,
		PhotosSuccessful: 0,
		LogSummary:       "Failed while processing photo ID 42",
	}

	err := client.CreateFailureIssue(context.Background(), report)
	if err != nil {
		t.Fatalf("unexpected error creating failure issue: %v", err)
	}

	if capturedPayload["title"] != "Syncer failed" {
		t.Errorf("expected title 'Syncer failed', got %v", capturedPayload["title"])
	}

	labels, ok := capturedPayload["labels"].([]interface{})
	if !ok || len(labels) != 1 || labels[0] != "seraphine-bug" {
		t.Errorf("expected labels ['seraphine-bug'], got %v", capturedPayload["labels"])
	}

	assignees, ok := capturedPayload["assignees"].([]interface{})
	if !ok || len(assignees) != 1 || assignees[0] != "brotherlogic-automation" {
		t.Errorf("expected assignees ['brotherlogic-automation'], got %v", capturedPayload["assignees"])
	}

	bodyStr, ok := capturedPayload["body"].(string)
	if !ok {
		t.Fatalf("expected string body, got %T", capturedPayload["body"])
	}

	for _, expectedSubstring := range []string{
		"Photo Ingestion",
		"connection reset by peer",
		"2026-09-27T15:00:00Z",
		"Photos Fetched",
		"Photos Attempted",
		"Photos Successful",
		"Failed while processing photo ID 42",
	} {
		if !strings.Contains(bodyStr, expectedSubstring) {
			t.Errorf("expected body to contain %q, but body was:\n%s", expectedSubstring, bodyStr)
		}
	}
}

func TestCreateFailureIssue_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message": "Resource not accessible by integration"}`))
	}))
	defer server.Close()

	client := github.NewClient("test-token",
		github.WithBaseURL(server.URL),
		github.WithRepository("brotherlogic/rose"),
		github.WithHTTPClient(server.Client()),
	)

	report := github.FailureReport{
		Stage:            "Initialization",
		Timestamp:        time.Now().UTC(),
		Error:            errors.New("unauthorized"),
		PhotosFetched:    0,
		PhotosAttempted:  0,
		PhotosSuccessful: 0,
		LogSummary:       "Init error",
	}

	err := client.CreateFailureIssue(context.Background(), report)
	if err == nil {
		t.Fatal("expected error on 403 Forbidden, got nil")
	}
}
