import { describe, it, expect, vi, beforeEach } from 'vitest';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

vi.mock('next/font/google', () => ({
  Geist: () => ({ variable: '--font-geist-sans' }),
  Geist_Mono: () => ({ variable: '--font-geist-mono' }),
}));

vi.mock('next/link', () => ({
  default: ({ href, children, ...props }: { href: string; children: React.ReactNode; [key: string]: unknown }) =>
    React.createElement('a', { href, ...props }, children),
}));

// Mock nfs module for getArtworks
const mockGetArtworks = vi.fn();
vi.mock('@/lib/nfs', () => ({
  getArtworks: () => mockGetArtworks(),
}));

import RootLayout, { metadata } from '../app/layout';
import Home from '../app/page';
import Footer from '../app/components/Footer';

describe('Editorial Gallery & Museum Shell Integration Suite (#255)', () => {
  beforeEach(() => {
    mockGetArtworks.mockReset();
  });

  describe('Austere Curatorial Footer (app/components/Footer.tsx)', () => {
    it('should render semantic <footer role="contentinfo">', () => {
      const markup = renderToStaticMarkup(React.createElement(Footer));
      expect(markup).toContain('<footer');
      expect(markup).toContain('role="contentinfo"');
    });

    it('should display institutional credits, curatorial copyright notices, and archive colophon', () => {
      const markup = renderToStaticMarkup(React.createElement(Footer));
      expect(markup).toContain('Rose Seraphine Tucker');
      expect(markup).toMatch(/Institutional Credits|Curatorial Archive|Archive/i);
      expect(markup).toMatch(/©\s*2026|Copyright/i);
      expect(markup).toMatch(/Colophon/i);
      expect(markup).toMatch(/archival syncer|catalog/i);
    });
  });

  describe('Homepage Composition & Editorial Sections (app/page.tsx)', () => {
    it('should compose EditorialStatement, EditorialBiography, and #collection section', () => {
      mockGetArtworks.mockReturnValue([]);
      const markup = renderToStaticMarkup(React.createElement(Home));

      // Statement section
      expect(markup).toContain('id="statement"');
      expect(markup).toContain('Curatorial Statement');

      // Biography section
      expect(markup).toContain('id="biography"');
      expect(markup).toContain('Historical Movements Taxonomy');

      // Collection section
      expect(markup).toContain('id="collection"');
    });

    it('should render all 5 canonical biography movements on homepage', () => {
      mockGetArtworks.mockReturnValue([]);
      const markup = renderToStaticMarkup(React.createElement(Home));

      expect(markup).toContain('The Crayon Period');
      expect(markup).toContain('Domestic Destructionism');
      expect(markup).toContain('Kinetic Scribblism');
      expect(markup).toContain('Found Object Assemblage');
      expect(markup).toContain('Monochrome Nihilism');
    });

    it('should render the austere, museum-grade empty state card when artworks.length === 0', () => {
      mockGetArtworks.mockReturnValue([]);
      const markup = renderToStaticMarkup(React.createElement(Home));

      expect(markup).toContain('id="collection"');
      expect(markup).toContain('No Works Currently Cataloged');
      expect(markup).toMatch(/waiting for archival syncer ingestion/i);
    });

    it('should render cataloged artworks in VirtualizedGalleryGrid when artworks are present', () => {
      mockGetArtworks.mockReturnValue([
        {
          id: 'artwork-001',
          title: 'Study in Crayon No. 4',
          medium: 'Wax crayon on salvaged panel',
          imagePath: '/images/study-4.jpg',
        },
      ]);
      const markup = renderToStaticMarkup(React.createElement(Home));

      expect(markup).not.toContain('No Works Currently Cataloged');
      expect(markup).toContain('Study in Crayon No. 4');
      expect(markup).toContain('Wax crayon on salvaged panel');
      expect(markup).toContain('/images/study-4.jpg');
      expect(markup).toContain('aspect-square');
    });

    it('should prioritize thumbnailPath over imagePath when present in loaded archive state', () => {
      mockGetArtworks.mockReturnValue([
        {
          id: 'artwork-002',
          title: 'Domestic Construction No. 2',
          medium: 'Torn cardboard and adhesive',
          thumbnailPath: '/data/thumbnails/artwork-002.webp',
          imagePath: '/data/images/artwork-002.jpg',
        },
      ]);
      const markup = renderToStaticMarkup(React.createElement(Home));

      expect(markup).toContain('Domestic Construction No. 2');
      expect(markup).toContain('Torn cardboard and adhesive');
      expect(markup).toContain('/data/thumbnails/artwork-002.webp');
    });

    it('should fallback to placeholder.svg when thumbnail and image paths are absent', () => {
      mockGetArtworks.mockReturnValue([
        {
          id: 'artwork-003',
          title: 'Untitled Composition',
        },
      ]);
      const markup = renderToStaticMarkup(React.createElement(Home));

      expect(markup).toContain('Untitled Composition');
      expect(markup).toContain('/placeholder.svg');
    });
  });

  describe('Semantic Landmarks & Shell Integration', () => {
    it('should render semantic landmarks: header role="banner", nav, main role="main", footer role="contentinfo"', () => {
      mockGetArtworks.mockReturnValue([]);
      const markup = renderToStaticMarkup(
        React.createElement(
          RootLayout,
          null,
          React.createElement(Home)
        )
      );

      // Header landmark
      expect(markup).toContain('role="banner"');
      expect(markup).toContain('<header');

      // Nav landmark
      expect(markup).toContain('<nav');

      // Main landmark
      expect(markup).toContain('role="main"');
      expect(markup).toContain('<main');

      // Footer landmark
      expect(markup).toContain('role="contentinfo"');
      expect(markup).toContain('<footer');
    });

    it('should render desktop navigation links and mobile drawer toggle accessibility', () => {
      mockGetArtworks.mockReturnValue([]);
      const markup = renderToStaticMarkup(
        React.createElement(
          RootLayout,
          null,
          React.createElement(Home)
        )
      );

      // Desktop navigation anchors
      expect(markup).toContain('href="#statement"');
      expect(markup).toContain('href="#biography"');
      expect(markup).toContain('href="#collection"');

      // Mobile drawer toggle accessibility
      expect(markup).toContain('aria-label="Toggle navigation menu"');
      expect(markup).toContain('aria-expanded="false"');
    });

    it('should verify metadata export integrity in app/layout.tsx', () => {
      expect(metadata.title).toBe('Rose Seraphine Tucker — Retrospective');
      expect(metadata.description).toBe(
        'An austere contemporary retrospective presenting the foundational and conceptual works of Rose Seraphine Tucker.'
      );
      expect(metadata.openGraph).toBeDefined();
      expect(metadata.openGraph?.title).toBe('Rose Seraphine Tucker — Retrospective');
      expect(metadata.openGraph?.siteName).toBe('Rose Seraphine Tucker Retrospective');
    });
  });
});
