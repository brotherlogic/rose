package proto

import (
	"testing"
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
