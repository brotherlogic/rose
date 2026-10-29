import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import React, { act } from 'react';
import { createRoot } from 'react-dom/client';
import { renderToStaticMarkup } from 'react-dom/server';
import * as fs from 'fs';
import * as path from 'path';
import CuratedGalleryCollection, { CuratedGalleryCollectionProps } from '../app/components/CuratedGalleryCollection';
import ArtworkLightboxModal, { ArtworkLightboxModalProps } from '../app/components/ArtworkLightboxModal';
import CuratorialPlaque from '../app/components/CuratorialPlaque';
import ArtworkCard from '../app/components/ArtworkCard';
import VirtualizedGalleryGrid from '../app/components/VirtualizedGalleryGrid';
import { useArtworkLightbox } from '../app/hooks/useArtworkLightbox';
import * as useArtworkLightboxModule from '../app/hooks/useArtworkLightbox';
import * as useGalleryFilterModule from '../app/hooks/useGalleryFilter';
import type { gallery } from '../lib/proto/gallery';

(globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

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

// Helper to render CuratedGalleryCollection inside React SSR dispatcher context and capture the element tree
function renderAndCaptureCollection(props: CuratedGalleryCollectionProps): React.ReactElement {
  let captured: React.ReactElement | null = null;
  function Wrapper() {
    const el = CuratedGalleryCollection(props);
    captured = el;
    return el;
  }
  renderToStaticMarkup(React.createElement(Wrapper));
  if (!captured) {
    throw new Error('Failed to capture rendered CuratedGalleryCollection tree');
  }
  return captured;
}

// Helper to render ArtworkLightboxModal inside React SSR dispatcher context and capture the element tree
function renderAndCaptureModal(props: ArtworkLightboxModalProps): React.ReactElement {
  let captured: React.ReactElement | null = null;
  function Wrapper() {
    const el = ArtworkLightboxModal(props);
    captured = el;
    return el;
  }
  renderToStaticMarkup(React.createElement(Wrapper));
  if (!captured) {
    throw new Error('Failed to capture rendered ArtworkLightboxModal tree');
  }
  return captured;
}

// Helper for testing React hooks in Node environment
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

describe('End-to-End Lightbox Verification Suite (#299)', () => {
  const originalWindow = globalThis.window;
  const originalDocument = globalThis.document;

  const sampleArtworks: gallery.IArtwork[] = [
    {
      id: 'art-001',
      title: 'Disintegration of Form in Yellow',
      medium: 'Wax Crayon on Reclaimed Cardboard',
      description: 'A study in material decay and existential resignation.',
      theme: 'The Crayon Period',
      themeId: 'The Crayon Period',
      imagePath: '/data/images/art-001.jpg',
      thumbnailPath: '/data/thumbnails/art-001.webp',
      timestamp: 1712707200, // 2024
    },
    {
      id: 'art-002',
      title: 'Torn Cardboard Sculpture',
      medium: 'Cardboard and Industrial Adhesive',
      description: 'The architectural fragility of domestic structures.',
      theme: 'Domestic Destructionism',
      themeId: 'Domestic Destructionism',
      imagePath: '/data/images/art-002.jpg',
      thumbnailPath: '/data/thumbnails/art-002.webp',
      timestamp: 1700000000, // 2023
    },
    {
      id: 'art-003',
      title: 'Kinetic Scribble No. 4',
      medium: 'Graphite on Newsprint',
      description: 'Unchecked juvenile momentum resisting curatorial order.',
      theme: 'The Crayon Period',
      themeId: 'The Crayon Period',
      imagePath: '/data/images/art-003.jpg',
      thumbnailPath: '/data/thumbnails/art-003.webp',
      timestamp: 1715000000, // 2024
    },
  ];

  let mockWin: {
    location: { pathname: string; search: string };
    history: { pushState: ReturnType<typeof vi.fn>; replaceState: ReturnType<typeof vi.fn> };
    addEventListener: ReturnType<typeof vi.fn>;
    removeEventListener: ReturnType<typeof vi.fn>;
    HTMLIFrameElement: () => void;
  };

  const modalSourcePath = path.join(process.cwd(), 'app/components/ArtworkLightboxModal.tsx');
  const modalSource = fs.readFileSync(modalSourcePath, 'utf8');

  beforeEach(() => {
    mockWin = {
      location: {
        pathname: '/',
        search: '?movement=The+Crayon+Period&sort=desc',
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
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      HTMLIFrameElement: function () {},
    };

    const mockDoc = {
      nodeType: 9,
      body: {
        classList: {
          add: vi.fn(),
          remove: vi.fn(),
        },
      },
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

  describe('1. Full User Journey: Triggering Lightbox from Artwork Card', () => {
    it('should wire ArtworkCard trigger button to openLightbox with correct artwork index and element ref', () => {
      const openLightboxMock = vi.fn();
      vi.spyOn(useArtworkLightboxModule, 'useArtworkLightbox').mockReturnValue({
        activeIndex: null,
        isPlaqueCollapsed: false,
        triggerElementRef: { current: null },
        openLightbox: openLightboxMock,
        closeLightbox: vi.fn(),
        nextArtwork: vi.fn(),
        prevArtwork: vi.fn(),
        togglePlaque: vi.fn(),
      });

      const tree = renderAndCaptureCollection({ artworks: sampleArtworks });
      const gridEl = findElements(tree, (el) => el.type === VirtualizedGalleryGrid)[0];
      expect(gridEl).toBeDefined();

      const mockTriggerEl = { tagName: 'BUTTON', focus: vi.fn() } as unknown as HTMLElement;
      gridEl.props.onSelectArtwork(sampleArtworks[1], 1, mockTriggerEl);

      expect(openLightboxMock).toHaveBeenCalledWith(1, mockTriggerEl);
    });

    it('should render ArtworkCard with accessible dialog trigger button and keyboard activation support', () => {
      const onSelectMock = vi.fn();
      const cardMarkup = renderToStaticMarkup(
        React.createElement(ArtworkCard, {
          artwork: sampleArtworks[0],
          onSelect: onSelectMock,
        })
      );

      expect(cardMarkup).toContain('aria-haspopup="dialog"');
      expect(cardMarkup).toContain('aria-label="View artwork in fullscreen"');
      expect(cardMarkup).toContain('Disintegration of Form in Yellow');
      expect(cardMarkup).toContain('Wax Crayon on Reclaimed Cardboard');

      // Test keyboard interaction on ArtworkCard
      let capturedCard: React.ReactElement | null = null;
      function CardWrapper() {
        const el = ArtworkCard({
          artwork: sampleArtworks[0],
          onSelect: onSelectMock,
        });
        capturedCard = el;
        return el;
      }
      renderToStaticMarkup(React.createElement(CardWrapper));

      const buttonEl = findElements(capturedCard, (el) => el.type === 'button')[0];
      expect(buttonEl).toBeDefined();

      const mockButton = { tagName: 'BUTTON' } as HTMLElement;
      buttonEl.props.onKeyDown?.({ key: 'Enter', preventDefault: vi.fn(), currentTarget: mockButton });
      expect(onSelectMock).toHaveBeenCalledWith(sampleArtworks[0], mockButton);

      buttonEl.props.onKeyDown?.({ key: ' ', preventDefault: vi.fn(), currentTarget: mockButton });
      expect(onSelectMock).toHaveBeenCalledWith(sampleArtworks[0], mockButton);
    });
  });

  describe('2. Fullscreen Image Display & Curatorial Plaque Contents', () => {
    it('should render fullscreen high-resolution image with aspect-ratio preservation and curatorial plaque details', () => {
      const modalProps: ArtworkLightboxModalProps = {
        artwork: sampleArtworks[0],
        currentIndex: 0,
        totalCount: 3,
        isPlaqueCollapsed: false,
        onTogglePlaque: vi.fn(),
        onNext: vi.fn(),
        onPrev: vi.fn(),
        onClose: vi.fn(),
      };

      const markup = renderToStaticMarkup(React.createElement(ArtworkLightboxModal, modalProps));

      // Lightbox presentation & image
      expect(markup).toContain('role="dialog"');
      expect(markup).toContain('aria-modal="true"');
      expect(markup).toContain('aria-labelledby="curatorial-plaque-title"');
      expect(markup).toContain('object-contain max-h-[65vh] max-w-[90vw]');
      expect(markup).toContain('src="/data/images/art-001.jpg"');
      expect(markup).toContain('1 / 3');
      expect(markup).not.toContain('absolute bottom-4');

      // Verify CuratorialPlaque sits below the image in vertical flex layout
      const modalTree = renderAndCaptureModal(modalProps);
      const contentEl = findElements(modalTree, (el) => el.props['data-testid'] === 'lightbox-content')[0];
      expect(contentEl.props.className).toContain('flex-col');
      expect(contentEl.props.className).toContain('gap-4');
      const contentChildren = React.Children.toArray(contentEl.props.children) as React.ReactElement[];
      expect(contentChildren[1].props.className).not.toContain('absolute');

      // Embedded Curatorial Plaque contents
      expect(markup).toContain('Disintegration of Form in Yellow');
      expect(markup).toContain('2024');
      expect(markup).toMatch(/Wax Crayon on Reclaimed Cardboard/i);
      expect(markup).toContain('A study in material decay and existential resignation.');
      expect(markup).toContain('The Crayon Period');
    });

    it('should handle undated and fallback artwork properties in curatorial plaque cleanly', () => {
      const incompleteArtwork: gallery.IArtwork = {
        id: 'art-fallback',
        title: '',
        medium: '',
        description: '',
        theme: '',
        timestamp: 0,
        imagePath: '',
        thumbnailPath: '',
      };

      const plaqueMarkup = renderToStaticMarkup(
        React.createElement(CuratorialPlaque, {
          artwork: incompleteArtwork,
          isCollapsed: false,
          onToggleCollapse: vi.fn(),
        })
      );

      expect(plaqueMarkup).toContain('Untitled');
      expect(plaqueMarkup).toContain('Undated');
      expect(plaqueMarkup).toContain('Mixed Media');
      expect(plaqueMarkup).toContain('Catalog description pending curatorial annotation.');
      expect(plaqueMarkup).toContain('Early Works');
    });
  });

  describe('3. Focus Trapping Inside Modal & Focus Restoration on Close', () => {
    it('should restore focus to the originating card trigger element when lightbox is closed', () => {
      const mockTriggerEl = {
        tagName: 'BUTTON',
        focus: vi.fn(),
      };

      const { result, unmount } = renderHook(() => useArtworkLightbox(sampleArtworks));

      // Open lightbox on item 1 with trigger element
      act(() => {
        result.current.openLightbox(1, mockTriggerEl as unknown as HTMLElement);
      });
      expect(result.current.activeIndex).toBe(1);
      expect(result.current.triggerElementRef.current).toBe(mockTriggerEl);

      // Close lightbox
      act(() => {
        result.current.closeLightbox();
      });
      expect(result.current.activeIndex).toBe(null);
      expect(mockTriggerEl.focus).toHaveBeenCalledTimes(1);

      unmount();
    });

    it('should implement accessible Tab focus trapping and Escape dismissal within modal', () => {
      // Verify implementation in component source
      expect(modalSource).toContain("e.key === 'Escape'");
      expect(modalSource).toContain("e.key === 'Tab'");
      expect(modalSource).toContain('querySelectorAll');
      expect(modalSource).toContain('e.shiftKey');
      expect(modalSource).toContain('firstElement');
      expect(modalSource).toContain('lastElement');
    });
  });

  describe('4. Circular Traversal Across Filtered Thematic Subsets', () => {
    it('should circularly loop forward and backward across filtered thematic subset', () => {
      // The Crayon Period contains art-001 and art-003 (2 items)
      const filteredSubset = [sampleArtworks[0], sampleArtworks[2]];
      const { result, unmount } = renderHook(() => useArtworkLightbox(filteredSubset));

      // Open at index 0 (art-001)
      act(() => {
        result.current.openLightbox(0);
      });
      expect(result.current.activeIndex).toBe(0);

      // Navigate forward -> index 1 (art-003)
      act(() => {
        result.current.nextArtwork();
      });
      expect(result.current.activeIndex).toBe(1);

      // Navigate forward at end -> wraps circularly to index 0 (art-001)
      act(() => {
        result.current.nextArtwork();
      });
      expect(result.current.activeIndex).toBe(0);

      // Navigate backward at start -> wraps circularly to index 1 (art-003)
      act(() => {
        result.current.prevArtwork();
      });
      expect(result.current.activeIndex).toBe(1);

      unmount();
    });

    it('should scope navigation strictly to active filtered exhibition subset in CuratedGalleryCollection', () => {
      const lightboxSpy = vi.spyOn(useArtworkLightboxModule, 'useArtworkLightbox');

      vi.spyOn(useGalleryFilterModule, 'useGalleryFilter').mockReturnValue({
        selectedMovement: 'Domestic Destructionism',
        sortDirection: 'desc',
        setMovement: vi.fn(),
        setSortDirection: vi.fn(),
        resetFilter: vi.fn(),
      });

      renderAndCaptureCollection({ artworks: sampleArtworks });

      // Only art-002 belongs to Domestic Destructionism
      expect(lightboxSpy).toHaveBeenCalledWith([sampleArtworks[1]]);
    });
  });

  describe('5. Persistent Plaque Collapse Toggle State Across Navigation', () => {
    it('should maintain plaque collapse state across previous and next navigation transitions', () => {
      const { result, unmount } = renderHook(() => useArtworkLightbox(sampleArtworks));

      act(() => {
        result.current.openLightbox(0);
      });
      expect(result.current.isPlaqueCollapsed).toBe(false);

      // Toggle plaque to collapsed
      act(() => {
        result.current.togglePlaque();
      });
      expect(result.current.isPlaqueCollapsed).toBe(true);

      // Navigate to next artwork
      act(() => {
        result.current.nextArtwork();
      });
      expect(result.current.activeIndex).toBe(1);
      // Plaque collapsed state persists
      expect(result.current.isPlaqueCollapsed).toBe(true);

      // Navigate to previous artwork
      act(() => {
        result.current.prevArtwork();
      });
      expect(result.current.activeIndex).toBe(0);
      // Plaque collapsed state still persists
      expect(result.current.isPlaqueCollapsed).toBe(true);

      // Toggle plaque back to expanded
      act(() => {
        result.current.togglePlaque();
      });
      expect(result.current.isPlaqueCollapsed).toBe(false);

      unmount();
    });
  });

  describe('6. Dismissal Triggers: Escape Key, Close Button, Outside Backdrop Click', () => {
    it('should dismiss modal when close button is clicked', () => {
      const onCloseMock = vi.fn();
      const modalTree = renderAndCaptureModal({
        artwork: sampleArtworks[0],
        currentIndex: 0,
        totalCount: 3,
        isPlaqueCollapsed: false,
        onTogglePlaque: vi.fn(),
        onNext: vi.fn(),
        onPrev: vi.fn(),
        onClose: onCloseMock,
      });

      const buttons = findElements(modalTree, (el) => el.type === 'button');
      const closeBtn = buttons.find((b) => b.props['aria-label'] === 'Close modal');
      expect(closeBtn).toBeDefined();

      closeBtn?.props.onClick?.({} as React.MouseEvent);
      expect(onCloseMock).toHaveBeenCalledTimes(1);
    });

    it('should dismiss modal on backdrop click, but ignore clicks within modal content container', () => {
      const onCloseMock = vi.fn();
      const modalTree = renderAndCaptureModal({
        artwork: sampleArtworks[0],
        currentIndex: 0,
        totalCount: 3,
        isPlaqueCollapsed: false,
        onTogglePlaque: vi.fn(),
        onNext: vi.fn(),
        onPrev: vi.fn(),
        onClose: onCloseMock,
      });

      // Backdrop element click where target === currentTarget
      const backdropEl = findElements(modalTree, (el) => el.props['data-testid'] === 'lightbox-backdrop')[0];
      expect(backdropEl).toBeDefined();

      const mockBackdropTarget = { id: 'backdrop' };
      backdropEl.props.onClick?.({
        target: mockBackdropTarget,
        currentTarget: mockBackdropTarget,
      } as unknown as React.MouseEvent);
      expect(onCloseMock).toHaveBeenCalledTimes(1);

      // Click on inner target (target !== currentTarget) should NOT trigger onClose
      backdropEl.props.onClick?.({
        target: { id: 'inner' },
        currentTarget: mockBackdropTarget,
      } as unknown as React.MouseEvent);
      expect(onCloseMock).toHaveBeenCalledTimes(1);

      // Content area stopPropagation
      const contentEl = findElements(modalTree, (el) => el.props['data-testid'] === 'lightbox-content')[0];
      expect(contentEl).toBeDefined();
      const stopPropMock = vi.fn();
      contentEl.props.onClick?.({
        stopPropagation: stopPropMock,
      } as unknown as React.MouseEvent);
      expect(stopPropMock).toHaveBeenCalled();
    });

    it('should bind arrow key and keyboard navigation in component source', () => {
      expect(modalSource).toContain("e.key === 'ArrowLeft'");
      expect(modalSource).toContain('onPrev()');
      expect(modalSource).toContain("e.key === 'ArrowRight'");
      expect(modalSource).toContain('onNext()');
    });
  });

  describe('7. URL Synchronization (window.location.search & Browser History)', () => {
    it('should synchronize ?artwork=<id> with pushState on open/close, replaceState on next/prev, and no history calls on plaque toggle', () => {
      const { result, unmount } = renderHook(() => useArtworkLightbox(sampleArtworks));

      // 1. Open Lightbox -> pushState with ?artwork=art-001 (and cleans canonical default sort=desc)
      act(() => {
        result.current.openLightbox(0);
      });
      expect(mockWin.history.pushState).toHaveBeenCalledWith(
        null,
        '',
        '/?movement=The+Crayon+Period&artwork=art-001'
      );
      expect(mockWin.history.replaceState).not.toHaveBeenCalled();
      expect(mockWin.location.search).toBe('?movement=The+Crayon+Period&artwork=art-001');

      // 2. Next Artwork -> replaceState with ?artwork=art-002
      act(() => {
        result.current.nextArtwork();
      });
      expect(mockWin.history.pushState).toHaveBeenCalledTimes(1);
      expect(mockWin.history.replaceState).toHaveBeenCalledWith(
        null,
        '',
        '/?movement=The+Crayon+Period&artwork=art-002'
      );
      expect(mockWin.location.search).toBe('?movement=The+Crayon+Period&artwork=art-002');

      // 3. Toggle Plaque -> no history calls or URL mutation
      vi.clearAllMocks();
      act(() => {
        result.current.togglePlaque();
      });
      expect(mockWin.location.search).toBe('?movement=The+Crayon+Period&artwork=art-002');
      expect(mockWin.history.pushState).not.toHaveBeenCalled();
      expect(mockWin.history.replaceState).not.toHaveBeenCalled();

      // 4. Previous Artwork -> replaceState with ?artwork=art-001
      act(() => {
        result.current.prevArtwork();
      });
      expect(mockWin.history.pushState).not.toHaveBeenCalled();
      expect(mockWin.history.replaceState).toHaveBeenCalledWith(
        null,
        '',
        '/?movement=The+Crayon+Period&artwork=art-001'
      );
      expect(mockWin.location.search).toBe('?movement=The+Crayon+Period&artwork=art-001');

      // 5. Close Lightbox -> pushState removing artwork while preserving movement
      act(() => {
        result.current.closeLightbox();
      });
      expect(mockWin.history.pushState).toHaveBeenCalledWith(
        null,
        '',
        '/?movement=The+Crayon+Period'
      );
      expect(mockWin.location.search).toBe('?movement=The+Crayon+Period');

      unmount();
    });
  });
});
