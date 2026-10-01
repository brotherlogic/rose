package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/HugoSmits86/nativewebp"
	"github.com/brotherlogic/rose/internal/github"
	"github.com/brotherlogic/rose/internal/metrics"
	"github.com/brotherlogic/rose/internal/photos"
	"github.com/brotherlogic/rose/internal/storage"
	"github.com/brotherlogic/rose/internal/thumbnail"
	gallery "github.com/brotherlogic/rose/proto"
	dto "github.com/prometheus/client_model/go"
	"google.golang.org/protobuf/proto"
)

type mockPhotoService struct {
	photos          []photos.Photo
	fetchErr        error
	downloadErr     error
	downloadErrByID map[string]error
	downloadedData  []byte
	downloadCalls   []string
	onFetchPhotos   func()
}

func (m *mockPhotoService) FetchPhotos(ctx context.Context, albumURL string) ([]photos.Photo, error) {
	if m.onFetchPhotos != nil {
		m.onFetchPhotos()
	}
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
	return createTestJPEG(200, 200), nil
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

type mockIssueReporter struct {
	hasActiveIssue bool
	hasActiveErr   error
	createErr      error

	hasActiveCalls atomic.Int32
	createCalls    atomic.Int32
	lastReport     github.FailureReport
	lastCtx        context.Context
	lastCtxErr     error
	mu             sync.Mutex
}

func (m *mockIssueReporter) HasActiveFailureIssue(ctx context.Context) (bool, error) {
	m.hasActiveCalls.Add(1)
	m.mu.Lock()
	m.lastCtx = ctx
	m.lastCtxErr = ctx.Err()
	m.mu.Unlock()
	if m.hasActiveErr != nil {
		return false, m.hasActiveErr
	}
	return m.hasActiveIssue, nil
}

func (m *mockIssueReporter) CreateFailureIssue(ctx context.Context, report github.FailureReport) error {
	m.createCalls.Add(1)
	m.mu.Lock()
	m.lastCtx = ctx
	m.lastCtxErr = ctx.Err()
	m.lastReport = report
	m.mu.Unlock()
	return m.createErr
}

func TestFullSyncPass(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	rawJPEG := createTestJPEG(800, 600)
	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "photo-1", DownloadURL: "https://photos.google.com/photo-1=w0-h0"},
			{ID: "photo-2", DownloadURL: "https://photos.google.com/photo-2=w0-h0"},
		},
		downloadedData: rawJPEG,
	}
	visionSvc := &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			return "Scenic mountain view", "Nature", nil
		},
	}

	albumURL := "https://photos.app.goo.gl/samplealbum"
	exitCode := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, nil, nil)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	if visionSvc.callCount != 2 {
		t.Errorf("expected vision service called 2 times, got %d", visionSvc.callCount)
	}

	// Verify vision service received actual image bytes
	for i, img := range visionSvc.receivedImg {
		if !bytes.Equal(img, rawJPEG) {
			t.Errorf("expected vision call %d to receive raw image bytes", i)
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

		// Verify raw image file in images/
		imgPath := filepath.Join(tempDir, "images", id+".jpg")
		imgBytes, err := os.ReadFile(imgPath)
		if err != nil {
			t.Fatalf("failed to read raw image file %s: %v", imgPath, err)
		}
		if !bytes.Equal(imgBytes, rawJPEG) {
			t.Errorf("image content mismatch for %s", id)
		}

		// Verify thumbnail file in thumbnails/
		thumbPath := filepath.Join(tempDir, "thumbnails", id+".webp")
		if _, err := os.Stat(thumbPath); err != nil {
			t.Errorf("expected thumbnail to exist at %s: %v", thumbPath, err)
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
		if artwork.GetImagePath() != "images/"+id+".jpg" {
			t.Errorf("expected image_path images/%s.jpg, got %s", id, artwork.GetImagePath())
		}
		if artwork.GetThumbnailPath() != "thumbnails/"+id+".webp" {
			t.Errorf("expected thumbnail_path thumbnails/%s.webp, got %s", id, artwork.GetThumbnailPath())
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
	code1 := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, nil, nil)
	if code1 != 0 {
		t.Fatalf("first pass failed with exit code %d", code1)
	}
	if visionSvc.callCount != 2 {
		t.Fatalf("expected 2 vision calls on first pass, got %d", visionSvc.callCount)
	}

	// Second pass
	code2 := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, nil, nil)
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
	visionCalls := 0
	visionSvc := &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			visionCalls++
			if visionCalls == 1 {
				return "", "", errors.New("vision AI transient error")
			}
			return "Ocean view", "Water", nil
		},
	}

	albumURL := "https://photos.app.goo.gl/samplealbum"
	exitCode := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, nil, nil)
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
	if _, err := os.Stat(filepath.Join(tempDir, "images", "photo-ok.jpg")); err != nil {
		t.Errorf("expected photo-ok.jpg to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "thumbnails", "photo-ok.webp")); err != nil {
		t.Errorf("expected photo-ok.webp to exist: %v", err)
	}
}

