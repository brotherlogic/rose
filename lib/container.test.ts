import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';

describe('Containerization Configuration', () => {
  const rootDir = path.resolve(__dirname, '..');
  const dockerfilePath = path.join(rootDir, 'Dockerfile');
  const nginxConfPath = path.join(rootDir, 'nginx.conf');
  const dockerignorePath = path.join(rootDir, '.dockerignore');

  describe('Dockerfile', () => {
    it('should exist', () => {
      expect(fs.existsSync(dockerfilePath)).toBe(true);
    });

    it('should be a multi-stage build with builder and runner stages', () => {
      const content = fs.readFileSync(dockerfilePath, 'utf-8');
      expect(content).toMatch(/FROM\s+node:22-alpine\s+AS\s+builder/i);
      expect(content).toMatch(/FROM\s+nginx:alpine/i);
    });

    it('should install dependencies with npm ci and build the app', () => {
      const content = fs.readFileSync(dockerfilePath, 'utf-8');
      expect(content).toMatch(/npm\s+ci/);
      expect(content).toMatch(/npm\s+run\s+build/);
    });

    it('should copy static bundle and custom nginx config, expose 80, and set CMD', () => {
      const content = fs.readFileSync(dockerfilePath, 'utf-8');
      expect(content).toMatch(/COPY\s+--from=builder\s+.*\/app\/out\s+\/usr\/share\/nginx\/html/);
      expect(content).toMatch(/COPY\s+nginx\.conf\s+\/etc\/nginx\/conf\.d\/default\.conf/);
      expect(content).toMatch(/EXPOSE\s+80/);
      expect(content).toMatch(/CMD\s+\["nginx",\s*"-g",\s*"daemon off;"\]/);
    });
  });

  describe('nginx.conf', () => {
    it('should exist', () => {
      expect(fs.existsSync(nginxConfPath)).toBe(true);
    });

    it('should configure gzip compression for text, css, json, js, xml, svg', () => {
      const content = fs.readFileSync(nginxConfPath, 'utf-8');
      expect(content).toMatch(/gzip\s+on;/);
      expect(content).toMatch(/gzip_types/);
      expect(content).toContain('text/plain');
      expect(content).toContain('text/css');
      expect(content).toContain('application/json');
      expect(content).toContain('application/javascript');
      expect(content).toContain('image/svg+xml');
    });

    it('should configure long-term caching for static assets under /_next/static/', () => {
      const content = fs.readFileSync(nginxConfPath, 'utf-8');
      expect(content).toMatch(/location\s+\/_next\/static\//);
      expect(content).toContain('public, max-age=31536000, immutable');
    });

    it('should configure media caching with max-age=604800', () => {
      const content = fs.readFileSync(nginxConfPath, 'utf-8');
      expect(content).toContain('public, max-age=604800');
    });

    it('should configure SPA routing fallback for static HTML pages', () => {
      const content = fs.readFileSync(nginxConfPath, 'utf-8');
      expect(content).toContain('try_files $uri $uri.html $uri/ /index.html =404;');
    });
  });

  describe('.dockerignore', () => {
    it('should exist', () => {
      expect(fs.existsSync(dockerignorePath)).toBe(true);
    });

    it('should exclude artifacts, caches, and dev configuration', () => {
      const content = fs.readFileSync(dockerignorePath, 'utf-8');
      const lines = content.split('\n').map((l) => l.trim()).filter((l) => l && !l.startsWith('#'));
      expect(lines).toContain('node_modules');
      expect(lines).toContain('.next');
      expect(lines).toContain('out');
      expect(lines).toContain('.git');
      expect(lines).toContain('.github');
      expect(lines).toContain('.devcontainer');
      expect(lines).toContain('*.log');
      expect(lines).toContain('*.bin');
    });
  });
});
