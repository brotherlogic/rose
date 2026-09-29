package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
)

func TestMetrics_Registration(t *testing.T) {
	m := NewMetrics()
	if m == nil {
		t.Fatal("expected non-nil Metrics instance")
	}

	// Verify all collectors are registered by gathering metrics from the registry
	mfs, err := m.registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather registered metrics: %v", err)
	}

	expectedMetrics := map[string]bool{
		"rose_syncer_photos_discovered_total":    false,
		"rose_syncer_photos_downloaded_total":    false,
		"rose_syncer_thumbnails_generated_total": false,
		"rose_syncer_storage_bytes":              false,
		"rose_syncer_sync_duration_seconds":      false,
		"rose_syncer_sync_errors_total":          false,
		"rose_syncer_photos_stored":              false,
		"rose_syncer_thumbnails_stored":          false,
	}

	for _, mf := range mfs {
		if _, ok := expectedMetrics[mf.GetName()]; ok {
			expectedMetrics[mf.GetName()] = true
		}
	}

	for name, found := range expectedMetrics {
		if !found {
			t.Errorf("expected metric %q to be registered in registry", name)
		}
	}
}

func TestNewMetrics_Registration(t *testing.T) {
	TestMetrics_Registration(t)
}

func TestMetrics_Handler(t *testing.T) {
	m := NewMetrics()
	m.SetDiscoveredPhotos(42)
	m.IncPhotosDownloaded()
	m.IncThumbnailsGenerated()
	m.IncSyncErrors()
	m.SetSyncDuration(1500 * time.Millisecond)

	server := httptest.NewServer(m.Handler())
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("failed to GET /metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	bodyStr := string(body)
	expectedStrings := []string{
		"rose_syncer_photos_discovered_total 42",
		"rose_syncer_photos_downloaded_total 1",
		"rose_syncer_thumbnails_generated_total 1",
		"rose_syncer_sync_errors_total 1",
		"rose_syncer_sync_duration_seconds 1.5",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(bodyStr, expected) {
			t.Errorf("response body does not contain expected string %q\nBody:\n%s", expected, bodyStr)
		}
	}
}

func TestMetrics_UpdateStorageBytes(t *testing.T) {
	m := NewMetrics()

	t.Run("nonexistent directory defaults to 0", func(t *testing.T) {
		nonExistent := filepath.Join(t.TempDir(), "does_not_exist")
		m.UpdateStorageBytes(nonExistent)

		mfs, err := m.registry.Gather()
		if err != nil {
			t.Fatalf("failed to gather metrics: %v", err)
		}

		var storageMf *dto.MetricFamily
		for _, mf := range mfs {
			if mf.GetName() == "rose_syncer_storage_bytes" {
				storageMf = mf
				break
			}
		}

		if storageMf == nil {
			t.Fatal("rose_syncer_storage_bytes metric family not found")
		}

		for _, metric := range storageMf.GetMetric() {
			var labelVal string
			for _, label := range metric.GetLabel() {
				if label.GetName() == "type" {
					labelVal = label.GetValue()
				}
			}
			val := metric.GetGauge().GetValue()
			if val != 0 {
				t.Errorf("expected 0 bytes for label %q, got %f", labelVal, val)
			}
		}
	})

	t.Run("calculates sizes accurately for images and thumbnails", func(t *testing.T) {
		tempDir := t.TempDir()
		imagesDir := filepath.Join(tempDir, "images")
		thumbnailsDir := filepath.Join(tempDir, "thumbnails")

		if err := os.MkdirAll(filepath.Join(imagesDir, "sub"), 0755); err != nil {
			t.Fatalf("failed to create images dir: %v", err)
		}
		if err := os.MkdirAll(thumbnailsDir, 0755); err != nil {
			t.Fatalf("failed to create thumbnails dir: %v", err)
		}

		// Write files: 100 bytes + 200 bytes in images = 300 bytes
		file1 := filepath.Join(imagesDir, "img1.jpg")
		file2 := filepath.Join(imagesDir, "sub", "img2.jpg")
		if err := os.WriteFile(file1, make([]byte, 100), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file2, make([]byte, 200), 0644); err != nil {
			t.Fatal(err)
		}

		// Write files: 50 bytes in thumbnails
		thumbFile := filepath.Join(thumbnailsDir, "img1.webp")
		if err := os.WriteFile(thumbFile, make([]byte, 50), 0644); err != nil {
			t.Fatal(err)
		}

		m.UpdateStorageBytes(tempDir)

		mfs, err := m.registry.Gather()
		if err != nil {
			t.Fatalf("failed to gather metrics: %v", err)
		}

		var storageMf *dto.MetricFamily
		for _, mf := range mfs {
			if mf.GetName() == "rose_syncer_storage_bytes" {
				storageMf = mf
				break
			}
		}

		if storageMf == nil {
			t.Fatal("rose_syncer_storage_bytes metric family not found")
		}

		gotBytes := make(map[string]float64)
		for _, metric := range storageMf.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "type" {
					gotBytes[label.GetValue()] = metric.GetGauge().GetValue()
				}
			}
		}

		if gotBytes["images"] != 300 {
			t.Errorf("expected 300 bytes for images, got %f", gotBytes["images"])
		}
		if gotBytes["thumbnails"] != 50 {
			t.Errorf("expected 50 bytes for thumbnails, got %f", gotBytes["thumbnails"])
		}
	})
}