func TestImmediateAbortOnRateLimit(t *testing.T) {
	t.Run("FetchPhotosRateLimitedGracefulExit", func(t *testing.T) {
		tempDir := t.TempDir()
		store := storage.NewStore(tempDir)
		photoSvc := &mockPhotoService{
			fetchErr: photos.ErrRateLimited,
		}
		visionSvc := &mockVisionService{}

		code := Run(context.Background(), "https://photos.app.goo.gl/samplealbum", tempDir, photoSvc, visionSvc, store, nil, nil)
		if code != 0 {
			t.Errorf("expected exit code 0 on fetch rate limit graceful exit, got %d", code)
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

		code := Run(context.Background(), "https://photos.app.goo.gl/samplealbum", tempDir, photoSvc, visionSvc, store, nil, nil)
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

		code := Run(context.Background(), "https://photos.app.goo.gl/samplealbum", tempDir, photoSvc, visionSvc, store, nil, nil)
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

		code := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, nil, nil)
		if code != 0 {
			t.Errorf("expected exit code 0 for empty photo list, got %d", code)
		}
	})

	t.Run("FetchPhotosError", func(t *testing.T) {
		tempDir := t.TempDir()
		store := storage.NewStore(tempDir)
		photoSvc := &mockPhotoService{fetchErr: errors.New("network failure")}
		visionSvc := &mockVisionService{}

		code := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, nil, nil)
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

		code := Run(ctx, albumURL, tempDir, photoSvc, visionSvc, store, nil, nil)
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

	t.Run("GitHubTokenFlagOverridesEnv", func(t *testing.T) {
		t.Setenv("PHOTOS_ALBUM_URL", "https://photos.app.goo.gl/env-album")
		t.Setenv("GITHUB_TOKEN", "env-token-xyz")

		cfg, err := parseConfig([]string{"-github-token", "flag-token-123"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.GitHubToken != "flag-token-123" {
			t.Errorf("expected GitHubToken 'flag-token-123', got %q", cfg.GitHubToken)
		}
	})

	t.Run("GitHubTokenEnvFallback", func(t *testing.T) {
		t.Setenv("PHOTOS_ALBUM_URL", "https://photos.app.goo.gl/env-album")
		t.Setenv("GITHUB_TOKEN", "env-token-xyz")

		cfg, err := parseConfig([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.GitHubToken != "env-token-xyz" {
			t.Errorf("expected GitHubToken 'env-token-xyz', got %q", cfg.GitHubToken)
		}
	})

	t.Run("GitHubTokenDefaultEmpty", func(t *testing.T) {
		t.Setenv("PHOTOS_ALBUM_URL", "https://photos.app.goo.gl/env-album")
		os.Unsetenv("GITHUB_TOKEN")

		cfg, err := parseConfig([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.GitHubToken != "" {
			t.Errorf("expected empty GitHubToken, got %q", cfg.GitHubToken)
		}
	})
}

func TestRun_AllSuccessful(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "p1", DownloadURL: "https://photos.google.com/p1"},
			{ID: "p2", DownloadURL: "https://photos.google.com/p2"},
		},
	}
	visionSvc := &mockVisionService{}
	reporter := &mockIssueReporter{}

	code := Run(context.Background(), "https://photos.app.goo.gl/test", tempDir, photoSvc, visionSvc, store, reporter, nil)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if reporter.createCalls.Load() != 0 {
		t.Errorf("expected 0 reporter create calls, got %d", reporter.createCalls.Load())
	}
	if reporter.hasActiveCalls.Load() != 0 {
		t.Errorf("expected 0 reporter hasActive calls, got %d", reporter.hasActiveCalls.Load())
	}
}

func TestRun_EmptyAlbum(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoSvc := &mockPhotoService{photos: []photos.Photo{}}
	visionSvc := &mockVisionService{}
	reporter := &mockIssueReporter{}

	code := Run(context.Background(), "https://photos.app.goo.gl/test", tempDir, photoSvc, visionSvc, store, reporter, nil)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if reporter.createCalls.Load() != 0 {
		t.Errorf("expected 0 reporter create calls, got %d", reporter.createCalls.Load())
	}
	if reporter.hasActiveCalls.Load() != 0 {
		t.Errorf("expected 0 reporter hasActive calls, got %d", reporter.hasActiveCalls.Load())
	}
}

func TestRun_AllAlreadyProcessed(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	_ = store.SaveProcessedPhoto("p1")
	_ = store.WriteImage("p1", []byte("img1"))
	_ = store.WriteThumbnail("p1", []byte("thumb1"))
	_ = store.SaveProcessedPhoto("p2")
	_ = store.WriteImage("p2", []byte("img2"))
	_ = store.WriteThumbnail("p2", []byte("thumb2"))

	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "p1", DownloadURL: "https://photos.google.com/p1"},
			{ID: "p2", DownloadURL: "https://photos.google.com/p2"},
		},
	}
	visionSvc := &mockVisionService{}
	reporter := &mockIssueReporter{}

	code := Run(context.Background(), "https://photos.app.goo.gl/test", tempDir, photoSvc, visionSvc, store, reporter, nil)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if reporter.createCalls.Load() != 0 {
		t.Errorf("expected 0 reporter create calls, got %d", reporter.createCalls.Load())
	}
	if reporter.hasActiveCalls.Load() != 0 {
		t.Errorf("expected 0 reporter hasActive calls, got %d", reporter.hasActiveCalls.Load())
	}
}

func TestRun_PartialSuccessWithErrors(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "p1", DownloadURL: "https://photos.google.com/p1"},
			{ID: "p2", DownloadURL: "https://photos.google.com/p2"},
		},
		downloadErrByID: map[string]error{
			"p2": errors.New("download transient error"),
		},
	}
	visionSvc := &mockVisionService{}
	reporter := &mockIssueReporter{}

	code := Run(context.Background(), "https://photos.app.goo.gl/test", tempDir, photoSvc, visionSvc, store, reporter, nil)
	if code != 1 {
		t.Fatalf("expected exit code 1 for partial success with errors, got %d", code)
	}
	if reporter.createCalls.Load() != 0 {
		t.Errorf("expected 0 reporter create calls for partial success, got %d", reporter.createCalls.Load())
	}
	if reporter.hasActiveCalls.Load() != 0 {
		t.Errorf("expected 0 reporter hasActive calls for partial success, got %d", reporter.hasActiveCalls.Load())
	}
}

func TestRun_ZeroSuccessFailure_FilesIssue(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "p1", DownloadURL: "https://photos.google.com/p1"},
			{ID: "p2", DownloadURL: "https://photos.google.com/p2"},
		},
		downloadErrByID: map[string]error{
			"p1": errors.New("download err 1"),
			"p2": errors.New("download err 2"),
		},
	}
	visionSvc := &mockVisionService{}
	reporter := &mockIssueReporter{}

	code := Run(context.Background(), "https://photos.app.goo.gl/test", tempDir, photoSvc, visionSvc, store, reporter, nil)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if reporter.hasActiveCalls.Load() != 1 {
		t.Errorf("expected 1 hasActive call, got %d", reporter.hasActiveCalls.Load())
	}
	if reporter.createCalls.Load() != 1 {
		t.Errorf("expected 1 create call, got %d", reporter.createCalls.Load())
	}
	if reporter.lastReport.Stage != "Photo Processing - 0 Synced" {
		t.Errorf("expected stage 'Photo Processing - 0 Synced', got %q", reporter.lastReport.Stage)
	}
	if reporter.lastReport.PhotosFetched != 2 {
		t.Errorf("expected 2 photos fetched, got %d", reporter.lastReport.PhotosFetched)
	}
	if reporter.lastReport.PhotosAttempted != 2 {
		t.Errorf("expected 2 photos attempted, got %d", reporter.lastReport.PhotosAttempted)
	}
	if reporter.lastReport.PhotosSuccessful != 0 {
		t.Errorf("expected 0 photos successful, got %d", reporter.lastReport.PhotosSuccessful)
	}
}

func TestRun_AlbumFetchFailure_FilesIssue(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoSvc := &mockPhotoService{
		fetchErr: errors.New("cannot fetch album"),
	}
	visionSvc := &mockVisionService{}
	reporter := &mockIssueReporter{}

	code := Run(context.Background(), "https://photos.app.goo.gl/test", tempDir, photoSvc, visionSvc, store, reporter, nil)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if reporter.hasActiveCalls.Load() != 1 {
		t.Errorf("expected 1 hasActive call, got %d", reporter.hasActiveCalls.Load())
	}
	if reporter.createCalls.Load() != 1 {
		t.Errorf("expected 1 create call, got %d", reporter.createCalls.Load())
	}
	if reporter.lastReport.Stage != "Album Fetch" {
		t.Errorf("expected stage 'Album Fetch', got %q", reporter.lastReport.Stage)
	}
}

func TestRun_ActiveIssueAlreadyExists_SkipsCreation(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoSvc := &mockPhotoService{
		fetchErr: errors.New("cannot fetch album"),
	}
	visionSvc := &mockVisionService{}
	reporter := &mockIssueReporter{
		hasActiveIssue: true,
	}

	code := Run(context.Background(), "https://photos.app.goo.gl/test", tempDir, photoSvc, visionSvc, store, reporter, nil)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if reporter.hasActiveCalls.Load() != 1 {
		t.Errorf("expected 1 hasActive call, got %d", reporter.hasActiveCalls.Load())
	}
	if reporter.createCalls.Load() != 0 {
		t.Errorf("expected 0 create calls due to deduplication, got %d", reporter.createCalls.Load())
	}
}

func TestRun_MissingGitHubToken_DoesNotPanic(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoSvc := &mockPhotoService{
		fetchErr: errors.New("cannot fetch album"),
	}
	visionSvc := &mockVisionService{}

	code := Run(context.Background(), "https://photos.app.goo.gl/test", tempDir, photoSvc, visionSvc, store, nil, nil)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestRun_ContextCancelled_FilesIssue(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "p1", DownloadURL: "https://photos.google.com/p1"},
		},
	}
	visionSvc := &mockVisionService{}
	reporter := &mockIssueReporter{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel context before run

	code := Run(ctx, "https://photos.app.goo.gl/test", tempDir, photoSvc, visionSvc, store, reporter, nil)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if reporter.hasActiveCalls.Load() != 1 {
		t.Errorf("expected 1 hasActive call, got %d", reporter.hasActiveCalls.Load())
	}
	if reporter.createCalls.Load() != 1 {
		t.Errorf("expected 1 create call, got %d", reporter.createCalls.Load())
	}
	if reporter.lastCtx == nil || reporter.lastCtxErr != nil {
		t.Errorf("expected decoupled context with nil error at call time, got err=%v", reporter.lastCtxErr)
	}
	if _, ok := reporter.lastCtx.Deadline(); !ok {
		t.Errorf("expected decoupled context to have a timeout deadline")
	}
}

