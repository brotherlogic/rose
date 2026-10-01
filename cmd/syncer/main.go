package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/brotherlogic/rose/internal/github"
	"github.com/brotherlogic/rose/internal/metrics"
	"github.com/brotherlogic/rose/internal/photos"
	"github.com/brotherlogic/rose/internal/storage"
	"github.com/brotherlogic/rose/internal/thumbnail"
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
	MetricsPort int
}

// parseConfig parses CLI flags and environment variables.
func parseConfig(args []string) (*Config, error) {
	defaultAlbum := os.Getenv("PHOTOS_ALBUM_URL")
	defaultStorage := os.Getenv("STORAGE_PATH")
	if defaultStorage == "" {
		defaultStorage = "/data"
	}
	defaultToken := os.Getenv("GITHUB_TOKEN")

	defaultMetricsPort := 8081
	if envPort := os.Getenv("METRICS_PORT"); envPort != "" {
		p, err := strconv.Atoi(envPort)
		if err != nil {
			return nil, fmt.Errorf("invalid METRICS_PORT: %w", err)
		}
		defaultMetricsPort = p
	}

	fs := flag.NewFlagSet("syncer", flag.ContinueOnError)
	albumURL := fs.String("album-url", defaultAlbum, "Google Photos public shared album URL")
	storagePath := fs.String("storage-path", defaultStorage, "Path to storage directory")
	githubToken := fs.String("github-token", defaultToken, "GitHub access token for failure reporting")
	metricsPort := fs.Int("metrics-port", defaultMetricsPort, "Port for Prometheus metrics HTTP server")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if *metricsPort < 1 || *metricsPort > 65535 {
		return nil, fmt.Errorf("invalid metrics port %d: must be between 1 and 65535", *metricsPort)
	}

	trimmedAlbum := strings.TrimSpace(*albumURL)
	if trimmedAlbum == "" {
		return nil, fmt.Errorf("album URL must be specified via -album-url flag or PHOTOS_ALBUM_URL environment variable")
	}

	return &Config{
		AlbumURL:    trimmedAlbum,
		StoragePath: *storagePath,
		GitHubToken: strings.TrimSpace(*githubToken),
		MetricsPort: *metricsPort,
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

// buildArtwork creates a gallery.Artwork proto instance, checking if a thumbnail exists in storage
// and populating ThumbnailPath if found, or falling back to empty string if absent or store is nil.
func buildArtwork(photoID, description, theme string, store *storage.Store) *gallery.Artwork {
	thumbnailPath := ""
	if store != nil && store.HasThumbnail(photoID) {
		thumbnailPath = "thumbnails/" + photoID + ".webp"
	}

	return &gallery.Artwork{
		Id:            photoID,
		Title:         description,
		Description:   description,
		ThemeId:       theme,
		Timestamp:     time.Now().Unix(),
		ImagePath:     "images/" + photoID + ".jpg",
		ThumbnailPath: thumbnailPath,
	}
}

// Run executes a single synchronization pass over photos from the shared album.
func Run(ctx context.Context, albumURL, storagePath string, photoSvc PhotoService, visionSvc VisionService, store *storage.Store, reporter github.IssueReporter, m *metrics.Metrics) int {
	if m == nil {
		m = metrics.NewMetrics()
	}

	startTime := time.Now()
	defer func() {
		m.ScanStorage(storagePath)
		m.SetSyncDuration(time.Since(startTime))
	}()

	if err := os.MkdirAll(storagePath, 0755); err != nil {
		m.IncSyncErrors()
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

	imagesDir := filepath.Join(storagePath, "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		m.IncSyncErrors()
		log.Printf("failed to ensure images directory %s: %v", imagesDir, err)
		reportFailure(reporter, github.FailureReport{
			Stage:            "Run Initialization",
			Timestamp:        time.Now().UTC(),
			Error:            err,
			PhotosFetched:    0,
			PhotosAttempted:  0,
			PhotosSuccessful: 0,
			LogSummary:       fmt.Sprintf("failed to ensure images directory %s: %v", imagesDir, err),
		})
		return 1
	}

	thumbnailsDir := filepath.Join(storagePath, "thumbnails")
	if err := os.MkdirAll(thumbnailsDir, 0755); err != nil {
		m.IncSyncErrors()
		log.Printf("failed to ensure thumbnails directory %s: %v", thumbnailsDir, err)
		reportFailure(reporter, github.FailureReport{
			Stage:            "Run Initialization",
			Timestamp:        time.Now().UTC(),
			Error:            err,
			PhotosFetched:    0,
			PhotosAttempted:  0,
			PhotosSuccessful: 0,
			LogSummary:       fmt.Sprintf("failed to ensure thumbnails directory %s: %v", thumbnailsDir, err),
		})
		return 1
	}

	m.ScanStorage(storagePath)

	photosList, err := photoSvc.FetchPhotos(ctx, albumURL)
	if err != nil {
		if isRateLimitError(err) {
			log.Printf("Rate limit encountered during album fetch: %v, exiting gracefully", err)
			return 0
		}
		m.IncSyncErrors()
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

	m.SetDiscoveredPhotos(len(photosList))

	fetchedCount := len(photosList)
	log.Printf("Fetched %d photo(s) to process", fetchedCount)
	if fetchedCount == 0 {
		return 0
	}

	var attemptedCount, successCount, errorCount int
	for _, photo := range photosList {
		if ctx.Err() != nil {
			m.IncSyncErrors()
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
			m.IncSyncErrors()
			log.Printf("Error checking sync status for %s: %v", photo.ID, err)
			attemptedCount++
			errorCount++
			continue
		}
		if processed {
			if !store.HasImage(photo.ID) && store.HasLegacyImage(photo.ID) {
				log.Printf("Photo %s found in legacy storage, migrating to images directory", photo.ID)
				if err := store.MigrateLegacyImage(photo.ID); err != nil {
					m.IncSyncErrors()
					log.Printf("Error migrating legacy image for %s: %v, aborting immediately", photo.ID, err)
					reportFailure(reporter, github.FailureReport{
						Stage:            "Storage Persistence",
						Timestamp:        time.Now().UTC(),
						Error:            err,
						PhotosFetched:    fetchedCount,
						PhotosAttempted:  attemptedCount,
						PhotosSuccessful: successCount,
						LogSummary:       fmt.Sprintf("error migrating legacy image for %s: %v", photo.ID, err),
					})
					return 1
				}
				m.IncStoredPhotos()
				m.UpdateStorageBytes(storagePath)
			}

			if store.HasImage(photo.ID) {
				if store.HasThumbnail(photo.ID) {
					log.Printf("Photo %s already processed, skipping", photo.ID)
					continue
				}

				log.Printf("Photo %s already processed but missing thumbnail, backfilling thumbnail", photo.ID)
				imgBytes, err := store.ReadImage(photo.ID)
				if err != nil {
					m.IncSyncErrors()
					log.Printf("Error reading existing image for %s: %v", photo.ID, err)
					errorCount++
					continue
				}

				thumbBytes, err := thumbnail.GenerateThumbnail(imgBytes)
				if err != nil {
					m.IncSyncErrors()
					log.Printf("Error generating thumbnail for %s: %v", photo.ID, err)
					errorCount++
					continue
				}

				if err := store.WriteThumbnail(photo.ID, thumbBytes); err != nil {
					m.IncSyncErrors()
					log.Printf("Error writing thumbnail %s: %v, aborting immediately", photo.ID, err)
					reportFailure(reporter, github.FailureReport{
						Stage:            "Storage Persistence",
						Timestamp:        time.Now().UTC(),
						Error:            err,
						PhotosFetched:    fetchedCount,
						PhotosAttempted:  attemptedCount,
						PhotosSuccessful: successCount,
						LogSummary:       fmt.Sprintf("error writing thumbnail %s: %v", photo.ID, err),
					})
					return 1
				}

				m.IncThumbnailsGenerated()
				m.IncStoredThumbnails()
				m.UpdateStorageBytes(storagePath)

				// Update artwork proto metadata with ThumbnailPath if needed.
				protoBytes, err := store.ReadArtworkProto(photo.ID)
				if err == nil {
					var artwork gallery.Artwork
					if err := proto.Unmarshal(protoBytes, &artwork); err == nil {
						expectedThumbPath := "thumbnails/" + photo.ID + ".webp"
						expectedImagePath := "images/" + photo.ID + ".jpg"
						needsUpdate := false
						if artwork.GetThumbnailPath() != expectedThumbPath {
							artwork.ThumbnailPath = expectedThumbPath
							needsUpdate = true
						}
						if artwork.GetImagePath() != expectedImagePath {
							artwork.ImagePath = expectedImagePath
							needsUpdate = true
						}
						if needsUpdate {
							updatedProtoBytes, err := proto.Marshal(&artwork)
							if err != nil {
								m.IncSyncErrors()
								log.Printf("Error marshaling updated artwork proto for %s: %v", photo.ID, err)
							} else {
								if err := store.WriteArtworkProto(photo.ID, updatedProtoBytes); err != nil {
									m.IncSyncErrors()
									log.Printf("Error writing updated artwork proto for %s: %v, aborting immediately", photo.ID, err)
									reportFailure(reporter, github.FailureReport{
										Stage:            "Storage Persistence",
										Timestamp:        time.Now().UTC(),
										Error:            err,
										PhotosFetched:    fetchedCount,
										PhotosAttempted:  attemptedCount,
										PhotosSuccessful: successCount,
										LogSummary:       fmt.Sprintf("error writing updated artwork proto for %s: %v", photo.ID, err),
									})
									return 1
								}
							}
						}
					} else {
						m.IncSyncErrors()
						log.Printf("Error unmarshaling artwork proto for %s: %v", photo.ID, err)
					}
				} else if !os.IsNotExist(err) {
					m.IncSyncErrors()
					log.Printf("Error reading artwork proto for %s: %v", photo.ID, err)
				}

				continue
			}

			log.Printf("Photo %s marked processed but missing image on disk, treating as unprocessed", photo.ID)
		}

		attemptedCount++

		// Stream raw image bytes
		imgBytes, err := photoSvc.DownloadImage(ctx, photo.DownloadURL)
		if err != nil {
			m.IncSyncErrors()
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
			m.IncSyncErrors()
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
		m.IncPhotosDownloaded()
		m.IncStoredPhotos()
		m.UpdateStorageBytes(storagePath)

		thumbBytes, err := thumbnail.GenerateThumbnail(imgBytes)
		if err != nil {
			m.IncSyncErrors()
			log.Printf("Error generating thumbnail for %s: %v", photo.ID, err)
			errorCount++
			continue
		}

		if err := store.WriteThumbnail(photo.ID, thumbBytes); err != nil {
			m.IncSyncErrors()
			log.Printf("Error writing thumbnail %s: %v, aborting immediately", photo.ID, err)
			reportFailure(reporter, github.FailureReport{
				Stage:            "Storage Persistence",
				Timestamp:        time.Now().UTC(),
				Error:            err,
				PhotosFetched:    fetchedCount,
				PhotosAttempted:  attemptedCount,
				PhotosSuccessful: successCount,
				LogSummary:       fmt.Sprintf("error writing thumbnail %s: %v", photo.ID, err),
			})
			return 1
		}
		m.IncThumbnailsGenerated()
		m.IncStoredThumbnails()
		m.UpdateStorageBytes(storagePath)

		// Pass raw image bytes to vision service
		desc, theme, err := visionSvc.AnalyzeImage(ctx, imgBytes)
		if err != nil {
			m.IncSyncErrors()
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

		artwork := buildArtwork(photo.ID, desc, theme, store)

		protoData, err := proto.Marshal(artwork)
		if err != nil {
			m.IncSyncErrors()
			log.Printf("Error marshaling proto for %s: %v", photo.ID, err)
			errorCount++
			continue
		}

		if err := store.WriteArtworkProto(photo.ID, protoData); err != nil {
			m.IncSyncErrors()
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
			m.IncSyncErrors()
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

// startMetricsServer binds the TCP listener on the configured port and starts an HTTP server serving the handler in a background goroutine.
func startMetricsServer(port int, handler http.Handler) (net.Listener, *http.Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to bind metrics port %d: %w", port, err)
	}

	server := &http.Server{Handler: handler}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("metrics server error: %v", err)
		}
	}()

	return listener, server, nil
}

// runGracePeriod waits for the specified scrape grace period or immediately terminates if context is cancelled.
func runGracePeriod(ctx context.Context, duration time.Duration) {
	select {
	case <-time.After(duration):
		if duration == 60*time.Second {
			log.Printf("Completed 60s Prometheus scrape grace period")
		} else {
			log.Printf("Completed %v Prometheus scrape grace period", duration)
		}
	case <-ctx.Done():
		log.Printf("Termination signal received during grace period, shutting down immediately")
	}
}

// shutdownServer gracefully shuts down the HTTP server within the specified timeout.
func shutdownServer(server *http.Server, timeout time.Duration) error {
	if server == nil {
		return nil
	}
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), timeout)
	defer cancelShutdown()
	return server.Shutdown(shutdownCtx)
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		log.Printf("configuration error: %v", err)
		os.Exit(1)
	}

	m := metrics.NewMetrics()
	m.ScanStorage(cfg.StoragePath)
	_, server, err := startMetricsServer(cfg.MetricsPort, m.Handler())
	if err != nil {
		log.Printf("fatal: metrics port %d already bound or cannot be listened on: %v", cfg.MetricsPort, err)
		os.Exit(1)
	}

	photoSvc := photos.NewService()
	visionSvc := vision.NewService()
	store := storage.NewStore(cfg.StoragePath)

	var reporter github.IssueReporter
	if cfg.GitHubToken != "" {
		reporter = github.NewClient(cfg.GitHubToken)
	}

	exitCode := Run(ctx, cfg.AlbumURL, cfg.StoragePath, photoSvc, visionSvc, store, reporter, m)

	runGracePeriod(ctx, 60*time.Second)

	_ = shutdownServer(server, 5*time.Second)
	os.Exit(exitCode)
}

