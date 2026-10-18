package dashboards_test

import (
	"encoding/json"
	"strings"
	"testing"
)

type pipelineTarget struct {
	Expr         string `json:"expr"`
	LegendFormat string `json:"legendFormat"`
	RefId        string `json:"refId"`
	Instant      bool   `json:"instant"`
}

type pipelineValueMappingOption struct {
	Text  string `json:"text"`
	Color string `json:"color"`
	Index int    `json:"index"`
}

type pipelineMapping struct {
	Type    string                                `json:"type"`
	Options map[string]pipelineValueMappingOption `json:"options"`
}

type pipelineFieldConfigDefaults struct {
	Thresholds Thresholds        `json:"thresholds"`
	Unit       string            `json:"unit"`
	Mappings   []pipelineMapping `json:"mappings"`
}

type pipelineFieldConfig struct {
	Defaults pipelineFieldConfigDefaults `json:"defaults"`
}

type pipelineDatasource struct {
	Type string `json:"type"`
	UID  string `json:"uid"`
}

type pipelinePanel struct {
	ID          int                 `json:"id"`
	Title       string              `json:"title"`
	Type        string              `json:"type"`
	Collapsed   bool                `json:"collapsed"`
	Datasource  *pipelineDatasource `json:"datasource,omitempty"`
	Targets     []pipelineTarget    `json:"targets"`
	FieldConfig pipelineFieldConfig `json:"fieldConfig"`
	Panels      []pipelinePanel     `json:"panels,omitempty"`
}

type pipelineDashboard struct {
	UID           string          `json:"uid"`
	Title         string          `json:"title"`
	SchemaVersion int             `json:"schemaVersion"`
	Refresh       string          `json:"refresh"`
	LiveNow       bool            `json:"liveNow"`
	Panels        []pipelinePanel `json:"panels"`
}

func loadPipelineDashboard(t *testing.T) (Dashboard, pipelineDashboard) {
	t.Helper()
	raw, base := loadDashboard(t, "rose-pipeline.json")
	var pd pipelineDashboard
	if err := json.Unmarshal(raw, &pd); err != nil {
		t.Fatalf("failed to unmarshal rose-pipeline.json into pipelineDashboard: %v", err)
	}
	return base, pd
}

func findPipelinePanelByTitle(t *testing.T, panels []pipelinePanel, title string) pipelinePanel {
	t.Helper()
	for _, p := range panels {
		if p.Title == title {
			return p
		}
	}
	t.Fatalf("expected panel with title %q not found", title)
	return pipelinePanel{}
}

func TestRosePipelineDashboard_SchemaAndMetadata(t *testing.T) {
	base, pd := loadPipelineDashboard(t)

	if pd.UID != "rose-pipeline" {
		t.Errorf("expected uid 'rose-pipeline', got %q", pd.UID)
	}
	if pd.Title != "Rose - Unified Pipeline Overview" {
		t.Errorf("expected title 'Rose - Unified Pipeline Overview', got %q", pd.Title)
	}
	if pd.SchemaVersion < 38 {
		t.Errorf("expected schemaVersion >= 38, got %d", pd.SchemaVersion)
	}
	if pd.Refresh != "30s" {
		t.Errorf("expected refresh '30s', got %q", pd.Refresh)
	}
	if !pd.LiveNow {
		t.Errorf("expected liveNow to be true")
	}

	allPanels := extractAllPanels(base.Panels)
	if len(allPanels) != 19 {
		t.Errorf("expected 19 total panels, got %d", len(allPanels))
	}

	allTargets := extractAllTargets(base.Panels)
	if len(allTargets) == 0 {
		t.Errorf("expected dashboard to contain query targets")
	}

	seenIDs := make(map[int]string)
	for _, p := range pd.Panels {
		if p.ID <= 0 {
			t.Errorf("panel %q has invalid ID %d", p.Title, p.ID)
		}
		if prevTitle, exists := seenIDs[p.ID]; exists {
			t.Errorf("duplicate panel ID %d on %q and %q", p.ID, prevTitle, p.Title)
		}
		seenIDs[p.ID] = p.Title

		if p.Type != "row" {
			if p.Datasource == nil || p.Datasource.Type != "prometheus" || p.Datasource.UID != "prometheus" {
				t.Errorf("expected data panel %q to have prometheus datasource, got %+v", p.Title, p.Datasource)
			}
		}
	}
}