func TestConfigParsing_MetricsPort(t *testing.T) {
	t.Run("DefaultValue", func(t *testing.T) {
		t.Setenv("PHOTOS_ALBUM_URL", "https://photos.app.goo.gl/sample")
		os.Unsetenv("METRICS_PORT")

		cfg, err := parseConfig([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.MetricsPort != 8081 {
			t.Errorf("expected default MetricsPort 8081, got %d", cfg.MetricsPort)
		}
	})

	t.Run("EnvVariable", func(t *testing.T) {
		t.Setenv("PHOTOS_ALBUM_URL", "https://photos.app.goo.gl/sample")
		t.Setenv("METRICS_PORT", "9090")

		cfg, err := parseConfig([]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.MetricsPort != 9090 {
			t.Errorf("expected MetricsPort 9090 from env, got %d", cfg.MetricsPort)
		}
	})

	t.Run("FlagOverridesEnv", func(t *testing.T) {
		t.Setenv("PHOTOS_ALBUM_URL", "https://photos.app.goo.gl/sample")
		t.Setenv("METRICS_PORT", "9090")

		cfg, err := parseConfig([]string{"-metrics-port", "9191"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.MetricsPort != 9191 {
			t.Errorf("expected MetricsPort 9191 from flag, got %d", cfg.MetricsPort)
		}
	})

	t.Run("InvalidPort_OutOfRangeLow", func(t *testing.T) {
		t.Setenv("PHOTOS_ALBUM_URL", "https://photos.app.goo.gl/sample")

		_, err := parseConfig([]string{"-metrics-port", "0"})
		if err == nil {
			t.Fatalf("expected error for port 0, got nil")
		}
	})

	t.Run("InvalidPort_OutOfRangeHigh", func(t *testing.T) {
		t.Setenv("PHOTOS_ALBUM_URL", "https://photos.app.goo.gl/sample")

		_, err := parseConfig([]string{"-metrics-port", "65536"})
		if err == nil {
			t.Fatalf("expected error for port 65536, got nil")
		}
	})

	t.Run("InvalidPort_Negative", func(t *testing.T) {
		t.Setenv("PHOTOS_ALBUM_URL", "https://photos.app.goo.gl/sample")

		_, err := parseConfig([]string{"-metrics-port", "-1"})
		if err == nil {
			t.Fatalf("expected error for negative port, got nil")
		}
	})

	t.Run("InvalidPort_NonNumericEnv", func(t *testing.T) {
		t.Setenv("PHOTOS_ALBUM_URL", "https://photos.app.goo.gl/sample")
		t.Setenv("METRICS_PORT", "not-a-port")

		_, err := parseConfig([]string{})
		if err == nil {
			t.Fatalf("expected error for non-numeric METRICS_PORT, got nil")
		}
	})
}

func TestFailFastMetricsPortBinding(t *testing.T) {
	// First bind a port locally to simulate port conflict
	occupiedListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind test listener: %v", err)
	}
	defer occupiedListener.Close()

	occupiedPort := occupiedListener.Addr().(*net.TCPAddr).Port

	m := metrics.NewMetrics()
	// Attempt to start metrics server on the occupied port
	listener, server, err := startMetricsServer(occupiedPort, m.Handler())
	if err == nil {
		if listener != nil {
			_ = listener.Close()
		}
		if server != nil {
			_ = server.Close()
		}
		t.Fatalf("expected error when port %d is already bound, got nil", occupiedPort)
	}
}

func TestStartMetricsServer_ServingAndGracefulShutdown(t *testing.T) {
	m := metrics.NewMetrics()
	listener, server, err := startMetricsServer(0, m.Handler())
	if err != nil {
		t.Fatalf("failed to start metrics server: %v", err)
	}
	defer func() {
		_ = shutdownServer(server, 2*time.Second)
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/metrics", port))
	if err != nil {
		t.Fatalf("failed to query /metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestScrapeGracePeriod_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately to test fail-fast abort of 60s grace period
	cancel()

	start := time.Now()
	runGracePeriod(ctx, 60*time.Second)
	elapsed := time.Since(start)

	if elapsed >= 2*time.Second {
		t.Errorf("grace period took %v to abort on cancelled context, expected < 2s", elapsed)
	}
}

func TestScrapeGracePeriod_CompletesOnDuration(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	runGracePeriod(ctx, 20*time.Millisecond)
	elapsed := time.Since(start)

	if elapsed < 20*time.Millisecond {
		t.Errorf("expected grace period to wait for full duration, finished in %v", elapsed)
	}
}

func gatherMetricValues(t *testing.T, m *metrics.Metrics) (map[string]float64, map[string]float64) {
	t.Helper()
	var mfs []*dto.MetricFamily
	var err error
	mfs, err = m.Registry().Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	values := make(map[string]float64)
	storageBytes := make(map[string]float64)
	for _, mf := range mfs {
		name := mf.GetName()
		for _, metric := range mf.GetMetric() {
			if name == "rose_syncer_storage_bytes" {
				for _, label := range metric.GetLabel() {
					if label.GetName() == "type" {
						storageBytes[label.GetValue()] = metric.GetGauge().GetValue()
					}
				}
			} else if metric.GetGauge() != nil {
				values[name] = metric.GetGauge().GetValue()
			} else if metric.GetCounter() != nil {
				values[name] = metric.GetCounter().GetValue()
			}
		}
	}
	return values, storageBytes
}

func TestRun_MetricsInstrumentation_AllSuccessful(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	rawJPEG := createTestJPEG(200, 200)
	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "p1", DownloadURL: "https://photos.app.goo.gl/dl1"},
			{ID: "p2", DownloadURL: "https://photos.app.goo.gl/dl2"},
		},
		downloadedData: rawJPEG,
	}
	visionSvc := &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			return "A lovely cat", "pets", nil
		},
	}

	imagesDir := filepath.Join(tempDir, "images")
	thumbnailsDir := filepath.Join(tempDir, "thumbnails")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		t.Fatalf("failed to create images dir: %v", err)
	}
	if err := os.MkdirAll(thumbnailsDir, 0755); err != nil {
		t.Fatalf("failed to create thumbnails dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imagesDir, "sample.jpg"), make([]byte, 100), 0644); err != nil {
		t.Fatalf("failed to write sample image: %v", err)
	}
	if err := os.WriteFile(filepath.Join(thumbnailsDir, "sample.webp"), make([]byte, 50), 0644); err != nil {
		t.Fatalf("failed to write sample thumbnail: %v", err)
	}

	m := metrics.NewMetrics()
	exitCode := Run(context.Background(), "https://photos.app.goo.gl/sample", tempDir, photoSvc, visionSvc, store, nil, m)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	values, storageBytes := gatherMetricValues(t, m)

	if got := values["rose_syncer_photos_discovered_total"]; got != 2 {
		t.Errorf("expected 2 discovered photos, got %f", got)
	}
	if got := values["rose_syncer_photos_downloaded_total"]; got != 2 {
		t.Errorf("expected 2 downloaded photos, got %f", got)
	}
	if got := values["rose_syncer_thumbnails_generated_total"]; got != 2 {
		t.Errorf("expected 2 thumbnails generated, got %f", got)
	}
	if got := values["rose_syncer_sync_errors_total"]; got != 0 {
		t.Errorf("expected 0 sync errors, got %f", got)
	}
	if got := values["rose_syncer_sync_duration_seconds"]; got <= 0 {
		t.Errorf("expected sync duration > 0, got %f", got)
	}
	expectedImages := 100 + 2*float64(len(rawJPEG))
	if storageBytes["images"] != expectedImages {
		t.Errorf("expected images storage bytes %f, got %f", expectedImages, storageBytes["images"])
	}
	if storageBytes["thumbnails"] <= 50 {
		t.Errorf("expected thumbnails storage bytes > 50, got %f", storageBytes["thumbnails"])
	}
}

func TestRun_MetricsInstrumentation_DownloadAndVisionErrors(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	validJPEG := createTestJPEG(200, 200)
	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "err-dl", DownloadURL: "https://photos.app.goo.gl/err-dl"},
			{ID: "err-vis", DownloadURL: "https://photos.app.goo.gl/err-vis"},
			{ID: "ok-photo", DownloadURL: "https://photos.app.goo.gl/ok-photo"},
		},
		downloadErrByID: map[string]error{
			"err-dl": errors.New("download failed"),
		},
		downloadedData: validJPEG,
	}
	visionSvc := &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			// Fail vision for photo err-vis
			// We can inspect mock's call count or store
			if len(photoSvc.downloadCalls) == 2 {
				return "", "", errors.New("vision transient error")
			}
			return "Valid photo", "general", nil
		},
	}

	m := metrics.NewMetrics()
	exitCode := Run(context.Background(), "https://photos.app.goo.gl/sample", tempDir, photoSvc, visionSvc, store, nil, m)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 due to errors, got %d", exitCode)
	}

	values, _ := gatherMetricValues(t, m)

	if got := values["rose_syncer_photos_discovered_total"]; got != 3 {
		t.Errorf("expected 3 discovered photos, got %f", got)
	}
	// err-dl fails download, err-vis downloads and fails vision, ok-photo downloads and succeeds
	if got := values["rose_syncer_photos_downloaded_total"]; got != 2 {
		t.Errorf("expected 2 downloaded photos, got %f", got)
	}
	if got := values["rose_syncer_sync_errors_total"]; got != 2 {
		t.Errorf("expected 2 sync errors (1 dl + 1 vision), got %f", got)
	}
}

