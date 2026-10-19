import { describe, it, expect } from 'vitest';
import { execSync } from 'child_process';

interface IssueRecord {
  number: number;
  title: string;
  state: string;
  labels: { name: string }[];
  body: string;
}

function queryProdIssue(searchTitle: string): IssueRecord {
  try {
    const rawOutput = execSync(
      `gh issue list -R brotherlogic/prod --search "${searchTitle}" --json number,title,state,labels,body --state all`,
      { encoding: 'utf-8', timeout: 30000 }
    );
    const issues: IssueRecord[] = JSON.parse(rawOutput);
    const issue = issues.find((i) => i.title.trim() === searchTitle.trim());
    if (!issue) {
      throw new Error(`Tracking issue with title "${searchTitle}" was not found in brotherlogic/prod`);
    }
    return issue;
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err);
    throw new Error(`Failed to query brotherlogic/prod for "${searchTitle}": ${msg}`);
  }
}

describe('Milestone 4 Production Infrastructure Tracking Suite (#323)', () => {
  it('validates PVC Migration Tracking Issue exists with required metadata, manifest targets, and cross-references', () => {
    const title = '[Turnup] Migrate Rose PVC to ceph-filesystem RWX storage class';
    const issue = queryProdIssue(title);

    expect(issue).toBeDefined();
    const labelNames = issue.labels.map((l) => l.name);
    expect(
      labelNames.some((name) =>
        ['container-ready', 'container-creating', 'container-asleep'].includes(name)
      )
    ).toBe(true);
    expect(
      issue.state === 'CLOSED' ||
        labelNames.includes('seraphine-bug') ||
        labelNames.includes('seraphine-ready-to-implement')
    ).toBe(true);

    const body = issue.body;
    expect(body).toContain('projects/rose/pvc.yaml');
    expect(body).toContain('ceph-filesystem');
    expect(body).toContain('ReadWriteMany');
    expect(body).toContain('50Gi');
    expect(body).toContain('brotherlogic/rose#315');
    expect(body).toContain('brotherlogic/rose#316');
    expect(body).toContain('brotherlogic/rose#356');
    expect(body).toContain('brotherlogic/rose#370');
  });

  it('validates Gallery Deployment Volume Mount Tracking Issue exists with required metadata, manifest targets, and cross-references', () => {
    const title = '[Turnup] Update Rose gallery deployment with rose-data-pvc mount and fsGroup';
    const issue = queryProdIssue(title);

    expect(issue).toBeDefined();
    const labelNames = issue.labels.map((l) => l.name);
    expect(
      labelNames.some((name) =>
        ['container-ready', 'container-creating', 'container-asleep'].includes(name)
      )
    ).toBe(true);
    expect(
      issue.state === 'CLOSED' ||
        labelNames.includes('seraphine-bug') ||
        labelNames.includes('seraphine-ready-to-implement')
    ).toBe(true);

    const body = issue.body;
    expect(body).toContain('projects/rose/deploy.yaml');
    expect(body).toContain('rose-data-pvc');
    expect(body).toContain('/data');
    expect(body).toContain('readOnly: true');
    expect(body).toContain('fsGroup: 65532');
    expect(body).toContain('brotherlogic/rose#315');
    expect(body).toContain('brotherlogic/rose#316');
    expect(body).toContain('brotherlogic/rose#357');
    expect(body).toContain('brotherlogic/rose#370');
  });

  it('validates Unified Dashboard ConfigMap Tracking Issue exists with required metadata, manifest targets, and cross-references', () => {
    const title = '[Turnup] Deploy unified Milestone 4 Grafana dashboard ConfigMap for Rose';
    const issue = queryProdIssue(title);

    expect(issue).toBeDefined();
    const labelNames = issue.labels.map((l) => l.name);
    expect(
      labelNames.some((name) =>
        ['container-ready', 'container-creating', 'container-asleep'].includes(name)
      )
    ).toBe(true);

    const body = issue.body;
    expect(body).toContain('projects/rose/dashboard-unified.yaml');
    expect(body).toContain('rose-dashboard-unified');
    expect(body).toMatch(/ingestion sync status/i);
    expect(body).toMatch(/storage allocation/i);
    expect(body).toMatch(/AI vision inference/i);
    expect(body).toMatch(/web catalog health/i);
    expect(body).toContain('brotherlogic/rose#315');
    expect(body).toContain('brotherlogic/rose#316');
  });

  it('validates Ingress & TLS Verification Tracking Issue exists with required metadata, manifest targets, and cross-references', () => {
    const title = '[Turnup] Verify Traefik IngressRoute and TLS for roseseraphinetucker.com';
    const issue = queryProdIssue(title);

    expect(issue).toBeDefined();
    const labelNames = issue.labels.map((l) => l.name);
    expect(
      labelNames.some((name) =>
        ['container-ready', 'container-creating', 'container-asleep'].includes(name)
      )
    ).toBe(true);

    const body = issue.body;
    expect(body).toContain('projects/rose/ingress.yaml');
    expect(body).toContain('roseseraphinetucker.com');
    expect(body).toContain('Traefik IngressRoute');
    expect(body).toMatch(/cert-manager|letsencrypt-prod/i);
    expect(body).toMatch(/DNS/i);
    expect(body).toContain('brotherlogic/rose#315');
    expect(body).toContain('brotherlogic/rose#316');
  });
});
