import { describe, it, expect, vi, afterEach } from 'vitest';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import CuratedGalleryCollection, { CuratedGalleryCollectionProps } from '../app/components/CuratedGalleryCollection';
import ExhibitionEmptyState from '../app/components/ExhibitionEmptyState';
import ExhibitionFilterBar from '../app/components/ExhibitionFilterBar';
import VirtualizedGalleryGrid from '../app/components/VirtualizedGalleryGrid';
import ArtworkLightboxModal from '../app/components/ArtworkLightboxModal';
import * as galleryFilterModule from '../lib/gallery-filter';
import * as useGalleryFilterModule from '../app/hooks/useGalleryFilter';
import * as useArtworkLightboxModule from '../app/hooks/useArtworkLightbox';
import * as useGalleryCatalogModule from '../app/hooks/useGalleryCatalog';
import type { gallery } from '../lib/proto/gallery';

(globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT: boolean }).IS_REACT_ACT_ENVIRONMENT = true;

// Helper to render CuratedGalleryCollection inside React SSR context and capture element tree
function renderAndCapture(props: CuratedGalleryCollectionProps): React.ReactElement {
  let captured: React.ReactElement | null = null;
  function Wrapper() {
    const el = CuratedGalleryCollection(props);
    captured = el;
    return el;
  }
  renderToStaticMarkup(React.createElement(Wrapper));
  if (!captured) {
    throw new Error('Failed to capture rendered element tree');
  }
  return captured;
}

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

