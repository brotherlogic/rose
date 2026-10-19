import { describe, it, expect } from 'vitest';
import { execSync } from 'child_process';

describe('Production PVC Migration Tracking Issue (#319)', () => {
  it('should find the tracking issue in brotherlogic/prod with correct metadata and specifications', () => {
    const rawOutput = execSync(
      'gh issue list -R brotherlogic/prod --search "[Turnup] Migrate Rose PVC to ceph-filesystem RWX storage class" --json number,title,state,labels,body --state all',
      { encoding: 'utf-8' }
    );
    const issues = JSON.parse(rawOutput);
    expect(issues.length).toBeGreaterThan(0);

    const issue = issues.find(
      (i: { title: string }) =>
        i.title === '[Turnup] Migrate Rose PVC to ceph-filesystem RWX storage class'
    );
    expect(issue).toBeDefined();

    const labelNames = issue.labels.map((l: { name: string }) => l.name);
    expect(
      labelNames.some((name: string) =>
        ['container-ready', 'container-creating', 'container-asleep'].includes(name)
      )
    ).toBe(true);
    expect(
      issue.state === 'CLOSED' ||
        labelNames.includes('seraphine-bug') ||
        labelNames.includes('seraphine-ready-to-implement')
    ).toBe(true);

    const body = issue.body;
    // Manifest target
    expect(body).toContain('projects/rose/pvc.yaml');
    expect(body).toContain('name: rose-data-pvc');

    // StorageClass, Access Mode & Storage Request
    expect(body).toContain('ceph-filesystem');
    expect(body).toContain('ReadWriteMany');
    expect(body).toContain('50Gi');

    // Cross-references
    expect(body).toContain('brotherlogic/rose#315');
    expect(body).toContain('brotherlogic/rose#316');
    expect(body).toContain('brotherlogic/rose#319');
    expect(body).toContain('brotherlogic/rose#356');
    expect(body).toContain('brotherlogic/rose#370');
  });
});
