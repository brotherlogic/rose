import { describe, it, expect } from 'vitest';
import { execSync } from 'child_process';

describe('Production Unified Dashboard ConfigMap Tracking Issue (#321)', () => {
  it('should find the tracking issue in brotherlogic/prod with correct metadata and specifications', () => {
    const rawOutput = execSync(
      'gh issue list -R brotherlogic/prod --search "[Turnup] Deploy unified Milestone 4 Grafana dashboard ConfigMap for Rose" --json number,title,labels,body --state all',
      { encoding: 'utf-8' }
    );
    const issues = JSON.parse(rawOutput);
    expect(issues.length).toBeGreaterThan(0);

    const issue = issues.find(
      (i: { title: string }) =>
        i.title === '[Turnup] Deploy unified Milestone 4 Grafana dashboard ConfigMap for Rose'
    );
    expect(issue).toBeDefined();

    const labelNames = issue.labels.map((l: { name: string }) => l.name);
    expect(
      labelNames.some((name: string) =>
        ['container-ready', 'container-creating', 'container-asleep'].includes(name)
      )
    ).toBe(true);

    const body = issue.body;
    // Manifest target & ConfigMap name
    expect(body).toContain('projects/rose/dashboard-unified.yaml');
    expect(body).toContain('rose-dashboard-unified');

    // Metrics / Observability Coverage
    expect(body).toMatch(/ingestion sync status/i);
    expect(body).toMatch(/storage allocation/i);
    expect(body).toMatch(/AI vision inference/i);
    expect(body).toMatch(/web catalog health/i);

    // Cross-references
    expect(body).toContain('brotherlogic/rose#315');
    expect(body).toContain('brotherlogic/rose#316');
    expect(body).toContain('brotherlogic/rose#321');
  });
});
