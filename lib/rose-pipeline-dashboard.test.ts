import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';

interface DashboardTarget {
  expr: string;
  legendFormat?: string;
  refId: string;
  instant?: boolean;
}

interface ValueMappingOption {
  text: string;
  color?: string;
  index?: number;
}

interface DashboardMapping {
  type: string;
  options: Record<string, ValueMappingOption>;
}

interface ThresholdStep {
  color: string;
  value: number | null;
}

interface DashboardPanel {
  id: number;
  title: string;
  type: string;
  collapsed?: boolean;
  datasource?: {
    type: string;
    uid: string;
  };
  targets?: DashboardTarget[];
  fieldConfig?: {
    defaults?: {
      unit?: string;
      mappings?: DashboardMapping[];
      thresholds?: {
        mode?: string;
        steps?: ThresholdStep[];
      };
    };
  };
}

interface GrafanaDashboard {
  uid: string;
  title: string;
  schemaVersion: number;
  refresh: string;
  liveNow: boolean;
  panels: DashboardPanel[];
}

describe('Unified Grafana Pipeline Dashboard Validation Suite (#363)', () => {
  const dashboardsDir = path.resolve(__dirname, '..', 'dashboards');
  const pipelineDashboardPath = path.join(dashboardsDir, 'rose-pipeline.json');

  const loadDashboard = (fileName: string): GrafanaDashboard => {
    const filePath = path.join(dashboardsDir, fileName);
    const raw = fs.readFileSync(filePath, 'utf-8');
    return JSON.parse(raw) as GrafanaDashboard;
  };

  describe('1. Cross-Dashboard Coexistence & UID Uniqueness', () => {
    it('should contain all expected .json dashboard files in dashboards/', () => {
      const jsonFiles = fs
        .readdirSync(dashboardsDir)
        .filter((file) => file.endsWith('.json'))
        .sort();

      expect(jsonFiles).toEqual([
        'rose-curation.json',
        'rose-pipeline.json',
        'rose-syncer-m1.json',
      ]);
    });

    it('should parse every .json file in dashboards/ cleanly with unique UIDs and titles', () => {
      const jsonFiles = fs
        .readdirSync(dashboardsDir)
        .filter((file) => file.endsWith('.json'));

      const uids = new Set<string>();
      const titles = new Set<string>();

      for (const file of jsonFiles) {
        const dashboard = loadDashboard(file);
        expect(dashboard.uid).toBeTruthy();
        expect(dashboard.title).toBeTruthy();
        uids.add(dashboard.uid);
        titles.add(dashboard.title);
      }

      expect(uids.size).toBe(jsonFiles.length);
      expect(titles.size).toBe(jsonFiles.length);
      expect(uids).toEqual(
        new Set(['rose-m1-ingestion', 'rose-curation', 'rose-pipeline'])
      );
    });
  });

  describe('2. Unified Dashboard Schema & Section Organization', () => {
    it('should satisfy Grafana v10+ schema metadata requirements', () => {
      expect(fs.existsSync(pipelineDashboardPath)).toBe(true);
      const dashboard = loadDashboard('rose-pipeline.json');

      expect(dashboard.schemaVersion).toBeGreaterThanOrEqual(38);
      expect(dashboard.uid).toBe('rose-pipeline');
      expect(dashboard.title).toBe('Rose - Unified Pipeline Overview');
      expect(dashboard.refresh).toBe('30s');
      expect(dashboard.liveNow).toBe(true);
    });

    it('should contain 19 panels with unique IDs (1..19)', () => {
      const dashboard = loadDashboard('rose-pipeline.json');
      expect(dashboard.panels).toHaveLength(19);

      const ids = dashboard.panels.map((p) => p.id);
      const uniqueIds = new Set(ids);
      expect(uniqueIds.size).toBe(19);

      for (let expectedId = 1; expectedId <= 19; expectedId++) {
        expect(uniqueIds.has(expectedId)).toBe(true);
      }
    });

    it('should include all 4 uncollapsed operational section row headers', () => {
      const dashboard = loadDashboard('rose-pipeline.json');
      const expectedSections = [
        'Section 1: Ingestion & Sync',
        'Section 2: Storage & Assets',
        'Section 3: AI Vision Curation',
        'Section 4: Web Gallery Delivery',
      ];

      for (const sectionTitle of expectedSections) {
        const rowPanel = dashboard.panels.find((p) => p.title === sectionTitle);
        expect(rowPanel).toBeDefined();
        expect(rowPanel?.type).toBe('row');
        expect(rowPanel?.collapsed).toBe(false);
      }
    });
  });

  describe('3. Ephemeral CronJob Pod Isolation & Edge-Case Guards', () => {
    it('should enforce last_over_time(...[7d]) and instant: true on all point-in-time rose_syncer_* stat, gauge, and piechart targets', () => {
      const dashboard = loadDashboard('rose-pipeline.json');
      const pointInTimeTypes = new Set(['stat', 'gauge', 'piechart']);

      const syncerTargets: { panelTitle: string; panelType: string; target: DashboardTarget }[] = [];

      for (const panel of dashboard.panels) {
        if (!pointInTimeTypes.has(panel.type) || !panel.targets) {
          continue;
        }
        for (const target of panel.targets) {
          if (target.expr.includes('rose_syncer_')) {
            syncerTargets.push({
              panelTitle: panel.title,
              panelType: panel.type,
              target,
            });
          }
        }
      }

      expect(syncerTargets.length).toBeGreaterThan(0);

      for (const { target } of syncerTargets) {
        expect(target.instant).toBe(true);
        expect(target.expr).toMatch(/last_over_time\(rose_syncer_[a-z_]+\{[^}]*\}\[7d\]\)/);
      }
    });

    it('should include "or vector(0)" cold-start fallbacks on all stat and gauge targets', () => {
      const dashboard = loadDashboard('rose-pipeline.json');
      const statAndGaugePanels = dashboard.panels.filter(
        (p) => p.type === 'stat' || p.type === 'gauge'
      );

      expect(statAndGaugePanels.length).toBeGreaterThan(0);

      for (const panel of statAndGaugePanels) {
        expect(panel.targets?.length).toBeGreaterThan(0);
        for (const target of panel.targets ?? []) {
          expect(target.expr).toContain('or vector(0)');
        }
      }
    });

    it('should protect the Ingestion Progress gauge against division by zero using clamp_min', () => {
      const dashboard = loadDashboard('rose-pipeline.json');
      const progressPanel = dashboard.panels.find(
        (p) => p.title === 'Ingestion Progress'
      );

      expect(progressPanel).toBeDefined();
      expect(progressPanel?.type).toBe('gauge');
      expect(progressPanel?.fieldConfig?.defaults?.unit).toBe('percent');
      expect(progressPanel?.targets).toHaveLength(1);

      const expr = progressPanel?.targets?.[0]?.expr ?? '';
      expect(expr).toContain('clamp_min(');
      expect(expr).toContain('rose_syncer_photos_downloaded_total{namespace="rose"}[7d]');
      expect(expr).toContain('rose_syncer_photos_discovered_total{namespace="rose"}[7d]');
      expect(expr).toContain('or vector(0)');
    });

    it('should configure Artistic Movement Breakdown piechart with topk(10) ({{theme}}) and > 0 consolidated "Other" filtering', () => {
      const dashboard = loadDashboard('rose-pipeline.json');
      const movementPanel = dashboard.panels.find(
        (p) => p.title === 'Artistic Movement Breakdown'
      );

      expect(movementPanel).toBeDefined();
      expect(movementPanel?.type).toBe('piechart');
      expect(movementPanel?.targets).toHaveLength(2);

      const topkTarget = movementPanel?.targets?.find(
        (t) => t.legendFormat === '{{theme}}'
      );
      expect(topkTarget).toBeDefined();
      expect(topkTarget?.instant).toBe(true);
      expect(topkTarget?.expr).toBe(
        'topk(10, max by (theme) (last_over_time(rose_syncer_artistic_movements_total{namespace="rose"}[7d])))'
      );

      const otherTarget = movementPanel?.targets?.find(
        (t) => t.legendFormat === 'Other'
      );
      expect(otherTarget).toBeDefined();
      expect(otherTarget?.instant).toBe(true);
      expect(otherTarget?.expr).toContain(
        'sum(max by (theme) (last_over_time(rose_syncer_artistic_movements_total{namespace="rose"}[7d])))'
      );
      expect(otherTarget?.expr).toContain(
        'sum(topk(10, max by (theme) (last_over_time(rose_syncer_artistic_movements_total{namespace="rose"}[7d]))))'
      );
      expect(otherTarget?.expr).toMatch(/>\s*0$/);
    });
  });

  describe('4. Section 4 Web Gallery Delivery Telemetry', () => {
    it('should derive Total Cataloged Artworks from rose_syncer_artworks_annotated_total', () => {
      const dashboard = loadDashboard('rose-pipeline.json');
      const catalogedPanel = dashboard.panels.find(
        (p) => p.title === 'Total Cataloged Artworks'
      );

      expect(catalogedPanel).toBeDefined();
      expect(catalogedPanel?.type).toBe('stat');
      expect(catalogedPanel?.targets).toHaveLength(1);
      expect(catalogedPanel?.targets?.[0]?.instant).toBe(true);
      expect(catalogedPanel?.targets?.[0]?.expr).toBe(
        'max(last_over_time(rose_syncer_artworks_annotated_total{namespace="rose"}[7d])) or vector(0)'
      );
    });

    it('should configure Web Gallery Availability with up / probe_success selectors, Healthy/Down mappings, and red/green thresholds', () => {
      const dashboard = loadDashboard('rose-pipeline.json');
      const availabilityPanel = dashboard.panels.find(
        (p) => p.title === 'Web Gallery Availability'
      );

      expect(availabilityPanel).toBeDefined();
      expect(availabilityPanel?.type).toBe('stat');
      expect(availabilityPanel?.targets).toHaveLength(1);

      const target = availabilityPanel?.targets?.[0];
      expect(target?.instant).toBe(true);
      expect(target?.expr).toContain('up{namespace="rose", app="rose-gallery"}');
      expect(target?.expr).toContain('probe_success{namespace="rose"}');
      expect(target?.expr).toContain('or vector(0)');

      const defaults = availabilityPanel?.fieldConfig?.defaults;
      const valueMapping = defaults?.mappings?.find((m) => m.type === 'value');
      expect(valueMapping?.options?.['0']?.text).toBe('Down');
      expect(valueMapping?.options?.['1']?.text).toBe('Healthy');

      const steps = defaults?.thresholds?.steps ?? [];
      expect(steps).toEqual(
        expect.arrayContaining([
          expect.objectContaining({ color: 'red', value: null }),
          expect.objectContaining({ color: 'green', value: 1 }),
        ])
      );
    });

    it('should configure Last Catalog Refresh Timestamp using sync duration timestamp * 1000, dateTimeAsIso unit, and 0 -> "Never" mapping', () => {
      const dashboard = loadDashboard('rose-pipeline.json');
      const refreshPanel = dashboard.panels.find(
        (p) => p.title === 'Last Catalog Refresh Timestamp'
      );

      expect(refreshPanel).toBeDefined();
      expect(refreshPanel?.type).toBe('stat');
      expect(refreshPanel?.targets).toHaveLength(1);

      const target = refreshPanel?.targets?.[0];
      expect(target?.instant).toBe(true);
      expect(target?.expr).toContain(
        'timestamp(last_over_time(rose_syncer_sync_duration_seconds{namespace="rose"}[7d]))'
      );
      expect(target?.expr).toContain('* 1000');
      expect(target?.expr).toContain('or vector(0)');

      const defaults = refreshPanel?.fieldConfig?.defaults;
      expect(defaults?.unit).toBe('dateTimeAsIso');

      const valueMapping = defaults?.mappings?.find((m) => m.type === 'value');
      expect(valueMapping?.options?.['0']?.text).toBe('Never');
    });
  });
});
