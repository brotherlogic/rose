package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brotherlogic/rose/internal/photos"
	"github.com/brotherlogic/rose/internal/storage"
	gallery "github.com/brotherlogic/rose/proto"
	"google.golang.org/protobuf/proto"
)

type mockPhotoService struct {
	photos          []photos.Photo
	fetchErr        error
	downloadErr     error
	downloadErrByID map[string]error
	downloadedData  []byte
	downloadCalls   []string
}

func (m *mockPhotoService) FetchPhotos(ctx context.Context, albumURL string) ([]photos.Photo, error) {
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return m.photos, nil
}

func (m *mockPhotoService) DownloadImage(ctx context.Context, downloadURL string) ([]byte, error) {
	m.downloadCalls = append(m.downloadCalls, downloadURL)
	if m.downloadErr != nil {
		return nil, m.downloadErr
	}
	for id, err := range m.downloadErrByID {
		if strings.Contains(downloadURL, id) {
			return nil, err
		}
	}
	if len(m.downloadedData) > 0 {
		return m.downloadedData, nil
	}
	return []byte("fake-image-bytes-" + downloadURL), nil
}

type mockVisionService struct {
	analyzeFunc func(ctx context.Context, img []byte) (string, string, error)
	callCount   int
	receivedImg [][]byte
}

func (m *mockVisionService) AnalyzeImage(ctx context.Context, img []byte) (string, string, error) {
	m.callCount++
	m.receivedImg = append(m.receivedImg, img)
	if m.analyzeFunc != nil {
		return m.analyzeFunc(ctx, img)
	}
	return "A beautiful sunset", "Landscape", nil
}

func TestFullSyncPass(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "photo-1", DownloadURL: "https://photos.google.com/photo-1=w0-h0"},
			{ID: "photo-2", DownloadURL: "https://photos.google.com/photo-2=w0-h0"},
		},
		downloadedData: []byte("sample-raw-bytes"),
	}
	visionSvc := &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			return "Scenic mountain view", "Nature", nil
		},
	}

	albumURL := "https://photos.app.goo.gl/samplealbum"
	exitCode := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	if visionSvc.callCount != 2 {
		t.Errorf("expected vision service called 2 times, got %d", visionSvc.callCount)
	}

	// Verify vision service received actual image bytes
	for i, img := range visionSvc.receivedImg {
		if string(img) != "sample-raw-bytes" {
			t.Errorf("expected vision call %d to receive raw image bytes, got %q", i, string(img))
		}
	}

	// Verify sync state, images written, and protos
	for _, id := range []string{"photo-1", "photo-2"} {
		processed, err := store.IsPhotoProcessed(id)
		if err != nil {
			t.Fatalf("error checking sync state for %s: %v", id, err)
		}
		if !processed {
			t.Errorf("expected %s to be marked processed", id)
		}

		// Verify raw image file
		imgPath := filepath.Join(tempDir, id+".jpg")
		imgBytes, err := os.ReadFile(imgPath)
		if err != nil {
			t.Fatalf("failed to read raw image file %s: %v", imgPath, err)
		}
		if string(imgBytes) != "sample-raw-bytes" {
			t.Errorf("expected image content 'sample-raw-bytes', got %q", string(imgBytes))
		}

		// Verify proto
		protoPath := filepath.Join(tempDir, id+".proto.bin")
		data, err := os.ReadFile(protoPath)
		if err != nil {
			t.Fatalf("failed to read proto bin file %s: %v", protoPath, err)
		}

		var artwork gallery.Artwork
		if err := proto.Unmarshal(data, &artwork); err != nil {
			t.Fatalf("failed to unmarshal proto for %s: %v", id, err)
		}

		if artwork.GetId() != id {
			t.Errorf("expected artwork id %s, got %s", id, artwork.GetId())
		}
		if artwork.GetTitle() == "" {
			t.Errorf("expected non-empty title for %s", id)
		}
		if artwork.GetDescription() != "Scenic mountain view" {
			t.Errorf("expected description 'Scenic mountain view', got %s", artwork.GetDescription())
		}
		if artwork.GetThemeId() != "Nature" {
			t.Errorf("expected theme_id 'Nature', got %s", artwork.GetThemeId())
		}
		if artwork.GetImagePath() != id+".jpg" {
			t.Errorf("expected image_path %s.jpg, got %s", id, artwork.GetImagePath())
		}
		if artwork.GetTimestamp() <= 0 {
			t.Errorf("expected positive timestamp, got %d", artwork.GetTimestamp())
		}
	}
}

