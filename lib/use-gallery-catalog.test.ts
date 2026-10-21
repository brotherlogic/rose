import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import React, { act } from 'react';
import { createRoot } from 'react-dom/client';
import { renderToStaticMarkup } from 'react-dom/server';
import type { gallery } from './proto/gallery';
import {
  CATALOG_ENDPOINT,
  normalizeMediaPath,
  normalizeCatalogArtwork,
  useGalleryCatalog,
} from '../app/hooks/useGalleryCatalog';

(globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

function renderHook<T>(hook: () => T) {
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
  const result = { current: null as unknown as T };

  function TestHarness() {
    result.current = hook();
    return null;
  }

  act(() => {
    root.render(React.createElement(TestHarness));
  });

  return {
    result,
    rerender: () => {
      act(() => {
        root.render(React.createElement(TestHarness));
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
    },
  };
}

describe('useGalleryCatalog Hook & Normalization Suite (#344)', () => {
  const originalWindow = globalThis.window;
  const originalDocument = globalThis.document;
  const originalFetch = globalThis.fetch;

  const baselineArtworks: gallery.IArtwork[] = [
    {
      id: 'static-001',
      title: 'Static Baseline Work',
      medium: 'Crayon on paper',
      description: 'Built into static export.',
      themeId: 'The Crayon Period',
      timestamp: 1700000000,
      imagePath: '/data/images/static-001.jpg',
      thumbnailPath: '/data/thumbnails/static-001.webp',
    },
  ];

  beforeEach(() => {
    const mockWin = {
      location: {
        pathname: '/',
        search: '',
      },
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

  describe('Constants & Exports', () => {
    it('exports CATALOG_ENDPOINT as /data/catalog.json', () => {
      expect(CATALOG_ENDPOINT).toBe('/data/catalog.json');
    });
  });

  describe('normalizeMediaPath', () => {
    it('prefixes relative images/ and thumbnails/ paths with /data/', () => {
      expect(normalizeMediaPath('images/photo-123.jpg')).toBe('/data/images/photo-123.jpg');
      expect(normalizeMediaPath('thumbnails/photo-123.webp')).toBe('/data/thumbnails/photo-123.webp');
      expect(normalizeMediaPath('  images/trimmed.jpg  ')).toBe('/data/images/trimmed.jpg');
    });

    it('normalizes bare <id>.jpg image filenames to /data/images/<id>.jpg and bare <id>.webp to /data/thumbnails/<id>.webp', () => {
      expect(normalizeMediaPath('AF1QipMMgSANgQO9t3rSDH7ALFTBkizu6iz-LHCW_R79.jpg')).toBe(
        '/data/images/AF1QipMMgSANgQO9t3rSDH7ALFTBkizu6iz-LHCW_R79.jpg'
      );
      expect(normalizeMediaPath('  AF1QipPDF_AkW2XnwtEMyQ7_AUi1rC_MIeKgijyMPWdp.jpg  ')).toBe(
        '/data/images/AF1QipPDF_AkW2XnwtEMyQ7_AUi1rC_MIeKgijyMPWdp.jpg'
      );
      expect(normalizeMediaPath('legacy-thumb.webp')).toBe('/data/thumbnails/legacy-thumb.webp');
    });

    it('preserves paths already starting with / or http(s):// unchanged', () => {
      expect(normalizeMediaPath('/data/images/photo-123.jpg')).toBe('/data/images/photo-123.jpg');
      expect(normalizeMediaPath('/data/thumbnails/photo-123.webp')).toBe('/data/thumbnails/photo-123.webp');
      expect(normalizeMediaPath('/placeholder.svg')).toBe('/placeholder.svg');
      expect(normalizeMediaPath('https://cdn.example.com/art.jpg')).toBe('https://cdn.example.com/art.jpg');
      expect(normalizeMediaPath('http://example.com/art.jpg')).toBe('http://example.com/art.jpg');
    });

    it('returns empty string for non-string or blank values', () => {
      expect(normalizeMediaPath('')).toBe('');
      expect(normalizeMediaPath('   ')).toBe('');
      expect(normalizeMediaPath(null)).toBe('');
      expect(normalizeMediaPath(undefined)).toBe('');
      expect(normalizeMediaPath(42)).toBe('');
    });
  });

  describe('normalizeCatalogArtwork', () => {
    it('normalizes snake_case catalog JSON fields and relative /data/ paths', () => {
      const normalized = normalizeCatalogArtwork({
        id: 'art-snake-1',
        title: 'Crimson Scribble',
        description: 'A study in wax.',
        theme_id: 'Kinetic Scribblism',
        timestamp: 1712707200,
        image_path: 'images/art-snake-1.jpg',
        thumbnail_path: 'thumbnails/art-snake-1.webp',
        medium: 'Wax crayon on wall',
      });

      expect(normalized).toEqual({
        id: 'art-snake-1',
        title: 'Crimson Scribble',
        description: 'A study in wax.',
        themeId: 'Kinetic Scribblism',
        timestamp: 1712707200,
        imagePath: '/data/images/art-snake-1.jpg',
        thumbnailPath: '/data/thumbnails/art-snake-1.webp',
        medium: 'Wax crayon on wall',
      });
    });

    it('normalizes camelCase fields including theme fallback', () => {
      const normalized = normalizeCatalogArtwork({
        id: 'art-camel-1',
        title: 'Cardboard Tower',
        description: 'Structural collapse.',
        theme: 'Domestic Destructionism',
        timestamp: '1712800000',
        imagePath: '/data/images/art-camel-1.jpg',
        thumbnailPath: '/data/thumbnails/art-camel-1.webp',
        medium: 'Cardboard and tape',
      });

      expect(normalized).toEqual({
        id: 'art-camel-1',
        title: 'Cardboard Tower',
        description: 'Structural collapse.',
        themeId: 'Domestic Destructionism',
        timestamp: 1712800000,
        imagePath: '/data/images/art-camel-1.jpg',
        thumbnailPath: '/data/thumbnails/art-camel-1.webp',
        medium: 'Cardboard and tape',
      });
    });

    it('applies curatorial defaults consistent with lib/nfs.ts when optional fields are missing or blank', () => {
      const withImageOnly = normalizeCatalogArtwork({
        id: '  art-defaults-1  ',
        title: '   ',
        medium: '',
        description: null,
        theme_id: '  ',
        timestamp: 'not-a-number',
        image_path: 'images/art-defaults-1.jpg',
        thumbnail_path: '',
      });

      expect(withImageOnly).toEqual({
        id: 'art-defaults-1',
        title: 'Untitled',
        description: '',
        themeId: '',
        timestamp: 0,
        imagePath: '/data/images/art-defaults-1.jpg',
        thumbnailPath: '/data/images/art-defaults-1.jpg',
        medium: 'Mixed Media',
      });

      const withNoPaths = normalizeCatalogArtwork({
        id: 'art-defaults-2',
      });

      expect(withNoPaths).toEqual({
        id: 'art-defaults-2',
        title: 'Untitled',
        description: '',
        themeId: '',
        timestamp: 0,
        imagePath: '',
        thumbnailPath: '/placeholder.svg',
        medium: 'Mixed Media',
      });
    });

    it('rejects non-object items or items missing a non-empty string id', () => {
      expect(normalizeCatalogArtwork(null)).toBeNull();
      expect(normalizeCatalogArtwork(undefined)).toBeNull();
      expect(normalizeCatalogArtwork('invalid')).toBeNull();
      expect(normalizeCatalogArtwork(123)).toBeNull();
      expect(normalizeCatalogArtwork([])).toBeNull();
      expect(normalizeCatalogArtwork({})).toBeNull();
      expect(normalizeCatalogArtwork({ id: '' })).toBeNull();
      expect(normalizeCatalogArtwork({ id: '   ' })).toBeNull();
      expect(normalizeCatalogArtwork({ id: 123 })).toBeNull();
    });

    it('preserves is_representative: true and isRepresentative: true as isRepresentative: true and omits the property when false or absent', () => {
      const fromSnakeTrue = normalizeCatalogArtwork({
        id: 'rep-snake',
        is_representative: true,
      });
      expect(fromSnakeTrue).toMatchObject({
        id: 'rep-snake',
        isRepresentative: true,
      });

      const fromCamelTrue = normalizeCatalogArtwork({
        id: 'rep-camel',
        isRepresentative: true,
      });
      expect(fromCamelTrue).toMatchObject({
        id: 'rep-camel',
        isRepresentative: true,
      });

      const fromFalse = normalizeCatalogArtwork({
        id: 'rep-false',
        is_representative: false,
        isRepresentative: false,
      });
      expect(fromFalse).not.toBeNull();
      expect('isRepresentative' in (fromFalse as Record<string, unknown>)).toBe(false);

      const fromAbsent = normalizeCatalogArtwork({
        id: 'rep-absent',
      });
      expect(fromAbsent).not.toBeNull();
      expect('isRepresentative' in (fromAbsent as Record<string, unknown>)).toBe(false);
    });
  });

  describe('useGalleryCatalog Hydration Lifecycle', () => {
    it('initializes synchronously with initialArtworks during SSR renderToStaticMarkup', () => {
      function SsrComponent() {
        const { artworks, isHydrated } = useGalleryCatalog(baselineArtworks);
        return React.createElement(
          'div',
          { 'data-hydrated': String(isHydrated) },
          artworks.map((a) => a.title).join(', ')
        );
      }

      const html = renderToStaticMarkup(React.createElement(SsrComponent));
      expect(html).toContain('data-hydrated="false"');
      expect(html).toContain('Static Baseline Work');
    });

    it('fetches /data/catalog.json with cache: no-store on mount and completely replaces initialArtworks when valid non-empty catalog is returned', async () => {
      const remotePayload = [
        {
          id: 'dynamic-001',
          title: 'Dynamic Masterpiece I',
          description: 'Freshly synced from Google Photos.',
          theme_id: 'The Crayon Period',
          timestamp: 1720000000,
          image_path: 'images/dynamic-001.jpg',
          thumbnail_path: 'thumbnails/dynamic-001.webp',
          medium: 'Wax on paper',
        },
        {
          id: 'dynamic-002',
          title: 'Dynamic Masterpiece II',
          description: '',
          theme_id: 'Monochrome Nihilism',
          timestamp: 1715000000,
          image_path: '/data/images/dynamic-002.jpg',
          thumbnail_path: '/data/thumbnails/dynamic-002.webp',
          medium: 'Charcoal',
        },
      ];

      const fetchMock = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => remotePayload,
      });
      globalThis.fetch = fetchMock as unknown as typeof fetch;

      const { result, unmount } = renderHook(() => useGalleryCatalog(baselineArtworks));

      await act(async () => {
        await Promise.resolve();
      });

      expect(fetchMock).toHaveBeenCalledTimes(1);
      expect(fetchMock).toHaveBeenCalledWith(
        '/data/catalog.json',
        expect.objectContaining({
          cache: 'no-store',
          signal: expect.any(AbortSignal),
        })
      );

      expect(result.current.isHydrated).toBe(true);
      expect(result.current.artworks).toHaveLength(2);
      expect(result.current.artworks[0]).toEqual({
        id: 'dynamic-001',
        title: 'Dynamic Masterpiece I',
        description: 'Freshly synced from Google Photos.',
        themeId: 'The Crayon Period',
        timestamp: 1720000000,
        imagePath: '/data/images/dynamic-001.jpg',
        thumbnailPath: '/data/thumbnails/dynamic-001.webp',
        medium: 'Wax on paper',
      });
      expect(result.current.artworks.find((a) => a.id === 'static-001')).toBeUndefined();

      unmount();
    });

    it('retains initialArtworks on HTTP 404 or HTTP 500 responses', async () => {
      for (const status of [404, 500]) {
        const fetchMock = vi.fn().mockResolvedValue({
          ok: false,
          status,
          json: async () => ({ error: 'Failed' }),
        });
        globalThis.fetch = fetchMock as unknown as typeof fetch;

        const { result, unmount } = renderHook(() => useGalleryCatalog(baselineArtworks));

        await act(async () => {
          await Promise.resolve();
        });

        expect(result.current.isHydrated).toBe(false);
        expect(result.current.artworks).toEqual(baselineArtworks);
        unmount();
      }
    });

    it('retains initialArtworks on network failure, malformed JSON, or empty array []', async () => {
      // 1. Network failure
      globalThis.fetch = vi.fn().mockRejectedValue(new Error('Network offline')) as unknown as typeof fetch;
      const hookNetErr = renderHook(() => useGalleryCatalog(baselineArtworks));
      await act(async () => {
        await Promise.resolve();
      });
      expect(hookNetErr.result.current.isHydrated).toBe(false);
      expect(hookNetErr.result.current.artworks).toEqual(baselineArtworks);
      hookNetErr.unmount();

      // 2. Malformed JSON
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => {
          throw new SyntaxError('Unexpected token < in JSON');
        },
      }) as unknown as typeof fetch;
      const hookBadJson = renderHook(() => useGalleryCatalog(baselineArtworks));
      await act(async () => {
        await Promise.resolve();
      });
      expect(hookBadJson.result.current.isHydrated).toBe(false);
      expect(hookBadJson.result.current.artworks).toEqual(baselineArtworks);
      hookBadJson.unmount();

      // 3. Empty array []
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => [],
      }) as unknown as typeof fetch;
      const hookEmpty = renderHook(() => useGalleryCatalog(baselineArtworks));
      await act(async () => {
        await Promise.resolve();
      });
      expect(hookEmpty.result.current.isHydrated).toBe(false);
      expect(hookEmpty.result.current.artworks).toEqual(baselineArtworks);
      hookEmpty.unmount();
    });

    it('synchronizes state with initialArtworks prop updates prior to remote hydration', async () => {
      // Keep fetch pending so hook remains unhydrated
      globalThis.fetch = vi.fn().mockReturnValue(new Promise(() => {})) as unknown as typeof fetch;

      let currentProp: gallery.IArtwork[] = [];
      const { result, rerender, unmount } = renderHook(() => useGalleryCatalog(currentProp));

      expect(result.current.artworks).toEqual([]);
      expect(result.current.isHydrated).toBe(false);

      currentProp = baselineArtworks;
      rerender();

      expect(result.current.artworks).toEqual(baselineArtworks);
      expect(result.current.isHydrated).toBe(false);

      unmount();
    });

    it('deduplicates concurrent fetch(/data/catalog.json) calls when multiple hooks mount together', async () => {
      const remotePayload = [
        {
          id: 'shared-001',
          title: 'Shared Representative Work',
          description: 'Fetched once across concurrent hooks.',
          theme_id: 'The Crayon Period',
          timestamp: 1730000000,
          image_path: 'images/shared-001.jpg',
          thumbnail_path: 'thumbnails/shared-001.webp',
          medium: 'Wax on panel',
          is_representative: true,
        },
      ];

      let resolveFetch!: (value: unknown) => void;
      const fetchMock = vi.fn().mockImplementation(
        () =>
          new Promise((resolve) => {
            resolveFetch = resolve;
          })
      );
      globalThis.fetch = fetchMock as unknown as typeof fetch;

      const hookA = renderHook(() => useGalleryCatalog(baselineArtworks));
      const hookB = renderHook(() => useGalleryCatalog(baselineArtworks));

      expect(fetchMock).toHaveBeenCalledTimes(1);

      await act(async () => {
        resolveFetch({
          ok: true,
          status: 200,
          json: async () => remotePayload,
        });
        await Promise.resolve();
      });

      expect(hookA.result.current.isHydrated).toBe(true);
      expect(hookB.result.current.isHydrated).toBe(true);
      expect(hookA.result.current.artworks).toEqual(hookB.result.current.artworks);
      expect(hookA.result.current.artworks[0]).toMatchObject({
        id: 'shared-001',
        isRepresentative: true,
      });

      hookA.unmount();
      hookB.unmount();
    });
  });
});
