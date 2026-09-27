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

	"github.com/brotherlogic/rose/internal/github"
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
	GitHubToken string
}

// parseConfig parses CLI flags and environment variables.
func parseConfig(args []string) (*Config, error) {
	defaultAlbum := os.Getenv("PHOTOS_ALBUM_URL")
	defaultStorage := os.Getenv("STORAGE_PATH")
	if defaultStorage == "" {
		defaultStorage = "/data"
	}
	defaultToken := os.Getenv("GITHUB_TOKEN")

	fs := flag.NewFlagSet("syncer", flag.ContinueOnError)
	albumURL := fs.String("album-url", defaultAlbum, "Google Photos public shared album URL")
	storagePath := fs.String("storage-path", defaultStorage, "Path to storage directory")
	githubToken := fs.String("github-token", defaultToken, "GitHub access token for failure reporting")

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
		GitHubToken: strings.TrimSpace(*githubToken),
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

func reportFailure(reporter github.IssueReporter, report github.FailureReport) {
	if reporter == nil {
		fmt.Fprintln(os.Stderr, "warning: GitHub token is empty or reporter not configured, skipping failure reporting")
		return
	}

	reportCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	hasActive, err := reporter.HasActiveFailureIssue(reportCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to check active failure issues: %v\n", err)
		return
	}
	if hasActive {
		log.Printf("Active failure issue already exists, skipping issue creation")
		return
	}

	if err := reporter.CreateFailureIssue(reportCtx, report); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create failure issue: %v\n", err)
		return
	}
	log.Printf("Successfully created failure issue for stage: %s", report.Stage)
}

