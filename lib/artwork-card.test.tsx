import { describe, it, expect, vi } from 'vitest';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import * as fs from 'fs';
import * as path from 'path';
import ArtworkCard from '../app/components/ArtworkCard';
import { gallery } from './proto/gallery';

// Helper to recursively find elements by predicate
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

describe('Museum Artwork Card Component Suite (#264)', () => {
  const mockArtwork: gallery.IArtwork = {
    id: 'artwork-001',
    title: 'Study in Crimson and Charcoal',
    medium: 'Oil and pastel on archival linen',
    description: 'An exploration of domesticity and destructive marks.',
    thumbnailPath: '/data/thumbnails/artwork-001.webp',
    imagePath: '/data/images/artwork-001.jpg',
    themeId: 'kinetic-scribblism',
    timestamp: 1712707200,
  };

  describe('Curatorial Fallback Asset (public/placeholder.svg)', () => {
    const placeholderPath = path.join(process.cwd(), 'public', 'placeholder.svg');

    it('should exist in public directory', () => {
      expect(fs.existsSync(placeholderPath)).toBe(true);
    });

    it('should be a valid minimalist monochrome SVG matching gallery aesthetic', () => {
      const svgContent = fs.readFileSync(placeholderPath, 'utf8');
      expect(svgContent).toContain('<svg');
      expect(svgContent).toContain('</svg>');
      // Monochrome palette check
      expect(svgContent).toMatch(/#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})|currentColor|none/);
    });
  });

  describe('ArtworkCard Component (app/components/ArtworkCard.tsx)', () => {
    describe('Aspect Ratio Container & CLS Prevention', () => {
      it('should render a fixed aspect ratio container with subtle border for zero CLS', () => {
        const markup = renderToStaticMarkup(React.createElement(ArtworkCard, { artwork: mockArtwork }));
        expect(markup).toContain('aspect-square');
        expect(markup).toMatch(/border\s+border-\[var\(--border-subtle\)\]/);
      });
    });

    describe('Skeleton Shimmer Placeholder & Image Fade-In', () => {
      it('should render background skeleton shimmer placeholder and lazy-loaded image', () => {
        const markup = renderToStaticMarkup(React.createElement(ArtworkCard, { artwork: mockArtwork }));
        // Background skeleton shimmer placeholder while image loads
        expect(markup).toMatch(/animate-pulse|shimmer/);
        // Image element with loading="lazy" and WebP thumbnail
        expect(markup).toContain('loading="lazy"');
        expect(markup).toContain('/data/thumbnails/artwork-001.webp');
      });

      it('should include smooth fade-in transition classes (opacity-0 initial state)', () => {
        const markup = renderToStaticMarkup(React.createElement(ArtworkCard, { artwork: mockArtwork }));
        expect(markup).toContain('opacity-0');
        expect(markup).toMatch(/transition-opacity|transition/);
      });
    });

    describe('Image Fallback & Error Handling', () => {
      it('should fallback to /placeholder.svg when thumbnailPath is absent', () => {
        const artworkWithoutThumbnail: gallery.IArtwork = {
          id: 'artwork-no-thumb',
          title: 'Monochrome Void',
          medium: 'Gouache on paper',
        };
        const markup = renderToStaticMarkup(React.createElement(ArtworkCard, { artwork: artworkWithoutThumbnail }));
        expect(markup).toContain('/placeholder.svg');
      });

      it('should gracefully switch to /placeholder.svg on image load error', () => {
        const markup = renderToStaticMarkup(
          React.createElement(ArtworkCard, {
            artwork: mockArtwork,
            initialError: true,
          })
        );
        expect(markup).toContain('src="/placeholder.svg"');
        expect(markup).toContain('opacity-100');
      });

      it('should bind onError event handler to switch to /placeholder.svg', () => {
        const componentPath = path.join(process.cwd(), 'app/components/ArtworkCard.tsx');
        const componentCode = fs.readFileSync(componentPath, 'utf8');
        expect(componentCode).toContain('onError');
        expect(componentCode).toContain('/placeholder.svg');
      });
    });

    describe('Archival Typography & Truncation', () => {
      it('should render avant-garde typography for title with truncation', () => {
        const markup = renderToStaticMarkup(React.createElement(ArtworkCard, { artwork: mockArtwork }));
        expect(markup).toContain('Study in Crimson and Charcoal');
        expect(markup).toMatch(/font-\[family-name:var\(--font-serif\)\]/);
        expect(markup).toContain('text-base');
        expect(markup).toContain('truncate');
      });

      it('should render monospace uppercase caption for medium with truncation', () => {
        const markup = renderToStaticMarkup(React.createElement(ArtworkCard, { artwork: mockArtwork }));
        expect(markup).toContain('Oil and pastel on archival linen');
        expect(markup).toContain('font-mono');
        expect(markup).toContain('text-xs');
        expect(markup).toMatch(/text-\[var\(--text-secondary\)\]/);
        expect(markup).toContain('uppercase');
        expect(markup).toContain('tracking-wider');
        expect(markup).toContain('truncate');
      });

      it('should fallback gracefully when title or medium are missing', () => {
        const emptyArtwork: gallery.IArtwork = {
          id: 'empty-artwork',
        };
        const markup = renderToStaticMarkup(React.createElement(ArtworkCard, { artwork: emptyArtwork }));
        expect(markup).toContain('Untitled');
        expect(markup).toContain('Mixed Media');
      });

      it('should gracefully truncate excessively long titles and mediums', () => {
        const longArtwork: gallery.IArtwork = {
          id: 'artwork-long',
          title: 'A Very Long Archival Artwork Title That Exceeds Standard Container Boundaries In Curatorial Exhibitions',
          medium: 'Mixed Media Assemblage With Domestic Objects, Wax, Crayons, And Found Ephemera On Distressed Archival Canvas',
          thumbnailPath: '/data/thumbnails/artwork-long.webp',
        };
        const markup = renderToStaticMarkup(React.createElement(ArtworkCard, { artwork: longArtwork }));
        expect(markup).toContain('A Very Long Archival Artwork Title');
        expect(markup).toContain('truncate');
      });
    });

    describe('Interactive Dialog Trigger Button & Selection (#298)', () => {
      it('should wrap the card image container in an accessible interactive trigger button', () => {
        const markup = renderToStaticMarkup(React.createElement(ArtworkCard, { artwork: mockArtwork }));
        expect(markup).toContain('aria-haspopup="dialog"');
        expect(markup).toMatch(/<button[^>]+type="button"[^>]+aria-haspopup="dialog"/);
      });

      it('should invoke onSelect callback with artwork and trigger element when clicked', () => {
        const onSelectMock = vi.fn();
        let captured: React.ReactElement | null = null;
        function Wrapper() {
          const el = ArtworkCard({ artwork: mockArtwork, onSelect: onSelectMock });
          captured = el;
          return el;
        }
        renderToStaticMarkup(React.createElement(Wrapper));
        const buttons = findElements(captured, (el) => el.type === 'button' && el.props['aria-haspopup'] === 'dialog');
        expect(buttons.length).toBe(1);
        const triggerBtn = buttons[0];
        const mockTarget = { tagName: 'BUTTON' } as unknown as HTMLElement;
        triggerBtn.props.onClick({ currentTarget: mockTarget, preventDefault: vi.fn() });
        expect(onSelectMock).toHaveBeenCalledWith(mockArtwork, mockTarget);
      });

      it('should invoke onSelect callback on Enter and Space keyboard events', () => {
        const onSelectMock = vi.fn();
        let captured: React.ReactElement | null = null;
        function Wrapper() {
          const el = ArtworkCard({ artwork: mockArtwork, onSelect: onSelectMock });
          captured = el;
          return el;
        }
        renderToStaticMarkup(React.createElement(Wrapper));
        const buttons = findElements(captured, (el) => el.type === 'button' && el.props['aria-haspopup'] === 'dialog');
        expect(buttons.length).toBe(1);
        const triggerBtn = buttons[0];
        const mockTarget = { tagName: 'BUTTON' } as unknown as HTMLElement;

        triggerBtn.props.onKeyDown({ key: 'Enter', preventDefault: vi.fn(), currentTarget: mockTarget });
        expect(onSelectMock).toHaveBeenCalledTimes(1);
        expect(onSelectMock).toHaveBeenCalledWith(mockArtwork, mockTarget);

        triggerBtn.props.onKeyDown({ key: ' ', preventDefault: vi.fn(), currentTarget: mockTarget });
        expect(onSelectMock).toHaveBeenCalledTimes(2);
        expect(onSelectMock).toHaveBeenLastCalledWith(mockArtwork, mockTarget);
      });
    });
  });
});

