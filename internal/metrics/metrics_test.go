package metrics

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gallerypb "github.com/brotherlogic/rose/proto"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/protobuf/proto"
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

func TestAnnotationMetrics_RegistrationAndRecording(t *testing.T) {
	m := NewMetrics()
	if m == nil {
		t.Fatal("expected non-nil Metrics instance")
	}

	// Verify nil-receiver safety
	var nilM *Metrics
	nilM.IncPhotosAnnotated("new")
	nilM.ObserveAnnotationDuration(time.Second)
	nilM.IncAnnotationErrors()
	nilM.RecordTheme("portrait")

	// Record test metrics
	m.IncPhotosAnnotated("new")
	m.IncPhotosAnnotated("new")
	m.IncPhotosAnnotated("backfill")
	m.ObserveAnnotationDuration(1500 * time.Millisecond)
	m.IncAnnotationErrors()
	m.RecordTheme("impressionism")
	m.RecordTheme("impressionism")
	m.RecordTheme("renaissance")

	mfs, err := m.Registry().Gather()
	if err != nil {
		t.Fatalf("failed to gather registered metrics: %v", err)
	}

	mfMap := make(map[string]*dto.MetricFamily)
	for _, mf := range mfs {
		mfMap[mf.GetName()] = mf
	}

	expectedMetrics := []string{
		"rose_syncer_photos_annotated_total",
		"rose_syncer_annotation_duration_seconds",
		"rose_syncer_annotation_errors_total",
		"rose_syncer_themes_total",
	}

	for _, name := range expectedMetrics {
		if _, ok := mfMap[name]; !ok {
			t.Errorf("expected metric %q to be registered in registry", name)
		}
	}

	// Verify rose_syncer_photos_annotated_total
	if mf, ok := mfMap["rose_syncer_photos_annotated_total"]; ok {
		counts := make(map[string]float64)
		for _, metric := range mf.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "source" {
					counts[label.GetValue()] = metric.GetCounter().GetValue()
				}
			}
		}
		if counts["new"] != 2 {
			t.Errorf("expected source=new count=2, got %f", counts["new"])
		}
		if counts["backfill"] != 1 {
			t.Errorf("expected source=backfill count=1, got %f", counts["backfill"])
		}
	}

	// Verify rose_syncer_annotation_duration_seconds
	if mf, ok := mfMap["rose_syncer_annotation_duration_seconds"]; ok {
		for _, metric := range mf.GetMetric() {
			hist := metric.GetHistogram()
			if hist.GetSampleCount() != 1 {
				t.Errorf("expected duration sample count 1, got %d", hist.GetSampleCount())
			}
			if hist.GetSampleSum() != 1.5 {
				t.Errorf("expected duration sample sum 1.5, got %f", hist.GetSampleSum())
			}
		}
	}

	// Verify rose_syncer_annotation_errors_total
	if mf, ok := mfMap["rose_syncer_annotation_errors_total"]; ok {
		for _, metric := range mf.GetMetric() {
			val := metric.GetCounter().GetValue()
			if val != 1 {
				t.Errorf("expected annotation errors=1, got %f", val)
			}
		}
	}

	// Verify rose_syncer_themes_total
	if mf, ok := mfMap["rose_syncer_themes_total"]; ok {
		themeCounts := make(map[string]float64)
		for _, metric := range mf.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "theme" {
					themeCounts[label.GetValue()] = metric.GetCounter().GetValue()
				}
			}
		}
		if themeCounts["impressionism"] != 2 {
			t.Errorf("expected impressionism count=2, got %f", themeCounts["impressionism"])
		}
		if themeCounts["renaissance"] != 1 {
			t.Errorf("expected renaissance count=1, got %f", themeCounts["renaissance"])
		}
	}
}

