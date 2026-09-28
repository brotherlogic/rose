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
		storageBytes,
		syncDuration,
		syncErrors,
	)

	storageBytes.WithLabelValues("images").Set(0)
	storageBytes.WithLabelValues("thumbnails").Set(0)

	return &Metrics{
		registry:            reg,
		photosDiscovered:    photosDiscovered,
		photosDownloaded:    photosDownloaded,
		thumbnailsGenerated: thumbnailsGenerated,
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

// UpdateStorageBytes recursively calculates directory sizes for images and thumbnails and updates the gauge.
func (m *Metrics) UpdateStorageBytes(basePath string) {
	if m == nil || m.storageBytes == nil {
		return
	}

	imagesSize := dirSize(filepath.Join(basePath, "images"))
	thumbnailsSize := dirSize(filepath.Join(basePath, "thumbnails"))

	m.storageBytes.WithLabelValues("images").Set(imagesSize)
	m.storageBytes.WithLabelValues("thumbnails").Set(thumbnailsSize)
}

func dirSize(path string) float64 {
	var totalSize int64
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				totalSize += info.Size()
			}
		}
		return nil
	})
	return float64(totalSize)
}