func TestMetrics_Helpers(t *testing.T) {
	m := NewMetrics()

	m.SetDiscoveredPhotos(10)
	m.IncPhotosDownloaded()
	m.IncPhotosDownloaded()
	m.IncThumbnailsGenerated()
	m.IncSyncErrors()
	m.IncSyncErrors()
	m.IncSyncErrors()
	m.SetSyncDuration(2500 * time.Millisecond)

	mfs, err := m.registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	values := make(map[string]float64)
	for _, mf := range mfs {
		name := mf.GetName()
		for _, metric := range mf.GetMetric() {
			if metric.GetGauge() != nil {
				values[name] = metric.GetGauge().GetValue()
			} else if metric.GetCounter() != nil {
				values[name] = metric.GetCounter().GetValue()
			}
		}
	}

	if values["rose_syncer_photos_discovered_total"] != 10 {
		t.Errorf("photos_discovered: expected 10, got %f", values["rose_syncer_photos_discovered_total"])
	}
	if values["rose_syncer_photos_downloaded_total"] != 2 {
		t.Errorf("photos_downloaded: expected 2, got %f", values["rose_syncer_photos_downloaded_total"])
	}
	if values["rose_syncer_thumbnails_generated_total"] != 1 {
		t.Errorf("thumbnails_generated: expected 1, got %f", values["rose_syncer_thumbnails_generated_total"])
	}
	if values["rose_syncer_sync_errors_total"] != 3 {
		t.Errorf("sync_errors: expected 3, got %f", values["rose_syncer_sync_errors_total"])
	}
	if values["rose_syncer_sync_duration_seconds"] != 2.5 {
		t.Errorf("sync_duration: expected 2.5, got %f", values["rose_syncer_sync_duration_seconds"])
	}
}

