import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
import { execSync } from 'child_process';

describe('Milestone 3 End-to-End Container Verification Script (#289)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const scriptPath = path.join(rootDir, 'scripts', 'verify-milestone3-e2e.sh');

  it('should exist and be executable', () => {
    expect(fs.existsSync(scriptPath)).toBe(true);
    const stats = fs.statSync(scriptPath);
    expect(stats.mode & 0o111).toBeGreaterThan(0);
  });

  it('should have bash shebang and set -euo pipefail', () => {
    const content = fs.readFileSync(scriptPath, 'utf-8');
    const lines = content.split('\n');
    expect(lines[0]).toBe('#!/usr/bin/env bash');
    expect(content).toContain('set -euo pipefail');
  });

  it('should pass bash syntax validation (bash -n)', () => {
    const result = execSync(`bash -n "${scriptPath}"`, { encoding: 'utf-8' });
    expect(result).toBe('');
  });

  it('should implement Step 1: Verify static export build artifacts', () => {
    const content = fs.readFileSync(scriptPath, 'utf-8');
    expect(content).toMatch(/out\/index\.html/);
    expect(content).toMatch(/out\/_next\/static/);
  });

  it('should implement Step 2: Validate Nginx configuration syntax', () => {
    const content = fs.readFileSync(scriptPath, 'utf-8');
    expect(content).toMatch(/nginx\s+-t/);
  });

  it('should implement Step 3: Validate container healthcheck probe endpoint (/healthz)', () => {
    const content = fs.readFileSync(scriptPath, 'utf-8');
    expect(content).toContain('/healthz');
    expect(content).toMatch(/200/);
    expect(content).toMatch(/OK/);
  });

  it('should implement Step 4: Validate homepage HTML content and curatorial landmarks', () => {
    const content = fs.readFileSync(scriptPath, 'utf-8');
    expect(content).toContain('ROSE SERAPHINE TUCKER');
    expect(content).toContain('The Permanent Collection');
    expect(content).toContain('Editorial Statement');
    expect(content).toContain('Biography');
  });

  it('should implement Step 5: Validate media 404 isolation for nonexistent assets', () => {
    const content = fs.readFileSync(scriptPath, 'utf-8');
    expect(content).toMatch(/\/data\/thumbnails\//);
    expect(content).toMatch(/404/);
    expect(content).toContain('<!DOCTYPE html>');
  });

  it('should implement Step 6: Validate client routing deep links', () => {
    const content = fs.readFileSync(scriptPath, 'utf-8');
    expect(content).toContain('The%20Crayon%20Period');
  });
});
