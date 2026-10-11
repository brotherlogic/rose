import { describe, it, expect, vi, afterEach } from 'vitest';
import React, { act } from 'react';
import { createRoot } from 'react-dom/client';
import { renderToStaticMarkup } from 'react-dom/server';
import { useGalleryFilter, type UseGalleryFilterReturn } from '../app/hooks/useGalleryFilter';

// Setup React 19 act environment
(globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

interface MockWindowOptions {
  pathname?: string;
  search?: string;
}

function createMockEnvironment(options: MockWindowOptions = {}) {
  const listeners: Record<string, ((event: Event) => void)[]> = {};

  const win = {
    location: {
      pathname: options.pathname ?? '/',
      search: options.search ?? '',
      get href() {
        return `http://localhost${this.pathname}${this.search}`;
      },
    },
    history: {
      pushState: vi.fn((_data: unknown, _unused: string, url?: string | URL | null) => {
        if (typeof url === 'string') {
          const parsed = new URL(url, 'http://localhost');
          win.location.pathname = parsed.pathname;
          win.location.search = parsed.search;
        }
      }),
      replaceState: vi.fn((_data: unknown, _unused: string, url?: string | URL | null) => {
        if (typeof url === 'string') {
          const parsed = new URL(url, 'http://localhost');
          win.location.pathname = parsed.pathname;
          win.location.search = parsed.search;
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

  const doc = {
    nodeType: 9,
    createElement: (tag: string) => ({
      nodeType: 1,
      tagName: tag.toUpperCase(),
      ownerDocument: doc,
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
    defaultView: win,
  };

  return { win, doc, listeners };
}

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
    unmount: () => {
      act(() => {
        root.unmount();
      });
    },
  };
}

describe('useGalleryFilter Hook Suite (#276)', () => {
  const originalWindow = globalThis.window;
  const originalDocument = globalThis.document;

  afterEach(() => {
    vi.restoreAllMocks();
    (globalThis as unknown as { window: unknown }).window = originalWindow;
    (globalThis as unknown as { document: unknown }).document = originalDocument;
  });

  describe('Initial State Parsing from URL Search Parameters', () => {
    it('should initialize with default values when URL has no search parameters', () => {
      const { win, doc } = createMockEnvironment({ search: '' });
      (globalThis as unknown as { window: unknown }).window = win;
      (globalThis as unknown as { document: unknown }).document = doc;

      const { result, unmount } = renderHook(() => useGalleryFilter());

      expect(result.current.selectedMovement).toBeNull();
      expect(result.current.sortDirection).toBe('desc');
      unmount();
    });

    it('should parse movement and sort direction from URL search parameters on mount', () => {
      const { win, doc } = createMockEnvironment({
        search: '?movement=The%20Crayon%20Period&sort=asc',
      });
      (globalThis as unknown as { window: unknown }).window = win;
      (globalThis as unknown as { document: unknown }).document = doc;

      const { result, unmount } = renderHook(() => useGalleryFilter());

      expect(result.current.selectedMovement).toBe('The Crayon Period');
      expect(result.current.sortDirection).toBe('asc');
      unmount();
    });

    it('should normalize "All Works" movement parameter to null', () => {
      const { win, doc } = createMockEnvironment({
        search: '?movement=All%20Works&sort=asc',
      });
      (globalThis as unknown as { window: unknown }).window = win;
      (globalThis as unknown as { document: unknown }).document = doc;

      const { result, unmount } = renderHook(() => useGalleryFilter());

      expect(result.current.selectedMovement).toBeNull();
      expect(result.current.sortDirection).toBe('asc');
      unmount();
    });

    it('should fallback to default "desc" sort when sort parameter is omitted or invalid', () => {
      const { win, doc } = createMockEnvironment({
        search: '?movement=Domestic%20Destructionism&sort=unknown',
      });
      (globalThis as unknown as { window: unknown }).window = win;
      (globalThis as unknown as { document: unknown }).document = doc;

      const { result, unmount } = renderHook(() => useGalleryFilter());

      expect(result.current.selectedMovement).toBe('Domestic Destructionism');
      expect(result.current.sortDirection).toBe('desc');
      unmount();
    });

    it('should execute safely on server/SSR when window is undefined without crashing', () => {
      (globalThis as unknown as { window: unknown }).window = undefined;
      (globalThis as unknown as { document: unknown }).document = undefined;

      let hookOutput: UseGalleryFilterReturn | null = null;
      function TestComponent() {
        hookOutput = useGalleryFilter();
        return null;
      }

      renderToStaticMarkup(React.createElement(TestComponent));

      expect(hookOutput).not.toBeNull();
      expect(hookOutput!.selectedMovement).toBeNull();
      expect(hookOutput!.sortDirection).toBe('desc');
      expect(typeof hookOutput!.setMovement).toBe('function');
      expect(typeof hookOutput!.setSortDirection).toBe('function');
      expect(typeof hookOutput!.resetFilter).toBe('function');

      // Actions should be safe during SSR without throwing
      expect(() => hookOutput!.setMovement('Test')).not.toThrow();
      expect(() => hookOutput!.setSortDirection('asc')).not.toThrow();
      expect(() => hookOutput!.resetFilter()).not.toThrow();
    });
  });

  describe('State Actions and URL Synchronization via window.history.pushState', () => {
    it('should update selected movement and push clean URL search params', () => {
      const { win, doc } = createMockEnvironment({ pathname: '/gallery', search: '' });
      (globalThis as unknown as { window: unknown }).window = win;
      (globalThis as unknown as { document: unknown }).document = doc;

      const { result, unmount } = renderHook(() => useGalleryFilter());

      act(() => {
        result.current.setMovement('Domestic Destructionism');
      });

      expect(result.current.selectedMovement).toBe('Domestic Destructionism');
      expect(win.history.pushState).toHaveBeenCalledWith(
        null,
        '',
        '/gallery?movement=Domestic+Destructionism'
      );
      unmount();
    });

    it('should toggle off selected movement when same movement is selected again', () => {
      const { win, doc } = createMockEnvironment({
        pathname: '/gallery',
        search: '?movement=Domestic%20Destructionism',
      });
      (globalThis as unknown as { window: unknown }).window = win;
      (globalThis as unknown as { document: unknown }).document = doc;

      const { result, unmount } = renderHook(() => useGalleryFilter());
      expect(result.current.selectedMovement).toBe('Domestic Destructionism');

      act(() => {
        result.current.setMovement('Domestic Destructionism');
      });

      expect(result.current.selectedMovement).toBeNull();
      // Since movement is toggled off and sort is default 'desc', canonical URL has no query params
      expect(win.history.pushState).toHaveBeenCalledWith(null, '', '/gallery');
      unmount();
    });

    it('should update chronological sort direction and reflect in URL', () => {
      const { win, doc } = createMockEnvironment({
        pathname: '/',
        search: '?movement=The%20Crayon%20Period',
      });
      (globalThis as unknown as { window: unknown }).window = win;
      (globalThis as unknown as { document: unknown }).document = doc;

      const { result, unmount } = renderHook(() => useGalleryFilter());

      act(() => {
        result.current.setSortDirection('asc');
      });

      expect(result.current.sortDirection).toBe('asc');
      expect(win.history.pushState).toHaveBeenCalledWith(
        null,
        '',
        '/?movement=The+Crayon+Period&sort=asc'
      );
      unmount();
    });

    it('should omit default parameters from the URL query string to maintain canonical URLs', () => {
      const { win, doc } = createMockEnvironment({
        pathname: '/',
        search: '?movement=Kinetic%20Scribblism&sort=asc',
      });
      (globalThis as unknown as { window: unknown }).window = win;
      (globalThis as unknown as { document: unknown }).document = doc;

      const { result, unmount } = renderHook(() => useGalleryFilter());

      // Changing sort back to 'desc' (default) should omit the sort parameter
      act(() => {
        result.current.setSortDirection('desc');
      });

      expect(result.current.sortDirection).toBe('desc');
      expect(win.history.pushState).toHaveBeenCalledWith(null, '', '/?movement=Kinetic+Scribblism');

      // Setting movement to 'All Works' should omit movement parameter completely
      act(() => {
        result.current.setMovement('All Works');
      });

      expect(result.current.selectedMovement).toBeNull();
      expect(win.history.pushState).toHaveBeenCalledWith(null, '', '/');
      unmount();
    });
  });

  describe('Browser Popstate Navigation Handling', () => {
    it('should update hook state when browser popstate events occur', () => {
      const { win, doc } = createMockEnvironment({
        pathname: '/',
        search: '?movement=Early%20Works&sort=asc',
      });
      (globalThis as unknown as { window: unknown }).window = win;
      (globalThis as unknown as { document: unknown }).document = doc;

      const { result, unmount } = renderHook(() => useGalleryFilter());
      expect(result.current.selectedMovement).toBe('Early Works');
      expect(result.current.sortDirection).toBe('asc');

      // Simulate user clicking back button in browser:
      // Browser updates window.location.search and dispatches popstate
      win.location.search = '?movement=The%20Crayon%20Period&sort=desc';
      act(() => {
        win.dispatchEvent(new Event('popstate'));
      });

      expect(result.current.selectedMovement).toBe('The Crayon Period');
      expect(result.current.sortDirection).toBe('desc');
      unmount();
    });

    it('should clean up the popstate event listener on unmount', () => {
      const { win, doc } = createMockEnvironment();
      (globalThis as unknown as { window: unknown }).window = win;
      (globalThis as unknown as { document: unknown }).document = doc;

      const { unmount } = renderHook(() => useGalleryFilter());

      expect(win.addEventListener).toHaveBeenCalledWith('popstate', expect.any(Function));

      unmount();

      expect(win.removeEventListener).toHaveBeenCalledWith('popstate', expect.any(Function));
    });
  });

  describe('resetFilter Action', () => {
    it('should clear query parameters and return state to defaults', () => {
      const { win, doc } = createMockEnvironment({
        pathname: '/gallery',
        search: '?movement=Domestic%20Destructionism&sort=asc',
      });
      (globalThis as unknown as { window: unknown }).window = win;
      (globalThis as unknown as { document: unknown }).document = doc;

      const { result, unmount } = renderHook(() => useGalleryFilter());
      expect(result.current.selectedMovement).toBe('Domestic Destructionism');
      expect(result.current.sortDirection).toBe('asc');

      act(() => {
        result.current.resetFilter();
      });

      expect(result.current.selectedMovement).toBeNull();
      expect(result.current.sortDirection).toBe('desc');
      expect(win.history.pushState).toHaveBeenCalledWith(null, '', '/gallery');
      unmount();
    });
  });
});