func TestMetrics_ScanStorage(t *testing.T) {
	t.Run("non-existent directories default to 0", func(t *testing.T) {
		m := NewMetrics()
		nonExistent := filepath.Join(t.TempDir(), "does_not_exist")
		m.ScanStorage(nonExistent)

		mfs, err := m.registry.Gather()
		if err != nil {
			t.Fatalf("failed to gather metrics: %v", err)
		}

		values := make(map[string]float64)
		for _, mf := range mfs {
			name := mf.GetName()
			for _, metric := range mf.GetMetric() {
				if metric.GetGauge() != nil {
					values[name] = metric.GetGauge().GetValue()
				}
			}
		}

		if val := values["rose_syncer_photos_stored"]; val != 0 {
			t.Errorf("expected rose_syncer_photos_stored=0, got %f", val)
		}
		if val := values["rose_syncer_thumbnails_stored"]; val != 0 {
			t.Errorf("expected rose_syncer_thumbnails_stored=0, got %f", val)
		}
	})

	t.Run("empty directories default to 0", func(t *testing.T) {
		m := NewMetrics()
		tempDir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(tempDir, "images"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(tempDir, "thumbnails"), 0755); err != nil {
			t.Fatal(err)
		}

		m.ScanStorage(tempDir)

		mfs, err := m.registry.Gather()
		if err != nil {
			t.Fatalf("failed to gather metrics: %v", err)
		}

		values := make(map[string]float64)
		for _, mf := range mfs {
			name := mf.GetName()
			for _, metric := range mf.GetMetric() {
				if metric.GetGauge() != nil {
					values[name] = metric.GetGauge().GetValue()
				}
			}
		}

		if val := values["rose_syncer_photos_stored"]; val != 0 {
			t.Errorf("expected rose_syncer_photos_stored=0, got %f", val)
		}
		if val := values["rose_syncer_thumbnails_stored"]; val != 0 {
			t.Errorf("expected rose_syncer_thumbnails_stored=0, got %f", val)
		}
	})

	t.Run("directories containing non-regular files and subdirectories are ignored", func(t *testing.T) {
		m := NewMetrics()
		tempDir := t.TempDir()
		imagesDir := filepath.Join(tempDir, "images")
		thumbnailsDir := filepath.Join(tempDir, "thumbnails")

		// Create subdirectories inside images and thumbnails
		if err := os.MkdirAll(filepath.Join(imagesDir, "nested_dir"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(thumbnailsDir, "nested_dir"), 0755); err != nil {
			t.Fatal(err)
		}

		// Create a symlink in images
		dummyTarget := filepath.Join(tempDir, "target.txt")
		if err := os.WriteFile(dummyTarget, []byte("target file content"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(dummyTarget, filepath.Join(imagesDir, "symlink.jpg")); err != nil {
			t.Logf("symlink creation failed (skipping symlink): %v", err)
		}

		m.ScanStorage(tempDir)

		mfs, err := m.registry.Gather()
		if err != nil {
			t.Fatalf("failed to gather metrics: %v", err)
		}

		values := make(map[string]float64)
		for _, mf := range mfs {
			name := mf.GetName()
			for _, metric := range mf.GetMetric() {
				if metric.GetGauge() != nil {
					values[name] = metric.GetGauge().GetValue()
				}
			}
		}

		if val := values["rose_syncer_photos_stored"]; val != 0 {
			t.Errorf("expected rose_syncer_photos_stored=0 for non-regular files and subdirectories, got %f", val)
		}
		if val := values["rose_syncer_thumbnails_stored"]; val != 0 {
			t.Errorf("expected rose_syncer_thumbnails_stored=0, got %f", val)
		}
	})

	t.Run("valid image and thumbnail files compute correct counts and byte sizes", func(t *testing.T) {
		m := NewMetrics()
		tempDir := t.TempDir()
		imagesDir := filepath.Join(tempDir, "images")
		thumbnailsDir := filepath.Join(tempDir, "thumbnails")

		if err := os.MkdirAll(imagesDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(thumbnailsDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Write 2 images: 100 bytes and 250 bytes
		if err := os.WriteFile(filepath.Join(imagesDir, "photo1.jpg"), make([]byte, 100), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(imagesDir, "photo2.jpg"), make([]byte, 250), 0644); err != nil {
			t.Fatal(err)
		}

		// Write 3 thumbnails: 40 bytes each = 120 bytes
		for _, name := range []string{"t1.webp", "t2.webp", "t3.webp"} {
			if err := os.WriteFile(filepath.Join(thumbnailsDir, name), make([]byte, 40), 0644); err != nil {
				t.Fatal(err)
			}
		}

		m.ScanStorage(tempDir)

		mfs, err := m.registry.Gather()
		if err != nil {
			t.Fatalf("failed to gather metrics: %v", err)
		}

		values := make(map[string]float64)
		storageBytes := make(map[string]float64)
		for _, mf := range mfs {
			name := mf.GetName()
			for _, metric := range mf.GetMetric() {
				if metric.GetGauge() != nil {
					if name == "rose_syncer_storage_bytes" {
						for _, label := range metric.GetLabel() {
							if label.GetName() == "type" {
								storageBytes[label.GetValue()] = metric.GetGauge().GetValue()
							}
						}
					} else {
						values[name] = metric.GetGauge().GetValue()
					}
				}
			}
		}

		if val := values["rose_syncer_photos_stored"]; val != 2 {
			t.Errorf("expected rose_syncer_photos_stored=2, got %f", val)
		}
		if val := values["rose_syncer_thumbnails_stored"]; val != 3 {
			t.Errorf("expected rose_syncer_thumbnails_stored=3, got %f", val)
		}
		if val := storageBytes["images"]; val != 350 {
			t.Errorf("expected storageBytes[images]=350, got %f", val)
		}
		if val := storageBytes["thumbnails"]; val != 120 {
			t.Errorf("expected storageBytes[thumbnails]=120, got %f", val)
		}
	})
}

func TestMetrics_IncrementalHelpers(t *testing.T) {
	m := NewMetrics()

	m.SetStoredPhotos(10)
	m.SetStoredThumbnails(5)
	m.IncStoredPhotos()
	m.IncStoredThumbnails()

	mfs, err := m.registry.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	values := make(map[string]float64)
	for _, mf := range mfs {
		name := mf.GetName()
		for _, metric := range mf.GetMetric() {
			if metric.GetGauge() != nil {
				values[name] = metric.GetGauge().GetValue()
			}
		}
	}

	if values["rose_syncer_photos_stored"] != 11 {
		t.Errorf("expected rose_syncer_photos_stored=11, got %f", values["rose_syncer_photos_stored"])
	}
	if values["rose_syncer_thumbnails_stored"] != 6 {
		t.Errorf("expected rose_syncer_thumbnails_stored=6, got %f", values["rose_syncer_thumbnails_stored"])
	}
}