func TestMetrics_CurationRegistration(t *testing.T) {
	m := NewMetrics()
	if m == nil {
		t.Fatal("expected non-nil Metrics instance")
	}

	// Verify nil-receiver safety
	var nilM *Metrics
	nilM.SetArtworksAnnotated(10)
	nilM.SetArtisticMovement("cubism", 3)
	nilM.ResetArtisticMovements()

	// Initial default verification: artworksAnnotated should be registered and 0
	mfs, err := m.Registry().Gather()
	if err != nil {
		t.Fatalf("failed to gather registered metrics: %v", err)
	}

	mfMap := make(map[string]*dto.MetricFamily)
	for _, mf := range mfs {
		mfMap[mf.GetName()] = mf
	}

	if _, ok := mfMap["rose_syncer_artworks_annotated_total"]; !ok {
		t.Fatalf("expected metric rose_syncer_artworks_annotated_total to be registered in registry")
	}

	if val := mfMap["rose_syncer_artworks_annotated_total"].GetMetric()[0].GetGauge().GetValue(); val != 0 {
		t.Errorf("expected default artworksAnnotated=0, got %f", val)
	}

	// Set values for curation metrics
	m.SetArtworksAnnotated(42)
	m.SetArtisticMovement("surrealism", 12)
	m.SetArtisticMovement("impressionism", 7)

	mfs, err = m.Registry().Gather()
	if err != nil {
		t.Fatalf("failed to gather registered metrics: %v", err)
	}
	mfMap = make(map[string]*dto.MetricFamily)
	for _, mf := range mfs {
		mfMap[mf.GetName()] = mf
	}

	if _, ok := mfMap["rose_syncer_artworks_annotated_total"]; !ok {
		t.Fatalf("expected metric rose_syncer_artworks_annotated_total to be gathered")
	}
	if _, ok := mfMap["rose_syncer_artistic_movements_total"]; !ok {
		t.Fatalf("expected metric rose_syncer_artistic_movements_total to be gathered")
	}

	if val := mfMap["rose_syncer_artworks_annotated_total"].GetMetric()[0].GetGauge().GetValue(); val != 42 {
		t.Errorf("expected artworksAnnotated=42, got %f", val)
	}

	movements := make(map[string]float64)
	for _, metric := range mfMap["rose_syncer_artistic_movements_total"].GetMetric() {
		for _, label := range metric.GetLabel() {
			if label.GetName() == "theme" {
				movements[label.GetValue()] = metric.GetGauge().GetValue()
			}
		}
	}
	if movements["surrealism"] != 12 {
		t.Errorf("expected surrealism=12, got %f", movements["surrealism"])
	}
	if movements["impressionism"] != 7 {
		t.Errorf("expected impressionism=7, got %f", movements["impressionism"])
	}

	// Reset artistic movements and re-populate
	m.ResetArtisticMovements()
	mfs, err = m.Registry().Gather()
	if err != nil {
		t.Fatalf("failed to gather registered metrics after reset: %v", err)
	}
	mfMap = make(map[string]*dto.MetricFamily)
	for _, mf := range mfs {
		mfMap[mf.GetName()] = mf
	}
	if mf, ok := mfMap["rose_syncer_artistic_movements_total"]; ok && len(mf.GetMetric()) > 0 {
		t.Errorf("expected artisticMovements to be empty after reset, got %d metrics", len(mf.GetMetric()))
	}

	m.SetArtisticMovement("modernism", 5)
	mfs, err = m.Registry().Gather()
	if err != nil {
		t.Fatalf("failed to gather registered metrics after set: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() == "rose_syncer_artistic_movements_total" {
			if len(mf.GetMetric()) != 1 || mf.GetMetric()[0].GetGauge().GetValue() != 5 {
				t.Errorf("expected single metric with value 5 for modernism")
			}
		}
	}
}

