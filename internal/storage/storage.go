package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	filePath := filepath.Join(s.BasePath, id+".proto.bin")
	return os.WriteFile(filePath, data, 0644)
}
