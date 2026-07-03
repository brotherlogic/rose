package vision

import "context"

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) AnalyzeImage(ctx context.Context, img []byte) (string, string, error) {
	// Mock implementation
	return "A beautiful landscape", "Nature", nil
}