func TestMetrics_Handler_CurationExposition(t *testing.T) {
	m := NewMetrics()
	m.SetArtworksAnnotated(15)
	m.SetArtisticMovement("renaissance", 8)

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
		"# TYPE rose_syncer_artworks_annotated_total gauge",
		"rose_syncer_artworks_annotated_total 15",
		"# TYPE rose_syncer_artistic_movements_total gauge",
		`rose_syncer_artistic_movements_total{theme="renaissance"} 8`,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(bodyStr, expected) {
			t.Errorf("response body does not contain expected string %q\nBody:\n%s", expected, bodyStr)
		}
	}
}

func writeTestArtworkProto(t *testing.T, dir, filename string, artwork *gallerypb.Artwork) string {
	t.Helper()
	data, err := proto.Marshal(artwork)
	if err != nil {
		t.Fatalf("failed to marshal artwork proto: %v", err)
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write artwork proto file %s: %v", path, err)
	}
	return path
}

func getGatheredGaugeValue(t *testing.T, m *Metrics, name string) (float64, bool) {
	t.Helper()
	mfs, err := m.Registry().Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() == name {
			if len(mf.GetMetric()) > 0 && mf.GetMetric()[0].GetGauge() != nil {
				return mf.GetMetric()[0].GetGauge().GetValue(), true
			}
		}
	}
	return 0, false
}

func getGatheredCounterValue(t *testing.T, m *Metrics, name string) (float64, bool) {
	t.Helper()
	mfs, err := m.Registry().Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() == name {
			if len(mf.GetMetric()) > 0 && mf.GetMetric()[0].GetCounter() != nil {
				return mf.GetMetric()[0].GetCounter().GetValue(), true
			}
		}
	}
	return 0, false
}

func getGatheredArtisticMovements(t *testing.T, m *Metrics) map[string]float64 {
	t.Helper()
	mfs, err := m.Registry().Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}
	movements := make(map[string]float64)
	for _, mf := range mfs {
		if mf.GetName() == "rose_syncer_artistic_movements_total" {
			for _, metric := range mf.GetMetric() {
				for _, label := range metric.GetLabel() {
					if label.GetName() == "theme" {
						movements[label.GetValue()] = metric.GetGauge().GetValue()
					}
				}
			}
		}
	}
	return movements
}

func TestMetrics_ScanStorage_ArtworksAndMovements(t *testing.T) {
	tempDir := t.TempDir()
	m := NewMetrics()

	writeTestArtworkProto(t, tempDir, "art1.proto.bin", &gallerypb.Artwork{
		Id:      "art1",
		ThemeId: "Impressionism",
	})
	writeTestArtworkProto(t, tempDir, "art2.proto.bin", &gallerypb.Artwork{
		Id:      "art2",
		ThemeId: "Impressionism",
	})
	writeTestArtworkProto(t, tempDir, "art3.proto.bin", &gallerypb.Artwork{
		Id:      "art3",
		ThemeId: "Renaissance",
	})

	m.ScanStorage(tempDir)

	val, found := getGatheredGaugeValue(t, m, "rose_syncer_artworks_annotated_total")
	if !found || val != 3 {
		t.Errorf("expected rose_syncer_artworks_annotated_total=3, found=%v val=%f", found, val)
	}

	movements := getGatheredArtisticMovements(t, m)
	if movements["Impressionism"] != 2 {
		t.Errorf("expected Impressionism=2, got %f", movements["Impressionism"])
	}
	if movements["Renaissance"] != 1 {
		t.Errorf("expected Renaissance=1, got %f", movements["Renaissance"])
	}
}

