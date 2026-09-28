package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/brotherlogic/rose/internal/github"
	"github.com/brotherlogic/rose/internal/photos"
	"github.com/brotherlogic/rose/internal/storage"
	gallery "github.com/brotherlogic/rose/proto"
	"google.golang.org/protobuf/proto"
)

type integrationRoundTripper struct {
	targetHost string
}

func (rt *integrationRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = "http"
	cloned.URL.Host = rt.targetHost
	return http.DefaultTransport.RoundTrip(cloned)
}

func TestEndToEndSyncerIntegration(t *testing.T) {
	var downloadCalls atomic.Int32

	albumHTML := `
<!DOCTYPE html>
<html>
<head><title>Integration Test Album</title></head>
<body>
<script>
AF_initDataCallback({
  key: 'ds:1',
  hash: '2',
  data: [
    null,
    [
      ["photo-e2e-1", ["https://lh3.googleusercontent.com/pw/e2e_photo_1_base", 1920, 1080]],
      ["video-e2e-1", ["https://lh3.googleusercontent.com/pw/e2e_video_1_base", 1920, 1080], null, null, null, null, null, null, null, null, null, null, null, null, null, ["video/mp4"]],
      ["photo-e2e-2", ["https://lh3.googleusercontent.com/pw/e2e_photo_2_base=w800-h600", 800, 600]]
    ]
  ]
});
</script>
</body>
</html>`

	rawImg1 := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'P', 'E', 'G', '1'}
	rawImg2 := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'P', 'E', 'G', '2'}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/shortlink":
			http.Redirect(w, r, "https://photos.google.com/share/full-album-e2e", http.StatusFound)
		case r.URL.Path == "/share/full-album-e2e":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(albumHTML))
		case strings.Contains(r.URL.Path, "e2e_photo_1_base"):
			downloadCalls.Add(1)
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(rawImg1)
		case strings.Contains(r.URL.Path, "e2e_photo_2_base"):
			downloadCalls.Add(1)
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(rawImg2)
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	customTransport := &integrationRoundTripper{
		targetHost: mockServer.Listener.Addr().String(),
	}
	httpClient := &http.Client{
		Transport: customTransport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	photoSvc := photos.NewServiceWithClient(httpClient)
	visionSvc := &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			if string(img) == string(rawImg1) {
				return "Mountain Lake at Sunrise", "Landscape", nil
			}
			if string(img) == string(rawImg2) {
				return "Urban Skyline at Twilight", "Architecture", nil
			}
			return "Unknown scenery", "General", nil
		},
	}

	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	albumURL := "https://photos.app.goo.gl/shortlink"

	mockReporter := &mockIssueReporter{}

	// 1. Initial Synchronization Pass
	exitCode := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, mockReporter, nil)
	if exitCode != 0 {
		t.Fatalf("first Run pass failed with exit code %d", exitCode)
	}
	if mockReporter.createCalls.Load() != 0 || mockReporter.hasActiveCalls.Load() != 0 {
		t.Errorf("expected zero reporter interaction on successful first pass, got create=%d hasActive=%d",
			mockReporter.createCalls.Load(), mockReporter.hasActiveCalls.Load())
	}

	// Verify vision service calls
	if visionSvc.callCount != 2 {
		t.Errorf("expected vision service called 2 times, got %d", visionSvc.callCount)
	}

	// Verify mock server download calls
	if int(downloadCalls.Load()) != 2 {
		t.Errorf("expected 2 image downloads from mock HTTP server, got %d", downloadCalls.Load())
	}

	// 2. Verify Filesystem Artifacts
	for _, id := range []string{"photo-e2e-1", "photo-e2e-2"} {
		imgPath := filepath.Join(tempDir, id+".jpg")
		imgData, err := os.ReadFile(imgPath)
		if err != nil {
			t.Fatalf("failed to read raw image %s: %v", imgPath, err)
		}

		expectedBytes := rawImg1
		expectedDesc := "Mountain Lake at Sunrise"
		expectedTheme := "Landscape"
		if id == "photo-e2e-2" {
			expectedBytes = rawImg2
			expectedDesc = "Urban Skyline at Twilight"
			expectedTheme = "Architecture"
		}

		if string(imgData) != string(expectedBytes) {
			t.Errorf("image content mismatch for %s: got %v, want %v", id, imgData, expectedBytes)
		}

		protoPath := filepath.Join(tempDir, id+".proto.bin")
		protoBytes, err := os.ReadFile(protoPath)
		if err != nil {
			t.Fatalf("failed to read artwork proto %s: %v", protoPath, err)
		}

		var artwork gallery.Artwork
		if err := proto.Unmarshal(protoBytes, &artwork); err != nil {
			t.Fatalf("failed to unmarshal artwork proto for %s: %v", id, err)
		}

		if artwork.GetId() != id {
			t.Errorf("artwork ID mismatch: got %s, want %s", artwork.GetId(), id)
		}
		if artwork.GetDescription() != expectedDesc {
			t.Errorf("artwork Description mismatch: got %s, want %s", artwork.GetDescription(), expectedDesc)
		}
		if artwork.GetThemeId() != expectedTheme {
			t.Errorf("artwork ThemeId mismatch: got %s, want %s", artwork.GetThemeId(), expectedTheme)
		}
		if artwork.GetImagePath() != id+".jpg" {
			t.Errorf("artwork ImagePath mismatch: got %s, want %s.jpg", artwork.GetImagePath(), id)
		}
		if artwork.GetTimestamp() <= 0 {
			t.Errorf("expected positive timestamp on artwork %s", id)
		}
	}

	// 3. Verify .sync-state.json
	statePath := filepath.Join(tempDir, ".sync-state.json")
	stateBytes, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("failed to read .sync-state.json: %v", err)
	}
	var stateMap map[string]bool
	if err := json.Unmarshal(stateBytes, &stateMap); err != nil {
		t.Fatalf("failed to parse .sync-state.json: %v", err)
	}
	if !stateMap["photo-e2e-1"] || !stateMap["photo-e2e-2"] {
		t.Errorf("expected photo-e2e-1 and photo-e2e-2 in sync state, got: %+v", stateMap)
	}

	// 4. Verify Idempotency (Second Pass)
	pass2ExitCode := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, mockReporter, nil)
	if pass2ExitCode != 0 {
		t.Fatalf("second Run pass failed with exit code %d", pass2ExitCode)
	}
	if mockReporter.createCalls.Load() != 0 || mockReporter.hasActiveCalls.Load() != 0 {
		t.Errorf("expected zero reporter interaction on idempotent second pass, got create=%d hasActive=%d",
			mockReporter.createCalls.Load(), mockReporter.hasActiveCalls.Load())
	}

	// Zero redundant downloads
	if int(downloadCalls.Load()) != 2 {
		t.Errorf("expected zero redundant image downloads (still 2), but got %d", downloadCalls.Load())
	}

	// Zero redundant vision calls
	if visionSvc.callCount != 2 {
		t.Errorf("expected zero redundant vision calls (still 2), but got %d", visionSvc.callCount)
	}
}

