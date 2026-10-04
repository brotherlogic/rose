package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gallerypb "github.com/brotherlogic/rose/proto"
	"google.golang.org/protobuf/proto"
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

func TestStorage_HasLegacyImage(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	// Initially false
	if store.HasLegacyImage("photo1") {
		t.Errorf("expected HasLegacyImage to be false initially")
	}

	// Nil store
	var nilStore *Store
	if nilStore.HasLegacyImage("photo1") {
		t.Errorf("expected nil store HasLegacyImage to be false")
	}

	// Invalid ID
	if store.HasLegacyImage("../bad") {
		t.Errorf("expected HasLegacyImage to be false for invalid id")
	}

	// Write legacy image directly to root
	legacyFile := filepath.Join(tempDir, "photo1.jpg")
	if err := os.WriteFile(legacyFile, []byte("legacy-jpg-data"), 0644); err != nil {
		t.Fatalf("failed to write legacy image file: %v", err)
	}
	if !store.HasLegacyImage("photo1") {
		t.Errorf("expected HasLegacyImage to be true after writing file")
	}

	// Directory instead of file
	dirPath := filepath.Join(tempDir, "dir-legacy.jpg")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if store.HasLegacyImage("dir-legacy") {
		t.Errorf("expected HasLegacyImage to be false when path is a directory")
	}
}

func TestStorage_MigrateLegacyImage(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	// Nil store
	var nilStore *Store
	if err := nilStore.MigrateLegacyImage("photo1"); err == nil {
		t.Errorf("expected error migrating with nil store")
	}

	// Invalid ID
	if err := store.MigrateLegacyImage("../bad"); err == nil {
		t.Errorf("expected error migrating invalid id")
	}

	// Non-existent legacy file
	if err := store.MigrateLegacyImage("nonexistent"); err == nil {
		t.Errorf("expected error migrating nonexistent file")
	}

	// Create legacy file
	legacyData := []byte("legacy-image-content")
	legacyFile := filepath.Join(tempDir, "photo1.jpg")
	if err := os.WriteFile(legacyFile, legacyData, 0644); err != nil {
		t.Fatalf("failed to write legacy file: %v", err)
	}

	if !store.HasLegacyImage("photo1") {
		t.Fatalf("expected HasLegacyImage to be true")
	}
	if store.HasImage("photo1") {
		t.Fatalf("expected HasImage to be false before migration")
	}

	// Migrate
	if err := store.MigrateLegacyImage("photo1"); err != nil {
		t.Fatalf("failed to migrate legacy image: %v", err)
	}

	// Legacy file should no longer exist
	if store.HasLegacyImage("photo1") {
		t.Errorf("expected HasLegacyImage to be false after migration")
	}

	// Image should now exist in images/
	if !store.HasImage("photo1") {
		t.Errorf("expected HasImage to be true after migration")
	}

	data, err := store.ReadImage("photo1")
	if err != nil {
		t.Fatalf("failed to read migrated image: %v", err)
	}
	if string(data) != string(legacyData) {
		t.Errorf("expected %q, got %q", string(legacyData), string(data))
	}
}

func TestStorage_ArtworkProtoRoundTripWithMedium(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	// 1. Populated medium test case
	const expectedMedium = "Wax pigment on reclaimed cardboard"
	artwork := &gallerypb.Artwork{
		Id:            "art-medium-roundtrip-1",
		Title:         "Abstract Composition #4",
		Description:   "Avant-garde mixed media exploration",
		ThemeId:       "postmodern",
		Timestamp:     1710000000,
		ImagePath:     "images/art-medium-roundtrip-1.jpg",
		ThumbnailPath: "thumbnails/art-medium-roundtrip-1.webp",
		Medium:        expectedMedium,
	}

	data, err := proto.Marshal(artwork)
	if err != nil {
		t.Fatalf("failed to marshal artwork: %v", err)
	}

	if err := store.WriteArtworkProto(artwork.GetId(), data); err != nil {
		t.Fatalf("failed to write artwork proto: %v", err)
	}

	readBytes, err := store.ReadArtworkProto(artwork.GetId())
	if err != nil {
		t.Fatalf("failed to read artwork proto: %v", err)
	}

	unmarshaled := &gallerypb.Artwork{}
	if err := proto.Unmarshal(readBytes, unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal artwork proto: %v", err)
	}

	if unmarshaled.GetMedium() != expectedMedium {
		t.Errorf("Medium mismatch: got %q, want %q", unmarshaled.GetMedium(), expectedMedium)
	}
	if unmarshaled.GetId() != artwork.GetId() {
		t.Errorf("ID mismatch: got %q, want %q", unmarshaled.GetId(), artwork.GetId())
	}
	if unmarshaled.GetTitle() != artwork.GetTitle() {
		t.Errorf("Title mismatch: got %q, want %q", unmarshaled.GetTitle(), artwork.GetTitle())
	}

	// 2. Legacy/empty payload verification: persist artwork without medium specified
	legacyArtwork := &gallerypb.Artwork{
		Id:            "art-legacy-roundtrip-2",
		Title:         "Legacy Without Medium",
		Description:   "Artwork without medium set",
		ThemeId:       "classic",
		Timestamp:     1650000000,
		ImagePath:     "images/art-legacy-roundtrip-2.jpg",
		ThumbnailPath: "thumbnails/art-legacy-roundtrip-2.webp",
	}

	legacyData, err := proto.Marshal(legacyArtwork)
	if err != nil {
		t.Fatalf("failed to marshal legacy artwork: %v", err)
	}

	if err := store.WriteArtworkProto(legacyArtwork.GetId(), legacyData); err != nil {
		t.Fatalf("failed to write legacy artwork proto: %v", err)
	}

	readLegacyBytes, err := store.ReadArtworkProto(legacyArtwork.GetId())
	if err != nil {
		t.Fatalf("failed to read legacy artwork proto: %v", err)
	}

	unmarshaledLegacy := &gallerypb.Artwork{}
	if err := proto.Unmarshal(readLegacyBytes, unmarshaledLegacy); err != nil {
		t.Fatalf("failed to unmarshal legacy artwork proto: %v", err)
	}

	if unmarshaledLegacy.GetMedium() != "" {
		t.Errorf("expected GetMedium() to safely default to empty string for legacy payload, got %q", unmarshaledLegacy.GetMedium())
	}
}