func TestMetrics_ScanStorage_EmptyAndWhitespaceThemes(t *testing.T) {
	tempDir := t.TempDir()
	m := NewMetrics()

	writeTestArtworkProto(t, tempDir, "art-empty.proto.bin", &gallerypb.Artwork{
		Id:      "art-empty",
		ThemeId: "",
	})
	writeTestArtworkProto(t, tempDir, "art-whitespace.proto.bin", &gallerypb.Artwork{
		Id:      "art-whitespace",
		ThemeId: "   \t\n ",
	})

	m.ScanStorage(tempDir)

	val, found := getGatheredGaugeValue(t, m, "rose_syncer_artworks_annotated_total")
	if !found || val != 2 {
		t.Errorf("expected rose_syncer_artworks_annotated_total=2, found=%v val=%f", found, val)
	}

	movements := getGatheredArtisticMovements(t, m)
	if movements["Uncategorized"] != 2 {
		t.Errorf("expected Uncategorized=2, got %f", movements["Uncategorized"])
	}
}

func TestMetrics_ScanStorage_SpecialCharacterThemes(t *testing.T) {
	tempDir := t.TempDir()
	m := NewMetrics()

	writeTestArtworkProto(t, tempDir, "art-brut.proto.bin", &gallerypb.Artwork{
		Id:      "art-brut",
		ThemeId: "L'art brut",
	})
	writeTestArtworkProto(t, tempDir, "art-avant.proto.bin", &gallerypb.Artwork{
		Id:      "art-avant",
		ThemeId: "Avant-Garde",
	})

	m.ScanStorage(tempDir)

	val, found := getGatheredGaugeValue(t, m, "rose_syncer_artworks_annotated_total")
	if !found || val != 2 {
		t.Errorf("expected rose_syncer_artworks_annotated_total=2, found=%v val=%f", found, val)
	}

	movements := getGatheredArtisticMovements(t, m)
	if movements["L'art brut"] != 1 {
		t.Errorf("expected L'art brut=1, got %f", movements["L'art brut"])
	}
	if movements["Avant-Garde"] != 1 {
		t.Errorf("expected Avant-Garde=1, got %f", movements["Avant-Garde"])
	}

	server := httptest.NewServer(m.Handler())
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("failed to GET /metrics: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	bodyStr := string(body)

	expectedStrings := []string{
		`rose_syncer_artistic_movements_total{theme="L'art brut"} 1`,
		`rose_syncer_artistic_movements_total{theme="Avant-Garde"} 1`,
	}
	for _, expected := range expectedStrings {
		if !strings.Contains(bodyStr, expected) {
			t.Errorf("expected metrics exposition to contain %q, body:\n%s", expected, bodyStr)
		}
	}
}

func TestMetrics_ScanStorage_IgnoresTemporaryFiles(t *testing.T) {
	tempDir := t.TempDir()
	m := NewMetrics()

	writeTestArtworkProto(t, tempDir, "valid.proto.bin", &gallerypb.Artwork{
		Id:      "valid",
		ThemeId: "Pop Art",
	})

	// Temporary atomic files and dotfiles
	atomicTmp := filepath.Join(tempDir, fmt.Sprintf(".%s.proto.bin.tmp-%d", "atomic", time.Now().UnixNano()))
	if err := os.WriteFile(atomicTmp, []byte("temp atomic content"), 0644); err != nil {
		t.Fatalf("failed to write atomic temp file: %v", err)
	}

	rawTmp := filepath.Join(tempDir, "image.tmp")
	if err := os.WriteFile(rawTmp, []byte("temp content"), 0644); err != nil {
		t.Fatalf("failed to write tmp file: %v", err)
	}

	dotProto := filepath.Join(tempDir, ".hidden.proto.bin")
	if err := os.WriteFile(dotProto, []byte("dotfile content"), 0644); err != nil {
		t.Fatalf("failed to write dot proto file: %v", err)
	}

	tmpSuffix := filepath.Join(tempDir, "photo.proto.bin.tmp-999")
	if err := os.WriteFile(tmpSuffix, []byte("tmp suffix content"), 0644); err != nil {
		t.Fatalf("failed to write tmp suffix file: %v", err)
	}

	m.ScanStorage(tempDir)

	val, found := getGatheredGaugeValue(t, m, "rose_syncer_artworks_annotated_total")
	if !found || val != 1 {
		t.Errorf("expected rose_syncer_artworks_annotated_total=1, found=%v val=%f", found, val)
	}

	movements := getGatheredArtisticMovements(t, m)
	if movements["Pop Art"] != 1 {
		t.Errorf("expected Pop Art=1, got %f", movements["Pop Art"])
	}
	if len(movements) != 1 {
		t.Errorf("expected exactly 1 movement, got %d", len(movements))
	}

	errCount, _ := getGatheredCounterValue(t, m, "rose_syncer_annotation_errors_total")
	if errCount != 0 {
		t.Errorf("expected rose_syncer_annotation_errors_total=0, got %f", errCount)
	}
}

func TestMetrics_ScanStorage_CorruptProtoResilience(t *testing.T) {
	tempDir := t.TempDir()
	m := NewMetrics()

	writeTestArtworkProto(t, tempDir, "valid.proto.bin", &gallerypb.Artwork{
		Id:      "valid",
		ThemeId: "Baroque",
	})

	corruptPath := filepath.Join(tempDir, "corrupt.proto.bin")
	if err := os.WriteFile(corruptPath, []byte{0xff, 0xff, 0xff, 0xff, 0x00}, 0644); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}

	m.ScanStorage(tempDir)

	val, found := getGatheredGaugeValue(t, m, "rose_syncer_artworks_annotated_total")
	if !found || val != 1 {
		t.Errorf("expected rose_syncer_artworks_annotated_total=1, found=%v val=%f", found, val)
	}

	movements := getGatheredArtisticMovements(t, m)
	if movements["Baroque"] != 1 {
		t.Errorf("expected Baroque=1, got %f", movements["Baroque"])
	}

	errCount, foundErr := getGatheredCounterValue(t, m, "rose_syncer_annotation_errors_total")
	if !foundErr || errCount != 1 {
		t.Errorf("expected rose_syncer_annotation_errors_total=1, found=%v val=%f", foundErr, errCount)
	}
}

