package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	gallerypb "github.com/brotherlogic/rose/proto"
	"google.golang.org/protobuf/proto"
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
	if s == nil {
		return fmt.Errorf("nil store")
	}
	if err := os.MkdirAll(s.BasePath, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	tmpPath := filepath.Join(s.BasePath, fmt.Sprintf(".sync-state.json.tmp-%d", time.Now().UnixNano()))
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}

	cleanUp := true
	defer func() {
		if cleanUp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, s.syncStatePath()); err != nil {
		return err
	}

	cleanUp = false
	return nil
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

func (s *Store) WriteArtworkProtoAtomic(id string, data []byte) error {
	if s == nil {
		return fmt.Errorf("nil store")
	}
	if err := validateID(id); err != nil {
		return err
	}
	if err := os.MkdirAll(s.BasePath, 0755); err != nil {
		return err
	}

	tmpPath := filepath.Join(s.BasePath, fmt.Sprintf(".%s.proto.bin.tmp-%d", id, time.Now().UnixNano()))
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}

	cleanUp := true
	defer func() {
		if cleanUp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	destPath := filepath.Join(s.BasePath, id+".proto.bin")
	if err := os.Rename(tmpPath, destPath); err != nil {
		return err
	}

	cleanUp = false
	return nil
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

// ReadImage reads the raw image bytes for the given photo ID from the images directory.
func (s *Store) ReadImage(id string) ([]byte, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	filePath := filepath.Join(s.BasePath, "images", id+".jpg")
	return os.ReadFile(filePath)
}

// HasThumbnail returns true if a non-directory thumbnail file exists for the given photo ID.
func (s *Store) HasThumbnail(id string) bool {
	if s == nil {
		return false
	}
	if err := validateID(id); err != nil {
		return false
	}
	filePath := filepath.Join(s.BasePath, "thumbnails", id+".webp")
	fi, err := os.Stat(filePath)
	return err == nil && !fi.IsDir()
}

// ReadArtworkProto reads the serialized protobuf bytes for the given photo ID.
func (s *Store) ReadArtworkProto(id string) ([]byte, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	filePath := filepath.Join(s.BasePath, id+".proto.bin")
	return os.ReadFile(filePath)
}

// HasArtworkProto returns true if a non-directory artwork proto file exists for the given photo ID.
func (s *Store) HasArtworkProto(id string) bool {
	if s == nil {
		return false
	}
	if err := validateID(id); err != nil {
		return false
	}
	filePath := filepath.Join(s.BasePath, id+".proto.bin")
	fi, err := os.Stat(filePath)
	return err == nil && !fi.IsDir()
}

// HasImage returns true if a non-directory raw image file exists for the given photo ID.
func (s *Store) HasImage(id string) bool {
	if s == nil {
		return false
	}
	if err := validateID(id); err != nil {
		return false
	}
	filePath := filepath.Join(s.BasePath, "images", id+".jpg")
	fi, err := os.Stat(filePath)
	return err == nil && !fi.IsDir()
}

// HasLegacyImage returns true if a non-directory legacy image file exists at <basePath>/<id>.jpg.
func (s *Store) HasLegacyImage(id string) bool {
	if s == nil {
		return false
	}
	if err := validateID(id); err != nil {
		return false
	}
	filePath := filepath.Join(s.BasePath, id+".jpg")
	fi, err := os.Stat(filePath)
	return err == nil && !fi.IsDir()
}

// MigrateLegacyImage moves a legacy image file from <basePath>/<id>.jpg to <basePath>/images/<id>.jpg.
func (s *Store) MigrateLegacyImage(id string) error {
	if s == nil {
		return fmt.Errorf("nil store")
	}
	if err := validateID(id); err != nil {
		return err
	}
	legacyPath := filepath.Join(s.BasePath, id+".jpg")
	imagesDir := filepath.Join(s.BasePath, "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return err
	}
	targetPath := filepath.Join(imagesDir, id+".jpg")
	if err := os.Rename(legacyPath, targetPath); err != nil {
		data, err := os.ReadFile(legacyPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return err
		}
		_ = os.Remove(legacyPath)
	}
	return nil
}

// BackfillCandidate represents a photo candidate for metadata backfill.
type BackfillCandidate struct {
	ID        string
	Timestamp int64
}

// FindUnannotatedPhotos scans the store for photos that are missing annotations
// (missing .proto.bin or empty medium field) and returns them sorted chronologically.
func (s *Store) FindUnannotatedPhotos() ([]BackfillCandidate, error) {
	if s == nil {
		return nil, fmt.Errorf("nil store")
	}

	imagesDir := filepath.Join(s.BasePath, "images")
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackfillCandidate{}, nil
		}
		return nil, err
	}

	candidates := make([]BackfillCandidate, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jpg") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".jpg")
		if err := validateID(id); err != nil {
			continue
		}

		info, err := entry.Info()
		if err != nil || info.IsDir() {
			continue
		}
		modTime := info.ModTime().Unix()

		protoBytes, err := s.ReadArtworkProto(id)
		if err != nil {
			if os.IsNotExist(err) {
				candidates = append(candidates, BackfillCandidate{
					ID:        id,
					Timestamp: modTime,
				})
				continue
			}
			return nil, err
		}

		var artwork gallerypb.Artwork
		if err := proto.Unmarshal(protoBytes, &artwork); err != nil {
			candidates = append(candidates, BackfillCandidate{
				ID:        id,
				Timestamp: modTime,
			})
			continue
		}

		if artwork.GetMedium() == "" {
			ts := artwork.GetTimestamp()
			if ts == 0 {
				ts = modTime
			}
			candidates = append(candidates, BackfillCandidate{
				ID:        id,
				Timestamp: ts,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Timestamp != candidates[j].Timestamp {
			return candidates[i].Timestamp < candidates[j].Timestamp
		}
		return candidates[i].ID < candidates[j].ID
	})

	return candidates, nil
}