func TestRosePipelineDashboard_FourOperationalSections(t *testing.T) {
	_, pd := loadPipelineDashboard(t)

	expectedSections := []string{
		"Section 1: Ingestion & Sync",
		"Section 2: Storage & Assets",
		"Section 3: AI Vision Curation",
		"Section 4: Web Gallery Delivery",
	}

	for _, secTitle := range expectedSections {
		p := findPipelinePanelByTitle(t, pd.Panels, secTitle)
		if p.Type != "row" {
			t.Errorf("expected section %q to have type 'row', got %q", secTitle, p.Type)
		}
		if p.Collapsed {
			t.Errorf("expected section %q to have collapsed == false", secTitle)
		}
	}
}

func TestRosePipelineDashboard_Section1IngestionAndSync(t *testing.T) {
	_, pd := loadPipelineDashboard(t)

	// 1. Album Photos Discovered
	discovered := findPipelinePanelByTitle(t, pd.Panels, "Album Photos Discovered")
	if discovered.Type != "stat" {
		t.Errorf("expected 'Album Photos Discovered' type 'stat', got %q", discovered.Type)
	}
	if discovered.FieldConfig.Defaults.Unit != "short" {
		t.Errorf("expected 'Album Photos Discovered' unit 'short', got %q", discovered.FieldConfig.Defaults.Unit)
	}
	if len(discovered.Targets) != 1 || !discovered.Targets[0].Instant ||
		!strings.Contains(discovered.Targets[0].Expr, "last_over_time(rose_syncer_photos_discovered_total{namespace=\"rose\"}[7d])") ||
		!strings.Contains(discovered.Targets[0].Expr, "or vector(0)") {
		t.Errorf("unexpected target for 'Album Photos Discovered': %+v", discovered.Targets)
	}

	// 2. Photos Downloaded
	downloaded := findPipelinePanelByTitle(t, pd.Panels, "Photos Downloaded")
	if downloaded.Type != "stat" {
		t.Errorf("expected 'Photos Downloaded' type 'stat', got %q", downloaded.Type)
	}
	if len(downloaded.Targets) != 1 || !downloaded.Targets[0].Instant ||
		!strings.Contains(downloaded.Targets[0].Expr, "last_over_time(rose_syncer_photos_downloaded_total{namespace=\"rose\"}[7d])") ||
		!strings.Contains(downloaded.Targets[0].Expr, "or vector(0)") {
		t.Errorf("unexpected target for 'Photos Downloaded': %+v", downloaded.Targets)
	}

	// 3. Ingestion Progress
	progress := findPipelinePanelByTitle(t, pd.Panels, "Ingestion Progress")
	if progress.Type != "gauge" {
		t.Errorf("expected 'Ingestion Progress' type 'gauge', got %q", progress.Type)
	}
	if progress.FieldConfig.Defaults.Unit != "percent" {
		t.Errorf("expected 'Ingestion Progress' unit 'percent', got %q", progress.FieldConfig.Defaults.Unit)
	}
	if len(progress.Targets) != 1 || !progress.Targets[0].Instant ||
		!strings.Contains(progress.Targets[0].Expr, "clamp_min") ||
		!strings.Contains(progress.Targets[0].Expr, "or vector(0)") {
		t.Errorf("unexpected target for 'Ingestion Progress': %+v", progress.Targets)
	}
	has50 := false
	has95 := false
	for _, s := range progress.FieldConfig.Defaults.Thresholds.Steps {
		if s.Value != nil && *s.Value == 50 && s.Color == "yellow" {
			has50 = true
		}
		if s.Value != nil && *s.Value == 95 && s.Color == "green" {
			has95 = true
		}
	}
	if !has50 || !has95 {
		t.Errorf("expected 'Ingestion Progress' thresholds at 50 (yellow) and 95 (green), got %+v", progress.FieldConfig.Defaults.Thresholds.Steps)
	}

	// 4. Sync Errors
	syncErrors := findPipelinePanelByTitle(t, pd.Panels, "Sync Errors")
	if syncErrors.Type != "stat" {
		t.Errorf("expected 'Sync Errors' type 'stat', got %q", syncErrors.Type)
	}
	if len(syncErrors.Targets) != 1 || !syncErrors.Targets[0].Instant ||
		!strings.Contains(syncErrors.Targets[0].Expr, "last_over_time(rose_syncer_sync_errors_total{namespace=\"rose\"}[7d])") {
		t.Errorf("unexpected target for 'Sync Errors': %+v", syncErrors.Targets)
	}
	hasRed1 := false
	for _, s := range syncErrors.FieldConfig.Defaults.Thresholds.Steps {
		if s.Color == "red" && s.Value != nil && *s.Value == 1 {
			hasRed1 = true
		}
	}
	if !hasRed1 {
		t.Errorf("expected 'Sync Errors' red threshold at 1, got %+v", syncErrors.FieldConfig.Defaults.Thresholds.Steps)
	}

	// 5. Sync Execution Duration
	syncDuration := findPipelinePanelByTitle(t, pd.Panels, "Sync Execution Duration")
	if syncDuration.Type != "timeseries" {
		t.Errorf("expected 'Sync Execution Duration' type 'timeseries', got %q", syncDuration.Type)
	}
	if syncDuration.FieldConfig.Defaults.Unit != "s" {
		t.Errorf("expected 'Sync Execution Duration' unit 's', got %q", syncDuration.FieldConfig.Defaults.Unit)
	}
	if len(syncDuration.Targets) != 1 ||
		!strings.Contains(syncDuration.Targets[0].Expr, "last_over_time(rose_syncer_sync_duration_seconds{namespace=\"rose\"}[7d])") {
		t.Errorf("unexpected target for 'Sync Execution Duration': %+v", syncDuration.Targets)
	}
}

