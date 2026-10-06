package dashboards_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type Target struct {
	Expr         string `json:"expr"`
	LegendFormat string `json:"legendFormat"`
	RefId        string `json:"refId"`
}

type Step struct {
	Color string   `json:"color"`
	Value *float64 `json:"value"`
}

type Thresholds struct {
	Mode  string `json:"mode"`
	Steps []Step `json:"steps"`
}

type FieldConfigDefaults struct {
	Thresholds Thresholds `json:"thresholds"`
	Unit       string     `json:"unit"`
}

type FieldConfig struct {
	Defaults FieldConfigDefaults `json:"defaults"`
}

type Panel struct {
	ID          int         `json:"id"`
	Title       string      `json:"title"`
	Type        string      `json:"type"`
	Targets     []Target    `json:"targets"`
	FieldConfig FieldConfig `json:"fieldConfig"`
	Panels      []Panel     `json:"panels,omitempty"`
}

type Dashboard struct {
	UID           string  `json:"uid"`
	Title         string  `json:"title"`
	SchemaVersion int     `json:"schemaVersion"`
	Panels        []Panel `json:"panels"`
}

func loadDashboard(t *testing.T, filename string) ([]byte, Dashboard) {
	t.Helper()
	paths := []string{
		filename,
		filepath.Join("dashboards", filename),
		filepath.Join("..", "dashboards", filename),
	}

	var data []byte
	var err error
	found := false
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("could not find %s: last err: %v", filename, err)
	}

	var d Dashboard
	if err := json.Unmarshal(data, &d); err != nil {
		t.Fatalf("failed to parse dashboard JSON %s: %v", filename, err)
	}
	return data, d
}

func extractAllTargets(panels []Panel) []Target {
	var targets []Target
	for _, p := range panels {
		targets = append(targets, p.Targets...)
		if len(p.Panels) > 0 {
			targets = append(targets, extractAllTargets(p.Panels)...)
		}
	}
	return targets
}

func extractAllPanels(panels []Panel) []Panel {
	var all []Panel
	for _, p := range panels {
		all = append(all, p)
		if len(p.Panels) > 0 {
			all = append(all, extractAllPanels(p.Panels)...)
		}
	}
	return all
}

func TestRoseCurationDashboard_SchemaAndMetadata(t *testing.T) {
	_, d := loadDashboard(t, "rose-curation.json")

	if d.UID != "rose-curation" {
		t.Errorf("expected dashboard UID 'rose-curation', got '%s'", d.UID)
	}

	expectedTitle := "Rose - AI Curation Overview"
	if d.Title != expectedTitle {
		t.Errorf("expected dashboard title '%s', got '%s'", expectedTitle, d.Title)
	}

	if d.SchemaVersion < 30 {
		t.Errorf("expected Grafana v10+ schema version (>=30), got %d", d.SchemaVersion)
	}
}

func TestRoseCurationDashboard_RequiredPanelMetricQueries(t *testing.T) {
	_, d := loadDashboard(t, "rose-curation.json")

	requiredMetrics := []string{
		"rose_syncer_artworks_annotated_total",
		"rose_syncer_annotation_errors_total",
		"rose_syncer_annotation_duration_seconds_bucket",
		"rose_syncer_artistic_movements_total",
	}

	targets := extractAllTargets(d.Panels)
	if len(targets) == 0 {
		t.Fatalf("dashboard has no panel query targets")
	}

	for _, metric := range requiredMetrics {
		found := false
		for _, target := range targets {
			if strings.Contains(target.Expr, metric) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("required metric query '%s' not referenced in any panel target", metric)
		}
	}
}

