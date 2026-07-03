package vision

import (
	"context"
	"testing"
)

func TestAnalyzeImage_Success(t *testing.T) {
	service := NewService()
	desc, theme, err := service.AnalyzeImage(context.Background(), []byte("fake-image"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if desc == "" || theme == "" {
		t.Errorf("expected description and theme, got empty strings")
	}
}
