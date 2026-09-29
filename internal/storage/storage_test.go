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

func TestWriteImage_DirectoryStructure(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	data := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00}
	err := store.WriteImage("photo123", data)
	if err != nil {
		t.Fatalf("expected WriteImage to succeed, got error: %v", err)
	}

	expectedPath := filepath.Join(tempDir, "images", "photo123.jpg")
	readData, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("expected image file to exist at %s: %v", expectedPath, err)
	}

	if string(readData) != string(data) {
		t.Errorf("image content mismatch: got %v, want %v", readData, data)
	}

	info, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatalf("failed to stat written file: %v", err)
	}
	if info.Mode().Perm() != 0644 {
		t.Errorf("expected file mode 0644, got %v", info.Mode().Perm())
	}
}

func TestWriteImage_DirectoryTraversal(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	data := []byte("image payload")
	maliciousIDs := []string{
		"../photo",
		"../../etc/passwd",
		"/root/image",
		"sub/dir/img",
		"..",
		".",
		"",
		"nested\\file",
		"foo/../bar",
	}

	for _, malID := range maliciousIDs {
		t.Run(malID, func(t *testing.T) {
			err := store.WriteImage(malID, data)
			if err == nil {
				t.Fatalf("expected error for malicious ID %q, but got nil", malID)
			}
		})
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read tempDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no files written for malicious IDs, found %d entries", len(entries))
	}
}

func TestWriteThumbnail_DirectoryStructure(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	data := []byte("fake webp thumbnail data")
	err := store.WriteThumbnail("photo123", data)
	if err != nil {
		t.Fatalf("expected WriteThumbnail to succeed, got error: %v", err)
	}

	expectedPath := filepath.Join(tempDir, "thumbnails", "photo123.webp")
	readData, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("expected thumbnail file to exist at %s: %v", expectedPath, err)
	}

	if string(readData) != string(data) {
		t.Errorf("thumbnail content mismatch: got %v, want %v", readData, data)
	}

	info, err := os.Stat(expectedPath)
	if err != nil {
		t.Fatalf("failed to stat written file: %v", err)
	}
	if info.Mode().Perm() != 0644 {
		t.Errorf("expected file mode 0644, got %v", info.Mode().Perm())
	}
}

func TestWriteThumbnail_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	data := []byte("thumbnail payload")
	maliciousIDs := []string{
		"../id",
		"..",
		"/etc/passwd",
		"foo/bar",
		"\\test",
		"",
		".",
		"nested\\file",
		"foo/../bar",
	}

	for _, malID := range maliciousIDs {
		t.Run(malID, func(t *testing.T) {
			err := store.WriteThumbnail(malID, data)
			if err == nil {
				t.Fatalf("expected error for malicious ID %q, but got nil", malID)
			}
		})
	}

	thumbnailsDir := filepath.Join(tempDir, "thumbnails")
	if entries, err := os.ReadDir(thumbnailsDir); err == nil && len(entries) != 0 {
		t.Errorf("expected no files written for malicious IDs, found %d entries", len(entries))
	}
}

func TestWrite_AutoCreateDirectories(t *testing.T) {
	tempDir := t.TempDir()
	nestedDir := filepath.Join(tempDir, "nested", "deeply", "storage")
	store := NewStore(nestedDir)

	err := store.WriteImage("img1", []byte("img-bytes"))
	if err != nil {
		t.Fatalf("WriteImage failed: %v", err)
	}

	err = store.WriteThumbnail("thumb1", []byte("thumb-bytes"))
	if err != nil {
		t.Fatalf("WriteThumbnail failed: %v", err)
	}

	imagesDir := filepath.Join(nestedDir, "images")
	imgInfo, err := os.Stat(imagesDir)
	if err != nil {
		t.Fatalf("expected images directory to exist at %s: %v", imagesDir, err)
	}
	if !imgInfo.IsDir() {
		t.Fatalf("expected %s to be a directory", imagesDir)
	}
	if imgInfo.Mode().Perm() != 0755 {
		t.Errorf("expected images directory mode 0755, got %v", imgInfo.Mode().Perm())
	}

	thumbnailsDir := filepath.Join(nestedDir, "thumbnails")
	thumbInfo, err := os.Stat(thumbnailsDir)
	if err != nil {
		t.Fatalf("expected thumbnails directory to exist at %s: %v", thumbnailsDir, err)
	}
	if !thumbInfo.IsDir() {
		t.Fatalf("expected %s to be a directory", thumbnailsDir)
	}
	if thumbInfo.Mode().Perm() != 0755 {
		t.Errorf("expected thumbnails directory mode 0755, got %v", thumbInfo.Mode().Perm())
	}
}