func TestRosePipelineDashboard_Section2StorageAndAssets(t *testing.T) {
	_, pd := loadPipelineDashboard(t)

	// 1. Storage Consumed
	storage := findPipelinePanelByTitle(t, pd.Panels, "Storage Consumed")
	if storage.Type != "stat" {
		t.Errorf("expected 'Storage Consumed' type 'stat', got %q", storage.Type)
	}
	if storage.FieldConfig.Defaults.Unit != "bytes" {
		t.Errorf("expected 'Storage Consumed' unit 'bytes', got %q", storage.FieldConfig.Defaults.Unit)
	}
	hasTotal := false
	hasImages := false
	hasThumbnails := false
	for _, tr := range storage.Targets {
		if !tr.Instant {
			t.Errorf("expected 'Storage Consumed' target %s to have instant: true", tr.RefId)
		}
		if tr.LegendFormat == "Total" && strings.Contains(tr.Expr, "rose_syncer_storage_bytes{namespace=\"rose\"}[7d]") {
			hasTotal = true
		}
		if tr.LegendFormat == "Images" && strings.Contains(tr.Expr, "type=\"images\"") {
			hasImages = true
		}
		if tr.LegendFormat == "Thumbnails" && strings.Contains(tr.Expr, "type=\"thumbnails\"") {
			hasThumbnails = true
		}
	}
	if !hasTotal || !hasImages || !hasThumbnails {
		t.Errorf("expected 'Storage Consumed' targets for Total, Images, and Thumbnails; got %+v", storage.Targets)
	}

	// 2. Original Images Stored
	origImages := findPipelinePanelByTitle(t, pd.Panels, "Original Images Stored")
	if origImages.Type != "stat" {
		t.Errorf("expected 'Original Images Stored' type 'stat', got %q", origImages.Type)
	}
	if len(origImages.Targets) != 1 || !origImages.Targets[0].Instant ||
		!strings.Contains(origImages.Targets[0].Expr, "last_over_time(rose_syncer_photos_stored{namespace=\"rose\"}[7d])") {
		t.Errorf("unexpected target for 'Original Images Stored': %+v", origImages.Targets)
	}

	// 3. Thumbnails Stored & Generated
	thumbs := findPipelinePanelByTitle(t, pd.Panels, "Thumbnails Stored & Generated")
	if thumbs.Type != "stat" {
		t.Errorf("expected 'Thumbnails Stored & Generated' type 'stat', got %q", thumbs.Type)
	}
	hasThumbsStored := false
	hasThumbsGenerated := false
	for _, tr := range thumbs.Targets {
		if !tr.Instant {
			t.Errorf("expected 'Thumbnails Stored & Generated' target %s to have instant: true", tr.RefId)
		}
		if strings.Contains(tr.Expr, "rose_syncer_thumbnails_stored{namespace=\"rose\"}[7d]") {
			hasThumbsStored = true
		}
		if strings.Contains(tr.Expr, "rose_syncer_thumbnails_generated_total{namespace=\"rose\"}[7d]") {
			hasThumbsGenerated = true
		}
	}
	if !hasThumbsStored || !hasThumbsGenerated {
		t.Errorf("expected 'Thumbnails Stored & Generated' targets for stored and generated; got %+v", thumbs.Targets)
	}
}

