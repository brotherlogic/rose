package metrics

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics encapsulates Prometheus collectors and a dedicated registry for rose-syncer.
type Metrics struct {
	registry            *prometheus.Registry
	photosDiscovered    prometheus.Gauge
	photosDownloaded    prometheus.Counter
	thumbnailsGenerated prometheus.Counter
	photosStored        prometheus.Gauge
	thumbnailsStored    prometheus.Gauge
	storageBytes        *prometheus.GaugeVec
	syncDuration        prometheus.Gauge
	syncErrors          prometheus.Counter
}

// NewMetrics initializes and registers all syncer operational telemetry collectors.
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()

	photosDiscovered := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rose_syncer_photos_discovered_total",
		Help: "Total photos discovered in the Google Photos shared album during the pass.",
	})

	photosDownloaded := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rose_syncer_photos_downloaded_total",
		Help: "Total photos successfully downloaded to disk.",
	})

	thumbnailsGenerated := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rose_syncer_thumbnails_generated_total",
		Help: "Total WebP thumbnails generated.",
	})

	photosStored := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rose_syncer_photos_stored",
		Help: "Total photos physically stored on disk.",
	})

	thumbnailsStored := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rose_syncer_thumbnails_stored",
		Help: "Total thumbnails physically stored on disk.",
	})

	storageBytes := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "rose_syncer_storage_bytes",
		Help: "Total bytes consumed in storage directories.",
	}, []string{"type"})

	syncDuration := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rose_syncer_sync_duration_seconds",
		Help: "Elapsed execution time of the synchronization run.",
	})

	syncErrors := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rose_syncer_sync_errors_total",
		Help: "Count of errors encountered during discovery, download, or thumbnail generation.",
	})

	reg.MustRegister(
		photosDiscovered,
		photosDownloaded,
		thumbnailsGenerated,
		photosStored,
		thumbnailsStored,
		storageBytes,
		syncDuration,
		syncErrors,
	)

	photosStored.Set(0)
	thumbnailsStored.Set(0)
	storageBytes.WithLabelValues("images").Set(0)
	storageBytes.WithLabelValues("thumbnails").Set(0)

	return &Metrics{
		registry:            reg,
		photosDiscovered:    photosDiscovered,
		photosDownloaded:    photosDownloaded,
		thumbnailsGenerated: thumbnailsGenerated,
		photosStored:        photosStored,
		thumbnailsStored:    thumbnailsStored,
		storageBytes:        storageBytes,
		syncDuration:        syncDuration,
		syncErrors:          syncErrors,
	}
}

// Registry returns the underlying Prometheus registry.
func (m *Metrics) Registry() *prometheus.Registry {
	if m == nil {
		return nil
	}
	return m.registry
}

// Handler returns an HTTP handler serving Prometheus metrics.
func (m *Metrics) Handler() http.Handler {
	if m == nil || m.registry == nil {
		return http.NotFoundHandler()
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// SetDiscoveredPhotos sets the count of photos discovered in the album.
func (m *Metrics) SetDiscoveredPhotos(count int) {
	if m == nil || m.photosDiscovered == nil {
		return
	}
	m.photosDiscovered.Set(float64(count))
}

// IncPhotosDownloaded increments the count of downloaded photos.
func (m *Metrics) IncPhotosDownloaded() {
	if m == nil || m.photosDownloaded == nil {
		return
	}
	m.photosDownloaded.Inc()
}

// IncThumbnailsGenerated increments the count of generated thumbnails.
func (m *Metrics) IncThumbnailsGenerated() {
	if m == nil || m.thumbnailsGenerated == nil {
		return
	}
	m.thumbnailsGenerated.Inc()
}

// IncSyncErrors increments the count of synchronization errors.
func (m *Metrics) IncSyncErrors() {
	if m == nil || m.syncErrors == nil {
		return
	}
	m.syncErrors.Inc()
}

// SetSyncDuration records the elapsed synchronization execution duration in seconds.
func (m *Metrics) SetSyncDuration(d time.Duration) {
	if m == nil || m.syncDuration == nil {
		return
	}
	m.syncDuration.Set(d.Seconds())
}

// SetStoredPhotos sets the count of photos physically stored on disk.
func (m *Metrics) SetStoredPhotos(count int) {
	if m == nil || m.photosStored == nil {
		return
	}
	m.photosStored.Set(float64(count))
}

// SetStoredThumbnails sets the count of thumbnails physically stored on disk.
func (m *Metrics) SetStoredThumbnails(count int) {
	if m == nil || m.thumbnailsStored == nil {
		return
	}
	m.thumbnailsStored.Set(float64(count))
}

// IncStoredPhotos increments stored photos gauge by 1.
func (m *Metrics) IncStoredPhotos() {
	if m == nil || m.photosStored == nil {
		return
	}
	m.photosStored.Inc()
}

// IncStoredThumbnails increments stored thumbnails gauge by 1.
func (m *Metrics) IncStoredThumbnails() {
	if m == nil || m.thumbnailsStored == nil {
		return
	}
	m.thumbnailsStored.Inc()
}

// ScanStorage performs a single walk of images and thumbnails directories,
// calculating both counts and total bytes, setting all respective gauges.
func (m *Metrics) ScanStorage(basePath string) {
	if m == nil {
		return
	}

	imagesCount, imagesBytes := scanDir(filepath.Join(basePath, "images"))
	thumbnailsCount, thumbnailsBytes := scanDir(filepath.Join(basePath, "thumbnails"))

	if m.photosStored != nil {
		m.photosStored.Set(float64(imagesCount))
	}
	if m.thumbnailsStored != nil {
		m.thumbnailsStored.Set(float64(thumbnailsCount))
	}
	if m.storageBytes != nil {
		m.storageBytes.WithLabelValues("images").Set(imagesBytes)
		m.storageBytes.WithLabelValues("thumbnails").Set(thumbnailsBytes)
	}
}

// UpdateStorageBytes recursively calculates directory sizes for images and thumbnails and updates the gauge.
func (m *Metrics) UpdateStorageBytes(basePath string) {
	if m == nil || m.storageBytes == nil {
		return
	}

	_, imagesBytes := scanDir(filepath.Join(basePath, "images"))
	_, thumbnailsBytes := scanDir(filepath.Join(basePath, "thumbnails"))

	m.storageBytes.WithLabelValues("images").Set(imagesBytes)
	m.storageBytes.WithLabelValues("thumbnails").Set(thumbnailsBytes)
}

func scanDir(path string) (int, float64) {
	var count int
	var totalSize int64
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		count++
		totalSize += info.Size()
		return nil
	})
	return count, float64(totalSize)
}

func dirSize(path string) float64 {
	_, size := scanDir(path)
	return size
}