func TestRoseCurationDashboard_PanelsDetailedRequirements(t *testing.T) {
	_, d := loadDashboard(t, "rose-curation.json")
	panels := extractAllPanels(d.Panels)

	// 1. Artworks Annotated: single-stat panel referencing rose_syncer_artworks_annotated_total
	// and rose_syncer_photos_stored for side-by-side comparison
	var artworksPanel *Panel
	for i := range panels {
		p := &panels[i]
		for _, tr := range p.Targets {
			if strings.Contains(tr.Expr, "rose_syncer_artworks_annotated_total") {
				artworksPanel = p
				break
			}
		}
	}
	if artworksPanel == nil {
		t.Fatal("panel with metric rose_syncer_artworks_annotated_total not found")
	}
	if artworksPanel.Type != "stat" {
		t.Errorf("expected artworks panel type 'stat', got '%s'", artworksPanel.Type)
	}
	hasArtworksAnnotatedRef := false
	hasPhotoStoredRef := false
	for _, tr := range artworksPanel.Targets {
		if strings.Contains(tr.Expr, "rose_syncer_artworks_annotated_total") && tr.LegendFormat == "Curated Artworks" {
			hasArtworksAnnotatedRef = true
		}
		if strings.Contains(tr.Expr, "rose_syncer_photos_stored") && tr.LegendFormat == "Total Stored Photos" {
			hasPhotoStoredRef = true
		}
	}
	if !hasArtworksAnnotatedRef {
		t.Errorf("expected artworks panel to reference rose_syncer_artworks_annotated_total with legend 'Curated Artworks'")
	}
	if !hasPhotoStoredRef {
		t.Errorf("expected artworks panel to reference rose_syncer_photos_stored with legend 'Total Stored Photos'")
	}

	// 2. Annotation Failures: stat / counter with prominent red threshold when value > 0
	var errorPanel *Panel
	for i := range panels {
		p := &panels[i]
		for _, tr := range p.Targets {
			if strings.Contains(tr.Expr, "rose_syncer_annotation_errors_total") {
				errorPanel = p
				break
			}
		}
	}
	if errorPanel == nil {
		t.Fatal("panel with metric rose_syncer_annotation_errors_total not found")
	}
	if errorPanel.Type != "stat" {
		t.Errorf("expected error panel type 'stat', got '%s'", errorPanel.Type)
	}
	hasRedStep := false
	for _, s := range errorPanel.FieldConfig.Defaults.Thresholds.Steps {
		if strings.ToLower(s.Color) == "red" && (s.Value == nil || *s.Value >= 0) {
			hasRedStep = true
			break
		}
	}
	if !hasRedStep {
		t.Errorf("expected error panel to configure red threshold highlights")
	}

	// 3. Annotation Latency: timeseries with p50 and p95 queries
	var latencyPanel *Panel
	for i := range panels {
		p := &panels[i]
		for _, tr := range p.Targets {
			if strings.Contains(tr.Expr, "rose_syncer_annotation_duration_seconds_bucket") {
				latencyPanel = p
				break
			}
		}
	}
	if latencyPanel == nil {
		t.Fatal("panel with latency metric rose_syncer_annotation_duration_seconds_bucket not found")
	}
	if latencyPanel.Type != "timeseries" {
		t.Errorf("expected latency panel type 'timeseries', got '%s'", latencyPanel.Type)
	}
	hasP50 := false
	hasP95 := false
	for _, tr := range latencyPanel.Targets {
		if strings.Contains(tr.Expr, "0.5") || strings.Contains(tr.Expr, "0.50") {
			hasP50 = true
		}
		if strings.Contains(tr.Expr, "0.95") {
			hasP95 = true
		}
	}
	if !hasP50 || !hasP95 {
		t.Errorf("expected latency panel to include both p50 and p95 quantile queries; got p50=%v, p95=%v", hasP50, hasP95)
	}

	// 4. Artistic Movement Breakdown: piechart or bargauge with theme legend
	var movementsPanel *Panel
	for i := range panels {
		p := &panels[i]
		for _, tr := range p.Targets {
			if strings.Contains(tr.Expr, "rose_syncer_artistic_movements_total") {
				movementsPanel = p
				break
			}
		}
	}
	if movementsPanel == nil {
		t.Fatal("panel with metric rose_syncer_artistic_movements_total not found")
	}
	if movementsPanel.Type != "piechart" && movementsPanel.Type != "bargauge" && movementsPanel.Type != "bar-gauge" {
		t.Errorf("expected movements panel type 'piechart' or 'bargauge', got '%s'", movementsPanel.Type)
	}
	hasTopKTarget := false
	hasOtherTarget := false
	for _, tr := range movementsPanel.Targets {
		if strings.Contains(tr.Expr, "topk(10,") && strings.Contains(tr.Expr, "rose_syncer_artistic_movements_total") && tr.LegendFormat == "{{theme}}" {
			hasTopKTarget = true
		}
		if strings.Contains(tr.Expr, "sum(rose_syncer_artistic_movements_total") &&
			strings.Contains(tr.Expr, "sum(topk(10, rose_syncer_artistic_movements_total") &&
			tr.LegendFormat == "Other" {
			hasOtherTarget = true
		}
	}
	if !hasTopKTarget {
		t.Errorf("expected movements panel to contain topk(10, rose_syncer_artistic_movements_total) query with legend '{{theme}}'")
	}
	if !hasOtherTarget {
		t.Errorf("expected movements panel to contain consolidated Other target expression sum(...) - sum(topk(...)) with legend 'Other'")
	}
}