func TestIdempotency(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoList := []photos.Photo{
		{ID: "photo-1", DownloadURL: "https://photos.google.com/photo-1=w0-h0"},
		{ID: "photo-2", DownloadURL: "https://photos.google.com/photo-2=w0-h0"},
	}
	photoSvc := &mockPhotoService{photos: photoList}
	visionSvc := &mockVisionService{}

	albumURL := "https://photos.app.goo.gl/samplealbum"

	// First pass
	code1 := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store)
	if code1 != 0 {
		t.Fatalf("first pass failed with exit code %d", code1)
	}
	if visionSvc.callCount != 2 {
		t.Fatalf("expected 2 vision calls on first pass, got %d", visionSvc.callCount)
	}

	// Second pass
	code2 := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store)
	if code2 != 0 {
		t.Fatalf("second pass failed with exit code %d", code2)
	}
	// Call count should remain 2 (no new photos processed)
	if visionSvc.callCount != 2 {
		t.Errorf("expected idempotency with 0 new vision calls, got %d total calls", visionSvc.callCount)
	}
}

func TestErrorContinuation(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoList := []photos.Photo{
		{ID: "photo-dl-err", DownloadURL: "https://photos.google.com/photo-dl-err=w0-h0"},
		{ID: "photo-vision-err", DownloadURL: "https://photos.google.com/photo-vision-err=w0-h0"},
		{ID: "photo-ok", DownloadURL: "https://photos.google.com/photo-ok=w0-h0"},
	}
	photoSvc := &mockPhotoService{
		photos: photoList,
		downloadErrByID: map[string]error{
			"photo-dl-err": errors.New("download timeout"),
		},
	}
	visionSvc := &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			if strings.Contains(string(img), "photo-vision-err") {
				return "", "", errors.New("vision AI transient error")
			}
			return "Ocean view", "Water", nil
		},
	}

	albumURL := "https://photos.app.goo.gl/samplealbum"
	exitCode := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 due to partial failure, got %d", exitCode)
	}

	// photo-dl-err should NOT be marked processed
	processed1, err := store.IsPhotoProcessed("photo-dl-err")
	if err != nil {
		t.Fatalf("unexpected error checking state: %v", err)
	}
	if processed1 {
		t.Errorf("download-failed photo should not be marked processed")
	}

	// photo-vision-err should NOT be marked processed
	processed2, err := store.IsPhotoProcessed("photo-vision-err")
	if err != nil {
		t.Fatalf("unexpected error checking state: %v", err)
	}
	if processed2 {
		t.Errorf("vision-failed photo should not be marked processed")
	}

	// photo-ok SHOULD be processed despite earlier errors
	processed3, err := store.IsPhotoProcessed("photo-ok")
	if err != nil {
		t.Fatalf("unexpected error checking state: %v", err)
	}
	if !processed3 {
		t.Errorf("subsequent photo should be processed despite earlier errors")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "photo-ok.proto.bin")); err != nil {
		t.Errorf("expected photo-ok.proto.bin to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "photo-ok.jpg")); err != nil {
		t.Errorf("expected photo-ok.jpg to exist: %v", err)
	}
}

func TestImmediateAbortOnRateLimit(t *testing.T) {
	t.Run("FetchPhotosRateLimited", func(t *testing.T) {
		tempDir := t.TempDir()
		store := storage.NewStore(tempDir)
		photoSvc := &mockPhotoService{
			fetchErr: photos.ErrRateLimited,
		}
		visionSvc := &mockVisionService{}

		code := Run(context.Background(), "https://photos.app.goo.gl/samplealbum", tempDir, photoSvc, visionSvc, store)
		if code != 1 {
			t.Errorf("expected exit code 1 on fetch rate limit, got %d", code)
		}
	})

	t.Run("DownloadRateLimitedAbortsImmediately", func(t *testing.T) {
		tempDir := t.TempDir()
		store := storage.NewStore(tempDir)
		photoList := []photos.Photo{
			{ID: "photo-1", DownloadURL: "https://photos.google.com/photo-1=w0-h0"},
			{ID: "photo-2", DownloadURL: "https://photos.google.com/photo-2=w0-h0"},
		}
		photoSvc := &mockPhotoService{
			photos:      photoList,
			downloadErr: photos.ErrRateLimited,
		}
		visionSvc := &mockVisionService{}

		code := Run(context.Background(), "https://photos.app.goo.gl/samplealbum", tempDir, photoSvc, visionSvc, store)
		if code != 1 {
			t.Errorf("expected exit code 1 on download rate limit abort, got %d", code)
		}

		// Verify it aborted immediately without trying remaining photos
		if len(photoSvc.downloadCalls) != 1 {
			t.Errorf("expected immediate abort on first rate limit error (1 call), got %d calls", len(photoSvc.downloadCalls))
		}
		if visionSvc.callCount != 0 {
			t.Errorf("expected 0 vision calls on immediate abort, got %d", visionSvc.callCount)
		}
	})

	t.Run("DiskWriteErrorAbortsImmediately", func(t *testing.T) {
		tempDir := t.TempDir()
		store := storage.NewStore(tempDir)
		photoList := []photos.Photo{
			{ID: "../invalid-id", DownloadURL: "https://photos.google.com/photo-1=w0-h0"},
			{ID: "photo-2", DownloadURL: "https://photos.google.com/photo-2=w0-h0"},
		}
		photoSvc := &mockPhotoService{photos: photoList}
		visionSvc := &mockVisionService{}

		code := Run(context.Background(), "https://photos.app.goo.gl/samplealbum", tempDir, photoSvc, visionSvc, store)
		if code != 1 {
			t.Errorf("expected exit code 1 on disk write error abort, got %d", code)
		}
		if visionSvc.callCount != 0 {
			t.Errorf("expected 0 vision calls after disk write error, got %d", visionSvc.callCount)
		}
		if len(photoSvc.downloadCalls) != 1 {
			t.Errorf("expected syncer to abort immediately on disk write error, but processed %d downloads", len(photoSvc.downloadCalls))
		}
	})
}

