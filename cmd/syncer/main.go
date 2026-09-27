package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/brotherlogic/rose/internal/photos"
	"github.com/brotherlogic/rose/internal/storage"
	"github.com/brotherlogic/rose/internal/vision"
	gallery "github.com/brotherlogic/rose/proto"
	"google.golang.org/protobuf/proto"
)

// PhotoService defines the interface for fetching photos and downloading image binaries.
type PhotoService interface {
	FetchPhotos(ctx context.Context, albumURL string) ([]photos.Photo, error)
	DownloadImage(ctx context.Context, downloadURL string) ([]byte, error)
}

// VisionService defines the interface for analyzing photo content.
type VisionService interface {
	AnalyzeImage(ctx context.Context, img []byte) (string, string, error)
}

// Config holds the configuration for syncer CLI execution.
type Config struct {
	AlbumURL    string
	StoragePath string
}

// parseConfig parses CLI flags and environment variables.
func parseConfig(args []string) (*Config, error) {
	defaultAlbum := os.Getenv("PHOTOS_ALBUM_URL")
	defaultStorage := os.Getenv("STORAGE_PATH")
	if defaultStorage == "" {
		defaultStorage = "/data"
	}

	fs := flag.NewFlagSet("syncer", flag.ContinueOnError)
	albumURL := fs.String("album-url", defaultAlbum, "Google Photos public shared album URL")
	storagePath := fs.String("storage-path", defaultStorage, "Path to storage directory")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	trimmedAlbum := strings.TrimSpace(*albumURL)
	if trimmedAlbum == "" {
		return nil, fmt.Errorf("album URL must be specified via -album-url flag or PHOTOS_ALBUM_URL environment variable")
	}

	return &Config{
		AlbumURL:    trimmedAlbum,
		StoragePath: *storagePath,
	}, nil
}

// isRateLimitError checks if an error represents an HTTP 429 rate limit.
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, photos.ErrRateLimited) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "429") || strings.Contains(strings.ToLower(msg), "rate limit")
}

// Run executes a single synchronization pass over photos from the shared album.
func Run(ctx context.Context, albumURL, storagePath string, photoSvc PhotoService, visionSvc VisionService, store *storage.Store) int {
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		log.Printf("failed to ensure storage directory %s: %v", storagePath, err)
		return 1
	}

	photosList, err := photoSvc.FetchPhotos(ctx, albumURL)
	if err != nil {
		log.Printf("failed to fetch photos: %v", err)
		return 1
	}

	log.Printf("Fetched %d photo(s) to process", len(photosList))
	if len(photosList) == 0 {
		return 0
	}

	var errCount int
	for _, photo := range photosList {
		if ctx.Err() != nil {
			log.Printf("Context cancelled: %v", ctx.Err())
			return 1
		}

		processed, err := store.IsPhotoProcessed(photo.ID)
		if err != nil {
			log.Printf("Error checking sync status for %s: %v", photo.ID, err)
			errCount++
			continue
		}
		if processed {
			log.Printf("Photo %s already processed, skipping", photo.ID)
			continue
		}

		// Stream raw image bytes
		imgBytes, err := photoSvc.DownloadImage(ctx, photo.DownloadURL)
		if err != nil {
			if isRateLimitError(err) {
				log.Printf("Rate limit encountered downloading %s: %v, aborting", photo.ID, err)
				return 1
			}
			log.Printf("Error downloading image %s: %v", photo.ID, err)
			errCount++
			continue
		}

		// Persist raw image to disk
		if err := store.WriteImage(photo.ID, imgBytes); err != nil {
			log.Printf("Error writing raw image %s: %v, aborting immediately", photo.ID, err)
			return 1
		}

		// Pass raw image bytes to vision service
		desc, theme, err := visionSvc.AnalyzeImage(ctx, imgBytes)
		if err != nil {
			if isRateLimitError(err) {
				log.Printf("Rate limit encountered during vision analysis for %s: %v, aborting", photo.ID, err)
				return 1
			}
			log.Printf("Error analyzing image %s: %v", photo.ID, err)
			errCount++
			continue
		}

		artwork := &gallery.Artwork{
			Id:          photo.ID,
			Title:       desc,
			Description: desc,
			ThemeId:     theme,
			Timestamp:   time.Now().Unix(),
			ImagePath:   photo.ID + ".jpg",
		}

		protoData, err := proto.Marshal(artwork)
		if err != nil {
			log.Printf("Error marshaling proto for %s: %v", photo.ID, err)
			errCount++
			continue
		}

		if err := store.WriteArtworkProto(photo.ID, protoData); err != nil {
			log.Printf("Error writing artwork proto for %s: %v, aborting immediately", photo.ID, err)
			return 1
		}

		if err := store.SaveProcessedPhoto(photo.ID); err != nil {
			log.Printf("Error saving processed state for %s: %v, aborting immediately", photo.ID, err)
			return 1
		}

		log.Printf("Successfully processed and stored photo %s", photo.ID)
	}

	if errCount > 0 {
		log.Printf("Syncer completed with %d error(s)", errCount)
		return 1
	}

	log.Printf("Syncer completed successfully")
	return 0
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		log.Printf("configuration error: %v", err)
		os.Exit(1)
	}

	photoSvc := photos.NewService()
	visionSvc := vision.NewService()
	store := storage.NewStore(cfg.StoragePath)

	exitCode := Run(ctx, cfg.AlbumURL, cfg.StoragePath, photoSvc, visionSvc, store)
	os.Exit(exitCode)
}
