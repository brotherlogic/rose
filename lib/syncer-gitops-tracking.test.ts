import { describe, it, expect } from 'vitest';
import { execSync } from 'child_process';

describe('Rose Syncer GitOps Image Automation Tracking Issue (#94)', () => {
  it('should find the tracking issue in brotherlogic/prod with correct metadata and specifications', () => {
    const rawOutput = execSync(
      'gh issue list -R brotherlogic/prod --search "[Turnup] Rose Syncer ImageRepository and ImagePolicy GitOps automation" --json number,title,labels,body --state all',
      { encoding: 'utf-8' }
    );
    const issues = JSON.parse(rawOutput);
    expect(issues.length).toBeGreaterThan(0);

    const issue = issues.find(
      (i: { title: string }) =>
        i.title === '[Turnup] Rose Syncer ImageRepository and ImagePolicy GitOps automation'
    );
    expect(issue).toBeDefined();

    const labelNames = issue.labels.map((l: { name: string }) => l.name);
    expect(labelNames).toContain('container-ready');
    expect(labelNames).toContain('infrastructure');

    // Body content specifications
    const body = issue.body;
    expect(body).toContain('ImageRepository');
    expect(body).toContain('name: rose-syncer');
    expect(body).toContain('namespace: flux-system');
    expect(body).toContain('image: ghcr.io/brotherlogic/rose-syncer');
    expect(body).toContain('interval: 1m0s');

    expect(body).toContain('ImagePolicy');
    expect(body).toContain('imageRepositoryRef:');
    expect(body).toContain('range: 0.x.0');

    // Integration and setter annotations
    expect(body).toContain('ImageUpdateAutomation');
    expect(body).toContain('strategy: Setters');
    expect(body).toContain('projects/rose/cronjob.yaml');
    expect(body).toContain(
      'image: ghcr.io/brotherlogic/rose-syncer:0.5.0 # {"$imagepolicy": "flux-system:rose-syncer"}'
    );

    // Parent references
    expect(body).toContain('brotherlogic/rose#79');
    expect(body).toContain('brotherlogic/rose#85');
    expect(body).toContain('brotherlogic/rose#92');
    expect(body).toContain('brotherlogic/rose#94');
  });
});