func TestExitCodes(t *testing.T) {
	albumURL := "https://photos.app.goo.gl/samplealbum"

	t.Run("EmptyPhotoList", func(t *testing.T) {
		tempDir := t.TempDir()
		store := storage.NewStore(tempDir)
		photoSvc := &mockPhotoService{photos: []photos.Photo{}}
		visionSvc := &mockVisionService{}

		code := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store)
		if code != 0 {
			t.Errorf("expected exit code 0 for empty photo list, got %d", code)
		}
	})

	t.Run("FetchPhotosError", func(t *testing.T) {
		tempDir := t.TempDir()
		store := storage.NewStore(tempDir)
		photoSvc := &mockPhotoService{fetchErr: errors.New("network failure")}
		visionSvc := &mockVisionService{}

		code := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store)
		if code != 1 {
			t.Errorf("expected exit code 1 for fetch error, got %d", code)
		}
	})

	t.Run("ContextCancelled", func(t *testing.T) {
		tempDir := t.TempDir()
		store := storage.NewStore(tempDir)
		photoSvc := &mockPhotoService{
			photos: []photos.Photo{
				{ID: "photo-1", DownloadURL: "https://photos.google.com/photo-1=w0-h0"},
				{ID: "photo-2", DownloadURL: "https://photos.google.com/photo-2=w0-h0"},
			},
		}
		visionSvc := &mockVisionService{}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel before run

		code := Run(ctx, albumURL, tempDir, photoSvc, visionSvc, store)
		if code != 1 {
			t.Errorf("expected exit code 1 for cancelled context, got %d", code)
		}
	})
}

func TestConfigParsing(t *testing.T) {
	t.Run("FlagOverridesEnv", func(t *testing.T) {
		t.Setenv("PHOTOS_ALBUM_URL", "https://photos.app.goo.gl/env-album")
		t.Setenv("STORAGE_PATH", "/env/storage")

		cfg, err := parseConfig([]string{"-album-url", "https://photos.app.goo.gl/flag-album", "-storage-path", "/flag/storage"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AlbumURL != "https://photos.app.goo.gl/flag-album" {
			t.Errorf("expected flag album url, got %s", cfg.AlbumURL)
		}
		if cfg.StoragePath != "/flag/storage" {
			t.Errorf("expected flag storage path, got %s", cfg.StoragePath)
		}
	})

	t.Run("EnvFallback", func(t *testing.T) {
		t.Setenv("PHOTOS_ALBUM_URL", "https://photos.app.goo.gl/env-album")
		t.Setenv("STORAGE_PATH", "/env/storage")

		cfg, err := parseConfig([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AlbumURL != "https://photos.app.goo.gl/env-album" {
			t.Errorf("expected env album url, got %s", cfg.AlbumURL)
		}
		if cfg.StoragePath != "/env/storage" {
			t.Errorf("expected env storage path, got %s", cfg.StoragePath)
		}
	})

	t.Run("DefaultStoragePath", func(t *testing.T) {
		os.Unsetenv("STORAGE_PATH")
		t.Setenv("PHOTOS_ALBUM_URL", "https://photos.app.goo.gl/env-album")

		cfg, err := parseConfig([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.StoragePath != "/data" {
			t.Errorf("expected default storage path /data, got %s", cfg.StoragePath)
		}
	})

	t.Run("MissingAlbumURLErrors", func(t *testing.T) {
		os.Unsetenv("PHOTOS_ALBUM_URL")
		_, err := parseConfig([]string{})
		if err == nil {
			t.Fatalf("expected error when album URL is missing, got nil")
		}
	})
}
