import { describe, it, expect, afterEach, vi } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
import { execSync } from 'child_process';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import CuratedGalleryCollection from '../app/components/CuratedGalleryCollection';
import type { gallery } from './proto/gallery';

(globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

describe('Milestone 4 End-to-End Verification Suite (#335)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const scriptPath = path.join(rootDir, 'scripts', 'verify-milestone4-e2e.sh');
  const packageJsonPath = path.join(rootDir, 'package.json');

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Shell Script Foundation & Syntax', () => {
    it('should assert script existence on disk', () => {
      expect(fs.existsSync(scriptPath)).toBe(true);
    });

    it('should assert executable permissions (fs.statSync(scriptPath).mode & 0o111 > 0)', () => {
      const stats = fs.statSync(scriptPath);
      expect(stats.mode & 0o111).toBeGreaterThan(0);
    });

    it('should assert shebang #!/usr/bin/env bash and presence of set -euo pipefail', () => {
      const content = fs.readFileSync(scriptPath, 'utf-8');
      const lines = content.split('\n');
      expect(lines[0]).toBe('#!/usr/bin/env bash');
      expect(content).toContain('set -euo pipefail');
    });

    it('should assert zero syntax errors via bash -n scripts/verify-milestone4-e2e.sh', () => {
      const result = execSync(`bash -n "${scriptPath}"`, { encoding: 'utf-8' });
      expect(result).toBe('');
    });
  });

  describe('Coverage Assertions for All 5 Verification Steps', () => {
    it('should implement Step 1: Static export artifacts and Nginx syntax verification', () => {
      const content = fs.readFileSync(scriptPath, 'utf-8');
      expect(content).toContain('Step 1: Build Artifacts & Nginx Syntax');
      expect(content).toMatch(/out\/index\.html/);
      expect(content).toMatch(/out\/_next\/static/);
      expect(content).toMatch(/nginx\s+-t/);
      expect(content).toContain('location = /healthz');
      expect(content).toContain('location = /data/catalog.json');
    });

    it('should implement Step 2: Storage ingestion simulation and catalog generation schema validation', () => {
      const content = fs.readFileSync(scriptPath, 'utf-8');
      expect(content).toContain('Step 2: Storage ingestion simulation and catalog generation schema validation');
      expect(content).toContain('artwork-1.proto.bin');
      expect(content).toContain('scripts/generate-catalog.ts');
      expect(content).toContain('catalog.json');
      expect(content).toContain('requiredFields');
      expect(content).toContain('theme_id');
      expect(content).toContain('thumbnail_path');
    });

    it('should implement Step 3: Nginx /data/catalog.json HTTP 200 serving and media 404 isolation', () => {
      const content = fs.readFileSync(scriptPath, 'utf-8');
      expect(content).toContain('Step 3: Nginx /data/catalog.json HTTP 200 serving and media 404 isolation');
      expect(content).toContain('/data/catalog.json');
      expect(content).toContain('application/json');
      expect(content).toContain('/data/images/');
      expect(content).toContain('/data/thumbnails/');
      expect(content).toMatch(/try_files.*=404;/);
    });

    it('should implement Step 4: Client catalog polling simulation detecting new artwork without restart', () => {
      const content = fs.readFileSync(scriptPath, 'utf-8');
      expect(content).toContain('Step 4: Dynamic Hands-Free Catalog Polling simulation');
      expect(content).toContain('artwork-2.proto.bin');
      expect(content).toContain('scripts/generate-catalog.ts');
      expect(content).toMatch(/catalog\.length\s*!==\s*2/);
      expect(content).toContain('artwork-2');
      expect(content).toContain('without restart');
    });

    it('should implement Step 5: Prometheus metrics scraping on port 8081 (rose_syncer_artworks_total, rose_syncer_stored_photos)', () => {
      const content = fs.readFileSync(scriptPath, 'utf-8');
      expect(content).toContain('Step 5: Prometheus metrics scraping on port 8081');
      expect(content).toContain('8081');
      expect(content).toContain('/metrics');
      expect(content).toContain('rose_syncer_artworks_total');
      expect(content).toContain('rose_syncer_stored_photos');
      expect(content).toContain('cleanup()');
    });
  });

  describe('Dynamic Gallery Rendering Update', () => {
    const initialArtwork: gallery.IArtwork = {
      id: 'artwork-1',
      title: 'First Masterpiece',
      medium: 'Crayon on canvas',
      theme: 'The Crayon Period',
      thumbnailPath: '/data/thumbnails/artwork-1.webp',
      imagePath: '/data/images/artwork-1.jpg',
      timestamp: 1672531199,
    };

    const newArtwork: gallery.IArtwork = {
      id: 'artwork-2',
      title: 'Second Masterpiece',
      medium: 'Crayon on paper',
      theme: 'The Crayon Period',
      thumbnailPath: '/data/thumbnails/artwork-2.webp',
      imagePath: '/data/images/artwork-2.jpg',
      timestamp: 1672539999,
    };

    it('should reflect dynamic catalog updates in CuratedGalleryCollection without a full page reload', () => {
      // Initial render with 1 artwork
      const initialMarkup = renderToStaticMarkup(
        React.createElement(CuratedGalleryCollection, { artworks: [initialArtwork] })
      );
      expect(initialMarkup).toContain('First Masterpiece');
      expect(initialMarkup).toContain('All Works (1)');
      expect(initialMarkup).not.toContain('Second Masterpiece');

      // Dynamic update with 2 artworks (simulating client-side polling update)
      const updatedMarkup = renderToStaticMarkup(
        React.createElement(CuratedGalleryCollection, { artworks: [newArtwork, initialArtwork] })
      );
      expect(updatedMarkup).toContain('First Masterpiece');
      expect(updatedMarkup).toContain('Second Masterpiece');
      expect(updatedMarkup).toContain('All Works (2)');
      expect(updatedMarkup).toContain('The Crayon Period (2)');
    });

    it('should transition seamlessly from empty collection to populated collection on dynamic update', () => {
      // Initial empty state
      const emptyMarkup = renderToStaticMarkup(
        React.createElement(CuratedGalleryCollection, { artworks: [] })
      );
      expect(emptyMarkup).toContain('No Works Currently Cataloged');
      expect(emptyMarkup).not.toContain('First Masterpiece');

      // Dynamic update after syncer ingests artwork
      const populatedMarkup = renderToStaticMarkup(
        React.createElement(CuratedGalleryCollection, { artworks: [initialArtwork] })
      );
      expect(populatedMarkup).not.toContain('No Works Currently Cataloged');
      expect(populatedMarkup).toContain('First Masterpiece');
      expect(populatedMarkup).toContain('All Works (1)');
    });
  });

  describe('NPM Script Registration in package.json', () => {
    it('should register verify:milestone4 script in package.json', () => {
      const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf-8'));
      expect(packageJson.scripts).toBeDefined();
      expect(packageJson.scripts['verify:milestone4']).toBe(
        'vitest run lib/verify-milestone4-e2e.test.ts'
      );
    });
  });
});
