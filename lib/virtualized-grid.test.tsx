import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import VirtualizedGalleryGrid, {
  getColumnCountForWidth,
  calculateGridTotalHeight,
} from '../app/components/VirtualizedGalleryGrid';
import { gallery } from './proto/gallery';

describe('Responsive Virtualized Gallery Grid Component Suite (#265)', () => {
  // Helper to generate N mock artworks
  const createMockArtworks = (count: number): gallery.IArtwork[] => {
    return Array.from({ length: count }, (_, i) => ({
      id: `artwork-${String(i + 1).padStart(4, '0')}`,
      title: `Archival Composition #${i + 1}`,
      medium: 'Mixed Media on Linen',
      description: `Description for artwork #${i + 1}`,
      thumbnailPath: `/data/thumbnails/artwork-${String(i + 1).padStart(4, '0')}.webp`,
      imagePath: `/data/images/artwork-${String(i + 1).padStart(4, '0')}.jpg`,
      timestamp: 1712707200 - i * 3600,
    }));
  };

  describe('Breakpoint-Based Responsive Column Calculation', () => {
    it('should assign 1 column for viewports < 640px', () => {
      expect(getColumnCountForWidth(320)).toBe(1);
      expect(getColumnCountForWidth(480)).toBe(1);
      expect(getColumnCountForWidth(639)).toBe(1);
    });

    it('should assign 2 columns for viewports between 640px and 1023px', () => {
      expect(getColumnCountForWidth(640)).toBe(2);
      expect(getColumnCountForWidth(768)).toBe(2);
      expect(getColumnCountForWidth(1023)).toBe(2);
    });

    it('should assign 3 columns for viewports between 1024px and 1439px', () => {
      expect(getColumnCountForWidth(1024)).toBe(3);
      expect(getColumnCountForWidth(1100)).toBe(3);
      expect(getColumnCountForWidth(1439)).toBe(3);
    });

    it('should assign 4 columns for viewports >= 1440px', () => {
      expect(getColumnCountForWidth(1440)).toBe(4);
      expect(getColumnCountForWidth(1600)).toBe(4);
      expect(getColumnCountForWidth(2560)).toBe(4);
    });
  });

  describe('Empty State Handling', () => {
    it('should render museum-grade empty state card when artworks array is empty', () => {
      const markup = renderToStaticMarkup(
        React.createElement(VirtualizedGalleryGrid, { artworks: [] })
      );
      expect(markup).toContain('No Works Currently Cataloged');
      expect(markup).toContain('Archival Status');
      expect(markup).toContain('Waiting for archival syncer ingestion');
    });
  });

  describe('Strict Bounding of Mounted DOM Nodes (1,000+ items)', () => {
    it('should strictly bound active mounted artwork card DOM nodes to < 50 items for 1,000+ items', () => {
      const artworks1000 = createMockArtworks(1000);
      const markup = renderToStaticMarkup(
        React.createElement(VirtualizedGalleryGrid, {
          artworks: artworks1000,
          columnCount: 4,
          rowHeight: 460,
          gap: 24,
        })
      );

      // Count occurrences of artwork article or title occurrences in markup
      const cardMatches = markup.match(/Archival Composition #\d+/g) || [];
      expect(cardMatches.length).toBeGreaterThan(0);
      expect(cardMatches.length).toBeLessThan(50);
    });

    it('should set total outer container height to totalRows * rowHeight + (totalRows - 1) * gap', () => {
      const artworks100 = createMockArtworks(100);
      const rowHeight = 400;
      const gap = 32;
      const columnCount = 4;
      const totalRows = Math.ceil(100 / columnCount); // 25 rows
      const expectedHeight = totalRows * rowHeight + (totalRows - 1) * gap; // 25*400 + 24*32 = 10000 + 768 = 10768

      expect(calculateGridTotalHeight(100, columnCount, rowHeight, gap)).toBe(expectedHeight);

      const markup = renderToStaticMarkup(
        React.createElement(VirtualizedGalleryGrid, {
          artworks: artworks100,
          columnCount,
          rowHeight,
          gap,
        })
      );
      expect(markup).toContain(`height:${expectedHeight}px`);
    });

    it('should position the rendered rows slice using absolute translation (transform: translateY)', () => {
      const artworks50 = createMockArtworks(50);
      const markup = renderToStaticMarkup(
        React.createElement(VirtualizedGalleryGrid, {
          artworks: artworks50,
          columnCount: 4,
          startIndex: 2,
          endIndex: 6,
          rowHeight: 400,
          gap: 20,
        })
      );

      // Start offset for row index 2: 2 * (400 + 20) = 840px
      expect(markup).toMatch(/transform:\s*translateY\(840px\)/);
    });
  });

  describe('Windowing Slice Updates and Scroll Simulation', () => {
    let mockScrollListeners: ((e: unknown) => void)[] = [];
    let mockResizeListeners: ((e: unknown) => void)[] = [];
    let rafCallbacks: ((timestamp: number) => void)[] = [];

    beforeEach(() => {
      mockScrollListeners = [];
      mockResizeListeners = [];
      rafCallbacks = [];

      vi.stubGlobal('requestAnimationFrame', (cb: (timestamp: number) => void) => {
        rafCallbacks.push(cb);
        return rafCallbacks.length;
      });

      vi.stubGlobal('cancelAnimationFrame', vi.fn());

      vi.stubGlobal('window', {
        innerHeight: 800,
        innerWidth: 1200,
        scrollY: 0,
        addEventListener: (event: string, listener: (e: unknown) => void) => {
          if (event === 'scroll') mockScrollListeners.push(listener);
          if (event === 'resize') mockResizeListeners.push(listener);
        },
        removeEventListener: (event: string, listener: (e: unknown) => void) => {
          if (event === 'scroll') {
            mockScrollListeners = mockScrollListeners.filter((l) => l !== listener);
          }
          if (event === 'resize') {
            mockResizeListeners = mockResizeListeners.filter((l) => l !== listener);
          }
        },
      });
    });

    afterEach(() => {
      vi.unstubAllGlobals();
    });

    it('should compute windowing slice with 2-row overscan buffer', () => {
      const artworks1000 = createMockArtworks(1000);
      const rowHeight = 400;
      const gap = 20; // effectiveRowHeight = 420
      const columnCount = 4; // totalRows = 250
      const overscan = 2;

      // Suppose scroll position is at row 10 (scrollTop = 4200)
      // visibleStartRow = Math.floor(4200 / 420) = 10
      // visibleEndRow = Math.ceil((4200 + 800) / 420) = 12
      // startIndex = Math.max(0, 10 - 2) = 8
      // endIndex = Math.min(250, 12 + 2) = 14
      // active rows = 14 - 8 = 6 rows = 24 items (< 50 items)
      const markup = renderToStaticMarkup(
        React.createElement(VirtualizedGalleryGrid, {
          artworks: artworks1000,
          columnCount,
          rowHeight,
          gap,
          overscan,
          initialScrollTop: 4200,
          viewportHeight: 800,
        })
      );

      // Row 8 starts at artwork index 8 * 4 = 32 -> Artwork #33
      expect(markup).toContain('Archival Composition #33');
      // Row 13 ends at artwork index 14 * 4 - 1 = 55 -> Artwork #56
      expect(markup).toContain('Archival Composition #56');
      // Artwork before overscan (e.g. Artwork #10) should NOT be mounted
      expect(markup).not.toContain('Archival Composition #10');
      // Artwork after overscan (e.g. Artwork #80) should NOT be mounted
      expect(markup).not.toContain('Archival Composition #80');
    });
  });
});
