package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brotherlogic/rose/internal/photos"
	"github.com/brotherlogic/rose/internal/storage"
	"github.com/brotherlogic/rose/internal/vision"
	gallery "github.com/brotherlogic/rose/proto"
	"google.golang.org/protobuf/proto"
)

// PhotoService defines the interface for fetching photos.
type PhotoService interface {
	FetchPhotos(ctx context.Context) ([]string, error)
}

// VisionService defines the interface for analyzing photo content.
type VisionService interface {
	AnalyzeImage(ctx context.Context, img []byte) (string, string, error)
}

// parseStoragePath parses the -storage-path flag with fallback to STORAGE_PATH env var, then /data.
func parseStoragePath(args []string) (string, error) {
	defaultStorage := os.Getenv("STORAGE_PATH")
	if defaultStorage == "" {
		defaultStorage = "/data"
	}

	fs := flag.NewFlagSet("syncer", flag.ContinueOnError)
	storagePath := fs.String("storage-path", defaultStorage, "Path to storage directory")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	return *storagePath, nil
}

// Run executes a single synchronization pass over photos.
func Run(ctx context.Context, storagePath string, photoSvc PhotoService, visionSvc VisionService, store *storage.Store) int {
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		log.Printf("failed to ensure storage directory %s: %v", storagePath, err)
		return 1
	}

	photosList, err := photoSvc.FetchPhotos(ctx)
	if err != nil {
		log.Printf("failed to fetch photos: %v", err)
		return 1
	}

	log.Printf("Fetched %d photo(s) to process", len(photosList))
	if len(photosList) == 0 {
		return 0
	}

	var errCount int
	for _, id := range photosList {
		select {
		case <-ctx.Done():
			log.Printf("Context cancelled: %v", ctx.Err())
			return 1
		default:
		}

		processed, err := store.IsPhotoProcessed(id)
		if err != nil {
			log.Printf("Error checking sync status for %s: %v", id, err)
			errCount++
			continue
		}
		if processed {
			log.Printf("Photo %s already processed, skipping", id)
			continue
		}

		desc, theme, err := visionSvc.AnalyzeImage(ctx, nil)
		if err != nil {
			log.Printf("Error analyzing image %s: %v", id, err)
			errCount++
			continue
		}

		artwork := &gallery.Artwork{
			Id:          id,
			Title:       desc,
			Description: desc,
			ThemeId:     theme,
			Timestamp:   time.Now().Unix(),
			ImagePath:   id,
		}

		data, err := proto.Marshal(artwork)
		if err != nil {
			log.Printf("Error marshaling proto for %s: %v", id, err)
			errCount++
			continue
		}

		if err := store.WriteArtworkProto(id, data); err != nil {
			log.Printf("Error writing artwork proto for %s: %v", id, err)
			errCount++
			continue
		}

		if err := store.SaveProcessedPhoto(id); err != nil {
			log.Printf("Error saving processed state for %s: %v", id, err)
			errCount++
			continue
		}

		log.Printf("Successfully processed and stored photo %s", id)
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

	storagePath, err := parseStoragePath(os.Args[1:])
	if err != nil {
		log.Fatalf("failed to parse flags: %v", err)
	}

	photoSvc := photos.NewService()
	visionSvc := vision.NewService()
	store := storage.NewStore(storagePath)

	exitCode := Run(ctx, storagePath, photoSvc, visionSvc, store)
	os.Exit(exitCode)
}
