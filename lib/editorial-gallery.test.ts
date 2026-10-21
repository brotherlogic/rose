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
import CuratedGalleryCollection from '../app/components/CuratedGalleryCollection';
import ExhibitionEmptyState from '../app/components/ExhibitionEmptyState';
import * as useGalleryFilterModule from '../app/hooks/useGalleryFilter';

function findElements(node: React.ReactNode, predicate: (el: React.ReactElement) => boolean): React.ReactElement[] {
  const matches: React.ReactElement[] = [];

  function traverse(current: React.ReactNode) {
    if (!current) return;
    if (Array.isArray(current)) {
      current.forEach(traverse);
      return;
    }
    if (React.isValidElement(current)) {
      if (predicate(current)) {
        matches.push(current);
      }
      if (current.props && (current.props as { children?: React.ReactNode }).children) {
        traverse((current.props as { children?: React.ReactNode }).children);
      }
    }
  }

  traverse(node);
  return matches;
}

describe('Editorial Gallery & Museum Shell Integration Suite (#255)', () => {
  beforeEach(() => {
    mockGetArtworks.mockReset();
    vi.restoreAllMocks();
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

      // Featured artwork omitted when artworks is empty
      expect(markup).not.toContain('id="featured-artwork"');

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

  describe('End-to-End Exhibition Filtering & Interactive Retrospective Suite (#279)', () => {
    const mockArtworks = [
      {
        id: 'art-001',
        title: 'Study in Crayon No. 1',
        medium: 'Wax crayon on salvaged paper',
        theme: 'The Crayon Period',
        thumbnailPath: '/data/thumbnails/art-001.webp',
        imagePath: '/data/images/art-001.jpg',
        timestamp: 1712707200, // Newer (2024-04-10)
      },
      {
        id: 'art-002',
        title: 'Domestic Construction No. 1',
        medium: 'Cardboard and packing tape',
        theme: 'Domestic Destructionism',
        thumbnailPath: '/data/thumbnails/art-002.webp',
        imagePath: '/data/images/art-002.jpg',
        timestamp: 1700000000, // Older (2023-11-14)
      },
      {
        id: 'art-003',
        title: 'Study in Crayon No. 2',
        medium: 'Crayon on recycled newsprint',
        theme: 'The Crayon Period',
        thumbnailPath: '/data/thumbnails/art-003.webp',
        imagePath: '/data/images/art-003.jpg',
        timestamp: 1710000000, // Mid (2024-03-09)
      },
    ];

    it('should verify complete homepage rendering with EditorialStatement, EditorialBiography, and #collection featuring CuratedGalleryCollection', () => {
      mockGetArtworks.mockReturnValue(mockArtworks);
      const markup = renderToStaticMarkup(React.createElement(Home));

      // EditorialStatement rendered
      const statementIdx = markup.indexOf('id="statement"');
      const featuredIdx = markup.indexOf('id="featured-artwork"');
      const biographyIdx = markup.indexOf('id="biography"');
      const collectionIdx = markup.indexOf('id="collection"');

      expect(statementIdx).toBeGreaterThan(-1);
      expect(featuredIdx).toBeGreaterThan(statementIdx);
      expect(biographyIdx).toBeGreaterThan(featuredIdx);
      expect(collectionIdx).toBeGreaterThan(biographyIdx);

      expect(markup).toContain('Curatorial Statement');

      // EditorialBiography rendered
      expect(markup).toContain('Historical Movements Taxonomy');

      // #collection rendered with CuratedGalleryCollection
      expect(markup).toContain('data-testid="curated-gallery-collection"');
      expect(markup).toContain('aria-label="Exhibition period filtering"');
      expect(markup).toContain('role="tablist"');
      expect(markup).toContain('All Works (3)');
      expect(markup).toContain('The Crayon Period (2)');
      expect(markup).toContain('Domestic Destructionism (1)');
      expect(markup).toContain('Newest First');
    });

    it('should verify movement filter pill selection filters artworks correctly', () => {
      mockGetArtworks.mockReturnValue(mockArtworks);
      vi.spyOn(useGalleryFilterModule, 'useGalleryFilter').mockReturnValue({
        selectedMovement: 'The Crayon Period',
        sortDirection: 'desc',
        setMovement: vi.fn(),
        setSortDirection: vi.fn(),
        resetFilter: vi.fn(),
      });

      const markup = renderToStaticMarkup(React.createElement(Home));
      const collectionMarkup = markup.slice(markup.indexOf('id="collection"'));

      expect(collectionMarkup).toContain('Study in Crayon No. 1');
      expect(collectionMarkup).toContain('Study in Crayon No. 2');
      expect(collectionMarkup).not.toContain('Domestic Construction No. 1');
    });

    it('should verify sort toggle reverses chronological order', () => {
      mockGetArtworks.mockReturnValue(mockArtworks);

      // Default desc (Newest first: art-001 [1712707200], art-003 [1710000000], art-002 [1700000000])
      vi.spyOn(useGalleryFilterModule, 'useGalleryFilter').mockReturnValue({
        selectedMovement: null,
        sortDirection: 'desc',
        setMovement: vi.fn(),
        setSortDirection: vi.fn(),
        resetFilter: vi.fn(),
      });
      const descMarkup = renderToStaticMarkup(React.createElement(Home));
      const descCollectionMarkup = descMarkup.slice(descMarkup.indexOf('id="collection"'));
      const descIdx1 = descCollectionMarkup.indexOf('Study in Crayon No. 1');
      const descIdx2 = descCollectionMarkup.indexOf('Domestic Construction No. 1');
      expect(descIdx1).toBeLessThan(descIdx2);

      // Ascending (Earliest first: art-002 [1700000000], art-003 [1710000000], art-001 [1712707200])
      vi.spyOn(useGalleryFilterModule, 'useGalleryFilter').mockReturnValue({
        selectedMovement: null,
        sortDirection: 'asc',
        setMovement: vi.fn(),
        setSortDirection: vi.fn(),
        resetFilter: vi.fn(),
      });
      const ascMarkup = renderToStaticMarkup(React.createElement(Home));
      const ascCollectionMarkup = ascMarkup.slice(ascMarkup.indexOf('id="collection"'));
      const ascIdx1 = ascCollectionMarkup.indexOf('Study in Crayon No. 1');
      const ascIdx2 = ascCollectionMarkup.indexOf('Domestic Construction No. 1');
      expect(ascIdx2).toBeLessThan(ascIdx1);
    });

    it('should verify filtering to empty movement displays curatorial empty notice and resets on button click', () => {
      mockGetArtworks.mockReturnValue(mockArtworks);
      const resetFilterMock = vi.fn();
      vi.spyOn(useGalleryFilterModule, 'useGalleryFilter').mockReturnValue({
        selectedMovement: 'Monochrome Nihilism',
        sortDirection: 'desc',
        setMovement: vi.fn(),
        setSortDirection: vi.fn(),
        resetFilter: resetFilterMock,
      });

      const markup = renderToStaticMarkup(React.createElement(Home));
      expect(markup).toContain('Monochrome Nihilism');
      expect(markup).toContain('No works from this movement are currently on public exhibition.');
      expect(markup).toContain('Return to All Works');

      // Verify reset action
      const collectionTree = CuratedGalleryCollection({ artworks: mockArtworks });
      const emptyStateEl = findElements(collectionTree, (el) => el.type === ExhibitionEmptyState)[0];
      expect(emptyStateEl).toBeDefined();
      emptyStateEl.props.onReset();
      expect(resetFilterMock).toHaveBeenCalledTimes(1);
    });

    it('should verify backward compatibility for artworks with missing or legacy theme_id', () => {
      mockGetArtworks.mockReturnValue([
        {
          id: 'art-legacy-01',
          title: 'Legacy Crayon Work',
          medium: 'Wax on wood',
          themeId: 'The Crayon Period', // Legacy property
          timestamp: 1600000000,
        },
        {
          id: 'art-legacy-02',
          title: 'Unthemed Early Work',
          medium: 'Pencil on scrap paper',
          // Missing theme / themeId
          timestamp: 1500000000,
        },
      ]);

      const markup = renderToStaticMarkup(React.createElement(Home));

      // Legacy themeId is properly aggregated under "The Crayon Period"
      expect(markup).toContain('The Crayon Period (1)');
      // Missing theme falls back gracefully to canonical "Early Works"
      expect(markup).toContain('Early Works (1)');
      expect(markup).toContain('Legacy Crayon Work');
      expect(markup).toContain('Unthemed Early Work');
    });
  });
});
