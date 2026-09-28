package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brotherlogic/rose/internal/metrics"
	"github.com/brotherlogic/rose/internal/photos"
	"github.com/brotherlogic/rose/internal/storage"
)

func parsePrometheusMetrics(text string) map[string]float64 {
	result := make(map[string]float64)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			metricKey := parts[0]
			val, err := strconv.ParseFloat(parts[1], 64)
			if err == nil {
				result[metricKey] = val
			}
		}
	}
	return result
}

type blockingPhotoService struct {
	mu             sync.Mutex
	photos         []photos.Photo
	downloadErrByID map[string]error
	downloadStarted chan struct{}
	allowDownload   chan struct{}
}

func (s *blockingPhotoService) FetchPhotos(ctx context.Context, albumURL string) ([]photos.Photo, error) {
	return s.photos, nil
}

func (s *blockingPhotoService) DownloadImage(ctx context.Context, downloadURL string) ([]byte, error) {
	s.mu.Lock()
	if s.downloadStarted != nil {
		select {
		case <-s.downloadStarted:
		default:
			close(s.downloadStarted)
		}
	}
	allow := s.allowDownload
	s.mu.Unlock()

	if allow != nil {
		select {
		case <-allow:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	for id, err := range s.downloadErrByID {
		if strings.Contains(downloadURL, id) {
			return nil, err
		}
	}

	return []byte("test-image-data-for-" + downloadURL), nil
}

func TestMetricsHTTPIntegration_EndToEnd(t *testing.T) {
	m := metrics.NewMetrics()
	listener, server, err := startMetricsServer(0, m.Handler())
	if err != nil {
		t.Fatalf("failed to start metrics server: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	metricsURL := fmt.Sprintf("http://localhost:%d/metrics", port)

	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	_ = os.MkdirAll(filepath.Join(tempDir, "images"), 0755)
	_ = os.MkdirAll(filepath.Join(tempDir, "thumbnails"), 0755)
	_ = os.WriteFile(filepath.Join(tempDir, "images", "img1.jpg"), make([]byte, 512), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "thumbnails", "thumb1.webp"), make([]byte, 256), 0644)

	photoSvc := &blockingPhotoService{
		photos: []photos.Photo{
			{ID: "p1", DownloadURL: "https://photos.google.com/p1"},
			{ID: "p2", DownloadURL: "https://photos.google.com/p2"},
			{ID: "p3-fail", DownloadURL: "https://photos.google.com/p3-fail"},
		},
		downloadErrByID: map[string]error{
			"p3-fail": fmt.Errorf("simulated download error"),
		},
		downloadStarted: make(chan struct{}),
		allowDownload:   make(chan struct{}),
	}

	visionSvc := &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			return "Scenic landscape", "Nature", nil
		},
	}

	reporter := &mockIssueReporter{}

	// Run syncer in background
	runDone := make(chan int, 1)
	go func() {
		code := Run(context.Background(), "https://photos.app.goo.gl/test-album", tempDir, photoSvc, visionSvc, store, reporter, m)
		runDone <- code
	}()

	// 1. Verify scraper queries during execution
	select {
	case <-photoSvc.downloadStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for photo download to start")
	}

	respDuring, err := http.Get(metricsURL)
	if err != nil {
		t.Fatalf("failed to scrape metrics during execution: %v", err)
	}
	if respDuring.StatusCode != http.StatusOK {
		t.Errorf("expected HTTP 200 during execution scrape, got %d", respDuring.StatusCode)
	}
	bodyDuringBytes, _ := io.ReadAll(respDuring.Body)
	_ = respDuring.Body.Close()
	duringMetrics := parsePrometheusMetrics(string(bodyDuringBytes))
	if duringMetrics["rose_syncer_photos_discovered_total"] != 3 {
		t.Errorf("expected 3 photos discovered during execution, got %f", duringMetrics["rose_syncer_photos_discovered_total"])
	}

	// Unblock download
	close(photoSvc.allowDownload)

	// Wait for Run to complete
	var exitCode int
	select {
	case exitCode = <-runDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Run to complete")
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 due to 1 failed download, got %d", exitCode)
	}

	// 2. Verify scraper queries during post-sync grace period
	graceCtx, graceCancel := context.WithCancel(context.Background())
	graceDone := make(chan struct{})
	go func() {
		runGracePeriod(graceCtx, 10*time.Second)
		close(graceDone)
	}()

	respGrace, err := http.Get(metricsURL)
	if err != nil {
		t.Fatalf("failed to scrape metrics during grace period: %v", err)
	}
	if respGrace.StatusCode != http.StatusOK {
		t.Errorf("expected HTTP 200 during grace period scrape, got %d", respGrace.StatusCode)
	}
	bodyGraceBytes, _ := io.ReadAll(respGrace.Body)
	_ = respGrace.Body.Close()

	finalMetrics := parsePrometheusMetrics(string(bodyGraceBytes))

	// Verify rose_syncer_photos_discovered_total matches discovered photo count (3)
	if finalMetrics["rose_syncer_photos_discovered_total"] != 3 {
		t.Errorf("expected rose_syncer_photos_discovered_total == 3, got %f", finalMetrics["rose_syncer_photos_discovered_total"])
	}

	// Verify rose_syncer_photos_downloaded_total matches downloaded count (2)
	if finalMetrics["rose_syncer_photos_downloaded_total"] != 2 {
		t.Errorf("expected rose_syncer_photos_downloaded_total == 2, got %f", finalMetrics["rose_syncer_photos_downloaded_total"])
	}

	// Verify rose_syncer_thumbnails_generated_total is initialized and exported (0)
	if val, ok := finalMetrics["rose_syncer_thumbnails_generated_total"]; !ok || val != 0 {
		t.Errorf("expected rose_syncer_thumbnails_generated_total == 0, got val=%f ok=%v", val, ok)
	}

	// Verify rose_syncer_storage_bytes{type="images"} and rose_syncer_storage_bytes{type="thumbnails"}
	if finalMetrics[`rose_syncer_storage_bytes{type="images"}`] != 512 {
		t.Errorf("expected rose_syncer_storage_bytes{type=\"images\"} == 512, got %f", finalMetrics[`rose_syncer_storage_bytes{type="images"}`])
	}
	if finalMetrics[`rose_syncer_storage_bytes{type="thumbnails"}`] != 256 {
		t.Errorf("expected rose_syncer_storage_bytes{type=\"thumbnails\"} == 256, got %f", finalMetrics[`rose_syncer_storage_bytes{type="thumbnails"}`])
	}

	// Verify rose_syncer_sync_duration_seconds is recorded (> 0)
	if finalMetrics["rose_syncer_sync_duration_seconds"] <= 0 {
		t.Errorf("expected rose_syncer_sync_duration_seconds > 0, got %f", finalMetrics["rose_syncer_sync_duration_seconds"])
	}

	// Verify rose_syncer_sync_errors_total accurately reflects error counts (1)
	if finalMetrics["rose_syncer_sync_errors_total"] != 1 {
		t.Errorf("expected rose_syncer_sync_errors_total == 1, got %f", finalMetrics["rose_syncer_sync_errors_total"])
	}

	// Terminate grace period
	graceCancel()
	<-graceDone

	// 3. Clean server shutdown
	if err := shutdownServer(server, 2*time.Second); err != nil {
		t.Fatalf("expected clean server shutdown, got %v", err)
	}

	// Verify endpoint is no longer accessible
	client := &http.Client{Timeout: 500 * time.Millisecond}
	_, errAfterShutdown := client.Get(metricsURL)
	if errAfterShutdown == nil {
		t.Errorf("expected error connecting to metrics endpoint after shutdown, got nil")
	}
}

func TestMetricsHTTPIntegration_AllSuccessfulPass(t *testing.T) {
	m := metrics.NewMetrics()
	listener, server, err := startMetricsServer(0, m.Handler())
	if err != nil {
		t.Fatalf("failed to start metrics server: %v", err)
	}
	defer func() {
		_ = shutdownServer(server, 2*time.Second)
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	metricsURL := fmt.Sprintf("http://localhost:%d/metrics", port)

	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	imagesDir := filepath.Join(tempDir, "images")
	thumbnailsDir := filepath.Join(tempDir, "thumbnails")
	_ = os.MkdirAll(imagesDir, 0755)
	_ = os.MkdirAll(thumbnailsDir, 0755)
	_ = os.WriteFile(filepath.Join(imagesDir, "photo.jpg"), make([]byte, 1024), 0644)
	_ = os.WriteFile(filepath.Join(thumbnailsDir, "photo_thumb.webp"), make([]byte, 128), 0644)

	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "photo-ok-1", DownloadURL: "https://photos.google.com/ok-1"},
			{ID: "photo-ok-2", DownloadURL: "https://photos.google.com/ok-2"},
		},
		downloadedData: []byte("sample-image-content"),
	}

	visionSvc := &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			return "Clear Mountain Vista", "Nature", nil
		},
	}

	reporter := &mockIssueReporter{}

	exitCode := Run(context.Background(), "https://photos.app.goo.gl/ok-album", tempDir, photoSvc, visionSvc, store, reporter, m)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 for successful run, got %d", exitCode)
	}

	// Scrape metrics
	resp, err := http.Get(metricsURL)
	if err != nil {
		t.Fatalf("failed to scrape metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	parsed := parsePrometheusMetrics(string(bodyBytes))

	if parsed["rose_syncer_photos_discovered_total"] != 2 {
		t.Errorf("expected 2 discovered photos, got %f", parsed["rose_syncer_photos_discovered_total"])
	}
	if parsed["rose_syncer_photos_downloaded_total"] != 2 {
		t.Errorf("expected 2 downloaded photos, got %f", parsed["rose_syncer_photos_downloaded_total"])
	}
	if parsed["rose_syncer_thumbnails_generated_total"] != 0 {
		t.Errorf("expected 0 thumbnails generated, got %f", parsed["rose_syncer_thumbnails_generated_total"])
	}
	if parsed[`rose_syncer_storage_bytes{type="images"}`] != 1024 {
		t.Errorf("expected 1024 bytes for images, got %f", parsed[`rose_syncer_storage_bytes{type="images"}`])
	}
	if parsed[`rose_syncer_storage_bytes{type="thumbnails"}`] != 128 {
		t.Errorf("expected 128 bytes for thumbnails, got %f", parsed[`rose_syncer_storage_bytes{type="thumbnails"}`])
	}
	if parsed["rose_syncer_sync_duration_seconds"] <= 0 {
		t.Errorf("expected sync duration > 0, got %f", parsed["rose_syncer_sync_duration_seconds"])
	}
	if parsed["rose_syncer_sync_errors_total"] != 0 {
		t.Errorf("expected 0 sync errors, got %f", parsed["rose_syncer_sync_errors_total"])
	}
}

