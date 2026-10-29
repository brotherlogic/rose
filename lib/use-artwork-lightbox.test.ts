import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import React, { act } from 'react';
import { createRoot } from 'react-dom/client';
import { useArtworkLightbox } from '../app/hooks/useArtworkLightbox';
import type { gallery } from '../lib/proto/gallery';

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

describe('useArtworkLightbox Hook Suite (#296)', () => {
  const originalWindow = globalThis.window;
  const originalDocument = globalThis.document;

  const mockArtworks: gallery.IArtwork[] = [
    {
      id: 'art-001',
      title: 'Study in Crimson and Charcoal',
      medium: 'Oil and pastel on linen',
      description: 'An exploration of domesticity and destructive marks.',
      thumbnailPath: '/data/thumbnails/art-001.webp',
      imagePath: '/data/images/art-001.jpg',
      themeId: 'kinetic-scribblism',
      timestamp: 1712707200,
    },
    {
      id: 'art-002',
      title: 'Torn Cardboard Sculpture',
      medium: 'Reclaimed cardboard and industrial adhesive',
      description: 'The fragility of structured forms.',
      thumbnailPath: '/data/thumbnails/art-002.webp',
      imagePath: '/data/images/art-002.jpg',
      themeId: 'domestic-destructionism',
      timestamp: 1712800000,
    },
    {
      id: 'art-003',
      title: 'Monochrome Void VII',
      medium: 'Wax pigment on newsprint',
      description: 'Solemn quietude in repetition.',
      thumbnailPath: '/data/thumbnails/art-003.webp',
      imagePath: '/data/images/art-003.jpg',
      themeId: 'the-crayon-period',
      timestamp: 1712900000,
    },
  ];

  beforeEach(() => {
    const listeners: Record<string, ((event: Event) => void)[]> = {};

    const mockWin = {
      location: {
        pathname: '/',
        search: '?movement=the-crayon-period',
      },
      history: {
        pushState: vi.fn((_data: unknown, _unused: string, url?: string | URL | null) => {
          if (typeof url === 'string') {
            const parsed = new URL(url, 'http://localhost');
            mockWin.location.pathname = parsed.pathname;
            mockWin.location.search = parsed.search;
          }
        }),
        replaceState: vi.fn((_data: unknown, _unused: string, url?: string | URL | null) => {
          if (typeof url === 'string') {
            const parsed = new URL(url, 'http://localhost');
            mockWin.location.pathname = parsed.pathname;
            mockWin.location.search = parsed.search;
          }
        }),
      },
      addEventListener: vi.fn((event: string, fn: (event: Event) => void) => {
        listeners[event] = listeners[event] || [];
        listeners[event].push(fn);
      }),
      removeEventListener: vi.fn((event: string, fn: (event: Event) => void) => {
        if (listeners[event]) {
          listeners[event] = listeners[event].filter((f) => f !== fn);
        }
      }),
      dispatchEvent: vi.fn((event: Event) => {
        if (listeners[event.type]) {
          listeners[event.type].forEach((fn) => fn(event));
          return true;
        }
        return false;
      }),
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
    vi.restoreAllMocks();
    (globalThis as unknown as { window: unknown }).window = originalWindow;
    (globalThis as unknown as { document: unknown }).document = originalDocument;
  });

  describe('Initial State', () => {
    it('should initialize with activeIndex null, isPlaqueCollapsed false, and triggerElementRef null', () => {
      const { result, unmount } = renderHook(() => useArtworkLightbox(mockArtworks));

      expect(result.current.activeIndex).toBeNull();
      expect(result.current.isPlaqueCollapsed).toBe(false);
      expect(result.current.triggerElementRef.current).toBeNull();

      unmount();
    });
  });

  describe('Opening and Closing Lightbox', () => {
    it('should update activeIndex and track origin button element upon openLightbox', () => {
      const { result, unmount } = renderHook(() => useArtworkLightbox(mockArtworks));
      const mockButton = { focus: vi.fn() } as unknown as HTMLElement;

      act(() => {
        result.current.openLightbox(1, mockButton);
      });

      expect(result.current.activeIndex).toBe(1);
      expect(result.current.triggerElementRef.current).toBe(mockButton);

      unmount();
    });

    it('should reset activeIndex to null and restore focus to triggerElementRef upon closeLightbox', () => {
      const { result, unmount } = renderHook(() => useArtworkLightbox(mockArtworks));
      const focusSpy = vi.fn();
      const mockButton = { focus: focusSpy } as unknown as HTMLElement;

      act(() => {
        result.current.openLightbox(2, mockButton);
      });

      expect(result.current.activeIndex).toBe(2);

      act(() => {
        result.current.closeLightbox();
      });

      expect(result.current.activeIndex).toBeNull();
      expect(focusSpy).toHaveBeenCalledTimes(1);

      unmount();
    });

    it('should safely handle closeLightbox when no trigger element was provided', () => {
      const { result, unmount } = renderHook(() => useArtworkLightbox(mockArtworks));

      act(() => {
        result.current.openLightbox(0);
      });

      expect(result.current.activeIndex).toBe(0);

      expect(() => {
        act(() => {
          result.current.closeLightbox();
        });
      }).not.toThrow();

      expect(result.current.activeIndex).toBeNull();

      unmount();
    });
  });

  describe('Circular Navigation', () => {
    it('should circularly advance forward with nextArtwork (wrapping from last item to 0)', () => {
      const { result, unmount } = renderHook(() => useArtworkLightbox(mockArtworks));

      act(() => {
        result.current.openLightbox(0);
      });
      expect(result.current.activeIndex).toBe(0);

      act(() => {
        result.current.nextArtwork();
      });
      expect(result.current.activeIndex).toBe(1);

      act(() => {
        result.current.nextArtwork();
      });
      expect(result.current.activeIndex).toBe(2);

      // Wraps back to 0
      act(() => {
        result.current.nextArtwork();
      });
      expect(result.current.activeIndex).toBe(0);

      unmount();
    });

    it('should circularly navigate backward with prevArtwork (wrapping from 0 to last item)', () => {
      const { result, unmount } = renderHook(() => useArtworkLightbox(mockArtworks));

      act(() => {
        result.current.openLightbox(0);
      });
      expect(result.current.activeIndex).toBe(0);

      // Wraps back to last item (index 2)
      act(() => {
        result.current.prevArtwork();
      });
      expect(result.current.activeIndex).toBe(2);

      act(() => {
        result.current.prevArtwork();
      });
      expect(result.current.activeIndex).toBe(1);

      act(() => {
        result.current.prevArtwork();
      });
      expect(result.current.activeIndex).toBe(0);

      unmount();
    });
  });

  describe('Safe Navigation with Edge-Case Collections', () => {
    it('should safely handle single-item subsets without index out-of-bounds errors', () => {
      const singleItem: gallery.IArtwork[] = [mockArtworks[0]];
      const { result, unmount } = renderHook(() => useArtworkLightbox(singleItem));

      act(() => {
        result.current.openLightbox(0);
      });
      expect(result.current.activeIndex).toBe(0);

      act(() => {
        result.current.nextArtwork();
      });
      expect(result.current.activeIndex).toBe(0);

      act(() => {
        result.current.prevArtwork();
      });
      expect(result.current.activeIndex).toBe(0);

      unmount();
    });

    it('should safely handle empty collections without runtime exceptions', () => {
      const emptyArtworks: gallery.IArtwork[] = [];
      const { result, unmount } = renderHook(() => useArtworkLightbox(emptyArtworks));

      expect(result.current.activeIndex).toBeNull();

      expect(() => {
        act(() => {
          result.current.openLightbox(0);
        });
      }).not.toThrow();
      expect(result.current.activeIndex).toBeNull();

      expect(() => {
        act(() => {
          result.current.nextArtwork();
        });
      }).not.toThrow();

      expect(() => {
        act(() => {
          result.current.prevArtwork();
        });
      }).not.toThrow();

      unmount();
    });

    it('should not navigate next/prev when activeIndex is null', () => {
      const { result, unmount } = renderHook(() => useArtworkLightbox(mockArtworks));

      expect(result.current.activeIndex).toBeNull();

      act(() => {
        result.current.nextArtwork();
      });
      expect(result.current.activeIndex).toBeNull();

      act(() => {
        result.current.prevArtwork();
      });
      expect(result.current.activeIndex).toBeNull();

      unmount();
    });
  });

  describe('Curatorial Plaque Collapse Persistence', () => {
    it('should toggle plaque collapsed state and persist across transitions', () => {
      const { result, unmount } = renderHook(() => useArtworkLightbox(mockArtworks));

      act(() => {
        result.current.openLightbox(0);
      });
      expect(result.current.isPlaqueCollapsed).toBe(false);

      act(() => {
        result.current.togglePlaque();
      });
      expect(result.current.isPlaqueCollapsed).toBe(true);

      // Persists through next artwork transition
      act(() => {
        result.current.nextArtwork();
      });
      expect(result.current.activeIndex).toBe(1);
      expect(result.current.isPlaqueCollapsed).toBe(true);

      // Persists through prev artwork transition
      act(() => {
        result.current.prevArtwork();
      });
      expect(result.current.activeIndex).toBe(0);
      expect(result.current.isPlaqueCollapsed).toBe(true);

      // Persists through direct openLightbox selection
      act(() => {
        result.current.openLightbox(2);
      });
      expect(result.current.activeIndex).toBe(2);
      expect(result.current.isPlaqueCollapsed).toBe(true);

      // Can be toggled back to expanded
      act(() => {
        result.current.togglePlaque();
      });
      expect(result.current.isPlaqueCollapsed).toBe(false);

      unmount();
    });
  });

  describe('URL Synchronization, Browser History & Deep-Linking', () => {
    it('should call window.history.pushState with ?artwork=<id> on openLightbox while preserving movement and sort=asc', () => {
      window.location.search = '?movement=the-crayon-period&sort=asc';
      const { result, unmount } = renderHook(() => useArtworkLightbox(mockArtworks));

      act(() => {
        result.current.openLightbox(1);
      });

      expect(result.current.activeIndex).toBe(1);
      expect(window.history.pushState).toHaveBeenCalledWith(
        null,
        '',
        '/?movement=the-crayon-period&sort=asc&artwork=art-002'
      );
      expect(window.location.search).toBe('?movement=the-crayon-period&sort=asc&artwork=art-002');

      unmount();
    });

    it('should call window.history.replaceState on nextArtwork and prevArtwork without calling pushState, and skip duplicate calls on single-item collections', () => {
      const { result, unmount } = renderHook(() => useArtworkLightbox(mockArtworks));

      act(() => {
        result.current.openLightbox(0);
      });
      expect(window.history.pushState).toHaveBeenCalledTimes(1);
      expect(window.history.replaceState).not.toHaveBeenCalled();

      act(() => {
        result.current.nextArtwork();
      });
      expect(result.current.activeIndex).toBe(1);
      expect(window.history.pushState).toHaveBeenCalledTimes(1);
      expect(window.history.replaceState).toHaveBeenCalledWith(
        null,
        '',
        '/?movement=the-crayon-period&artwork=art-002'
      );

      act(() => {
        result.current.prevArtwork();
      });
      expect(result.current.activeIndex).toBe(0);
      expect(window.history.pushState).toHaveBeenCalledTimes(1);
      expect(window.history.replaceState).toHaveBeenCalledWith(
        null,
        '',
        '/?movement=the-crayon-period&artwork=art-001'
      );

      unmount();

      // Single-item collection should not call replaceState or pushState on next/prev
      vi.clearAllMocks();
      window.location.search = '';
      const singleCollection = [mockArtworks[0]];
      const singleHook = renderHook(() => useArtworkLightbox(singleCollection));

      act(() => {
        singleHook.result.current.openLightbox(0);
      });
      expect(window.history.pushState).toHaveBeenCalledTimes(1);
      expect(window.history.replaceState).not.toHaveBeenCalled();

      act(() => {
        singleHook.result.current.nextArtwork();
        singleHook.result.current.prevArtwork();
      });
      expect(singleHook.result.current.activeIndex).toBe(0);
      expect(window.history.pushState).toHaveBeenCalledTimes(1);
      expect(window.history.replaceState).not.toHaveBeenCalled();

      singleHook.unmount();
    });

    it('should remove artwork from URL via window.history.pushState on closeLightbox while preserving movement and sort', () => {
      window.location.search = '?movement=the-crayon-period&sort=asc';
      const { result, unmount } = renderHook(() => useArtworkLightbox(mockArtworks));

      act(() => {
        result.current.openLightbox(2);
      });
      expect(window.location.search).toBe('?movement=the-crayon-period&sort=asc&artwork=art-003');

      act(() => {
        result.current.closeLightbox();
      });
      expect(result.current.activeIndex).toBeNull();
      expect(window.history.pushState).toHaveBeenLastCalledWith(
        null,
        '',
        '/?movement=the-crayon-period&sort=asc'
      );
      expect(window.location.search).toBe('?movement=the-crayon-period&sort=asc');

      unmount();
    });

    it('should never mutate window.location.search or call pushState/replaceState on togglePlaque', () => {
      const { result, unmount } = renderHook(() => useArtworkLightbox(mockArtworks));

      act(() => {
        result.current.openLightbox(0);
      });
      const searchAfterOpen = window.location.search;
      vi.clearAllMocks();

      act(() => {
        result.current.togglePlaque();
      });
      expect(result.current.isPlaqueCollapsed).toBe(true);
      expect(window.location.search).toBe(searchAfterOpen);
      expect(window.history.pushState).not.toHaveBeenCalled();
      expect(window.history.replaceState).not.toHaveBeenCalled();

      act(() => {
        result.current.togglePlaque();
      });
      expect(result.current.isPlaqueCollapsed).toBe(false);
      expect(window.location.search).toBe(searchAfterOpen);
      expect(window.history.pushState).not.toHaveBeenCalled();
      expect(window.history.replaceState).not.toHaveBeenCalled();

      unmount();
    });

    it('should synchronize with browser popstate events to close and re-open lightbox', () => {
      const focusSpy = vi.fn();
      const mockTrigger = { focus: focusSpy } as unknown as HTMLElement;
      const { result, unmount } = renderHook(() => useArtworkLightbox(mockArtworks));

      act(() => {
        result.current.openLightbox(0, mockTrigger);
      });
      expect(result.current.activeIndex).toBe(0);

      // Simulate Browser Back to grid URL without ?artwork=
      window.location.search = '?movement=the-crayon-period';
      act(() => {
        window.dispatchEvent(new Event('popstate'));
      });
      expect(result.current.activeIndex).toBeNull();
      expect(focusSpy).toHaveBeenCalledTimes(1);

      // Simulate Browser Forward to URL with ?artwork=art-003
      window.location.search = '?movement=the-crayon-period&artwork=art-003';
      act(() => {
        window.dispatchEvent(new Event('popstate'));
      });
      expect(result.current.activeIndex).toBe(2);

      unmount();
    });

    it('should open matching artwork with isPlaqueCollapsed false on deep-link mount', () => {
      window.location.search = '?movement=domestic-destructionism&artwork=art-002';
      const { result, unmount } = renderHook(() => useArtworkLightbox(mockArtworks));

      expect(result.current.activeIndex).toBe(1);
      expect(result.current.isPlaqueCollapsed).toBe(false);

      unmount();
    });

    it('should invoke onClearMovementFilter and strip movement via replaceState when deep-linked artwork is in catalogArtworks but excluded from artworks', () => {
      window.location.search = '?movement=the-crayon-period&sort=asc&artwork=art-002';
      const filteredArtworks = [mockArtworks[0], mockArtworks[2]]; // excludes art-002
      const onClearMovementFilter = vi.fn();

      const { result, unmount } = renderHook(() =>
        useArtworkLightbox(filteredArtworks, {
          catalogArtworks: mockArtworks,
          isCatalogReady: true,
          onClearMovementFilter,
        })
      );

      expect(result.current.activeIndex).toBeNull();
      expect(onClearMovementFilter).toHaveBeenCalledTimes(1);
      expect(window.history.replaceState).toHaveBeenCalledWith(
        null,
        '',
        '/?sort=asc&artwork=art-002'
      );

      unmount();
    });

    it('should wait while isCatalogReady is false, then strip invalid or empty ?artwork= parameter via replaceState once isCatalogReady is true', () => {
      window.location.search = '?movement=the-crayon-period&artwork=nonexistent';
      let isReady = false;
      let currentArtworks: gallery.IArtwork[] = [];

      const { result, rerender, unmount } = renderHook(() =>
        useArtworkLightbox(currentArtworks, {
          catalogArtworks: currentArtworks,
          isCatalogReady: isReady,
        })
      );

      // While pending async hydration, do not strip ?artwork=
      expect(result.current.activeIndex).toBeNull();
      expect(window.history.replaceState).not.toHaveBeenCalled();
      expect(window.location.search).toBe('?movement=the-crayon-period&artwork=nonexistent');

      // Once catalog hydration settles, strip unknown ?artwork=nonexistent
      isReady = true;
      currentArtworks = mockArtworks;
      rerender();

      expect(result.current.activeIndex).toBeNull();
      expect(window.history.replaceState).toHaveBeenCalledWith(
        null,
        '',
        '/?movement=the-crayon-period'
      );
      expect(window.location.search).toBe('?movement=the-crayon-period');

      unmount();

      // Also verify empty ?artwork= is stripped when ready
      vi.clearAllMocks();
      window.location.search = '?artwork=';
      const emptyParamHook = renderHook(() =>
        useArtworkLightbox(mockArtworks, {
          catalogArtworks: mockArtworks,
          isCatalogReady: true,
        })
      );
      expect(emptyParamHook.result.current.activeIndex).toBeNull();
      expect(window.history.replaceState).toHaveBeenCalledWith(null, '', '/');
      expect(window.location.search).toBe('');
      emptyParamHook.unmount();
    });
  });
});
