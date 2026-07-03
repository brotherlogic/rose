package photos

import "context"

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) FetchPhotos(ctx context.Context) ([]string, error) {
	// Mock implementation
	return []string{"photo1.jpg", "photo2.jpg"}, nil
}
