package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

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

	// 1. Initial Synchronization Pass
	exitCode := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, nil)
	if exitCode != 0 {
		t.Fatalf("first Run pass failed with exit code %d", exitCode)
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
	pass2ExitCode := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, nil)
	if pass2ExitCode != 0 {
		t.Fatalf("second Run pass failed with exit code %d", pass2ExitCode)
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