func TestRun_MetricsInstrumentation_AlbumFetchError(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoSvc := &mockPhotoService{
		fetchErr: errors.New("album not found"),
	}
	visionSvc := &mockVisionService{}

	m := metrics.NewMetrics()
	exitCode := Run(context.Background(), "https://photos.app.goo.gl/sample", tempDir, photoSvc, visionSvc, store, nil, m)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}

	values, _ := gatherMetricValues(t, m)

	if got := values["rose_syncer_sync_errors_total"]; got != 1 {
		t.Errorf("expected 1 sync error on album fetch error, got %f", got)
	}
	if got := values["rose_syncer_sync_duration_seconds"]; got <= 0 {
		t.Errorf("expected sync duration > 0, got %f", got)
	}
}

func TestRun_NilMetrics_DoesNotPanic(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "p1", DownloadURL: "https://photos.app.goo.gl/dl1"},
		},
		downloadedData: createTestJPEG(100, 100),
	}
	visionSvc := &mockVisionService{}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Run panicked when m is nil: %v", r)
		}
	}()

	exitCode := Run(context.Background(), "https://photos.app.goo.gl/sample", tempDir, photoSvc, visionSvc, store, nil, nil)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

func createTestJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

func TestRun_ThumbnailGenerationAndStorageLayout(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	validJPEG := createTestJPEG(800, 600)

	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "thumb-photo-1", DownloadURL: "https://photos.google.com/p1"},
		},
		downloadedData: validJPEG,
	}
	visionSvc := &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			return "A vibrant garden", "gardens", nil
		},
	}
	m := metrics.NewMetrics()

	exitCode := Run(context.Background(), "https://photos.app.goo.gl/test-album", tempDir, photoSvc, visionSvc, store, nil, m)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	// 1. Verify storage directory initialization (images and thumbnails directories exist)
	imagesDir := filepath.Join(tempDir, "images")
	if info, err := os.Stat(imagesDir); err != nil || !info.IsDir() {
		t.Fatalf("expected images directory to exist at %s, err: %v", imagesDir, err)
	}
	thumbnailsDir := filepath.Join(tempDir, "thumbnails")
	if info, err := os.Stat(thumbnailsDir); err != nil || !info.IsDir() {
		t.Fatalf("expected thumbnails directory to exist at %s, err: %v", thumbnailsDir, err)
	}

	// 2. Verify image and thumbnail files written
	fullImgPath := filepath.Join(imagesDir, "thumb-photo-1.jpg")
	if _, err := os.Stat(fullImgPath); err != nil {
		t.Fatalf("expected full-resolution image at %s: %v", fullImgPath, err)
	}
	thumbPath := filepath.Join(thumbnailsDir, "thumb-photo-1.webp")
	thumbBytes, err := os.ReadFile(thumbPath)
	if err != nil {
		t.Fatalf("expected thumbnail at %s: %v", thumbPath, err)
	}
	if len(thumbBytes) < 12 || string(thumbBytes[0:4]) != "RIFF" || string(thumbBytes[8:12]) != "WEBP" {
		t.Errorf("thumbnail does not have valid WebP RIFF header: %q", string(thumbBytes[:12]))
	}
	thumbCfg, err := nativewebp.DecodeConfig(bytes.NewReader(thumbBytes))
	if err != nil {
		t.Fatalf("failed to decode WebP thumbnail config: %v", err)
	}
	if thumbCfg.Width > 600 || thumbCfg.Height > 600 {
		t.Errorf("expected thumbnail to respect 600x600 bounding box, got %dx%d", thumbCfg.Width, thumbCfg.Height)
	}
	if thumbCfg.Width != 600 || thumbCfg.Height != 450 {
		t.Errorf("expected 800x600 image to scale to 600x450, got %dx%d", thumbCfg.Width, thumbCfg.Height)
	}

	// 3. Verify proto contains updated ImagePath: "images/" + photo.ID + ".jpg"
	protoPath := filepath.Join(tempDir, "thumb-photo-1.proto.bin")
	protoBytes, err := os.ReadFile(protoPath)
	if err != nil {
		t.Fatalf("failed to read proto file %s: %v", protoPath, err)
	}
	var artwork gallery.Artwork
	if err := proto.Unmarshal(protoBytes, &artwork); err != nil {
		t.Fatalf("failed to unmarshal proto: %v", err)
	}
	if artwork.GetImagePath() != "images/thumb-photo-1.jpg" {
		t.Errorf("expected ImagePath 'images/thumb-photo-1.jpg', got %q", artwork.GetImagePath())
	}

	// 4. Verify metrics: m.IncThumbnailsGenerated() was called
	values, _ := gatherMetricValues(t, m)
	if values["rose_syncer_thumbnails_generated_total"] != 1 {
		t.Errorf("expected 1 thumbnail generated metric, got %f", values["rose_syncer_thumbnails_generated_total"])
	}
}

func TestRun_ThumbnailGenerationFailure_Resilience(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	validJPEG := createTestJPEG(800, 600)

	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "corrupted-photo", DownloadURL: "https://photos.google.com/corrupted-photo"},
			{ID: "valid-photo", DownloadURL: "https://photos.google.com/valid-photo"},
		},
	}
	// Return corrupted non-image bytes for corrupted-photo and valid JPEG for valid-photo
	photoSvc.downloadErrByID = nil
	visionSvc := &mockVisionService{}

	// Custom download implementation in mock
	callIdx := 0
	_ = callIdx
	m := metrics.NewMetrics()

	// Use custom PhotoService for differing responses
	photoSvcWithCorrupted := &customPhotoService{
		photos: photoSvc.photos,
		dataMap: map[string][]byte{
			"corrupted-photo": []byte("corrupted-not-an-image-data"),
			"valid-photo":     validJPEG,
		},
	}

	exitCode := Run(context.Background(), "https://photos.app.goo.gl/test-album", tempDir, photoSvcWithCorrupted, visionSvc, store, nil, m)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 due to thumbnail generation error on corrupted photo, got %d", exitCode)
	}

	// corrupted-photo must NOT be marked processed in .sync-state.json
	processed1, err := store.IsPhotoProcessed("corrupted-photo")
	if err != nil {
		t.Fatalf("error checking processed state: %v", err)
	}
	if processed1 {
		t.Errorf("expected corrupted-photo to NOT be marked processed in sync state")
	}

	// valid-photo MUST be marked processed in .sync-state.json
	processed2, err := store.IsPhotoProcessed("valid-photo")
	if err != nil {
		t.Fatalf("error checking processed state: %v", err)
	}
	if !processed2 {
		t.Errorf("expected valid-photo to be marked processed in sync state")
	}

	// Verify valid-photo thumbnail was written and corrupted-photo thumbnail was NOT written
	validThumbPath := filepath.Join(tempDir, "thumbnails", "valid-photo.webp")
	validThumbBytes, err := os.ReadFile(validThumbPath)
	if err != nil {
		t.Fatalf("expected thumbnail to exist for valid-photo: %v", err)
	}
	validThumbCfg, err := nativewebp.DecodeConfig(bytes.NewReader(validThumbBytes))
	if err != nil {
		t.Fatalf("failed to decode valid-photo WebP thumbnail: %v", err)
	}
	if validThumbCfg.Width != 600 || validThumbCfg.Height != 450 {
		t.Errorf("expected valid-photo thumbnail dimensions 600x450, got %dx%d", validThumbCfg.Width, validThumbCfg.Height)
	}

	corruptedThumbPath := filepath.Join(tempDir, "thumbnails", "corrupted-photo.webp")
	if _, err := os.Stat(corruptedThumbPath); !os.IsNotExist(err) {
		t.Errorf("expected no thumbnail for corrupted-photo, got err: %v", err)
	}

	// Metrics check
	values, _ := gatherMetricValues(t, m)
	if values["rose_syncer_sync_errors_total"] < 1 {
		t.Errorf("expected at least 1 sync error for thumbnail failure, got %f", values["rose_syncer_sync_errors_total"])
	}
	if values["rose_syncer_thumbnails_generated_total"] != 1 {
		t.Errorf("expected 1 thumbnail generated for valid-photo, got %f", values["rose_syncer_thumbnails_generated_total"])
	}
}