func TestMetrics_ScanStorage_GaugeReconciliationOnDeletion(t *testing.T) {
	tempDir := t.TempDir()
	m := NewMetrics()

	file1 := writeTestArtworkProto(t, tempDir, "art1.proto.bin", &gallerypb.Artwork{
		Id:      "art1",
		ThemeId: "Impressionism",
	})
	file2 := writeTestArtworkProto(t, tempDir, "art2.proto.bin", &gallerypb.Artwork{
		Id:      "art2",
		ThemeId: "Renaissance",
	})

	m.ScanStorage(tempDir)

	val, found := getGatheredGaugeValue(t, m, "rose_syncer_artworks_annotated_total")
	if !found || val != 2 {
		t.Errorf("expected rose_syncer_artworks_annotated_total=2, found=%v val=%f", found, val)
	}

	movements := getGatheredArtisticMovements(t, m)
	if movements["Impressionism"] != 1 || movements["Renaissance"] != 1 {
		t.Errorf("expected Impressionism=1, Renaissance=1, got %v", movements)
	}

	// Delete file2 and re-scan
	_ = file1
	if err := os.Remove(file2); err != nil {
		t.Fatalf("failed to remove art2 file: %v", err)
	}

	m.ScanStorage(tempDir)

	val, found = getGatheredGaugeValue(t, m, "rose_syncer_artworks_annotated_total")
	if !found || val != 1 {
		t.Errorf("expected rose_syncer_artworks_annotated_total=1 after deletion, found=%v val=%f", found, val)
	}

	movements = getGatheredArtisticMovements(t, m)
	if movements["Impressionism"] != 1 {
		t.Errorf("expected Impressionism=1, got %f", movements["Impressionism"])
	}
	if _, ok := movements["Renaissance"]; ok {
		t.Errorf("expected Renaissance to be removed after deletion reconciliation, but still present in %v", movements)
	}
}


