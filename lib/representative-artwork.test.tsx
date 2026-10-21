import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import React, { act } from 'react';
import { createRoot } from 'react-dom/client';
import { renderToStaticMarkup } from 'react-dom/server';
import RepresentativeArtwork, {
  RepresentativeArtworkProps,
} from '../app/components/RepresentativeArtwork';
import type { gallery } from './proto/gallery';

(globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

function findElements(
  node: React.ReactNode,
  predicate: (el: React.ReactElement) => boolean
): React.ReactElement[] {
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

describe('RepresentativeArtwork Component Suite (#402)', () => {
  const originalWindow = globalThis.window;
  const originalDocument = globalThis.document;
  const originalFetch = globalThis.fetch;

  const mockArtworks: gallery.IArtwork[] = [
    {
      id: 'art-001',
      title: 'Minor Sketch',
      medium: 'Mixed Media',
      description: '',
      themeId: 'Early Works',
      imagePath: '/data/images/art-001.jpg',
      thumbnailPath: '/data/thumbnails/art-001.webp',
      timestamp: 1700000000,
    },
    {
      id: 'art-002',
      title: 'Featured Crayon Manifesto',
      medium: 'Wax crayon on salvaged wallboard',
      description: 'Definitive treatise of the early crayon period.',
      themeId: 'The Crayon Period',
      imagePath: '/data/images/art-002.jpg',
      thumbnailPath: '/data/thumbnails/art-002.webp',
      timestamp: 1712707200,
      ...( { isRepresentative: true } as Record<string, unknown> ),
    },
  ];

  beforeEach(() => {
    const mockWin = {
      location: { pathname: '/', search: '' },
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      HTMLIFrameElement: function () {},
    };

    const mockDoc = {
      nodeType: 9,
      createElement: () => ({
        nodeType: 1,
        style: {},
        setAttribute: () => {},
        removeAttribute: () => {},
        appendChild: () => {},
        removeChild: () => {},
        addEventListener: () => {},
        removeEventListener: () => {},
      }),
      addEventListener: () => {},
      removeEventListener: () => {},
      defaultView: mockWin,
    };

    (globalThis as unknown as { window: unknown }).window = mockWin;
    (globalThis as unknown as { document: unknown }).document = mockDoc;
  });

  afterEach(() => {
    (globalThis as unknown as { window: unknown }).window = originalWindow;
    (globalThis as unknown as { document: unknown }).document = originalDocument;
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it('renders null when artworks is empty (including direct invocation outside a React render root)', () => {
    const markup = renderToStaticMarkup(
      React.createElement(RepresentativeArtwork, { artworks: [] })
    );
    expect(markup).toBe('');

    const directResult = RepresentativeArtwork({ artworks: [] });
    expect(directResult).toBeNull();
  });

  it('renders the designated representative artwork in #featured-artwork using high-resolution imagePath first and minimal CuratorialPlaque', () => {
    const markup = renderToStaticMarkup(
      React.createElement(RepresentativeArtwork, { artworks: mockArtworks })
    );

    expect(markup).toContain('id="featured-artwork"');
    expect(markup).toContain('aria-label="Featured Archive Work"');
    expect(markup).toContain('data-testid="representative-artwork-section"');
    expect(markup).toContain('<figure');
    expect(markup).toContain('src="/data/images/art-002.jpg"');
    expect(markup).toContain('alt="Featured Crayon Manifesto"');

    // Minimal CuratorialPlaque rendered
    expect(markup).toContain('data-testid="minimal-curatorial-plaque"');
    expect(markup).toContain('Featured Crayon Manifesto');
    expect(markup).toContain('(2024)');
    expect(markup).toContain('Wax crayon on salvaged wallboard');
    expect(markup).toContain('The Crayon Period');

    // Critique and interactive lightbox controls must NOT be present
    expect(markup).not.toContain('Definitive treatise of the early crayon period.');
    expect(markup).not.toContain('<button');
    expect(markup).not.toContain('aria-haspopup="dialog"');
    expect(markup).not.toContain('role="dialog"');
  });

  it('resolves initial image source with fallback hierarchy: imagePath -> thumbnailPath -> /placeholder.svg', () => {
    const thumbOnlyMarkup = renderToStaticMarkup(
      React.createElement(RepresentativeArtwork, {
        artworks: [
          {
            id: 'art-thumb-only',
            title: 'Thumbnail Only Work',
            imagePath: '   ',
            thumbnailPath: '/data/thumbnails/art-thumb-only.webp',
          },
        ],
      })
    );
    expect(thumbOnlyMarkup).toContain('src="/data/thumbnails/art-thumb-only.webp"');

    const placeholderOnlyMarkup = renderToStaticMarkup(
      React.createElement(RepresentativeArtwork, {
        artworks: [
          {
            id: 'art-no-assets',
            title: 'Assetless Work',
            imagePath: '',
            thumbnailPath: '  ',
          },
        ],
      })
    );
    expect(placeholderOnlyMarkup).toContain('src="/placeholder.svg"');

    const initialErrorMarkup = renderToStaticMarkup(
      React.createElement(RepresentativeArtwork, {
        artworks: mockArtworks,
        initialError: true,
      })
    );
    expect(initialErrorMarkup).toContain('src="/placeholder.svg"');
  });

  it('transitions from imagePath to thumbnailPath and finally to /placeholder.svg on <img onError>', () => {
    globalThis.fetch = vi.fn().mockReturnValue(new Promise(() => {})) as unknown as typeof fetch;

    const container = {
      nodeType: 1,
      tagName: 'DIV',
      ownerDocument: globalThis.document,
      appendChild: () => {},
      removeChild: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
    };

    const root = createRoot(container as unknown as HTMLElement);
    let capturedImgProps: Record<string, unknown> = {};

    function Inspector(props: RepresentativeArtworkProps) {
      const tree = RepresentativeArtwork(props);
      const imgs = findElements(tree, (el) => el.type === 'img');
      if (imgs.length > 0) {
        capturedImgProps = imgs[0].props as Record<string, unknown>;
      }
      return null;
    }

    act(() => {
      root.render(React.createElement(Inspector, { artworks: mockArtworks }));
    });

    // 1. Starts with high-resolution imagePath
    expect(capturedImgProps.src).toBe('/data/images/art-002.jpg');
    expect(typeof capturedImgProps.onError).toBe('function');

    // 2. First onError transitions to thumbnailPath
    act(() => {
      (capturedImgProps.onError as () => void)();
    });
    expect(capturedImgProps.src).toBe('/data/thumbnails/art-002.webp');

    // 3. Second onError transitions to /placeholder.svg
    act(() => {
      (capturedImgProps.onError as () => void)();
    });
    expect(capturedImgProps.src).toBe('/placeholder.svg');

    act(() => {
      root.unmount();
    });
  });

  it('contains no interactive button, onClick handler, or lightbox trigger on section or figure', () => {
    let captured: React.ReactElement | null = null;
    function Wrapper() {
      const el = RepresentativeArtwork({ artworks: mockArtworks });
      captured = el;
      return el;
    }
    renderToStaticMarkup(React.createElement(Wrapper));

    const buttons = findElements(captured, (el) => el.type === 'button');
    expect(buttons).toHaveLength(0);

    const clickableElements = findElements(
      captured,
      (el) => typeof (el.props as { onClick?: unknown })?.onClick === 'function'
    );
    expect(clickableElements).toHaveLength(0);
  });
});
