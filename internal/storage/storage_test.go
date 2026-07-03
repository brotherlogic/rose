package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorageSyncState(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	// Should not be processed initially
	processed, err := store.IsPhotoProcessed("photo123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if processed {
		t.Errorf("expected photo to not be processed")
	}

	// Save processed
	err = store.SaveProcessedPhoto("photo123")
	if err != nil {
		t.Fatalf("failed to save processed photo: %v", err)
	}

	// Should be processed now
	processed, err = store.IsPhotoProcessed("photo123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !processed {
		t.Errorf("expected photo to be processed")
	}

	// Verify file is created
	_, err = os.Stat(filepath.Join(tempDir, ".sync-state.json"))
	if err != nil {
		t.Errorf(".sync-state.json not created: %v", err)
	}
}

func TestStorageWriteArtwork(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	data := []byte("fake protobuf data")
	err := store.WriteArtworkProto("photo123", data)
	if err != nil {
		t.Fatalf("failed to write artwork proto: %v", err)
	}

	// Verify file is created
	readData, err := os.ReadFile(filepath.Join(tempDir, "photo123.proto.bin"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if string(readData) != string(data) {
		t.Errorf("written data mismatch")
	}
}
