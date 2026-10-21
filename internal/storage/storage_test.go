package storage

import (
	"bytes"
	"encoding/json"
	"log"
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

func TestStorage_HasArtworkProto(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	// Initially false
	if store.HasArtworkProto("photo1") {
		t.Errorf("expected HasArtworkProto to be false initially")
	}

	// Nil store
	var nilStore *Store
	if nilStore.HasArtworkProto("photo1") {
		t.Errorf("expected nil store HasArtworkProto to be false")
	}

	// Invalid ID
	if store.HasArtworkProto("../bad") {
		t.Errorf("expected HasArtworkProto to be false for invalid id")
	}

	// Write proto
	if err := store.WriteArtworkProto("photo1", []byte("proto-bytes")); err != nil {
		t.Fatalf("failed to write artwork proto: %v", err)
	}
	if !store.HasArtworkProto("photo1") {
		t.Errorf("expected HasArtworkProto to be true after writing proto")
	}

	// Directory instead of file
	dirPath := filepath.Join(tempDir, "dir-proto.proto.bin")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if store.HasArtworkProto("dir-proto") {
		t.Errorf("expected HasArtworkProto to be false when path is a directory")
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

func TestFindUnannotatedPhotos_MissingProto(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	imgData := []byte("image-data-missing-proto")
	if err := store.WriteImage("photo1", imgData); err != nil {
		t.Fatalf("failed to write image: %v", err)
	}

	imgPath := filepath.Join(tempDir, "images", "photo1.jpg")
	fi, err := os.Stat(imgPath)
	if err != nil {
		t.Fatalf("failed to stat image: %v", err)
	}
	expectedTime := fi.ModTime().Unix()

	candidates, err := store.FindUnannotatedPhotos()
	if err != nil {
		t.Fatalf("FindUnannotatedPhotos failed: %v", err)
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}

	if candidates[0].ID != "photo1" {
		t.Errorf("expected candidate ID 'photo1', got %q", candidates[0].ID)
	}
	if candidates[0].Timestamp != expectedTime {
		t.Errorf("expected candidate timestamp %d, got %d", expectedTime, candidates[0].Timestamp)
	}
}

func TestFindUnannotatedPhotos_EmptyMedium(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	if err := store.WriteImage("photo-empty-medium", []byte("img-data")); err != nil {
		t.Fatalf("failed to write image: %v", err)
	}

	// Proto with empty medium and specific timestamp
	const customTimestamp = int64(1700000000)
	artwork := &gallerypb.Artwork{
		Id:        "photo-empty-medium",
		Title:     "Draft Piece",
		Medium:    "",
		Timestamp: customTimestamp,
	}
	protoBytes, err := proto.Marshal(artwork)
	if err != nil {
		t.Fatalf("failed to marshal artwork: %v", err)
	}
	if err := store.WriteArtworkProto("photo-empty-medium", protoBytes); err != nil {
		t.Fatalf("failed to write artwork proto: %v", err)
	}

	// Also add a second photo with empty medium and 0 timestamp (fallback to file modtime)
	if err := store.WriteImage("photo-zero-ts", []byte("img-data-2")); err != nil {
		t.Fatalf("failed to write second image: %v", err)
	}
	zeroArtwork := &gallerypb.Artwork{
		Id:        "photo-zero-ts",
		Title:     "Zero Timestamp Piece",
		Medium:    "",
		Timestamp: 0,
	}
	zeroBytes, err := proto.Marshal(zeroArtwork)
	if err != nil {
		t.Fatalf("failed to marshal zero artwork: %v", err)
	}
	if err := store.WriteArtworkProto("photo-zero-ts", zeroBytes); err != nil {
		t.Fatalf("failed to write zero artwork proto: %v", err)
	}

	imgPath := filepath.Join(tempDir, "images", "photo-zero-ts.jpg")
	fi, err := os.Stat(imgPath)
	if err != nil {
		t.Fatalf("failed to stat image: %v", err)
	}
	expectedZeroTs := fi.ModTime().Unix()

	candidates, err := store.FindUnannotatedPhotos()
	if err != nil {
		t.Fatalf("FindUnannotatedPhotos failed: %v", err)
	}

	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}

	foundMap := make(map[string]int64)
	for _, c := range candidates {
		foundMap[c.ID] = c.Timestamp
	}

	if ts, ok := foundMap["photo-empty-medium"]; !ok || ts != customTimestamp {
		t.Errorf("expected photo-empty-medium timestamp %d, got %d (found=%v)", customTimestamp, ts, ok)
	}
	if ts, ok := foundMap["photo-zero-ts"]; !ok || ts != expectedZeroTs {
		t.Errorf("expected photo-zero-ts timestamp %d, got %d (found=%v)", expectedZeroTs, ts, ok)
	}
}

func TestFindUnannotatedPhotos_FullyAnnotatedIgnored(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	if err := store.WriteImage("photo-annotated", []byte("img-data")); err != nil {
		t.Fatalf("failed to write image: %v", err)
	}

	artwork := &gallerypb.Artwork{
		Id:        "photo-annotated",
		Title:     "Masterpiece",
		Medium:    "Oil on canvas",
		Timestamp: 1600000000,
	}
	protoBytes, err := proto.Marshal(artwork)
	if err != nil {
		t.Fatalf("failed to marshal artwork: %v", err)
	}
	if err := store.WriteArtworkProto("photo-annotated", protoBytes); err != nil {
		t.Fatalf("failed to write artwork proto: %v", err)
	}

	candidates, err := store.FindUnannotatedPhotos()
	if err != nil {
		t.Fatalf("FindUnannotatedPhotos failed: %v", err)
	}

	if len(candidates) != 0 {
		t.Fatalf("expected 0 candidates for fully annotated photo, got %d: %+v", len(candidates), candidates)
	}
}

func TestFindUnannotatedPhotos_DeterministicSorting(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	type testItem struct {
		id string
		ts int64
	}
	items := []testItem{
		{"photo-d", 200},
		{"photo-b", 100},
		{"photo-a", 100},
		{"photo-c", 300},
		{"photo-e", 200},
	}

	for _, item := range items {
		if err := store.WriteImage(item.id, []byte("data-"+item.id)); err != nil {
			t.Fatalf("failed to write image %s: %v", item.id, err)
		}
		art := &gallerypb.Artwork{
			Id:        item.id,
			Timestamp: item.ts,
			Medium:    "", // unannotated
		}
		pb, err := proto.Marshal(art)
		if err != nil {
			t.Fatalf("failed to marshal %s: %v", item.id, err)
		}
		if err := store.WriteArtworkProto(item.id, pb); err != nil {
			t.Fatalf("failed to write proto %s: %v", item.id, err)
		}
	}

	candidates, err := store.FindUnannotatedPhotos()
	if err != nil {
		t.Fatalf("FindUnannotatedPhotos failed: %v", err)
	}

	expectedIDs := []string{"photo-a", "photo-b", "photo-d", "photo-e", "photo-c"}
	if len(candidates) != len(expectedIDs) {
		t.Fatalf("expected %d candidates, got %d", len(expectedIDs), len(candidates))
	}

	for i, expectedID := range expectedIDs {
		if candidates[i].ID != expectedID {
			t.Errorf("candidate at index %d: expected ID %q, got %q", i, expectedID, candidates[i].ID)
		}
	}
}

func TestFindUnannotatedPhotos_OrphanProtoSkipped(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	orphanArt := &gallerypb.Artwork{
		Id:        "orphan-photo",
		Timestamp: 1000,
		Medium:    "",
	}
	pb, err := proto.Marshal(orphanArt)
	if err != nil {
		t.Fatalf("failed to marshal orphan proto: %v", err)
	}
	if err := store.WriteArtworkProto("orphan-photo", pb); err != nil {
		t.Fatalf("failed to write orphan proto: %v", err)
	}

	if err := store.WriteImage("valid-photo", []byte("img-data")); err != nil {
		t.Fatalf("failed to write image: %v", err)
	}

	candidates, err := store.FindUnannotatedPhotos()
	if err != nil {
		t.Fatalf("FindUnannotatedPhotos failed: %v", err)
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].ID != "valid-photo" {
		t.Errorf("expected candidate 'valid-photo', got %q", candidates[0].ID)
	}
}

func TestFindUnannotatedPhotos_EdgeCases(t *testing.T) {
	var nilStore *Store
	_, err := nilStore.FindUnannotatedPhotos()
	if err == nil {
		t.Errorf("expected error for nil store, got nil")
	}

	tempDir := t.TempDir()
	emptyStore := NewStore(tempDir)
	candidates, err := emptyStore.FindUnannotatedPhotos()
	if err != nil {
		t.Fatalf("expected no error for nonexistent images directory, got %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates, got %d", len(candidates))
	}

	imagesDir := filepath.Join(tempDir, "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		t.Fatalf("failed to create images dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imagesDir, "photo.png"), []byte("png"), 0644); err != nil {
		t.Fatalf("failed to write png file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imagesDir, "photo.txt"), []byte("txt"), 0644); err != nil {
		t.Fatalf("failed to write txt file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(imagesDir, "subfolder.jpg"), 0755); err != nil {
		t.Fatalf("failed to create subfolder: %v", err)
	}

	candidates, err = emptyStore.FindUnannotatedPhotos()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("expected 0 candidates when only non-jpg / subdirectories exist, got %d", len(candidates))
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

func TestGenerateCatalog_EmptyDirectory(t *testing.T) {
	var nilStore *Store
	if err := nilStore.GenerateCatalog(); err == nil {
		t.Errorf("expected error for nil store, got nil")
	}

	tempDir := t.TempDir()
	nestedDir := filepath.Join(tempDir, "nested", "empty")
	store := NewStore(nestedDir)

	if err := store.GenerateCatalog(); err != nil {
		t.Fatalf("expected GenerateCatalog on empty directory to succeed, got: %v", err)
	}

	catalogPath := filepath.Join(nestedDir, "catalog.json")
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatalf("failed to read generated catalog.json: %v", err)
	}

	if strings.TrimSpace(string(data)) != "[]" {
		t.Errorf("expected empty catalog to serialize as \"[]\", got %q", string(data))
	}
}

func TestGenerateCatalog_AggregationSortingAndAllFields(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	artworks := []*gallerypb.Artwork{
		{
			Id:            "art-b",
			Title:         "Tie Timestamp B",
			Description:   "Second in tie-break",
			ThemeId:       "abstract",
			Timestamp:     1700000200,
			ImagePath:     "images/art-b.jpg",
			ThumbnailPath: "thumbnails/art-b.webp",
			Medium:        "Acrylic on canvas",
		},
		{
			Id:            "art-oldest",
			Title:         "Minimalist Empty Optionals",
			Description:   "",
			ThemeId:       "minimalism",
			Timestamp:     1700000100,
			ImagePath:     "images/art-oldest.jpg",
			ThumbnailPath: "thumbnails/art-oldest.webp",
			Medium:        "",
		},
		{
			Id:            "art-a",
			Title:         "Tie Timestamp A",
			Description:   "First in tie-break",
			ThemeId:       "abstract",
			Timestamp:     1700000200,
			ImagePath:     "images/art-a.jpg",
			ThumbnailPath: "thumbnails/art-a.webp",
			Medium:        "Oil on linen",
		},
		{
			Id:            "art-newest",
			Title:         "Newest Piece",
			Description:   "Latest addition",
			ThemeId:       "expressionism",
			Timestamp:     1700000300,
			ImagePath:     "images/art-newest.jpg",
			ThumbnailPath: "thumbnails/art-newest.webp",
			Medium:        "Mixed media",
		},
	}

	for _, art := range artworks {
		if err := store.WriteImage(art.GetId(), []byte("fake-jpeg-"+art.GetId())); err != nil {
			t.Fatalf("failed to write image %s: %v", art.GetId(), err)
		}
		pbBytes, err := proto.Marshal(art)
		if err != nil {
			t.Fatalf("failed to marshal artwork %s: %v", art.GetId(), err)
		}
		if err := store.WriteArtworkProtoAtomic(art.GetId(), pbBytes); err != nil {
			t.Fatalf("failed to write artwork proto %s: %v", art.GetId(), err)
		}
	}

	// Add noise entries that should be ignored: hidden file, temp file, and directory ending in .proto.bin
	if err := os.WriteFile(filepath.Join(tempDir, ".hidden.proto.bin"), []byte("hidden"), 0644); err != nil {
		t.Fatalf("failed to write hidden file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "art.proto.bin.tmp-123"), []byte("tmp"), 0644); err != nil {
		t.Fatalf("failed to write tmp file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "subdir.proto.bin"), 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	if err := store.GenerateCatalog(); err != nil {
		t.Fatalf("GenerateCatalog failed: %v", err)
	}

	catalogBytes, err := os.ReadFile(filepath.Join(tempDir, "catalog.json"))
	if err != nil {
		t.Fatalf("failed to read catalog.json: %v", err)
	}

	var parsed []CatalogArtwork
	if err := json.Unmarshal(catalogBytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal catalog.json into []CatalogArtwork: %v", err)
	}

	expectedOrder := []string{"art-newest", "art-a", "art-b", "art-oldest"}
	if len(parsed) != len(expectedOrder) {
		t.Fatalf("expected %d artworks in catalog, got %d", len(expectedOrder), len(parsed))
	}
	for i, wantID := range expectedOrder {
		if parsed[i].ID != wantID {
			t.Errorf("index %d: expected ID %q, got %q", i, wantID, parsed[i].ID)
		}
	}

	// Verify all 9 snake_case keys are explicitly present in raw JSON maps, even when empty
	var rawMaps []map[string]interface{}
	if err := json.Unmarshal(catalogBytes, &rawMaps); err != nil {
		t.Fatalf("failed to unmarshal catalog.json into raw maps: %v", err)
	}
	requiredKeys := []string{
		"id",
		"title",
		"description",
		"theme_id",
		"timestamp",
		"image_path",
		"thumbnail_path",
		"medium",
		"is_representative",
	}
	for idx, itemMap := range rawMaps {
		for _, key := range requiredKeys {
			if _, exists := itemMap[key]; !exists {
				t.Errorf("artwork at index %d (id=%v) is missing required JSON key %q", idx, itemMap["id"], key)
			}
		}
	}
}

func TestGenerateCatalog_SkipsCorruptedProto(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	validArt := &gallerypb.Artwork{
		Id:            "valid-art-1",
		Title:         "Valid Artwork",
		Description:   "Valid description",
		ThemeId:       "classic",
		Timestamp:     1700000500,
		ImagePath:     "images/valid-art-1.jpg",
		ThumbnailPath: "thumbnails/valid-art-1.webp",
		Medium:        "Watercolor",
	}
	if err := store.WriteImage(validArt.GetId(), []byte("fake-jpeg")); err != nil {
		t.Fatalf("failed to write valid artwork image: %v", err)
	}
	pbBytes, err := proto.Marshal(validArt)
	if err != nil {
		t.Fatalf("failed to marshal valid artwork: %v", err)
	}
	if err := store.WriteArtworkProtoAtomic(validArt.GetId(), pbBytes); err != nil {
		t.Fatalf("failed to write valid artwork proto: %v", err)
	}

	// Write corrupted .proto.bin file
	corruptPath := filepath.Join(tempDir, "corrupt-art.proto.bin")
	if err := os.WriteFile(corruptPath, []byte{0xFF, 0xFE, 0xFD, 0x00, 0x01}, 0644); err != nil {
		t.Fatalf("failed to write corrupt proto file: %v", err)
	}

	var logBuf bytes.Buffer
	origOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(origOutput)

	if err := store.GenerateCatalog(); err != nil {
		t.Fatalf("expected GenerateCatalog to succeed despite corrupt proto, got: %v", err)
	}

	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "warning: skipping corrupt artwork proto corrupt-art.proto.bin:") {
		t.Errorf("expected warning log for corrupt proto, got log: %q", logOutput)
	}

	catalogBytes, err := os.ReadFile(filepath.Join(tempDir, "catalog.json"))
	if err != nil {
		t.Fatalf("failed to read catalog.json: %v", err)
	}

	var parsed []CatalogArtwork
	if err := json.Unmarshal(catalogBytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal catalog.json: %v", err)
	}

	if len(parsed) != 1 || parsed[0].ID != "valid-art-1" {
		t.Errorf("expected only valid-art-1 in catalog, got %+v", parsed)
	}
}

func TestGenerateCatalog_SkipsOrphanedProtoAndNormalizesPaths(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	// 1. Orphaned test protos without corresponding images (like photo1.jpg.proto.bin and photo2.jpg.proto.bin)
	for _, orphanID := range []string{"photo1.jpg", "photo2.jpg", "orphan-no-image"} {
		orphanPb, err := proto.Marshal(&gallerypb.Artwork{
			Id:        orphanID,
			Title:     "Orphaned Test Proto",
			Timestamp: 1700000900,
			ImagePath: orphanID + ".jpg",
		})
		if err != nil {
			t.Fatalf("failed to marshal orphan %s: %v", orphanID, err)
		}
		if err := store.WriteArtworkProtoAtomic(orphanID, orphanPb); err != nil {
			t.Fatalf("failed to write orphan proto %s: %v", orphanID, err)
		}
	}

	// 2. Artwork with image in images/<id>.jpg and thumbnail in thumbnails/<id>.webp,
	//    but legacy bare ImagePath ("<id>.jpg") and empty ThumbnailPath in proto
	legacyPathID := "AF1QipMMgSANgQO9t3rSDH7ALFTBkizu6iz-LHCW_R79"
	if err := store.WriteImage(legacyPathID, []byte("jpeg-bytes")); err != nil {
		t.Fatalf("failed to write image for %s: %v", legacyPathID, err)
	}
	if err := store.WriteThumbnail(legacyPathID, []byte("webp-bytes")); err != nil {
		t.Fatalf("failed to write thumbnail for %s: %v", legacyPathID, err)
	}
	legacyPathPb, err := proto.Marshal(&gallerypb.Artwork{
		Id:            legacyPathID,
		Title:         "Legacy Path Artwork",
		Timestamp:     1700000800,
		ImagePath:     legacyPathID + ".jpg",
		ThumbnailPath: "",
		Medium:        "Crayon on paper",
	})
	if err != nil {
		t.Fatalf("failed to marshal legacyPathID: %v", err)
	}
	if err := store.WriteArtworkProtoAtomic(legacyPathID, legacyPathPb); err != nil {
		t.Fatalf("failed to write proto for %s: %v", legacyPathID, err)
	}

	// 3. Artwork with legacy root image (<basePath>/<id>.jpg via HasLegacyImage)
	legacyRootID := "legacy-root-img"
	if err := os.WriteFile(filepath.Join(tempDir, legacyRootID+".jpg"), []byte("root-jpeg"), 0644); err != nil {
		t.Fatalf("failed to write legacy root image: %v", err)
	}
	legacyRootPb, err := proto.Marshal(&gallerypb.Artwork{
		Id:        legacyRootID,
		Title:     "Legacy Root Image Artwork",
		Timestamp: 1700000700,
		ImagePath: legacyRootID + ".jpg",
	})
	if err != nil {
		t.Fatalf("failed to marshal legacyRootID: %v", err)
	}
	if err := store.WriteArtworkProtoAtomic(legacyRootID, legacyRootPb); err != nil {
		t.Fatalf("failed to write proto for %s: %v", legacyRootID, err)
	}

	if err := store.GenerateCatalog(); err != nil {
		t.Fatalf("GenerateCatalog failed: %v", err)
	}

	catalogBytes, err := os.ReadFile(filepath.Join(tempDir, "catalog.json"))
	if err != nil {
		t.Fatalf("failed to read catalog.json: %v", err)
	}

	var parsed []CatalogArtwork
	if err := json.Unmarshal(catalogBytes, &parsed); err != nil {
		t.Fatalf("failed to unmarshal catalog.json: %v", err)
	}

	if len(parsed) != 2 {
		t.Fatalf("expected 2 valid artworks in catalog (orphans excluded), got %d: %+v", len(parsed), parsed)
	}

	if parsed[0].ID != legacyPathID {
		t.Errorf("expected first artwork ID %q, got %q", legacyPathID, parsed[0].ID)
	}
	if parsed[0].ImagePath != "images/"+legacyPathID+".jpg" {
		t.Errorf("expected normalized ImagePath %q, got %q", "images/"+legacyPathID+".jpg", parsed[0].ImagePath)
	}
	if parsed[0].ThumbnailPath != "thumbnails/"+legacyPathID+".webp" {
		t.Errorf("expected normalized ThumbnailPath %q, got %q", "thumbnails/"+legacyPathID+".webp", parsed[0].ThumbnailPath)
	}

	if parsed[1].ID != legacyRootID {
		t.Errorf("expected second artwork ID %q, got %q", legacyRootID, parsed[1].ID)
	}
	if parsed[1].ImagePath != "images/"+legacyRootID+".jpg" {
		t.Errorf("expected normalized ImagePath %q, got %q", "images/"+legacyRootID+".jpg", parsed[1].ImagePath)
	}
}

func TestGenerateCatalog_AtomicWriteLeavesNoTmpFiles(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	art := &gallerypb.Artwork{
		Id:        "atomic-art-1",
		Title:     "Atomic Check",
		Timestamp: 1700000900,
	}
	if err := store.WriteImage(art.GetId(), []byte("fake-jpeg")); err != nil {
		t.Fatalf("failed to write artwork image: %v", err)
	}
	pbBytes, err := proto.Marshal(art)
	if err != nil {
		t.Fatalf("failed to marshal artwork: %v", err)
	}
	if err := store.WriteArtworkProtoAtomic(art.GetId(), pbBytes); err != nil {
		t.Fatalf("failed to write artwork proto: %v", err)
	}

	if err := store.GenerateCatalog(); err != nil {
		t.Fatalf("GenerateCatalog failed: %v", err)
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read tempDir: %v", err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp") {
			t.Errorf("found leftover temporary file after GenerateCatalog: %s", entry.Name())
		}
	}

	// Also verify cleanup when os.Rename onto catalog.json fails (e.g., catalog.json is a non-empty directory)
	failDir := filepath.Join(tempDir, "fail-rename")
	if err := os.MkdirAll(filepath.Join(failDir, "catalog.json", "nested"), 0755); err != nil {
		t.Fatalf("failed to create colliding catalog.json directory: %v", err)
	}
	failStore := NewStore(failDir)
	if err := failStore.GenerateCatalog(); err == nil {
		t.Fatalf("expected GenerateCatalog to fail when catalog.json is a non-empty directory")
	}

	failEntries, err := os.ReadDir(failDir)
	if err != nil {
		t.Fatalf("failed to read failDir: %v", err)
	}
	for _, entry := range failEntries {
		if strings.Contains(entry.Name(), ".tmp") {
			t.Errorf("found leftover temporary file after failed GenerateCatalog: %s", entry.Name())
		}
	}
}

func TestGenerateCatalog_RepresentativeScoringAndDesignation(t *testing.T) {
	t.Run("marks single highest-scoring artwork as representative with dominant movement bonus", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewStore(tempDir)

		// "Kinetic Scribblism" appears twice (dominant canonical movement), while "Monochrome Nihilism" appears once.
		artworks := []*gallerypb.Artwork{
			{
				Id:            "art-dominant-partial",
				Title:         "Untitled", // no +20 title bonus
				Description:   "",         // no +20 description bonus
				ThemeId:       "Kinetic Scribblism",
				Timestamp:     1700000500,
				ImagePath:     "images/art-dominant-partial.jpg",
				ThumbnailPath: "thumbnails/art-dominant-partial.webp",
				Medium:        "Mixed Media", // no +15 medium bonus
			},
			{
				Id:            "art-dominant-complete",
				Title:         "Velocity Study IV",
				Description:   "Rapid graphite spirals across torn paper.",
				ThemeId:       "Kinetic Scribblism",
				Timestamp:     1700000300,
				ImagePath:     "images/art-dominant-complete.jpg",
				ThumbnailPath: "thumbnails/art-dominant-complete.webp",
				Medium:        "Graphite and wax crayon on paper",
			},
			{
				Id:            "art-nondominant-complete",
				Title:         "Void Study I",
				Description:   "Dense black pastel layering.",
				ThemeId:       "Monochrome Nihilism",
				Timestamp:     1700000900, // newer, but lacks +40 dominant movement bonus
				ImagePath:     "images/art-nondominant-complete.jpg",
				ThumbnailPath: "thumbnails/art-nondominant-complete.webp",
				Medium:        "Charcoal on board",
			},
		}

		for _, art := range artworks {
			if err := store.WriteImage(art.GetId(), []byte("jpeg-"+art.GetId())); err != nil {
				t.Fatalf("failed to write image %s: %v", art.GetId(), err)
			}
			if err := store.WriteThumbnail(art.GetId(), []byte("webp-"+art.GetId())); err != nil {
				t.Fatalf("failed to write thumbnail %s: %v", art.GetId(), err)
			}
			pbBytes, err := proto.Marshal(art)
			if err != nil {
				t.Fatalf("failed to marshal %s: %v", art.GetId(), err)
			}
			if err := store.WriteArtworkProtoAtomic(art.GetId(), pbBytes); err != nil {
				t.Fatalf("failed to write proto %s: %v", art.GetId(), err)
			}
		}

		if err := store.GenerateCatalog(); err != nil {
			t.Fatalf("GenerateCatalog failed: %v", err)
		}

		catalogBytes, err := os.ReadFile(filepath.Join(tempDir, "catalog.json"))
		if err != nil {
			t.Fatalf("failed to read catalog.json: %v", err)
		}

		var rawItems []map[string]interface{}
		if err := json.Unmarshal(catalogBytes, &rawItems); err != nil {
			t.Fatalf("failed to unmarshal catalog.json: %v", err)
		}

		if len(rawItems) != 3 {
			t.Fatalf("expected 3 artworks, got %d", len(rawItems))
		}

		repCount := 0
		for _, item := range rawItems {
			id, _ := item["id"].(string)
			isRep, ok := item["is_representative"].(bool)
			if !ok {
				t.Errorf("artwork %q missing boolean is_representative field: %v", id, item["is_representative"])
				continue
			}
			if id == "art-dominant-complete" {
				if !isRep {
					t.Errorf("expected art-dominant-complete to have is_representative=true, got false")
				}
				repCount++
			} else if isRep {
				t.Errorf("expected %q to have is_representative=false, got true", id)
				repCount++
			}
		}
		if repCount != 1 {
			t.Errorf("expected exactly 1 representative artwork, got %d", repCount)
		}
	})

	t.Run("all unannotated/default artworks tie-break by positive timestamp DESC then id ASC", func(t *testing.T) {
		tempDir := t.TempDir()
		store := NewStore(tempDir)

		artworks := []*gallerypb.Artwork{
			{
				Id:        "unannotated-zero-ts",
				Title:     "Untitled",
				ThemeId:   "Early Works",
				Timestamp: 0,
				Medium:    "Mixed Media",
			},
			{
				Id:        "unannotated-b",
				Title:     "   untitled  ",
				ThemeId:   "  early works ",
				Timestamp: 1700000100,
				Medium:    "  mixed media ",
			},
			{
				Id:        "unannotated-a",
				Title:     "",
				ThemeId:   "",
				Timestamp: 1700000100,
				Medium:    "",
			},
		}

		for _, art := range artworks {
			if err := store.WriteImage(art.GetId(), []byte("jpeg-"+art.GetId())); err != nil {
				t.Fatalf("failed to write image %s: %v", art.GetId(), err)
			}
			pbBytes, err := proto.Marshal(art)
			if err != nil {
				t.Fatalf("failed to marshal %s: %v", art.GetId(), err)
			}
			if err := store.WriteArtworkProtoAtomic(art.GetId(), pbBytes); err != nil {
				t.Fatalf("failed to write proto %s: %v", art.GetId(), err)
			}
		}

		if err := store.GenerateCatalog(); err != nil {
			t.Fatalf("GenerateCatalog failed: %v", err)
		}

		catalogBytes, err := os.ReadFile(filepath.Join(tempDir, "catalog.json"))
		if err != nil {
			t.Fatalf("failed to read catalog.json: %v", err)
		}

		var rawItems []map[string]interface{}
		if err := json.Unmarshal(catalogBytes, &rawItems); err != nil {
			t.Fatalf("failed to unmarshal catalog.json: %v", err)
		}

		for _, item := range rawItems {
			id, _ := item["id"].(string)
			isRep, ok := item["is_representative"].(bool)
			if !ok {
				t.Errorf("artwork %q missing boolean is_representative field", id)
				continue
			}
			wantRep := id == "unannotated-a"
			if isRep != wantRep {
				t.Errorf("artwork %q: expected is_representative=%v, got %v", id, wantRep, isRep)
			}
		}
	})
}

func TestDesignateRepresentativeArtwork_ScoringAndTieBreaking(t *testing.T) {
	t.Run("canonical movement tie broken by earliest Phase I-V chronology", func(t *testing.T) {
		// Tie between "Found Object Assemblage" (index 3) and "Domestic Destructionism" (index 1).
		// "Domestic Destructionism" should win as dominantMovement.
		artworks := []CatalogArtwork{
			{
				ID:            "art-found",
				Title:         "Assemblage One",
				Description:   "Found objects",
				ThemeID:       "Found Object Assemblage",
				Timestamp:     1700000500,
				ImagePath:     "images/art-found.jpg",
				ThumbnailPath: "thumbnails/art-found.webp",
				Medium:        "Cardboard and tape",
			},
			{
				ID:            "art-domestic",
				Title:         "Wall Intervention",
				Description:   "Hallway markings",
				ThemeID:       "Domestic Destructionism",
				Timestamp:     1700000100, // older timestamp, but wins via +40 dominant movement bonus
				ImagePath:     "images/art-domestic.jpg",
				ThumbnailPath: "thumbnails/art-domestic.webp",
				Medium:        "Permanent marker on drywall",
			},
		}

		designateRepresentativeArtwork(artworks)

		if !artworks[1].IsRepresentative {
			t.Errorf("expected art-domestic (index 1) to be designated representative")
		}
		if artworks[0].IsRepresentative {
			t.Errorf("expected art-found (index 0) not to be designated representative")
		}
	})

	t.Run("curatorial completeness bonuses (+25 theme, +20 title, +20 description, +15 medium, +10 assets)", func(t *testing.T) {
		// Test individual bonus contributions when no canonical movement is present:
		// Candidate A: non-Early-Works custom theme (+25) = 25
		// Candidate B: non-Untitled title (+20) + description (+20) = 40
		// Candidate C: non-Untitled title (+20) + non-Mixed-Media medium (+15) + valid assets (+10) = 45
		artworks := []CatalogArtwork{
			{
				ID:            "cand-a",
				Title:         "Untitled",
				Description:   "",
				ThemeID:       "Custom Archival Study", // +25
				Timestamp:     1700000300,
				ImagePath:     "/placeholder.svg", // no +10
				ThumbnailPath: "thumbnails/cand-a.webp",
				Medium:        "Mixed Media",
			},
			{
				ID:            "cand-b",
				Title:         "Named Work",       // +20
				Description:   "Detailed critique", // +20
				ThemeID:       "Early Works",       // no +25
				Timestamp:     1700000200,
				ImagePath:     "images/cand-b.jpg",
				ThumbnailPath: "/placeholder.svg", // no +10
				Medium:        "mixed media",      // no +15
			},
			{
				ID:            "cand-c",
				Title:         "Named With Medium And Assets", // +20
				Description:   "   ",                          // no +20
				ThemeID:       "",                             // no +25
				Timestamp:     1700000100,
				ImagePath:     "images/cand-c.jpg",      // +10 (with valid thumbnail)
				ThumbnailPath: "thumbnails/cand-c.webp", // +10
				Medium:        "Oil pastel",             // +15
			},
		}

		designateRepresentativeArtwork(artworks)

		if !artworks[2].IsRepresentative {
			t.Errorf("expected cand-c (score 45) to be representative, got false")
		}
		if artworks[0].IsRepresentative || artworks[1].IsRepresentative {
			t.Errorf("expected cand-a and cand-b to have IsRepresentative=false")
		}
	})

	t.Run("non-positive timestamps rank after positive timestamps and tie-break by ID ascending", func(t *testing.T) {
		artworks := []CatalogArtwork{
			{
				ID:        "art-zero-ts",
				Title:     "Same Score",
				Timestamp: 0,
			},
			{
				ID:        "art-negative-ts",
				Title:     "Same Score",
				Timestamp: -500,
			},
			{
				ID:        "art-positive-ts",
				Title:     "Same Score",
				Timestamp: 100,
			},
		}

		designateRepresentativeArtwork(artworks)

		if !artworks[2].IsRepresentative {
			t.Errorf("expected art-positive-ts to beat zero/negative timestamps")
		}

		// When all timestamps are <= 0, tie-break by ID ascending
		nonPositiveOnly := []CatalogArtwork{
			{
				ID:        "art-z",
				Title:     "Same Score",
				Timestamp: 0,
			},
			{
				ID:        "art-a",
				Title:     "Same Score",
				Timestamp: -10,
			},
		}

		designateRepresentativeArtwork(nonPositiveOnly)

		if !nonPositiveOnly[1].IsRepresentative {
			t.Errorf("expected art-a to win ID tie-break when all timestamps are <= 0")
		}
		if nonPositiveOnly[0].IsRepresentative {
			t.Errorf("expected art-z to have IsRepresentative=false")
		}
	})

	t.Run("empty slice is a safe no-op", func(t *testing.T) {
		var empty []CatalogArtwork
		designateRepresentativeArtwork(empty)
	})
}









