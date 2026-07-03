package photos

import (
	"context"
	"testing"
)

func TestFetchPhotos_Success(t *testing.T) {
	service := NewService()
	photos, err := service.FetchPhotos(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(photos) == 0 {
		t.Errorf("expected photos, got none")
	}
}
