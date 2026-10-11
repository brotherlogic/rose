import { describe, it, expect } from 'vitest';
import { execSync } from 'child_process';

describe('Production Traefik IngressRoute and TLS Verification Tracking Issue (#322)', () => {
  it('should find the tracking issue in brotherlogic/prod with correct metadata and specifications', () => {
    const rawOutput = execSync(
      'gh issue list -R brotherlogic/prod --search "[Turnup] Verify Traefik IngressRoute and TLS for roseseraphinetucker.com" --json number,title,labels,body --state all',
      { encoding: 'utf-8' }
    );
    const issues = JSON.parse(rawOutput);
    expect(issues.length).toBeGreaterThan(0);

    const issue = issues.find(
      (i: { title: string }) =>
        i.title === '[Turnup] Verify Traefik IngressRoute and TLS for roseseraphinetucker.com'
    );
    expect(issue).toBeDefined();

    const labelNames = issue.labels.map((l: { name: string }) => l.name);
    expect(labelNames).toContain('container-ready');

    const body = issue.body;
    // Manifest target & domain
    expect(body).toContain('projects/rose/ingress.yaml');
    expect(body).toContain('roseseraphinetucker.com');

    // Ingress provider & TLS certificate provider
    expect(body).toContain('Traefik IngressRoute');
    expect(body).toContain('letsencrypt-prod');

    // DNS prerequisite
    expect(body).toMatch(/DNS/i);

    // Cross-references
    expect(body).toContain('brotherlogic/rose#315');
    expect(body).toContain('brotherlogic/rose#316');
    expect(body).toContain('brotherlogic/rose#322');
  });
});
