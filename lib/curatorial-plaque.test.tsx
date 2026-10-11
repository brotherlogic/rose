import { describe, it, expect, vi } from 'vitest';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import CuratorialPlaque from '../app/components/CuratorialPlaque';
import { gallery } from './proto/gallery';

// Helper to recursively find elements by predicate in a React element tree
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

describe('Museum Curatorial Plaque Component Suite (#295)', () => {
  const mockArtwork: gallery.IArtwork = {
    id: 'artwork-101',
    title: 'Disintegration of Form in Yellow',
    medium: 'Wax crayon on reclaimed cardboard',
    description: 'A study in material decay and existential resignation.',
    themeId: 'Monochrome Nihilism',
    timestamp: 1712707200, // 2024-04-10
  };

  describe('Archival Typography & Hierarchy', () => {
    it('should render avant-garde title in serif italic typography with id="curatorial-plaque-title"', () => {
      const markup = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: mockArtwork,
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );

      expect(markup).toContain('id="curatorial-plaque-title"');
      expect(markup).toContain('Disintegration of Form in Yellow');
      expect(markup).toMatch(/font-\[family-name:var\(--font-serif\)\]/);
      expect(markup).toContain('italic');
      expect(markup).toMatch(/text-lg|text-xl/);
      expect(markup).toContain('text-[var(--text-primary)]');
    });

    it('should render artistic medium in uppercase monospace tracking-wider', () => {
      const markup = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: mockArtwork,
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );

      expect(markup).toContain('Wax crayon on reclaimed cardboard');
      expect(markup).toMatch(/font-mono/);
      expect(markup).toMatch(/uppercase/);
      expect(markup).toMatch(/tracking-wider/);
      expect(markup).toMatch(/text-\[var\(--text-secondary\)\]/);
    });

    it('should render solemn deadpan philosophical critique text', () => {
      const markup = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: mockArtwork,
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );

      expect(markup).toContain('A study in material decay and existential resignation.');
    });

    it('should display thematic movement badge resolved from artwork.themeId', () => {
      const markup = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: mockArtwork,
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );

      expect(markup).toContain('Monochrome Nihilism');
    });
  });

  describe('Creation Year Formatting', () => {
    it('should format 4-digit calendar year from numeric timestamp (1712707200 -> "2024")', () => {
      const markup = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: { ...mockArtwork, timestamp: 1712707200 },
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );

      expect(markup).toContain('2024');
    });

    it('should handle timestamp objects with toNumber() method', () => {
      const markup = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: {
            ...mockArtwork,
            timestamp: { toNumber: () => 1712707200 } as unknown as number,
          },
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );

      expect(markup).toContain('2024');
    });

    it('should gracefully display "Undated" when timestamp is 0', () => {
      const markup = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: { ...mockArtwork, timestamp: 0 },
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );

      expect(markup).toContain('Undated');
      expect(markup).not.toContain('1970');
    });

    it('should gracefully display "Undated" when timestamp is missing or null', () => {
      const markup = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: { ...mockArtwork, timestamp: null },
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );

      expect(markup).toContain('Undated');
    });

    it('should gracefully display "Undated" when timestamp is invalid', () => {
      const markup = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: { ...mockArtwork, timestamp: NaN },
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );

      expect(markup).toContain('Undated');
    });
  });

  describe('Default Fallbacks for Missing or Empty Fields', () => {
    it('should fallback title to "Untitled" when missing, empty, or whitespace', () => {
      const markupEmpty = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: { ...mockArtwork, title: '   ' },
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );
      expect(markupEmpty).toContain('Untitled');

      const markupNull = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: { ...mockArtwork, title: null as unknown as string },
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );
      expect(markupNull).toContain('Untitled');
    });

    it('should fallback medium to "Mixed Media" when absent or whitespace', () => {
      const markupEmpty = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: { ...mockArtwork, medium: '   ' },
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );
      expect(markupEmpty).toContain('Mixed Media');

      const markupMissing = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: { ...mockArtwork, medium: undefined },
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );
      expect(markupMissing).toContain('Mixed Media');
    });

    it('should fallback curatorial critique to "Catalog description pending curatorial annotation." when empty or whitespace', () => {
      const markupEmpty = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: { ...mockArtwork, description: '   ' },
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );
      expect(markupEmpty).toContain('Catalog description pending curatorial annotation.');

      const markupNull = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: { ...mockArtwork, description: null as unknown as string },
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );
      expect(markupNull).toContain('Catalog description pending curatorial annotation.');
    });

    it('should fallback thematic movement badge to "Early Works" when themeId is absent or whitespace', () => {
      const markupEmpty = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: { ...mockArtwork, themeId: '   ' },
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );
      expect(markupEmpty).toContain('Early Works');

      const markupMissing = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: { ...mockArtwork, themeId: undefined },
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );
      expect(markupMissing).toContain('Early Works');
    });
  });

  describe('Collapsible Toggle & Accessibility States', () => {
    it('should render expanded state with aria-expanded="true" and aria-label="Collapse curatorial plaque"', () => {
      const markup = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: mockArtwork,
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );

      expect(markup).toContain('aria-expanded="true"');
      expect(markup).toContain('aria-label="Collapse curatorial plaque"');
      // Full details are visible
      expect(markup).toContain('Disintegration of Form in Yellow');
      expect(markup).toContain('Wax crayon on reclaimed cardboard');
      expect(markup).toContain('A study in material decay and existential resignation.');
    });

    it('should render collapsed state with aria-expanded="false" and aria-label="Expand curatorial plaque"', () => {
      const markup = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: mockArtwork,
          isCollapsed: true,
          onToggleCollapse: vi.fn(),
        })
      );

      expect(markup).toContain('aria-expanded="false"');
      expect(markup).toContain('aria-label="Expand curatorial plaque"');
      // In collapsed state, the expanded critique should not be rendered
      expect(markup).not.toContain('A study in material decay and existential resignation.');
      // id="curatorial-plaque-title" remains accessible for modal labeling
      expect(markup).toContain('id="curatorial-plaque-title"');
    });

    it('should trigger onToggleCollapse when clicking toggle button in expanded state', () => {
      const onToggleCollapse = vi.fn();
      const tree = CuratorialPlaque({
        artwork: mockArtwork,
        isCollapsed: false,
        onToggleCollapse,
      });

      const buttons = findElements(tree, (el) => el.type === 'button');
      const toggleBtn = buttons.find((btn) => btn.props['aria-label'] === 'Collapse curatorial plaque');
      expect(toggleBtn).toBeDefined();
      toggleBtn?.props.onClick?.({ preventDefault: () => {} });
      expect(onToggleCollapse).toHaveBeenCalledTimes(1);
    });

    it('should trigger onToggleCollapse when clicking toggle button in collapsed state', () => {
      const onToggleCollapse = vi.fn();
      const tree = CuratorialPlaque({
        artwork: mockArtwork,
        isCollapsed: true,
        onToggleCollapse,
      });

      const buttons = findElements(tree, (el) => el.type === 'button');
      const toggleBtn = buttons.find((btn) => btn.props['aria-label'] === 'Expand curatorial plaque');
      expect(toggleBtn).toBeDefined();
      toggleBtn?.props.onClick?.({ preventDefault: () => {} });
      expect(onToggleCollapse).toHaveBeenCalledTimes(1);
    });
  });
});