func TestStorage_ReadImage(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	data := []byte("image-data-payload")
	if err := store.WriteImage("photo1", data); err != nil {
		t.Fatalf("failed to write image: %v", err)
	}

	readBytes, err := store.ReadImage("photo1")
	if err != nil {
		t.Fatalf("failed to read image: %v", err)
	}
	if string(readBytes) != string(data) {
		t.Errorf("expected %q, got %q", string(data), string(readBytes))
	}

	// Missing image
	_, err = store.ReadImage("nonexistent")
	if err == nil {
		t.Errorf("expected error reading nonexistent image, got nil")
	}

	// Invalid ID / path traversal
	_, err = store.ReadImage("../traversal")
	if err == nil {
		t.Errorf("expected error for path traversal id, got nil")
	}
}

func TestStorage_HasThumbnail(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	// Initially false
	if store.HasThumbnail("photo1") {
		t.Errorf("expected HasThumbnail to be false initially")
	}

	// Nil store
	var nilStore *Store
	if nilStore.HasThumbnail("photo1") {
		t.Errorf("expected nil store HasThumbnail to be false")
	}

	// Invalid ID
	if store.HasThumbnail("../bad") {
		t.Errorf("expected HasThumbnail to be false for invalid id")
	}

	// Write thumbnail
	if err := store.WriteThumbnail("photo1", []byte("webp-data")); err != nil {
		t.Fatalf("failed to write thumbnail: %v", err)
	}
	if !store.HasThumbnail("photo1") {
		t.Errorf("expected HasThumbnail to be true after writing thumbnail")
	}

	// Directory instead of file
	dirPath := filepath.Join(tempDir, "thumbnails", "dir-thumb.webp")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if store.HasThumbnail("dir-thumb") {
		t.Errorf("expected HasThumbnail to be false when path is a directory")
	}
}

func TestStorage_ReadArtworkProto(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	protoData := []byte("proto-binary-bytes")
	if err := store.WriteArtworkProto("photo1", protoData); err != nil {
		t.Fatalf("failed to write artwork proto: %v", err)
	}

	readBytes, err := store.ReadArtworkProto("photo1")
	if err != nil {
		t.Fatalf("failed to read artwork proto: %v", err)
	}
	if string(readBytes) != string(protoData) {
		t.Errorf("expected %q, got %q", string(protoData), string(readBytes))
	}

	// Missing proto
	_, err = store.ReadArtworkProto("nonexistent")
	if err == nil {
		t.Errorf("expected error reading nonexistent proto, got nil")
	}

	// Invalid ID
	_, err = store.ReadArtworkProto("..")
	if err == nil {
		t.Errorf("expected error for invalid id, got nil")
	}
}

func TestStorage_HasImage(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	// Initially false
	if store.HasImage("photo1") {
		t.Errorf("expected HasImage to be false initially")
	}

	// Nil store
	var nilStore *Store
	if nilStore.HasImage("photo1") {
		t.Errorf("expected nil store HasImage to be false")
	}

	// Invalid ID
	if store.HasImage("../bad") {
		t.Errorf("expected HasImage to be false for invalid id")
	}

	// Write image
	if err := store.WriteImage("photo1", []byte("img-data")); err != nil {
		t.Fatalf("failed to write image: %v", err)
	}
	if !store.HasImage("photo1") {
		t.Errorf("expected HasImage to be true after writing image")
	}

	// Directory instead of file
	dirPath := filepath.Join(tempDir, "images", "dir-img.jpg")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if store.HasImage("dir-img") {
		t.Errorf("expected HasImage to be false when path is a directory")
	}
}



