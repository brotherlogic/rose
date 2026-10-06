import { describe, it, expect } from 'vitest';
import { execSync } from 'child_process';

describe('Production Dashboard ConfigMap Synchronization Tracking Issue (#233)', () => {
  it('should find the tracking issue in brotherlogic/prod with correct metadata and specifications', () => {
    const rawOutput = execSync(
      'gh issue list -R brotherlogic/prod --search "[Turnup] Sync updated AI Curation overview Grafana dashboard ConfigMap for Rose" --json number,title,labels,body --state all',
      { encoding: 'utf-8' }
    );
    const issues = JSON.parse(rawOutput);
    expect(issues.length).toBeGreaterThan(0);

    const issue = issues.find(
      (i: { title: string }) =>
        i.title === '[Turnup] Sync updated AI Curation overview Grafana dashboard ConfigMap for Rose'
    );
    expect(issue).toBeDefined();

    const labelNames = issue.labels.map((l: { name: string }) => l.name);
    expect(labelNames).toContain('container-ready');

    const body = issue.body;
    // ConfigMap specifications
    expect(body).toContain('projects/rose/dashboard-m2.yaml');
    expect(body).toContain('rose-m2-curation-dashboard.json');

    // Panel 1 specifications
    expect(body).toContain('Artworks Annotated');
    expect(body).toContain('rose_syncer_artworks_annotated_total');
    expect(body).toContain('rose_syncer_photos_stored');

    // Panel 4 specifications
    expect(body).toContain('Artistic Movement Breakdown');
    expect(body).toContain('rose_syncer_artistic_movements_total');

    // Cross-references
    expect(body).toContain('brotherlogic/rose#227');
    expect(body).toContain('brotherlogic/rose#230');
    expect(body).toContain('brotherlogic/rose#233');
  });
});