type customPhotoService struct {
	photos  []photos.Photo
	dataMap map[string][]byte
}

func (c *customPhotoService) FetchPhotos(ctx context.Context, albumURL string) ([]photos.Photo, error) {
	return c.photos, nil
}

func (c *customPhotoService) DownloadImage(ctx context.Context, downloadURL string) ([]byte, error) {
	for id, data := range c.dataMap {
		if strings.Contains(downloadURL, id) {
			return data, nil
		}
	}
	return []byte("dummy"), nil
}

func TestRun_LargeAlbumPaginationIntegration(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	rawJPEG := createTestJPEG(10, 10)

	const totalPhotos = 1200
	const preProcessed = 200

	photoList := make([]photos.Photo, totalPhotos)
	for i := 0; i < totalPhotos; i++ {
		id := fmt.Sprintf("large-photo-%04d", i)
		photoList[i] = photos.Photo{
			ID:          id,
			DownloadURL: fmt.Sprintf("https://photos.google.com/album/%s=w0-h0", id),
		}
	}

	// Seed 200 photo IDs as already processed in .sync-state.json and on disk
	for i := 0; i < preProcessed; i++ {
		if err := store.SaveProcessedPhoto(photoList[i].ID); err != nil {
			t.Fatalf("failed to seed processed photo %s: %v", photoList[i].ID, err)
		}
		if err := store.WriteImage(photoList[i].ID, rawJPEG); err != nil {
			t.Fatalf("failed to seed image %s: %v", photoList[i].ID, err)
		}
		if err := store.WriteThumbnail(photoList[i].ID, []byte("RIFFxxxxWEBP")); err != nil {
			t.Fatalf("failed to seed thumbnail %s: %v", photoList[i].ID, err)
		}
	}

	photoSvc := &mockPhotoService{
		photos:         photoList,
		downloadedData: rawJPEG,
	}
	visionSvc := &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			return "Large album landscape photo", "Nature", nil
		},
	}
	reporter := &mockIssueReporter{}
	m := metrics.NewMetrics()

	albumURL := "https://photos.app.goo.gl/large-album-1200"
	exitCode := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, reporter, m)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	// Assert no failure issues created
	if reporter.createCalls.Load() != 0 {
		t.Errorf("expected 0 failure issues created, got %d", reporter.createCalls.Load())
	}

	// Assert m.SetDiscoveredPhotos(1200) is recorded (verify gauge value via metrics exposition)
	values, _ := gatherMetricValues(t, m)
	if got := values["rose_syncer_photos_discovered_total"]; got != float64(totalPhotos) {
		t.Errorf("expected rose_syncer_photos_discovered_total == %d, got %f", totalPhotos, got)
	}

	// Assert the 200 already-processed photos are skipped and the remaining 1,000 photos are ingested
	expectedIngested := totalPhotos - preProcessed
	if len(photoSvc.downloadCalls) != expectedIngested {
		t.Errorf("expected %d download calls, got %d", expectedIngested, len(photoSvc.downloadCalls))
	}
	if visionSvc.callCount != expectedIngested {
		t.Errorf("expected %d vision service calls, got %d", expectedIngested, visionSvc.callCount)
	}

	if got := values["rose_syncer_photos_downloaded_total"]; got != float64(expectedIngested) {
		t.Errorf("expected %d photos downloaded metric, got %f", expectedIngested, got)
	}
	if got := values["rose_syncer_thumbnails_generated_total"]; got != float64(expectedIngested) {
		t.Errorf("expected %d thumbnails generated metric, got %f", expectedIngested, got)
	}
	if got := values["rose_syncer_sync_errors_total"]; got != 0 {
		t.Errorf("expected 0 sync errors, got %f", got)
	}

	// Verify all 1,200 photos are now marked as processed
	for _, p := range photoList {
		processed, err := store.IsPhotoProcessed(p.ID)
		if err != nil {
			t.Fatalf("error checking processed state for %s: %v", p.ID, err)
		}
		if !processed {
			t.Fatalf("expected photo %s to be marked processed", p.ID)
		}
	}
}

func TestRun_LargeAlbumRateLimitGracefulExit(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoSvc := &mockPhotoService{
		fetchErr: photos.ErrRateLimited,
	}
	visionSvc := &mockVisionService{}
	reporter := &mockIssueReporter{}
	m := metrics.NewMetrics()

	albumURL := "https://photos.app.goo.gl/large-album-ratelimited"
	exitCode := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, reporter, m)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 on rate limit graceful exit, got %d", exitCode)
	}

	// Assert no failure issues created
	if reporter.createCalls.Load() != 0 {
		t.Errorf("expected 0 CreateFailureIssue calls, got %d", reporter.createCalls.Load())
	}
}

func TestRun_ThumbnailPath_Serialization(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	validJPEG := createTestJPEG(400, 300)

	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "photo-thumb-test", DownloadURL: "https://photos.google.com/p1"},
		},
		downloadedData: validJPEG,
	}
	visionSvc := &mockVisionService{}

	exitCode := Run(context.Background(), "https://photos.app.goo.gl/test", tempDir, photoSvc, visionSvc, store, nil, nil)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	protoPath := filepath.Join(tempDir, "photo-thumb-test.proto.bin")
	protoBytes, err := os.ReadFile(protoPath)
	if err != nil {
		t.Fatalf("failed to read proto file %s: %v", protoPath, err)
	}
	var artwork gallery.Artwork
	if err := proto.Unmarshal(protoBytes, &artwork); err != nil {
		t.Fatalf("failed to unmarshal proto: %v", err)
	}

	if artwork.GetThumbnailPath() != "thumbnails/photo-thumb-test.webp" {
		t.Errorf("expected ThumbnailPath 'thumbnails/photo-thumb-test.webp', got %q", artwork.GetThumbnailPath())
	}
}