type mockGitHubIntegrationServer struct {
	mu           sync.Mutex
	getCalls     int
	postCalls    int
	hasOpenIssue bool
	lastHeaders  http.Header
	lastPayload  map[string]interface{}
}

func TestEndToEndSyncerFailureIntegration(t *testing.T) {
	ghState := &mockGitHubIntegrationServer{}

	var downloadShouldFail atomic.Bool
	downloadShouldFail.Store(true)

	albumHTML := `
<!DOCTYPE html>
<html>
<head><title>Failure Integration Album</title></head>
<body>
<script>
AF_initDataCallback({
  key: 'ds:1',
  hash: '2',
  data: [
    null,
    [
      ["photo-fail-1", ["https://lh3.googleusercontent.com/pw/fail_photo_1_base", 1920, 1080]],
      ["photo-fail-2", ["https://lh3.googleusercontent.com/pw/fail_photo_2_base", 1920, 1080]]
    ]
  ]
});
</script>
</body>
</html>`

	rawImg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'P', 'E', 'G'}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		// Mock GitHub Issues API
		case r.URL.Path == "/repos/brotherlogic/rose/issues":
			if r.Method == http.MethodGet {
				ghState.mu.Lock()
				ghState.getCalls++
				open := ghState.hasOpenIssue
				ghState.mu.Unlock()

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				if open {
					_ = json.NewEncoder(w).Encode([]map[string]interface{}{
						{"title": "Syncer failed", "state": "open"},
					})
				} else {
					_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
				}
				return
			}
			if r.Method == http.MethodPost {
				ghState.mu.Lock()
				ghState.postCalls++
				ghState.lastHeaders = r.Header.Clone()
				var payload map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&payload)
				ghState.lastPayload = payload
				ghState.mu.Unlock()

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":     1001,
					"number": 112,
					"title":  "Syncer failed",
					"state":  "open",
				})
				return
			}
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)

		// Mock Google Photos Album & Downloads
		case r.URL.Path == "/album-failure":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(albumHTML))
		case strings.Contains(r.URL.Path, "fail_photo_"):
			if downloadShouldFail.Load() {
				http.Error(w, "internal server error during photo download", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(rawImg)
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockServer.Close()

	customTransport := &integrationRoundTripper{
		targetHost: mockServer.Listener.Addr().String(),
	}
	httpClient := &http.Client{
		Transport: customTransport,
	}

	photoSvc := photos.NewServiceWithClient(httpClient)
	visionSvc := &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			return "Dramatic failure scenery", "Nature", nil
		},
	}

	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	albumURL := "https://photos.app.goo.gl/album-failure"

	reporter := github.NewClient("test-bearer-token",
		github.WithBaseURL(mockServer.URL),
		github.WithHTTPClient(mockServer.Client()),
	)

	// --- Step 1: Catastrophic failure across all items triggers GitHub Issue POST ---
	exitCode := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, reporter, nil)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 for catastrophic failure run, got %d", exitCode)
	}

	ghState.mu.Lock()
	postCalls := ghState.postCalls
	getCalls := ghState.getCalls
	lastHeaders := ghState.lastHeaders
	lastPayload := ghState.lastPayload
	ghState.mu.Unlock()

	if getCalls != 1 {
		t.Errorf("expected 1 GET request to check active issues, got %d", getCalls)
	}
	if postCalls != 1 {
		t.Fatalf("expected 1 POST request to create failure issue, got %d", postCalls)
	}

	// Verify GitHub API Headers
	if auth := lastHeaders.Get("Authorization"); auth != "Bearer test-bearer-token" {
		t.Errorf("expected Authorization header 'Bearer test-bearer-token', got '%s'", auth)
	}
	if accept := lastHeaders.Get("Accept"); accept != "application/vnd.github+json" {
		t.Errorf("expected Accept header 'application/vnd.github+json', got '%s'", accept)
	}
	if apiVer := lastHeaders.Get("X-GitHub-Api-Version"); apiVer != "2022-11-28" {
		t.Errorf("expected X-GitHub-Api-Version header '2022-11-28', got '%s'", apiVer)
	}

	// Verify GitHub Issue Payload
	if title, ok := lastPayload["title"].(string); !ok || title != "Syncer failed" {
		t.Errorf("expected payload title 'Syncer failed', got '%v'", lastPayload["title"])
	}

	labelsRaw, _ := lastPayload["labels"].([]interface{})
	var labels []string
	for _, l := range labelsRaw {
		if s, ok := l.(string); ok {
			labels = append(labels, s)
		}
	}
	if len(labels) != 1 || labels[0] != "seraphine-bug" {
		t.Errorf("expected labels ['seraphine-bug'], got %v", labels)
	}

	assigneesRaw, _ := lastPayload["assignees"].([]interface{})
	var assignees []string
	for _, a := range assigneesRaw {
		if s, ok := a.(string); ok {
			assignees = append(assignees, s)
		}
	}
	if len(assignees) != 1 || assignees[0] != "brotherlogic-automation" {
		t.Errorf("expected assignees ['brotherlogic-automation'], got %v", assignees)
	}

	body, _ := lastPayload["body"].(string)
	for _, expectedSubstring := range []string{
		"### Syncer Failure Report",
		"**Stage**: Photo Processing - 0 Synced",
		"**Photos Fetched**: 2",
		"**Photos Attempted**: 2",
		"**Photos Successful**: 0",
		"#### Log Summary",
	} {
		if !strings.Contains(body, expectedSubstring) {
			t.Errorf("missing expected substring %q in issue body:\n%s", expectedSubstring, body)
		}
	}

	// --- Step 2: Deduplication - Second run with active issue queries GitHub but skips POST ---
	ghState.mu.Lock()
	ghState.hasOpenIssue = true
	postCallsBefore := ghState.postCalls
	getCallsBefore := ghState.getCalls
	ghState.mu.Unlock()

	exitCode2 := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, reporter, nil)
	if exitCode2 != 1 {
		t.Fatalf("expected exit code 1 for second failure run, got %d", exitCode2)
	}

	ghState.mu.Lock()
	postCallsAfter := ghState.postCalls
	getCallsAfter := ghState.getCalls
	ghState.mu.Unlock()

	if getCallsAfter != getCallsBefore+1 {
		t.Errorf("expected GET calls to increment by 1 for active issue check, got before=%d after=%d", getCallsBefore, getCallsAfter)
	}
	if postCallsAfter != postCallsBefore {
		t.Errorf("expected duplicate POST to be skipped, got postCalls before=%d after=%d", postCallsBefore, postCallsAfter)
	}

	// --- Step 3: Successful run does not interact with GitHub issues API ---
	downloadShouldFail.Store(false)
	successTempDir := t.TempDir()
	successStore := storage.NewStore(successTempDir)

	ghState.mu.Lock()
	getCallsBeforeSuccess := ghState.getCalls
	postCallsBeforeSuccess := ghState.postCalls
	ghState.mu.Unlock()

	exitCodeSuccess := Run(context.Background(), albumURL, successTempDir, photoSvc, visionSvc, successStore, reporter, nil)
	if exitCodeSuccess != 0 {
		t.Fatalf("expected exit code 0 for successful run, got %d", exitCodeSuccess)
	}

	ghState.mu.Lock()
	getCallsAfterSuccess := ghState.getCalls
	postCallsAfterSuccess := ghState.postCalls
	ghState.mu.Unlock()

	if getCallsAfterSuccess != getCallsBeforeSuccess {
		t.Errorf("expected zero GET requests during successful run, got before=%d after=%d", getCallsBeforeSuccess, getCallsAfterSuccess)
	}
	if postCallsAfterSuccess != postCallsBeforeSuccess {
		t.Errorf("expected zero POST requests during successful run, got before=%d after=%d", postCallsBeforeSuccess, postCallsAfterSuccess)
	}
}