func TestRosePipelineDashboard_Section3AIVisionCuration(t *testing.T) {
	_, pd := loadPipelineDashboard(t)

	// 1. Artworks Annotated
	annotated := findPipelinePanelByTitle(t, pd.Panels, "Artworks Annotated")
	if annotated.Type != "stat" {
		t.Errorf("expected 'Artworks Annotated' type 'stat', got %q", annotated.Type)
	}
	hasCurated := false
	hasStored := false
	for _, tr := range annotated.Targets {
		if !tr.Instant {
			t.Errorf("expected 'Artworks Annotated' target %s to have instant: true", tr.RefId)
		}
		if tr.LegendFormat == "Curated Artworks" && strings.Contains(tr.Expr, "rose_syncer_artworks_annotated_total") {
			hasCurated = true
		}
		if tr.LegendFormat == "Total Stored Photos" && strings.Contains(tr.Expr, "rose_syncer_photos_stored") {
			hasStored = true
		}
	}
	if !hasCurated || !hasStored {
		t.Errorf("expected 'Artworks Annotated' to compare Curated Artworks and Total Stored Photos; got %+v", annotated.Targets)
	}

	// 2. Annotation Failures
	failures := findPipelinePanelByTitle(t, pd.Panels, "Annotation Failures")
	if failures.Type != "stat" {
		t.Errorf("expected 'Annotation Failures' type 'stat', got %q", failures.Type)
	}
	if len(failures.Targets) != 1 || !failures.Targets[0].Instant ||
		!strings.Contains(failures.Targets[0].Expr, "last_over_time(rose_syncer_annotation_errors_total{namespace=\"rose\"}[7d])") {
		t.Errorf("unexpected target for 'Annotation Failures': %+v", failures.Targets)
	}
	hasRed := false
	for _, s := range failures.FieldConfig.Defaults.Thresholds.Steps {
		if s.Color == "red" && s.Value != nil && *s.Value == 1 {
			hasRed = true
		}
	}
	if !hasRed {
		t.Errorf("expected 'Annotation Failures' red threshold at 1, got %+v", failures.FieldConfig.Defaults.Thresholds.Steps)
	}

	// 3. Annotation Latency
	latency := findPipelinePanelByTitle(t, pd.Panels, "Annotation Latency")
	if latency.Type != "timeseries" {
		t.Errorf("expected 'Annotation Latency' type 'timeseries', got %q", latency.Type)
	}
	if latency.FieldConfig.Defaults.Unit != "s" {
		t.Errorf("expected 'Annotation Latency' unit 's', got %q", latency.FieldConfig.Defaults.Unit)
	}
	hasP50 := false
	hasP95 := false
	for _, tr := range latency.Targets {
		if strings.Contains(tr.Expr, "histogram_quantile(0.50") && strings.Contains(tr.Expr, "rose_syncer_annotation_duration_seconds_bucket") {
			hasP50 = true
		}
		if strings.Contains(tr.Expr, "histogram_quantile(0.95") && strings.Contains(tr.Expr, "rose_syncer_annotation_duration_seconds_bucket") {
			hasP95 = true
		}
	}
	if !hasP50 || !hasP95 {
		t.Errorf("expected 'Annotation Latency' to include p50 and p95 histogram_quantile queries; got %+v", latency.Targets)
	}

	// 4. Artistic Movement Breakdown
	movements := findPipelinePanelByTitle(t, pd.Panels, "Artistic Movement Breakdown")
	if movements.Type != "piechart" {
		t.Errorf("expected 'Artistic Movement Breakdown' type 'piechart', got %q", movements.Type)
	}
	hasTopK := false
	hasOther := false
	for _, tr := range movements.Targets {
		if !tr.Instant {
			t.Errorf("expected 'Artistic Movement Breakdown' target %s to have instant: true", tr.RefId)
		}
		if tr.LegendFormat == "{{theme}}" && strings.Contains(tr.Expr, "topk(10, max by (theme) (last_over_time(rose_syncer_artistic_movements_total{namespace=\"rose\"}[7d])))") {
			hasTopK = true
		}
		if tr.LegendFormat == "Other" && strings.Contains(tr.Expr, "> 0") &&
			strings.Contains(tr.Expr, "sum(max by (theme) (last_over_time(rose_syncer_artistic_movements_total{namespace=\"rose\"}[7d])))") &&
			strings.Contains(tr.Expr, "sum(topk(10, max by (theme) (last_over_time(rose_syncer_artistic_movements_total{namespace=\"rose\"}[7d]))))") {
			hasOther = true
		}
	}
	if !hasTopK || !hasOther {
		t.Errorf("expected 'Artistic Movement Breakdown' topk(10) and consolidated Other > 0 targets; got %+v", movements.Targets)
	}
}