func TestWriteArtworkProtoAtomic_Success(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	data := []byte("binary-protobuf-payload-12345")
	id := "photo-atomic-success-1"

	err := store.WriteArtworkProtoAtomic(id, data)
	if err != nil {
		t.Fatalf("expected WriteArtworkProtoAtomic to succeed, got: %v", err)
	}

	targetPath := filepath.Join(tempDir, id+".proto.bin")
	fi, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("expected destination file %s to exist: %v", targetPath, err)
	}

	if fi.Mode().Perm() != 0644 {
		t.Errorf("expected file permissions 0644, got %v", fi.Mode().Perm())
	}

	readBytes, err := store.ReadArtworkProto(id)
	if err != nil {
		t.Fatalf("expected ReadArtworkProto to succeed: %v", err)
	}
	if string(readBytes) != string(data) {
		t.Errorf("read data mismatch: got %q, want %q", string(readBytes), string(data))
	}
}

func TestWriteArtworkProtoAtomic_TempFileCleanup(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	data := []byte("test-cleanup-bytes")
	id := "photo-atomic-cleanup"

	err := store.WriteArtworkProtoAtomic(id, data)
	if err != nil {
		t.Fatalf("WriteArtworkProtoAtomic failed: %v", err)
	}

	// Verify no dangling .tmp-* files remain in BasePath
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Errorf("found dangling temporary file: %s", entry.Name())
		}
	}

	// Verify cleanup when writing to an invalid or unwritable destination
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0755); err != nil {
		t.Fatalf("failed to create readonly dir: %v", err)
	}
	collidingDest := filepath.Join(readOnlyDir, "blocked-id.proto.bin")
	if err := os.MkdirAll(collidingDest, 0755); err != nil {
		t.Fatalf("failed to create colliding directory: %v", err)
	}

	roStore := NewStore(readOnlyDir)
	err = roStore.WriteArtworkProtoAtomic("blocked-id", data)
	if err == nil {
		t.Fatalf("expected WriteArtworkProtoAtomic to fail when renaming onto a non-empty directory")
	}

	roEntries, err := os.ReadDir(readOnlyDir)
	if err != nil {
		t.Fatalf("failed to read readonly dir: %v", err)
	}
	for _, entry := range roEntries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Errorf("found dangling temporary file after failure: %s", entry.Name())
		}
	}
}

func TestWriteArtworkProtoAtomic_InvalidID(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	data := []byte("proto-payload")
	invalidIDs := []string{
		"",
		".",
		"..",
		"../photo",
		"../../etc/passwd",
		"/root/file",
		"sub/dir",
		"path\\with\\backslash",
		"foo/../bar",
	}

	for _, id := range invalidIDs {
		t.Run(id, func(t *testing.T) {
			err := store.WriteArtworkProtoAtomic(id, data)
			if err == nil {
				t.Fatalf("expected error for invalid ID %q, got nil", id)
			}
		})
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read tempDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no files written in base directory for invalid IDs, found %d entries", len(entries))
	}
}

func TestSaveSyncStateAtomic(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	state := map[string]bool{
		"photo-1": true,
		"photo-2": false,
		"photo-3": true,
	}

	err := store.saveSyncState(state)
	if err != nil {
		t.Fatalf("expected saveSyncState to succeed, got: %v", err)
	}

	// Verify .sync-state.json exists and has 0644 permissions
	statPath := store.syncStatePath()
	fi, err := os.Stat(statPath)
	if err != nil {
		t.Fatalf("expected sync state file to exist: %v", err)
	}
	if fi.Mode().Perm() != 0644 {
		t.Errorf("expected sync state file permission 0644, got %v", fi.Mode().Perm())
	}

	// Verify no dangling .tmp-* files remain
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read dir: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Errorf("found dangling temporary file: %s", entry.Name())
		}
	}

	// Reload state via loadSyncState
	loadedState, err := store.loadSyncState()
	if err != nil {
		t.Fatalf("expected loadSyncState to succeed, got: %v", err)
	}

	if len(loadedState) != len(state) {
		t.Fatalf("loaded state count mismatch: got %d, want %d", len(loadedState), len(state))
	}
	for k, v := range state {
		if loadedState[k] != v {
			t.Errorf("loaded state mismatch for key %q: got %v, want %v", k, loadedState[k], v)
		}
	}
}







