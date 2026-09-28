package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	BasePath string
}

func NewStore(basePath string) *Store {
	return &Store{BasePath: basePath}
}

func (s *Store) syncStatePath() string {
	return filepath.Join(s.BasePath, ".sync-state.json")
}

func (s *Store) loadSyncState() (map[string]bool, error) {
	data, err := os.ReadFile(s.syncStatePath())
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]bool), nil
		}
		return nil, err
	}

	var state map[string]bool
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *Store) saveSyncState(state map[string]bool) error {
	if err := os.MkdirAll(s.BasePath, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(s.syncStatePath(), data, 0644)
}

func (s *Store) IsPhotoProcessed(id string) (bool, error) {
	state, err := s.loadSyncState()
	if err != nil {
		return false, err
	}
	return state[id], nil
}

func (s *Store) SaveProcessedPhoto(id string) error {
	state, err := s.loadSyncState()
	if err != nil {
		return err
	}
	state[id] = true
	return s.saveSyncState(state)
}

func (s *Store) WriteArtworkProto(id string, data []byte) error {
	if err := os.MkdirAll(s.BasePath, 0755); err != nil {
		return err
	}
	filePath := filepath.Join(s.BasePath, id+".proto.bin")
	return os.WriteFile(filePath, data, 0644)
}

func validateID(id string) error {
	if id == "" || id != filepath.Base(id) || id == "." || strings.Contains(id, "/") || strings.Contains(id, "\\") || strings.Contains(id, "..") {
		return fmt.Errorf("invalid photo id %q: path traversal or invalid characters detected", id)
	}
	return nil
}

func (s *Store) WriteImage(id string, data []byte) error {
	if err := validateID(id); err != nil {
		return err
	}

	imagesDir := filepath.Join(s.BasePath, "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(imagesDir, id+".jpg")
	return os.WriteFile(filePath, data, 0644)
}

func (s *Store) WriteThumbnail(id string, data []byte) error {
	if err := validateID(id); err != nil {
		return err
	}

	thumbnailsDir := filepath.Join(s.BasePath, "thumbnails")
	if err := os.MkdirAll(thumbnailsDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(thumbnailsDir, id+".webp")
	return os.WriteFile(filePath, data, 0644)
}