func TestRosePipelineDashboard_Section4WebGalleryDelivery(t *testing.T) {
	_, pd := loadPipelineDashboard(t)

	// 1. Total Cataloged Artworks
	cataloged := findPipelinePanelByTitle(t, pd.Panels, "Total Cataloged Artworks")
	if cataloged.Type != "stat" {
		t.Errorf("expected 'Total Cataloged Artworks' type 'stat', got %q", cataloged.Type)
	}
	if len(cataloged.Targets) != 1 || !cataloged.Targets[0].Instant ||
		!strings.Contains(cataloged.Targets[0].Expr, "last_over_time(rose_syncer_artworks_annotated_total{namespace=\"rose\"}[7d])") ||
		!strings.Contains(cataloged.Targets[0].Expr, "or vector(0)") {
		t.Errorf("unexpected target for 'Total Cataloged Artworks': %+v", cataloged.Targets)
	}

	// 2. Web Gallery Availability
	availability := findPipelinePanelByTitle(t, pd.Panels, "Web Gallery Availability")
	if availability.Type != "stat" {
		t.Errorf("expected 'Web Gallery Availability' type 'stat', got %q", availability.Type)
	}
	if len(availability.Targets) != 1 || !availability.Targets[0].Instant ||
		!strings.Contains(availability.Targets[0].Expr, "up{namespace=\"rose\", app=\"rose-gallery\"}") ||
		!strings.Contains(availability.Targets[0].Expr, "probe_success{namespace=\"rose\"}") {
		t.Errorf("unexpected target for 'Web Gallery Availability': %+v", availability.Targets)
	}
	hasRedBase := false
	hasGreen1 := false
	for _, s := range availability.FieldConfig.Defaults.Thresholds.Steps {
		if s.Color == "red" && s.Value == nil {
			hasRedBase = true
		}
		if s.Color == "green" && s.Value != nil && *s.Value == 1 {
			hasGreen1 = true
		}
	}
	if !hasRedBase || !hasGreen1 {
		t.Errorf("expected 'Web Gallery Availability' red (null) and green (1) thresholds, got %+v", availability.FieldConfig.Defaults.Thresholds.Steps)
	}
	hasDownMapping := false
	hasHealthyMapping := false
	for _, m := range availability.FieldConfig.Defaults.Mappings {
		if opt, ok := m.Options["0"]; ok && opt.Text == "Down" {
			hasDownMapping = true
		}
		if opt, ok := m.Options["1"]; ok && opt.Text == "Healthy" {
			hasHealthyMapping = true
		}
	}
	if !hasDownMapping || !hasHealthyMapping {
		t.Errorf("expected 'Web Gallery Availability' mappings for 0->'Down' and 1->'Healthy', got %+v", availability.FieldConfig.Defaults.Mappings)
	}

	// 3. Last Catalog Refresh Timestamp
	refreshTS := findPipelinePanelByTitle(t, pd.Panels, "Last Catalog Refresh Timestamp")
	if refreshTS.Type != "stat" {
		t.Errorf("expected 'Last Catalog Refresh Timestamp' type 'stat', got %q", refreshTS.Type)
	}
	if refreshTS.FieldConfig.Defaults.Unit != "dateTimeAsIso" {
		t.Errorf("expected 'Last Catalog Refresh Timestamp' unit 'dateTimeAsIso', got %q", refreshTS.FieldConfig.Defaults.Unit)
	}
	if len(refreshTS.Targets) != 1 || !refreshTS.Targets[0].Instant ||
		!strings.Contains(refreshTS.Targets[0].Expr, "timestamp(last_over_time(rose_syncer_sync_duration_seconds{namespace=\"rose\"}[7d]))") ||
		!strings.Contains(refreshTS.Targets[0].Expr, "* 1000") {
		t.Errorf("unexpected target for 'Last Catalog Refresh Timestamp': %+v", refreshTS.Targets)
	}
}
