import { describe, it, expect, vi } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

vi.mock('next/font/google', () => ({
  Geist: () => ({ variable: '--font-geist-sans' }),
  Geist_Mono: () => ({ variable: '--font-geist-mono' }),
}));

import RootLayout, { metadata } from '../app/layout';

describe('Root Layout Metadata & Design System Shell (#252)', () => {
  describe('Curatorial Metadata Exports', () => {
    it('should export the exact retrospective title and description', () => {
      expect(metadata.title).toBe('Rose Seraphine Tucker — Retrospective');
      expect(metadata.description).toBe(
        'An austere contemporary retrospective presenting the foundational and conceptual works of Rose Seraphine Tucker.'
      );
    });

    it('should configure OpenGraph metadata specifications accurately', () => {
      const og = metadata.openGraph as Record<string, unknown> | undefined;
      expect(og).toBeDefined();
      expect(og?.title).toBe('Rose Seraphine Tucker — Retrospective');
      expect(og?.description).toBe(
        'An austere contemporary retrospective presenting the foundational and conceptual works of Rose Seraphine Tucker.'
      );
      expect(og?.siteName).toBe('Rose Seraphine Tucker Retrospective');
      expect(og?.type).toBe('website');
      expect(og?.locale).toBe('en_US');
    });
  });

  describe('Root Layout Semantic Landmarks', () => {
    it('should render header (role="banner"), main (role="main"), and footer (role="contentinfo")', () => {
      const markup = renderToStaticMarkup(
        React.createElement(
          RootLayout,
          null,
          React.createElement('div', { id: 'test-content' }, 'Exhibition Gallery Body')
        )
      );

      // Verify semantic HTML landmarks
      expect(markup).toContain('role="banner"');
      expect(markup).toContain('<header');
      expect(markup).toContain('role="main"');
      expect(markup).toContain('<main');
      expect(markup).toContain('role="contentinfo"');
      expect(markup).toContain('<footer');

      // Verify children rendered inside
      expect(markup).toContain('id="test-content"');
      expect(markup).toContain('Exhibition Gallery Body');
    });
  });

  describe('Design System Tokens in globals.css', () => {
    const cssPath = path.resolve(__dirname, '../app/globals.css');
    const cssContent = fs.readFileSync(cssPath, 'utf-8');

    it('should define white cube light theme custom properties in :root', () => {
      expect(cssContent).toMatch(/--bg-primary:\s*#ffffff;/);
      expect(cssContent).toMatch(/--bg-secondary:\s*#f9f9f9;/);
      expect(cssContent).toMatch(/--text-primary:\s*#111111;/);
      expect(cssContent).toMatch(/--text-secondary:\s*#555555;/);
      expect(cssContent).toMatch(/--border-subtle:\s*#e5e5e5;/);
      expect(cssContent).toMatch(/--border-focus:\s*#000000;/);
    });

    it('should define dark luxury theme custom properties in @media (prefers-color-scheme: dark)', () => {
      expect(cssContent).toMatch(/--bg-primary:\s*#080808;/);
      expect(cssContent).toMatch(/--bg-secondary:\s*#121212;/);
      expect(cssContent).toMatch(/--text-primary:\s*#f0f0f0;/);
      expect(cssContent).toMatch(/--text-secondary:\s*#999999;/);
      expect(cssContent).toMatch(/--border-subtle:\s*#262626;/);
      expect(cssContent).toMatch(/--border-focus:\s*#ffffff;/);
    });

    it('should define typography font families and tracking tokens', () => {
      expect(cssContent).toMatch(/--font-serif:/);
      expect(cssContent).toMatch(/--font-sans:/);
      expect(cssContent).toMatch(/--tracking-tight:\s*-0\.02em;/);
      expect(cssContent).toMatch(/--tracking-wide:\s*0\.05em;/);
      expect(cssContent).toMatch(/--tracking-widest:\s*0\.15em;/);
    });

    it('should define header height and anchor scroll margin offsets for sections', () => {
      expect(cssContent).toMatch(/--header-height:\s*64px;/);
      expect(cssContent).toContain('#statement');
      expect(cssContent).toContain('#biography');
      expect(cssContent).toContain('#collection');
      expect(cssContent).toMatch(/scroll-margin-top:\s*calc\(var\(--header-height\)\s*\+\s*1\.5rem\);/);
    });
  });
});
