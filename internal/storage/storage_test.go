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

func TestStorageRobustness_NonExistentDirectory(t *testing.T) {
	tempDir := t.TempDir()
	nestedDir := filepath.Join(tempDir, "nonexistent", "nested", "path")
	store := NewStore(nestedDir)

	// Writing artwork proto should auto-create BasePath
	err := store.WriteArtworkProto("photo_auto", []byte("proto-content"))
	if err != nil {
		t.Fatalf("expected WriteArtworkProto to auto-create directory, got error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(nestedDir, "photo_auto.proto.bin")); err != nil {
		t.Errorf("expected photo_auto.proto.bin to exist: %v", err)
	}

	nestedDir2 := filepath.Join(tempDir, "another", "nested", "path")
	store2 := NewStore(nestedDir2)

	// Saving processed photo should auto-create BasePath
	err = store2.SaveProcessedPhoto("photo_state")
	if err != nil {
		t.Fatalf("expected SaveProcessedPhoto to auto-create directory, got error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(nestedDir2, ".sync-state.json")); err != nil {
		t.Errorf("expected .sync-state.json to exist: %v", err)
	}
}

