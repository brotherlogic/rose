package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/brotherlogic/rose/internal/storage"
	gallery "github.com/brotherlogic/rose/proto"
	"google.golang.org/protobuf/proto"
)

type mockPhotoService struct {
	photos []string
	err    error
}

func (m *mockPhotoService) FetchPhotos(ctx context.Context) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.photos, nil
}

type mockVisionService struct {
	analyzeFunc func(ctx context.Context, img []byte) (string, string, error)
	callCount   int
}

func (m *mockVisionService) AnalyzeImage(ctx context.Context, img []byte) (string, string, error) {
	m.callCount++
	if m.analyzeFunc != nil {
		return m.analyzeFunc(ctx, img)
	}
	return "A beautiful sunset", "Landscape", nil
}

func TestFullSyncPass(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photoSvc := &mockPhotoService{
		photos: []string{"photo-1", "photo-2"},
	}
	visionSvc := &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			return "Scenic mountain view", "Nature", nil
		},
	}

	exitCode := Run(context.Background(), tempDir, photoSvc, visionSvc, store)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	if visionSvc.callCount != 2 {
		t.Errorf("expected vision service called 2 times, got %d", visionSvc.callCount)
	}

	// Verify sync state
	for _, id := range []string{"photo-1", "photo-2"} {
		processed, err := store.IsPhotoProcessed(id)
		if err != nil {
			t.Fatalf("error checking sync state for %s: %v", id, err)
		}
		if !processed {
			t.Errorf("expected %s to be marked processed", id)
		}

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
		if artwork.GetImagePath() != id {
			t.Errorf("expected image_path %s, got %s", id, artwork.GetImagePath())
		}
		if artwork.GetTimestamp() <= 0 {
			t.Errorf("expected positive timestamp, got %d", artwork.GetTimestamp())
		}
	}
}

func TestIdempotency(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	photos := []string{"photo-1", "photo-2"}
	photoSvc := &mockPhotoService{photos: photos}
	visionSvc := &mockVisionService{}

	// First pass
	code1 := Run(context.Background(), tempDir, photoSvc, visionSvc, store)
	if code1 != 0 {
		t.Fatalf("first pass failed with exit code %d", code1)
	}
	if visionSvc.callCount != 2 {
		t.Fatalf("expected 2 vision calls on first pass, got %d", visionSvc.callCount)
	}

	// Second pass
	code2 := Run(context.Background(), tempDir, photoSvc, visionSvc, store)
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
	photos := []string{"photo-err", "photo-ok"}
	photoSvc := &mockPhotoService{photos: photos}
	var visionSvc *mockVisionService
	visionSvc = &mockVisionService{
		analyzeFunc: func(ctx context.Context, img []byte) (string, string, error) {
			if visionSvc.callCount == 1 {
				return "", "", errors.New("vision API timeout")
			}
			return "Ocean view", "Water", nil
		},
	}

	exitCode := Run(context.Background(), tempDir, photoSvc, visionSvc, store)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 due to partial failure, got %d", exitCode)
	}

	// First photo should NOT be marked processed
	processed1, err := store.IsPhotoProcessed("photo-err")
	if err != nil {
		t.Fatalf("unexpected error checking state: %v", err)
	}
	if processed1 {
		t.Errorf("failed photo should not be marked processed")
	}

	// Second photo SHOULD be processed despite previous failure
	processed2, err := store.IsPhotoProcessed("photo-ok")
	if err != nil {
		t.Fatalf("unexpected error checking state: %v", err)
	}
	if !processed2 {
		t.Errorf("subsequent photo should be processed despite earlier error")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "photo-ok.proto.bin")); err != nil {
		t.Errorf("expected photo-ok.proto.bin to exist: %v", err)
	}
}

func TestExitCodes(t *testing.T) {
	t.Run("EmptyPhotoList", func(t *testing.T) {
		tempDir := t.TempDir()
		store := storage.NewStore(tempDir)
		photoSvc := &mockPhotoService{photos: []string{}}
		visionSvc := &mockVisionService{}

		code := Run(context.Background(), tempDir, photoSvc, visionSvc, store)
		if code != 0 {
			t.Errorf("expected exit code 0 for empty photo list, got %d", code)
		}
	})

	t.Run("FetchPhotosError", func(t *testing.T) {
		tempDir := t.TempDir()
		store := storage.NewStore(tempDir)
		photoSvc := &mockPhotoService{err: errors.New("network failure")}
		visionSvc := &mockVisionService{}

		code := Run(context.Background(), tempDir, photoSvc, visionSvc, store)
		if code != 1 {
			t.Errorf("expected exit code 1 for fetch error, got %d", code)
		}
	})

	t.Run("ContextCancelled", func(t *testing.T) {
		tempDir := t.TempDir()
		store := storage.NewStore(tempDir)
		photoSvc := &mockPhotoService{photos: []string{"photo-1"}}
		visionSvc := &mockVisionService{}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel before run

		code := Run(ctx, tempDir, photoSvc, visionSvc, store)
		if code != 1 {
			t.Errorf("expected exit code 1 for cancelled context, got %d", code)
		}
	})
}

func TestStoragePathConfiguration(t *testing.T) {
	// Case 1: Flag overrides everything
	path, err := parseStoragePath([]string{"-storage-path", "/custom/flag/path"})
	if err != nil {
		t.Fatalf("unexpected error parsing flags: %v", err)
	}
	if path != "/custom/flag/path" {
		t.Errorf("expected /custom/flag/path, got %s", path)
	}

	// Case 2: Env var fallback
	t.Setenv("STORAGE_PATH", "/env/path")
	path, err = parseStoragePath([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/env/path" {
		t.Errorf("expected /env/path, got %s", path)
	}

	// Case 3: Default fallback to /data
	os.Unsetenv("STORAGE_PATH")
	path, err = parseStoragePath([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/data" {
		t.Errorf("expected default /data, got %s", path)
	}
}
