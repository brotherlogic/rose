import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
import { execSync } from 'child_process';

describe('Milestone 3 Container Delivery and Static Export Verification (#291)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const dockerfilePath = path.join(rootDir, 'Dockerfile');
  const nginxConfPath = path.join(rootDir, 'nginx.conf');
  const verifyScriptPath = path.join(rootDir, 'scripts', 'verify-milestone3-e2e.sh');
  const validatePrWorkflowPath = path.join(rootDir, '.github', 'workflows', 'validate-pr.yml');
  const outIndexPath = path.join(rootDir, 'out', 'index.html');
  const outStaticDir = path.join(rootDir, 'out', '_next', 'static');

  it('should verify Dockerfile has healthcheck and volume mount scaffolding', () => {
    const dockerfile = fs.readFileSync(dockerfilePath, 'utf-8');
    expect(dockerfile).toMatch(/FROM\s+node:22-alpine\s+AS\s+builder/i);
    expect(dockerfile).toMatch(/FROM\s+nginx:alpine/i);
    expect(dockerfile).toContain('RUN mkdir -p /data/images /data/thumbnails');
    expect(dockerfile).toMatch(/HEALTHCHECK\s+--interval=30s/);
    expect(dockerfile).toContain('/healthz');
  });

  it('should verify nginx.conf has /healthz, NFS media routes, and try_files =404', () => {
    const nginxConf = fs.readFileSync(nginxConfPath, 'utf-8');
    expect(nginxConf).toContain('location = /healthz');
    expect(nginxConf).toContain("return 200 'OK';");
    expect(nginxConf).toContain('location /data/images/');
    expect(nginxConf).toContain('location /data/thumbnails/');
    expect(nginxConf).toContain('try_files $uri =404;');
    expect(nginxConf).toContain('add_header Access-Control-Allow-Origin "*";');
  });

  it('should verify scripts/verify-milestone3-e2e.sh exists, is executable, and passes bash -n', () => {
    expect(fs.existsSync(verifyScriptPath)).toBe(true);
    const stats = fs.statSync(verifyScriptPath);
    expect(stats.mode & 0o111).toBeGreaterThan(0);
    const syntaxCheck = execSync(`bash -n "${verifyScriptPath}"`, { encoding: 'utf-8' });
    expect(syntaxCheck).toBe('');
  });

  it('should verify .github/workflows/validate-pr.yml implements container PR validation and healthcheck', () => {
    const workflow = fs.readFileSync(validatePrWorkflowPath, 'utf-8');
    expect(workflow).toContain('docker build');
    expect(workflow).toContain('docker run');
    expect(workflow).toContain('/healthz');
    expect(workflow).toContain('ROSE SERAPHINE TUCKER');
    expect(workflow).toContain('The Permanent Collection');
    expect(workflow).toContain('/data/thumbnails/nonexistent.webp');
    expect(workflow).toContain('docker stop');
    expect(workflow).toContain('docker rm');
  });

  it('should verify static export artifacts in out/ contain branding and curatorial landmarks', () => {
    expect(fs.existsSync(outIndexPath)).toBe(true);
    expect(fs.existsSync(outStaticDir)).toBe(true);
    const indexHtml = fs.readFileSync(outIndexPath, 'utf-8');
    expect(indexHtml).toContain('ROSE SERAPHINE TUCKER');
    expect(indexHtml).toContain('The Permanent Collection');
    expect(indexHtml).toContain('Biography');
  });

  it('should verify end-to-end verification script succeeds', () => {
    const output = execSync(`"${verifyScriptPath}"`, { encoding: 'utf-8', cwd: rootDir });
    expect(output).toContain('SUCCESS: Milestone 3 End-to-End Verification Passed!');
  });
});