describe('CuratedGalleryCollection Container Component Suite (#278)', () => {
  const sampleArtworks: gallery.IArtwork[] = [
    {
      id: 'art-001',
      title: 'Crayon Drawing No. 1',
      medium: 'Crayon on Construction Paper',
      theme: 'The Crayon Period',
      thumbnailPath: '/data/thumbnails/art-001.webp',
      imagePath: '/data/images/art-001.jpg',
      timestamp: 1712707200, // newer
    },
    {
      id: 'art-002',
      title: 'Torn Cardboard Sculpture',
      medium: 'Cardboard and Glue',
      theme: 'Domestic Destructionism',
      thumbnailPath: '/data/thumbnails/art-002.webp',
      imagePath: '/data/images/art-002.jpg',
      timestamp: 1700000000, // older
    },
    {
      id: 'art-003',
      title: 'Crayon Drawing No. 2',
      medium: 'Wax Crayon on Newsprint',
      theme: 'The Crayon Period',
      thumbnailPath: '/data/thumbnails/art-003.webp',
      imagePath: '/data/images/art-003.jpg',
      timestamp: 1710000000,
    },
  ];

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Empty Collection Handling (totalCount === 0)', () => {
    it('should render the global archival empty state when totalCount is 0 without crashing', () => {
      const markup = renderToStaticMarkup(
        React.createElement(CuratedGalleryCollection, { artworks: [] })
      );

      expect(markup).toContain('No Works Currently Cataloged');
      expect(markup).toContain('Waiting for archival syncer ingestion');
      // Should not render the filter bar when catalog is completely empty
      expect(markup).not.toContain('role="tablist"');
      expect(markup).not.toContain('Exhibition period filtering');
    });
  });

  describe('Curated Collection with Artworks (totalCount > 0)', () => {
    it('should render ExhibitionFilterBar and VirtualizedGalleryGrid with all artworks by default', () => {
      const markup = renderToStaticMarkup(
        React.createElement(CuratedGalleryCollection, { artworks: sampleArtworks })
      );

      // ExhibitionFilterBar should be present
      expect(markup).toContain('Exhibition period filtering');
      expect(markup).toContain('All Works (3)');
      expect(markup).toContain('The Crayon Period (2)');
      expect(markup).toContain('Domestic Destructionism (1)');

      // VirtualizedGalleryGrid items should be rendered
      expect(markup).toContain('Crayon Drawing No. 1');
      expect(markup).toContain('Torn Cardboard Sculpture');
      expect(markup).toContain('Crayon Drawing No. 2');
    });

    it('should call aggregateMovements and filterAndSortArtworks with correct arguments', () => {
      const aggregateSpy = vi.spyOn(galleryFilterModule, 'aggregateMovements');
      const filterSortSpy = vi.spyOn(galleryFilterModule, 'filterAndSortArtworks');

      renderToStaticMarkup(
        React.createElement(CuratedGalleryCollection, { artworks: sampleArtworks })
      );

      expect(aggregateSpy).toHaveBeenCalledWith(sampleArtworks);
      expect(filterSortSpy).toHaveBeenCalledWith(sampleArtworks, null, 'desc');
    });

    it('should wire ExhibitionFilterBar callbacks to useGalleryFilter actions', () => {
      const setMovementMock = vi.fn();
      const setSortDirectionMock = vi.fn();

      vi.spyOn(useGalleryFilterModule, 'useGalleryFilter').mockReturnValue({
        selectedMovement: null,
        sortDirection: 'desc',
        setMovement: setMovementMock,
        setSortDirection: setSortDirectionMock,
        resetFilter: vi.fn(),
      });

      const tree = renderAndCapture({ artworks: sampleArtworks });
      const filterBarEl = findElements(tree, (el) => el.type === ExhibitionFilterBar)[0];
      expect(filterBarEl).toBeDefined();

      filterBarEl.props.onSelectMovement('The Crayon Period');
      expect(setMovementMock).toHaveBeenCalledWith('The Crayon Period');

      filterBarEl.props.onToggleSort('asc');
      expect(setSortDirectionMock).toHaveBeenCalledWith('asc');
    });
  });

  describe('Filtering and Empty State Rendering', () => {
    it('should filter artworks when selectedMovement is active', () => {
      vi.spyOn(useGalleryFilterModule, 'useGalleryFilter').mockReturnValue({
        selectedMovement: 'The Crayon Period',
        sortDirection: 'desc',
        setMovement: vi.fn(),
        setSortDirection: vi.fn(),
        resetFilter: vi.fn(),
      });

      const markup = renderToStaticMarkup(
        React.createElement(CuratedGalleryCollection, { artworks: sampleArtworks })
      );

      // Only Crayon Period pieces should appear
      expect(markup).toContain('Crayon Drawing No. 1');
      expect(markup).toContain('Crayon Drawing No. 2');
      expect(markup).not.toContain('Torn Cardboard Sculpture');
    });

    it('should render ExhibitionEmptyState when selected movement yields 0 filtered artworks', () => {
      const resetFilterMock = vi.fn();
      vi.spyOn(useGalleryFilterModule, 'useGalleryFilter').mockReturnValue({
        selectedMovement: 'Monochrome Nihilism',
        sortDirection: 'desc',
        setMovement: vi.fn(),
        setSortDirection: vi.fn(),
        resetFilter: resetFilterMock,
      });

      const markup = renderToStaticMarkup(
        React.createElement(CuratedGalleryCollection, { artworks: sampleArtworks })
      );

      // Filter bar remains visible
      expect(markup).toContain('Exhibition period filtering');
      // Empty state displays movement name and curatorial message
      expect(markup).toContain('Monochrome Nihilism');
      expect(markup).toContain('No works from this movement are currently on public exhibition.');
      expect(markup).toContain('Return to All Works');
    });

    it('should trigger resetFilter when Return to All Works is clicked in empty state', () => {
      const resetFilterMock = vi.fn();
      vi.spyOn(useGalleryFilterModule, 'useGalleryFilter').mockReturnValue({
        selectedMovement: 'Monochrome Nihilism',
        sortDirection: 'desc',
        setMovement: vi.fn(),
        setSortDirection: vi.fn(),
        resetFilter: resetFilterMock,
      });

      const tree = renderAndCapture({ artworks: sampleArtworks });
      const emptyStateEl = findElements(tree, (el) => el.type === ExhibitionEmptyState)[0];
      expect(emptyStateEl).toBeDefined();
      expect(emptyStateEl.props.movementName).toBe('Monochrome Nihilism');

      // Test invoking the prop directly
      emptyStateEl.props.onReset();
      expect(resetFilterMock).toHaveBeenCalledTimes(1);

      // Test clicking the rendered button inside ExhibitionEmptyState
      const renderedChild = ExhibitionEmptyState(emptyStateEl.props);
      const buttons = findElements(renderedChild, (el) => el.type === 'button');
      const resetBtn = buttons.find((btn) => {
        const content = JSON.stringify(btn.props);
        return content.includes('Return to All Works');
      });

      expect(resetBtn).toBeDefined();
      resetBtn?.props.onClick?.({ preventDefault: () => {} });
      expect(resetFilterMock).toHaveBeenCalledTimes(2);
    });
  });

  describe('Sorting Integration', () => {
    it('should pass sortDirection="asc" to filterAndSortArtworks and render artworks sorted accordingly', () => {
      const filterSortSpy = vi.spyOn(galleryFilterModule, 'filterAndSortArtworks');
      vi.spyOn(useGalleryFilterModule, 'useGalleryFilter').mockReturnValue({
        selectedMovement: null,
        sortDirection: 'asc',
        setMovement: vi.fn(),
        setSortDirection: vi.fn(),
        resetFilter: vi.fn(),
      });

      renderToStaticMarkup(
        React.createElement(CuratedGalleryCollection, { artworks: sampleArtworks })
      );

      expect(filterSortSpy).toHaveBeenCalledWith(sampleArtworks, null, 'asc');
    });
  });

  describe('Lightbox Integration Suite (#298)', () => {
    it('should pass onSelectArtwork callback to VirtualizedGalleryGrid and not render modal initially', () => {
      const tree = renderAndCapture({ artworks: sampleArtworks });
      const gridEl = findElements(tree, (el) => el.type === VirtualizedGalleryGrid)[0];
      expect(gridEl).toBeDefined();
      expect(gridEl.props.onSelectArtwork).toBeDefined();
      expect(typeof gridEl.props.onSelectArtwork).toBe('function');

      const modalEl = findElements(tree, (el) => el.type === ArtworkLightboxModal)[0];
      expect(modalEl).toBeUndefined();
    });

    it('should wire onSelectArtwork to openLightbox with index and trigger element', () => {
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

      const tree = renderAndCapture({ artworks: sampleArtworks });
      const gridEl = findElements(tree, (el) => el.type === VirtualizedGalleryGrid)[0];
      expect(gridEl).toBeDefined();

      const mockTrigger = { tagName: 'BUTTON' } as unknown as HTMLElement;
      gridEl.props.onSelectArtwork(sampleArtworks[1], 1, mockTrigger);
      expect(openLightboxMock).toHaveBeenCalledWith(1, mockTrigger);
    });

    it('should render ArtworkLightboxModal when activeIndex is set, passing correct props and callbacks', () => {
      const closeLightboxMock = vi.fn();
      const nextArtworkMock = vi.fn();
      const prevArtworkMock = vi.fn();
      const togglePlaqueMock = vi.fn();

      vi.spyOn(useArtworkLightboxModule, 'useArtworkLightbox').mockReturnValue({
        activeIndex: 1,
        isPlaqueCollapsed: false,
        triggerElementRef: { current: null },
        openLightbox: vi.fn(),
        closeLightbox: closeLightboxMock,
        nextArtwork: nextArtworkMock,
        prevArtwork: prevArtworkMock,
        togglePlaque: togglePlaqueMock,
      });

      const tree = renderAndCapture({ artworks: sampleArtworks });
      const modalEl = findElements(tree, (el) => el.type === ArtworkLightboxModal)[0];
      expect(modalEl).toBeDefined();
      expect(modalEl.props.artwork.id).toBe('art-003');
      expect(modalEl.props.currentIndex).toBe(1);
      expect(modalEl.props.totalCount).toBe(3);

      modalEl.props.onNext();
      expect(nextArtworkMock).toHaveBeenCalledTimes(1);

      modalEl.props.onPrev();
      expect(prevArtworkMock).toHaveBeenCalledTimes(1);

      modalEl.props.onClose();
      expect(closeLightboxMock).toHaveBeenCalledTimes(1);

      modalEl.props.onTogglePlaque();
      expect(togglePlaqueMock).toHaveBeenCalledTimes(1);
    });

    it('should scope useArtworkLightbox strictly to the active filtered exhibition subset', () => {
      const lightboxSpy = vi.spyOn(useArtworkLightboxModule, 'useArtworkLightbox');

      vi.spyOn(useGalleryFilterModule, 'useGalleryFilter').mockReturnValue({
        selectedMovement: 'The Crayon Period',
        sortDirection: 'desc',
        setMovement: vi.fn(),
        setSortDirection: vi.fn(),
        resetFilter: vi.fn(),
      });

      renderAndCapture({ artworks: sampleArtworks });

      // Only art-001 and art-003 belong to The Crayon Period
      expect(lightboxSpy).toHaveBeenCalledWith(
        [sampleArtworks[0], sampleArtworks[2]],
        expect.objectContaining({
          catalogArtworks: sampleArtworks,
          isCatalogReady: true,
          onClearMovementFilter: expect.any(Function),
        })
      );
    });

    it('should forward catalogArtworks, isCatalogReady, and onClearMovementFilter across initial and hydrated states', () => {
      const clearMovementFilterMock = vi.fn();
      const setMovementMock = vi.fn();

      vi.spyOn(useGalleryCatalogModule, 'useGalleryCatalog').mockReturnValue({
        artworks: [],
        isHydrated: false,
        isCatalogReady: false,
      });

      vi.spyOn(useGalleryFilterModule, 'useGalleryFilter').mockReturnValue({
        selectedMovement: 'The Crayon Period',
        sortDirection: 'desc',
        setMovement: setMovementMock,
        setSortDirection: vi.fn(),
        resetFilter: vi.fn(),
        clearMovementFilter: clearMovementFilterMock,
      });

      const lightboxSpy = vi.spyOn(useArtworkLightboxModule, 'useArtworkLightbox');

      renderAndCapture({ artworks: [] });

      expect(lightboxSpy).toHaveBeenCalledWith(
        [],
        expect.objectContaining({
          catalogArtworks: [],
          isCatalogReady: false,
          onClearMovementFilter: expect.any(Function),
        })
      );

      const passedOptions = lightboxSpy.mock.calls[0][1];
      passedOptions?.onClearMovementFilter?.();
      expect(clearMovementFilterMock).toHaveBeenCalledTimes(1);
      expect(setMovementMock).not.toHaveBeenCalled();
    });

    it('should fall back to setMovement(null) in onClearMovementFilter when clearMovementFilter is not provided', () => {
      const setMovementMock = vi.fn();

      vi.spyOn(useGalleryFilterModule, 'useGalleryFilter').mockReturnValue({
        selectedMovement: 'The Crayon Period',
        sortDirection: 'desc',
        setMovement: setMovementMock,
        setSortDirection: vi.fn(),
        resetFilter: vi.fn(),
      });

      const lightboxSpy = vi.spyOn(useArtworkLightboxModule, 'useArtworkLightbox');

      renderAndCapture({ artworks: sampleArtworks });

      const passedOptions = lightboxSpy.mock.calls[0][1];
      expect(passedOptions).toBeDefined();
      passedOptions?.onClearMovementFilter?.();
      expect(setMovementMock).toHaveBeenCalledWith(null);
    });
  });

  describe('Dynamic Catalog Hydration Suite (#345)', () => {
    const hydratedArtworks: gallery.IArtwork[] = [
      ...sampleArtworks,
      {
        id: 'art-004',
        title: 'Midnight Fingerpaint Study',
        medium: 'Tempera on Plaster',
        themeId: 'Kinetic Scribblism',
        thumbnailPath: '/data/thumbnails/art-004.webp',
        imagePath: '/data/images/art-004.jpg',
        timestamp: 1720000000,
      },
    ];

    it('should pass the initial artworks prop into useGalleryCatalog', () => {
      const catalogSpy = vi.spyOn(useGalleryCatalogModule, 'useGalleryCatalog');

      renderToStaticMarkup(
        React.createElement(CuratedGalleryCollection, { artworks: sampleArtworks })
      );

      expect(catalogSpy).toHaveBeenCalledWith(sampleArtworks);
    });

    it('should dynamically update ExhibitionFilterBar movement counts, VirtualizedGalleryGrid items, and ArtworkLightboxModal navigation when useGalleryCatalog returns hydrated artworks', () => {
      vi.spyOn(useGalleryCatalogModule, 'useGalleryCatalog').mockReturnValue({
        artworks: hydratedArtworks,
        isHydrated: true,
      });

      const lightboxSpy = vi.spyOn(useArtworkLightboxModule, 'useArtworkLightbox').mockReturnValue({
        activeIndex: 0,
        isPlaqueCollapsed: false,
        triggerElementRef: { current: null },
        openLightbox: vi.fn(),
        closeLightbox: vi.fn(),
        nextArtwork: vi.fn(),
        prevArtwork: vi.fn(),
        togglePlaque: vi.fn(),
      });

      // Pass only a single initial static artwork while useGalleryCatalog returns 4 hydrated artworks
      const tree = renderAndCapture({ artworks: [sampleArtworks[0]] });

      const filterBarEl = findElements(tree, (el) => el.type === ExhibitionFilterBar)[0];
      expect(filterBarEl).toBeDefined();
      expect(filterBarEl.props.totalCount).toBe(4);
      expect(filterBarEl.props.movements).toEqual([
        { name: 'The Crayon Period', count: 2 },
        { name: 'Domestic Destructionism', count: 1 },
        { name: 'Kinetic Scribblism', count: 1 },
      ]);

      const gridEl = findElements(tree, (el) => el.type === VirtualizedGalleryGrid)[0];
      expect(gridEl).toBeDefined();
      expect(gridEl.props.artworks).toHaveLength(4);
      expect(gridEl.props.artworks[0].id).toBe('art-004');

      expect(lightboxSpy).toHaveBeenCalledWith(
        gridEl.props.artworks,
        expect.objectContaining({
          catalogArtworks: hydratedArtworks,
          isCatalogReady: true,
          onClearMovementFilter: expect.any(Function),
        })
      );

      const modalEl = findElements(tree, (el) => el.type === ArtworkLightboxModal)[0];
      expect(modalEl).toBeDefined();
      expect(modalEl.props.artwork.id).toBe('art-004');
      expect(modalEl.props.totalCount).toBe(4);
    });

    it('should transition from an empty initial static baseline to a populated dynamic catalog when useGalleryCatalog hydrates', () => {
      vi.spyOn(useGalleryCatalogModule, 'useGalleryCatalog').mockReturnValue({
        artworks: sampleArtworks,
        isHydrated: true,
      });

      const markup = renderToStaticMarkup(
        React.createElement(CuratedGalleryCollection, { artworks: [] })
      );

      expect(markup).not.toContain('No Works Currently Cataloged');
      expect(markup).toContain('Exhibition period filtering');
      expect(markup).toContain('All Works (3)');
      expect(markup).toContain('The Crayon Period (2)');
      expect(markup).toContain('Domestic Destructionism (1)');
      expect(markup).toContain('Crayon Drawing No. 1');
      expect(markup).toContain('Torn Cardboard Sculpture');
      expect(markup).toContain('Crayon Drawing No. 2');
    });
  });
});