func TestRun_ThumbnailPath_Fallback(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)

	t.Run("NilStoreFallback", func(t *testing.T) {
		artwork := buildArtwork("photo-nil-store", "A description", "A theme", nil)
		if artwork.GetThumbnailPath() != "" {
			t.Errorf("expected empty ThumbnailPath when store is nil, got %q", artwork.GetThumbnailPath())
		}
	})

	t.Run("MissingThumbnailFileFallback", func(t *testing.T) {
		artwork := buildArtwork("missing-thumb-photo", "A description", "A theme", store)
		if artwork.GetThumbnailPath() != "" {
			t.Errorf("expected empty ThumbnailPath when thumbnail file does not exist, got %q", artwork.GetThumbnailPath())
		}
	})

	t.Run("DirectoryInsteadOfFileFallback", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "thumbnails", "dir-photo.webp")
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		artwork := buildArtwork("dir-photo", "A description", "A theme", store)
		if artwork.GetThumbnailPath() != "" {
			t.Errorf("expected empty ThumbnailPath when target is a directory, got %q", artwork.GetThumbnailPath())
		}
	})

	t.Run("ExistingThumbnailFilePopulates", func(t *testing.T) {
		thumbnailsDir := filepath.Join(tempDir, "thumbnails")
		if err := os.MkdirAll(thumbnailsDir, 0755); err != nil {
			t.Fatalf("failed to create thumbnails dir: %v", err)
		}
		thumbFile := filepath.Join(thumbnailsDir, "existing-photo.webp")
		if err := os.WriteFile(thumbFile, []byte("fake-webp"), 0644); err != nil {
			t.Fatalf("failed to create fake thumbnail: %v", err)
		}

		artwork := buildArtwork("existing-photo", "A description", "A theme", store)
		if artwork.GetThumbnailPath() != "thumbnails/existing-photo.webp" {
			t.Errorf("expected 'thumbnails/existing-photo.webp', got %q", artwork.GetThumbnailPath())
		}
	})
}

func TestRun_BaselineStorageMetricsInitialization(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)

	imagesDir := filepath.Join(tempDir, "images")
	thumbnailsDir := filepath.Join(tempDir, "thumbnails")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(thumbnailsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create pre-existing files on disk
	if err := os.WriteFile(filepath.Join(imagesDir, "init1.jpg"), make([]byte, 100), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imagesDir, "init2.jpg"), make([]byte, 200), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(thumbnailsDir, "init1.webp"), make([]byte, 50), 0644); err != nil {
		t.Fatal(err)
	}

	m := metrics.NewMetrics()
	var baselinePhotosStored, baselineThumbnailsStored float64
	var baselineImagesBytes, baselineThumbnailsBytes float64
	var hookCalled bool

	photoSvc := &mockPhotoService{
		photos: []photos.Photo{},
		onFetchPhotos: func() {
			hookCalled = true
			values, storageBytes := gatherMetricValues(t, m)
			baselinePhotosStored = values["rose_syncer_photos_stored"]
			baselineThumbnailsStored = values["rose_syncer_thumbnails_stored"]
			baselineImagesBytes = storageBytes["images"]
			baselineThumbnailsBytes = storageBytes["thumbnails"]
		},
	}
	visionSvc := &mockVisionService{}

	exitCode := Run(context.Background(), "https://photos.app.goo.gl/baseline", tempDir, photoSvc, visionSvc, store, nil, m)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !hookCalled {
		t.Fatal("expected onFetchPhotos hook to be called during Run")
	}

	if baselinePhotosStored != 2 {
		t.Errorf("expected baseline photos_stored=2 before FetchPhotos, got %f", baselinePhotosStored)
	}
	if baselineThumbnailsStored != 1 {
		t.Errorf("expected baseline thumbnails_stored=1 before FetchPhotos, got %f", baselineThumbnailsStored)
	}
	if baselineImagesBytes != 300 {
		t.Errorf("expected baseline storage_bytes[images]=300 before FetchPhotos, got %f", baselineImagesBytes)
	}
	if baselineThumbnailsBytes != 50 {
		t.Errorf("expected baseline storage_bytes[thumbnails]=50 before FetchPhotos, got %f", baselineThumbnailsBytes)
	}
}

func TestRun_IngestionLoopStoredMetricsAndReconciliation(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)

	imagesDir := filepath.Join(tempDir, "images")
	thumbnailsDir := filepath.Join(tempDir, "thumbnails")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(thumbnailsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 1 pre-existing photo and thumbnail
	if err := os.WriteFile(filepath.Join(imagesDir, "existing.jpg"), make([]byte, 100), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(thumbnailsDir, "existing.webp"), make([]byte, 50), 0644); err != nil {
		t.Fatal(err)
	}

	rawJPEG := createTestJPEG(200, 200)
	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: "new-photo-1", DownloadURL: "https://photos.app.goo.gl/new1"},
		},
		downloadedData: rawJPEG,
	}
	visionSvc := &mockVisionService{}

	m := metrics.NewMetrics()
	exitCode := Run(context.Background(), "https://photos.app.goo.gl/ingestion", tempDir, photoSvc, visionSvc, store, nil, m)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	values, storageBytes := gatherMetricValues(t, m)
	if got := values["rose_syncer_photos_stored"]; got != 2 {
		t.Errorf("expected photos_stored=2 after ingestion, got %f", got)
	}
	if got := values["rose_syncer_thumbnails_stored"]; got != 2 {
		t.Errorf("expected thumbnails_stored=2 after ingestion, got %f", got)
	}
	if got := storageBytes["images"]; got != 100+float64(len(rawJPEG)) {
		t.Errorf("expected storage_bytes[images]=%f, got %f", 100+float64(len(rawJPEG)), got)
	}
	if got := storageBytes["thumbnails"]; got <= 50 {
		t.Errorf("expected storage_bytes[thumbnails] > 50, got %f", got)
	}
}

