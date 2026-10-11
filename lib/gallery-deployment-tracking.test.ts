import { describe, it, expect } from 'vitest';
import { execSync } from 'child_process';

describe('Production Rose Gallery Deployment Tracking Issue (#320)', () => {
  it('should find the tracking issue in brotherlogic/prod with correct metadata and specifications', () => {
    const rawOutput = execSync(
      'gh issue list -R brotherlogic/prod --search "[Turnup] Update Rose gallery deployment with rose-data-pvc mount and fsGroup" --json number,title,labels,body --state all',
      { encoding: 'utf-8' }
    );
    const issues = JSON.parse(rawOutput);
    expect(issues.length).toBeGreaterThan(0);

    const issue = issues.find(
      (i: { title: string }) =>
        i.title === '[Turnup] Update Rose gallery deployment with rose-data-pvc mount and fsGroup'
    );
    expect(issue).toBeDefined();

    const labelNames = issue.labels.map((l: { name: string }) => l.name);
    expect(labelNames).toContain('container-ready');

    const body = issue.body;
    // Target manifest
    expect(body).toContain('projects/rose/deploy.yaml');

    // Volume Mount & Mount Path
    expect(body).toContain('rose-data-pvc');
    expect(body).toContain('/data');

    // Pod Security Context fsGroup
    expect(body).toContain('fsGroup: 65532');

    // Cross-references
    expect(body).toContain('brotherlogic/rose#315');
    expect(body).toContain('brotherlogic/rose#316');
    expect(body).toContain('brotherlogic/rose#320');
  });
});
