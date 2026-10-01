package proto

import (
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestArtwork(t *testing.T) {
	artwork := &Artwork{
		Id:    "123",
		Title: "Test Artwork",
	}

	if artwork.GetId() != "123" {
		t.Errorf("Expected ID 123, got %s", artwork.GetId())
	}
}

func TestArtwork_ThumbnailPath(t *testing.T) {
	artwork := &Artwork{
		Id:            "art-1",
		Title:         "Thumbnail Test",
		ThumbnailPath: "thumbnails/art-1.webp",
	}

	if artwork.GetThumbnailPath() != "thumbnails/art-1.webp" {
		t.Errorf("Expected ThumbnailPath 'thumbnails/art-1.webp', got %q", artwork.GetThumbnailPath())
	}
}

func TestArtwork_RoundTrip(t *testing.T) {
	original := &Artwork{
		Id:            "round-trip-1",
		Title:         "Mona Lisa",
		Description:   "Famous painting",
		ThemeId:       "renaissance",
		Timestamp:     1700000000,
		ImagePath:     "images/mona-lisa.jpg",
		ThumbnailPath: "thumbnails/123.webp",
		Medium:        "Wax pigment on reclaimed cardboard",
	}

	data, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal Artwork: %v", err)
	}

	unmarshaled := &Artwork{}
	if err := proto.Unmarshal(data, unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal Artwork: %v", err)
	}

	if unmarshaled.GetId() != original.GetId() {
		t.Errorf("ID mismatch: got %q, want %q", unmarshaled.GetId(), original.GetId())
	}
	if unmarshaled.GetTitle() != original.GetTitle() {
		t.Errorf("Title mismatch: got %q, want %q", unmarshaled.GetTitle(), original.GetTitle())
	}
	if unmarshaled.GetDescription() != original.GetDescription() {
		t.Errorf("Description mismatch: got %q, want %q", unmarshaled.GetDescription(), original.GetDescription())
	}
	if unmarshaled.GetThemeId() != original.GetThemeId() {
		t.Errorf("ThemeId mismatch: got %q, want %q", unmarshaled.GetThemeId(), original.GetThemeId())
	}
	if unmarshaled.GetTimestamp() != original.GetTimestamp() {
		t.Errorf("Timestamp mismatch: got %d, want %d", unmarshaled.GetTimestamp(), original.GetTimestamp())
	}
	if unmarshaled.GetImagePath() != original.GetImagePath() {
		t.Errorf("ImagePath mismatch: got %q, want %q", unmarshaled.GetImagePath(), original.GetImagePath())
	}
	if unmarshaled.GetThumbnailPath() != original.GetThumbnailPath() {
		t.Errorf("ThumbnailPath mismatch: got %q, want %q", unmarshaled.GetThumbnailPath(), original.GetThumbnailPath())
	}
	if unmarshaled.GetMedium() != original.GetMedium() {
		t.Errorf("Medium mismatch: got %q, want %q", unmarshaled.GetMedium(), original.GetMedium())
	}
}

func TestArtwork_Medium(t *testing.T) {
	artwork := &Artwork{
		Id:     "art-medium-1",
		Title:  "Medium Test",
		Medium: "Wax pigment on reclaimed cardboard",
	}

	if artwork.GetMedium() != "Wax pigment on reclaimed cardboard" {
		t.Errorf("Expected Medium 'Wax pigment on reclaimed cardboard', got %q", artwork.GetMedium())
	}
}

func TestArtwork_BackwardCompatibility(t *testing.T) {
	legacyArtwork := &Artwork{
		Id:          "legacy-1",
		Title:       "Legacy Artwork",
		Description: "Encoded without field 7 and field 8",
		ThemeId:     "classic",
		Timestamp:   1600000000,
		ImagePath:   "images/legacy.jpg",
	}

	data, err := proto.Marshal(legacyArtwork)
	if err != nil {
		t.Fatalf("Failed to marshal legacy Artwork: %v", err)
	}

	unmarshaled := &Artwork{}
	if err := proto.Unmarshal(data, unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal legacy payload: %v", err)
	}

	if unmarshaled.GetThumbnailPath() != "" {
		t.Errorf("Expected empty ThumbnailPath for legacy payload, got %q", unmarshaled.GetThumbnailPath())
	}
	if unmarshaled.GetMedium() != "" {
		t.Errorf("Expected empty Medium for legacy payload, got %q", unmarshaled.GetMedium())
	}
}

func TestArtwork_MalformedPayload(t *testing.T) {
	corrupted := []byte{0xff, 0xff, 0xff, 0xff}
	artwork := &Artwork{}
	if err := proto.Unmarshal(corrupted, artwork); err == nil {
		t.Fatalf("Expected error unmarshaling corrupted payload, got nil")
	}
}