func TestRun_DeferredReconciliationOnEarlyExitOrError(t *testing.T) {
	t.Run("ReconciliationOnFetchError", func(t *testing.T) {
		tempDir := t.TempDir()
		store := storage.NewStore(tempDir)
		imagesDir := filepath.Join(tempDir, "images")
		thumbnailsDir := filepath.Join(tempDir, "thumbnails")
		if err := os.MkdirAll(imagesDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(thumbnailsDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Initial files
		if err := os.WriteFile(filepath.Join(imagesDir, "init1.jpg"), make([]byte, 100), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(thumbnailsDir, "init1.webp"), make([]byte, 50), 0644); err != nil {
			t.Fatal(err)
		}

		m := metrics.NewMetrics()
		photoSvc := &mockPhotoService{
			onFetchPhotos: func() {
				// Simulate out-of-band file written or altered before fetch failure occurs
				_ = os.WriteFile(filepath.Join(imagesDir, "out-of-band.jpg"), make([]byte, 250), 0644)
				_ = os.WriteFile(filepath.Join(thumbnailsDir, "out-of-band.webp"), make([]byte, 75), 0644)
			},
			fetchErr: errors.New("simulated fetch error"),
		}
		visionSvc := &mockVisionService{}

		exitCode := Run(context.Background(), "https://photos.app.goo.gl/fail", tempDir, photoSvc, visionSvc, store, nil, m)
		if exitCode != 1 {
			t.Fatalf("expected exit code 1 on fetch error, got %d", exitCode)
		}

		// Verify deferred ScanStorage ran and reconciled disk metrics despite early exit
		values, storageBytes := gatherMetricValues(t, m)
		if got := values["rose_syncer_photos_stored"]; got != 2 {
			t.Errorf("expected reconciled photos_stored=2, got %f", got)
		}
		if got := values["rose_syncer_thumbnails_stored"]; got != 2 {
			t.Errorf("expected reconciled thumbnails_stored=2, got %f", got)
		}
		if got := storageBytes["images"]; got != 350 {
			t.Errorf("expected reconciled storage_bytes[images]=350, got %f", got)
		}
		if got := storageBytes["thumbnails"]; got != 125 {
			t.Errorf("expected reconciled storage_bytes[thumbnails]=125, got %f", got)
		}
	})

	t.Run("ReconciliationOnDownloadRateLimitEarlyExit", func(t *testing.T) {
		tempDir := t.TempDir()
		store := storage.NewStore(tempDir)
		imagesDir := filepath.Join(tempDir, "images")
		thumbnailsDir := filepath.Join(tempDir, "thumbnails")
		if err := os.MkdirAll(imagesDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(thumbnailsDir, 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(imagesDir, "base.jpg"), make([]byte, 120), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(thumbnailsDir, "base.webp"), make([]byte, 60), 0644); err != nil {
			t.Fatal(err)
		}

		m := metrics.NewMetrics()
		photoSvc := &mockPhotoService{
			photos: []photos.Photo{
				{ID: "rl-photo", DownloadURL: "https://photos.app.goo.gl/rl"},
			},
			downloadErr: photos.ErrRateLimited,
		}
		visionSvc := &mockVisionService{}

		exitCode := Run(context.Background(), "https://photos.app.goo.gl/rl-album", tempDir, photoSvc, visionSvc, store, nil, m)
		if exitCode != 1 {
			t.Fatalf("expected exit code 1 on download rate limit abort, got %d", exitCode)
		}

		values, storageBytes := gatherMetricValues(t, m)
		if got := values["rose_syncer_photos_stored"]; got != 1 {
			t.Errorf("expected photos_stored=1, got %f", got)
		}
		if got := values["rose_syncer_thumbnails_stored"]; got != 1 {
			t.Errorf("expected thumbnails_stored=1, got %f", got)
		}
		if got := storageBytes["images"]; got != 120 {
			t.Errorf("expected storage_bytes[images]=120, got %f", got)
		}
		if got := storageBytes["thumbnails"]; got != 60 {
			t.Errorf("expected storage_bytes[thumbnails]=60, got %f", got)
		}
	})
}

func TestBackfillMissingThumbnails_ProcessedPhoto(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	rawJPEG := createTestJPEG(800, 600)

	photoID := "photo-legacy-1"

	// 1. Mark as processed
	if err := store.SaveProcessedPhoto(photoID); err != nil {
		t.Fatalf("failed to mark processed: %v", err)
	}

	// 2. Write original image to images/photo-legacy-1.jpg
	if err := store.WriteImage(photoID, rawJPEG); err != nil {
		t.Fatalf("failed to write original image: %v", err)
	}

	// 3. Write legacy artwork proto with empty ThumbnailPath
	legacyArtwork := &gallery.Artwork{
		Id:            photoID,
		Title:         "Legacy artwork",
		Description:   "A beautiful vintage capture",
		ThemeId:       "Vintage",
		Timestamp:     time.Now().Unix(),
		ImagePath:     "images/" + photoID + ".jpg",
		ThumbnailPath: "",
	}
	protoBytes, err := proto.Marshal(legacyArtwork)
	if err != nil {
		t.Fatalf("failed to marshal proto: %v", err)
	}
	if err := store.WriteArtworkProto(photoID, protoBytes); err != nil {
		t.Fatalf("failed to write artwork proto: %v", err)
	}

	// Ensure no thumbnail exists
	thumbFile := filepath.Join(tempDir, "thumbnails", photoID+".webp")
	if _, err := os.Stat(thumbFile); !os.IsNotExist(err) {
		t.Fatalf("expected thumbnail to not exist initially")
	}

	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: photoID, DownloadURL: "https://photos.google.com/legacy=w0-h0"},
		},
	}
	visionSvc := &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			t.Errorf("vision analysis should not be called for already processed photo")
			return "", "", nil
		},
	}

	m := metrics.NewMetrics()
	albumURL := "https://photos.app.goo.gl/samplealbum"

	exitCode := Run(context.Background(), albumURL, tempDir, photoSvc, visionSvc, store, nil, m)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	if len(photoSvc.downloadCalls) != 0 {
		t.Errorf("expected photoSvc.DownloadImage not to be called, got %d calls", len(photoSvc.downloadCalls))
	}
	if visionSvc.callCount != 0 {
		t.Errorf("expected visionSvc not to be called, got %d calls", visionSvc.callCount)
	}

	// Verify thumbnail was backfilled on disk
	if fi, err := os.Stat(thumbFile); err != nil {
		t.Errorf("expected thumbnail to be backfilled at %s: %v", thumbFile, err)
	} else if fi.Size() == 0 {
		t.Errorf("expected non-empty thumbnail file")
	}

	// Verify artwork proto was updated with ThumbnailPath
	updatedProtoBytes, err := os.ReadFile(filepath.Join(tempDir, photoID+".proto.bin"))
	if err != nil {
		t.Fatalf("failed to read proto file: %v", err)
	}
	var updatedArtwork gallery.Artwork
	if err := proto.Unmarshal(updatedProtoBytes, &updatedArtwork); err != nil {
		t.Fatalf("failed to unmarshal updated artwork: %v", err)
	}
	expectedThumbPath := "thumbnails/" + photoID + ".webp"
	if updatedArtwork.GetThumbnailPath() != expectedThumbPath {
		t.Errorf("expected updated ThumbnailPath %q, got %q", expectedThumbPath, updatedArtwork.GetThumbnailPath())
	}

	// Verify metrics
	values, _ := gatherMetricValues(t, m)
	if got := values["rose_syncer_thumbnails_generated_total"]; got != 1 {
		t.Errorf("expected thumbnails_generated_total=1, got %f", got)
	}
	if got := values["rose_syncer_thumbnails_stored"]; got != 1 {
		t.Errorf("expected thumbnails_stored=1, got %f", got)
	}
}

func TestBackfillMissingThumbnails_AlreadyHasThumbnail(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	rawJPEG := createTestJPEG(400, 300)
	photoID := "photo-with-thumb"

	if err := store.SaveProcessedPhoto(photoID); err != nil {
		t.Fatalf("failed to mark processed: %v", err)
	}
	if err := store.WriteImage(photoID, rawJPEG); err != nil {
		t.Fatalf("failed to write original image: %v", err)
	}
	thumbBytes, err := thumbnail.GenerateThumbnail(rawJPEG)
	if err != nil {
		t.Fatalf("failed to generate thumbnail: %v", err)
	}
	if err := store.WriteThumbnail(photoID, thumbBytes); err != nil {
		t.Fatalf("failed to write thumbnail: %v", err)
	}

	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: photoID, DownloadURL: "https://photos.google.com/thumb=w0-h0"},
		},
	}
	visionSvc := &mockVisionService{}
	m := metrics.NewMetrics()

	exitCode := Run(context.Background(), "https://photos.app.goo.gl/album", tempDir, photoSvc, visionSvc, store, nil, m)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	if len(photoSvc.downloadCalls) != 0 {
		t.Errorf("expected no download calls, got %d", len(photoSvc.downloadCalls))
	}
	if visionSvc.callCount != 0 {
		t.Errorf("expected no vision calls, got %d", visionSvc.callCount)
	}

	values, _ := gatherMetricValues(t, m)
	if got := values["rose_syncer_thumbnails_generated_total"]; got != 0 {
		t.Errorf("expected 0 thumbnails generated when already exists, got %f", got)
	}
}

func TestProcessedPhoto_MissingImage_Redownloaded(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoID := "photo-no-image"

	if err := store.SaveProcessedPhoto(photoID); err != nil {
		t.Fatalf("failed to mark processed: %v", err)
	}
	// Image file not written

	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: photoID, DownloadURL: "https://photos.google.com/missing=w0-h0"},
		},
	}
	visionSvc := &mockVisionService{}
	m := metrics.NewMetrics()

	exitCode := Run(context.Background(), "https://photos.app.goo.gl/album", tempDir, photoSvc, visionSvc, store, nil, m)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 when re-downloading missing image, got %d", exitCode)
	}

	if len(photoSvc.downloadCalls) != 1 {
		t.Errorf("expected 1 download call for missing image, got %d", len(photoSvc.downloadCalls))
	}
	if !store.HasImage(photoID) {
		t.Errorf("expected raw image to exist at images/%s.jpg", photoID)
	}
	if !store.HasThumbnail(photoID) {
		t.Errorf("expected thumbnail to exist at thumbnails/%s.webp", photoID)
	}

	values, _ := gatherMetricValues(t, m)
	if got := values["rose_syncer_sync_errors_total"]; got != 0 {
		t.Errorf("expected 0 sync errors, got %f", got)
	}
	if got := values["rose_syncer_photos_downloaded_total"]; got != 1 {
		t.Errorf("expected 1 photo downloaded, got %f", got)
	}
	if got := values["rose_syncer_thumbnails_generated_total"]; got != 1 {
		t.Errorf("expected 1 thumbnail generated, got %f", got)
	}
	if got := values["rose_syncer_photos_stored"]; got != 1 {
		t.Errorf("expected 1 stored photo, got %f", got)
	}
	if got := values["rose_syncer_thumbnails_stored"]; got != 1 {
		t.Errorf("expected 1 stored thumbnail, got %f", got)
	}
}