// Run executes a single synchronization pass over photos from the shared album.
func Run(ctx context.Context, albumURL, storagePath string, photoSvc PhotoService, visionSvc VisionService, store *storage.Store, reporter github.IssueReporter) int {
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		log.Printf("failed to ensure storage directory %s: %v", storagePath, err)
		reportFailure(reporter, github.FailureReport{
			Stage:            "Run Initialization",
			Timestamp:        time.Now().UTC(),
			Error:            err,
			PhotosFetched:    0,
			PhotosAttempted:  0,
			PhotosSuccessful: 0,
			LogSummary:       fmt.Sprintf("failed to ensure storage directory %s: %v", storagePath, err),
		})
		return 1
	}

	photosList, err := photoSvc.FetchPhotos(ctx, albumURL)
	if err != nil {
		log.Printf("failed to fetch photos: %v", err)
		stage := "Album Fetch"
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			stage = "Context Cancelled"
		}
		reportFailure(reporter, github.FailureReport{
			Stage:            stage,
			Timestamp:        time.Now().UTC(),
			Error:            err,
			PhotosFetched:    0,
			PhotosAttempted:  0,
			PhotosSuccessful: 0,
			LogSummary:       fmt.Sprintf("failed to fetch photos: %v", err),
		})
		return 1
	}

	fetchedCount := len(photosList)
	log.Printf("Fetched %d photo(s) to process", fetchedCount)
	if fetchedCount == 0 {
		return 0
	}

	var attemptedCount, successCount, errorCount int
	for _, photo := range photosList {
		if ctx.Err() != nil {
			log.Printf("Context cancelled: %v", ctx.Err())
			reportFailure(reporter, github.FailureReport{
				Stage:            "Context Cancelled",
				Timestamp:        time.Now().UTC(),
				Error:            ctx.Err(),
				PhotosFetched:    fetchedCount,
				PhotosAttempted:  attemptedCount,
				PhotosSuccessful: successCount,
				LogSummary:       fmt.Sprintf("context cancelled during photo processing: %v", ctx.Err()),
			})
			return 1
		}

		processed, err := store.IsPhotoProcessed(photo.ID)
		if err != nil {
			log.Printf("Error checking sync status for %s: %v", photo.ID, err)
			attemptedCount++
			errorCount++
			continue
		}
		if processed {
			log.Printf("Photo %s already processed, skipping", photo.ID)
			continue
		}

		attemptedCount++

		// Stream raw image bytes
		imgBytes, err := photoSvc.DownloadImage(ctx, photo.DownloadURL)
		if err != nil {
			if isRateLimitError(err) {
				log.Printf("Rate limit encountered downloading %s: %v, aborting", photo.ID, err)
				reportFailure(reporter, github.FailureReport{
					Stage:            "Photo Processing - Rate Limited",
					Timestamp:        time.Now().UTC(),
					Error:            err,
					PhotosFetched:    fetchedCount,
					PhotosAttempted:  attemptedCount,
					PhotosSuccessful: successCount,
					LogSummary:       fmt.Sprintf("rate limit encountered downloading %s: %v", photo.ID, err),
				})
				return 1
			}
			log.Printf("Error downloading image %s: %v", photo.ID, err)
			errorCount++
			continue
		}

		// Persist raw image to disk
		if err := store.WriteImage(photo.ID, imgBytes); err != nil {
			log.Printf("Error writing raw image %s: %v, aborting immediately", photo.ID, err)
			reportFailure(reporter, github.FailureReport{
				Stage:            "Storage Persistence",
				Timestamp:        time.Now().UTC(),
				Error:            err,
				PhotosFetched:    fetchedCount,
				PhotosAttempted:  attemptedCount,
				PhotosSuccessful: successCount,
				LogSummary:       fmt.Sprintf("error writing raw image %s: %v", photo.ID, err),
			})
			return 1
		}

		// Pass raw image bytes to vision service
		desc, theme, err := visionSvc.AnalyzeImage(ctx, imgBytes)
		if err != nil {
			if isRateLimitError(err) {
				log.Printf("Rate limit encountered during vision analysis for %s: %v, aborting", photo.ID, err)
				reportFailure(reporter, github.FailureReport{
					Stage:            "Photo Processing - Rate Limited",
					Timestamp:        time.Now().UTC(),
					Error:            err,
					PhotosFetched:    fetchedCount,
					PhotosAttempted:  attemptedCount,
					PhotosSuccessful: successCount,
					LogSummary:       fmt.Sprintf("rate limit encountered during vision analysis for %s: %v", photo.ID, err),
				})
				return 1
			}
			log.Printf("Error analyzing image %s: %v", photo.ID, err)
			errorCount++
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
			errorCount++
			continue
		}

		if err := store.WriteArtworkProto(photo.ID, protoData); err != nil {
			log.Printf("Error writing artwork proto for %s: %v, aborting immediately", photo.ID, err)
			reportFailure(reporter, github.FailureReport{
				Stage:            "Storage Persistence",
				Timestamp:        time.Now().UTC(),
				Error:            err,
				PhotosFetched:    fetchedCount,
				PhotosAttempted:  attemptedCount,
				PhotosSuccessful: successCount,
				LogSummary:       fmt.Sprintf("error writing artwork proto for %s: %v", photo.ID, err),
			})
			return 1
		}

		if err := store.SaveProcessedPhoto(photo.ID); err != nil {
			log.Printf("Error saving processed state for %s: %v, aborting immediately", photo.ID, err)
			reportFailure(reporter, github.FailureReport{
				Stage:            "Storage Persistence",
				Timestamp:        time.Now().UTC(),
				Error:            err,
				PhotosFetched:    fetchedCount,
				PhotosAttempted:  attemptedCount,
				PhotosSuccessful: successCount,
				LogSummary:       fmt.Sprintf("error saving processed state for %s: %v", photo.ID, err),
			})
			return 1
		}

		successCount++
		log.Printf("Successfully processed and stored photo %s", photo.ID)
	}

	if attemptedCount > 0 && successCount == 0 {
		log.Printf("Syncer failed: 0 of %d attempted photo(s) synced", attemptedCount)
		reportFailure(reporter, github.FailureReport{
			Stage:            "Photo Processing - 0 Synced",
			Timestamp:        time.Now().UTC(),
			Error:            fmt.Errorf("zero photos synced out of %d attempted", attemptedCount),
			PhotosFetched:    fetchedCount,
			PhotosAttempted:  attemptedCount,
			PhotosSuccessful: 0,
			LogSummary:       fmt.Sprintf("syncer failed to sync any photos (attempted %d, failed %d)", attemptedCount, errorCount),
		})
		return 1
	}

	if errorCount > 0 {
		log.Printf("Syncer completed with %d error(s)", errorCount)
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

	var reporter github.IssueReporter
	if cfg.GitHubToken != "" {
		reporter = github.NewClient(cfg.GitHubToken)
	}

	exitCode := Run(ctx, cfg.AlbumURL, cfg.StoragePath, photoSvc, visionSvc, store, reporter)
	os.Exit(exitCode)
}