func TestProcessedPhoto_LegacyImageMigrated(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	rawJPEG := createTestJPEG(400, 300)
	photoID := "legacy-photo-1"

	if err := store.SaveProcessedPhoto(photoID); err != nil {
		t.Fatalf("failed to mark processed: %v", err)
	}
	// Write legacy image directly into tempDir (basePath)
	legacyPath := filepath.Join(tempDir, photoID+".jpg")
	if err := os.WriteFile(legacyPath, rawJPEG, 0644); err != nil {
		t.Fatalf("failed to write legacy image: %v", err)
	}

	// Write initial artwork proto without ThumbnailPath
	initialArtwork := &gallery.Artwork{
		Id:        photoID,
		Title:     "Old Legacy Art",
		ImagePath: photoID + ".jpg",
	}
	protoBytes, err := proto.Marshal(initialArtwork)
	if err != nil {
		t.Fatalf("failed to marshal initial artwork: %v", err)
	}
	if err := store.WriteArtworkProto(photoID, protoBytes); err != nil {
		t.Fatalf("failed to write initial artwork proto: %v", err)
	}

	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: photoID, DownloadURL: "https://photos.google.com/legacy=w0-h0"},
		},
	}
	visionSvc := &mockVisionService{}
	m := metrics.NewMetrics()

	exitCode := Run(context.Background(), "https://photos.app.goo.gl/album", tempDir, photoSvc, visionSvc, store, nil, m)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0 when migrating legacy image, got %d", exitCode)
	}

	// Verify legacy file is moved
	if store.HasLegacyImage(photoID) {
		t.Errorf("expected legacy image file to no longer exist in root")
	}
	// Verify image is in images/
	if !store.HasImage(photoID) {
		t.Errorf("expected image to exist in images/")
	}
	// Verify thumbnail is in thumbnails/
	if !store.HasThumbnail(photoID) {
		t.Errorf("expected thumbnail to exist in thumbnails/")
	}
	// Verify no download call was made since legacy image was migrated
	if len(photoSvc.downloadCalls) != 0 {
		t.Errorf("expected 0 download calls, got %d", len(photoSvc.downloadCalls))
	}
	// Verify artwork proto updated with thumbnail path and image path
	updatedProtoBytes, err := store.ReadArtworkProto(photoID)
	if err != nil {
		t.Fatalf("failed to read updated artwork proto: %v", err)
	}
	var updatedArtwork gallery.Artwork
	if err := proto.Unmarshal(updatedProtoBytes, &updatedArtwork); err != nil {
		t.Fatalf("failed to unmarshal updated artwork proto: %v", err)
	}
	if updatedArtwork.GetThumbnailPath() != "thumbnails/"+photoID+".webp" {
		t.Errorf("expected ThumbnailPath %q, got %q", "thumbnails/"+photoID+".webp", updatedArtwork.GetThumbnailPath())
	}
	if updatedArtwork.GetImagePath() != "images/"+photoID+".jpg" {
		t.Errorf("expected ImagePath %q, got %q", "images/"+photoID+".jpg", updatedArtwork.GetImagePath())
	}

	values, _ := gatherMetricValues(t, m)
	if got := values["rose_syncer_sync_errors_total"]; got != 0 {
		t.Errorf("expected 0 sync errors, got %f", got)
	}
	if got := values["rose_syncer_photos_downloaded_total"]; got != 0 {
		t.Errorf("expected 0 photos downloaded, got %f", got)
	}
	if got := values["rose_syncer_thumbnails_generated_total"]; got != 1 {
		t.Errorf("expected 1 thumbnail generated, got %f", got)
	}
	if got := values["rose_syncer_photos_stored"]; got != 1 {
		t.Errorf("expected 1 stored photo gauge, got %f", got)
	}
	if got := values["rose_syncer_thumbnails_stored"]; got != 1 {
		t.Errorf("expected 1 stored thumbnail gauge, got %f", got)
	}
}

func TestProcessedPhoto_LegacyImageMigrationFailureAborts(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	rawJPEG := createTestJPEG(400, 300)
	photoID := "legacy-fail"

	if err := store.SaveProcessedPhoto(photoID); err != nil {
		t.Fatalf("failed to mark processed: %v", err)
	}
	legacyPath := filepath.Join(tempDir, photoID+".jpg")
	if err := os.WriteFile(legacyPath, rawJPEG, 0644); err != nil {
		t.Fatalf("failed to write legacy image: %v", err)
	}

	// Make images/ a read-only directory so migration (os.Rename/WriteFile) fails
	imagesDir := filepath.Join(tempDir, "images")
	if err := os.MkdirAll(imagesDir, 0555); err != nil {
		t.Fatalf("failed to make images dir: %v", err)
	}
	defer os.Chmod(imagesDir, 0755)

	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: photoID, DownloadURL: "https://photos.google.com/fail=w0-h0"},
		},
	}
	visionSvc := &mockVisionService{}
	reporter := &mockIssueReporter{}
	m := metrics.NewMetrics()

	exitCode := Run(context.Background(), "https://photos.app.goo.gl/album", tempDir, photoSvc, visionSvc, store, reporter, m)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 on migration failure abort, got %d", exitCode)
	}
	if reporter.createCalls.Load() != 1 {
		t.Errorf("expected 1 failure report, got %d", reporter.createCalls.Load())
	}
}


func TestBackfillMissingThumbnails_CorruptedOriginalImage(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoID := "photo-corrupt"

	if err := store.SaveProcessedPhoto(photoID); err != nil {
		t.Fatalf("failed to mark processed: %v", err)
	}
	// Write corrupt image bytes
	if err := store.WriteImage(photoID, []byte("not-a-valid-jpeg-image")); err != nil {
		t.Fatalf("failed to write image: %v", err)
	}

	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: photoID, DownloadURL: "https://photos.google.com/corrupt=w0-h0"},
		},
	}
	visionSvc := &mockVisionService{}
	m := metrics.NewMetrics()

	exitCode := Run(context.Background(), "https://photos.app.goo.gl/album", tempDir, photoSvc, visionSvc, store, nil, m)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 on corrupted image error, got %d", exitCode)
	}

	values, _ := gatherMetricValues(t, m)
	if got := values["rose_syncer_sync_errors_total"]; got != 1 {
		t.Errorf("expected 1 sync error for thumbnail generation failure, got %f", got)
	}
}

func TestBackfillMissingThumbnails_WriteThumbnailFailureAborts(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	rawJPEG := createTestJPEG(400, 300)
	photoID := "photo-write-fail"

	if err := store.SaveProcessedPhoto(photoID); err != nil {
		t.Fatalf("failed to mark processed: %v", err)
	}
	if err := store.WriteImage(photoID, rawJPEG); err != nil {
		t.Fatalf("failed to write image: %v", err)
	}

	// Make thumbnails/ a read-only directory so WriteThumbnail fails
	thumbnailsDir := filepath.Join(tempDir, "thumbnails")
	if err := os.MkdirAll(thumbnailsDir, 0555); err != nil {
		t.Fatalf("failed to make thumbnails dir: %v", err)
	}
	defer os.Chmod(thumbnailsDir, 0755)

	photoSvc := &mockPhotoService{
		photos: []photos.Photo{
			{ID: photoID, DownloadURL: "https://photos.google.com/writefail=w0-h0"},
		},
	}
	visionSvc := &mockVisionService{}
	reporter := &mockIssueReporter{}
	m := metrics.NewMetrics()

	exitCode := Run(context.Background(), "https://photos.app.goo.gl/album", tempDir, photoSvc, visionSvc, store, reporter, m)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 on storage persistence abort, got %d", exitCode)
	}
	if reporter.createCalls.Load() != 1 {
		t.Errorf("expected 1 failure report, got %d", reporter.createCalls.Load())
	}
}











