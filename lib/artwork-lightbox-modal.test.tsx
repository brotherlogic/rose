import { describe, it, expect, vi, afterEach } from 'vitest';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import * as fs from 'fs';
import * as path from 'path';
import ArtworkLightboxModal, { ArtworkLightboxModalProps } from '../app/components/ArtworkLightboxModal';
import { gallery } from './proto/gallery';

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

// Helper to render ArtworkLightboxModal inside a React SSR dispatcher context and capture the element tree
function renderAndCapture(props: ArtworkLightboxModalProps): React.ReactElement {
  let captured: React.ReactElement | null = null;
  function Wrapper() {
    const el = ArtworkLightboxModal(props);
    captured = el;
    return el;
  }
  renderToStaticMarkup(React.createElement(Wrapper));
  if (!captured) {
    throw new Error('Failed to capture rendered element tree');
  }
  return captured;
}

describe('Artwork Lightbox Modal Component Suite (#297)', () => {
  const mockArtwork: gallery.IArtwork = {
    id: 'art-001',
    title: 'Disintegration of Form in Yellow',
    medium: 'Wax crayon on reclaimed cardboard',
    description: 'A study in material decay and existential resignation.',
    themeId: 'Monochrome Nihilism',
    imagePath: '/data/images/art-001.jpg',
    thumbnailPath: '/data/thumbnails/art-001.webp',
    timestamp: 1712707200,
  };

  const defaultProps: ArtworkLightboxModalProps = {
    artwork: mockArtwork,
    currentIndex: 0,
    totalCount: 3,
    isPlaqueCollapsed: false,
    onTogglePlaque: vi.fn(),
    onNext: vi.fn(),
    onPrev: vi.fn(),
    onClose: vi.fn(),
  };

  const componentPath = path.join(process.cwd(), 'app/components/ArtworkLightboxModal.tsx');
  const componentSource = fs.readFileSync(componentPath, 'utf8');

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Modal Overlay, Presentation & Accessibility Semantics', () => {
    it('should render dialog role semantics: role="dialog", aria-modal="true", and aria-labelledby="curatorial-plaque-title"', () => {
      const markup = renderToStaticMarkup(React.createElement(ArtworkLightboxModal, defaultProps));

      expect(markup).toContain('role="dialog"');
      expect(markup).toContain('aria-modal="true"');
      expect(markup).toContain('aria-labelledby="curatorial-plaque-title"');
    });

    it('should render high-opacity dark backdrop with blur overlay', () => {
      const markup = renderToStaticMarkup(React.createElement(ArtworkLightboxModal, defaultProps));

      expect(markup).toMatch(/fixed\s+inset-0/);
      expect(markup).toContain('z-50');
      expect(markup).toMatch(/bg-black\/90/);
      expect(markup).toContain('backdrop-blur-sm');
    });

    it('should render responsive image container preserving aspect ratio with object-contain max-h-[65vh] max-w-[90vw]', () => {
      const markup = renderToStaticMarkup(React.createElement(ArtworkLightboxModal, defaultProps));

      expect(markup).toMatch(/object-contain/);
      expect(markup).toMatch(/max-h-\[65vh\]/);
      expect(markup).toMatch(/max-w-\[90vw\]/);
    });

    it('should render full-resolution image source (artwork.imagePath)', () => {
      const markup = renderToStaticMarkup(React.createElement(ArtworkLightboxModal, defaultProps));

      expect(markup).toContain('src="/data/images/art-001.jpg"');
      expect(markup).toContain('alt="Disintegration of Form in Yellow"');
    });

    it('should fallback to thumbnailPath when imagePath is absent', () => {
      const artworkWithoutImage: gallery.IArtwork = {
        ...mockArtwork,
        imagePath: undefined,
      };
      const markup = renderToStaticMarkup(
        React.createElement(ArtworkLightboxModal, {
          ...defaultProps,
          artwork: artworkWithoutImage,
        })
      );

      expect(markup).toContain('src="/data/thumbnails/art-001.webp"');
    });

    it('should fallback to /placeholder.svg when both imagePath and thumbnailPath are absent', () => {
      const artworkEmpty: gallery.IArtwork = {
        id: 'empty-art',
        title: 'Empty Void',
      };
      const markup = renderToStaticMarkup(
        React.createElement(ArtworkLightboxModal, {
          ...defaultProps,
          artwork: artworkEmpty,
        })
      );

      expect(markup).toContain('src="/placeholder.svg"');
    });

    it('should render fallback /placeholder.svg when initialError is true', () => {
      const markup = renderToStaticMarkup(
        React.createElement(ArtworkLightboxModal, {
          ...defaultProps,
          initialError: true,
        })
      );

      expect(markup).toContain('src="/placeholder.svg"');
    });

    it('should embed CuratorialPlaque with matching id="curatorial-plaque-title" and passed isPlaqueCollapsed', () => {
      const markup = renderToStaticMarkup(React.createElement(ArtworkLightboxModal, defaultProps));

      expect(markup).toContain('id="curatorial-plaque-title"');
      expect(markup).toContain('Disintegration of Form in Yellow');
      expect(markup).toContain('Wax crayon on reclaimed cardboard');
    });

    it('should position CuratorialPlaque below the artwork image in vertical flex flow without absolute overlap', () => {
      const tree = renderAndCapture(defaultProps);
      const content = findElements(tree, (el) => el.props['data-testid'] === 'lightbox-content')[0];
      expect(content).toBeDefined();
      expect(content.props.className).toContain('flex-col');
      expect(content.props.className).toContain('gap-4');

      const children = React.Children.toArray(content.props.children) as React.ReactElement[];
      expect(children.length).toBe(2);

      const imageWrapper = children[0];
      const plaqueWrapper = children[1];

      expect(findElements(imageWrapper, (el) => el.type === 'img').length).toBe(1);
      expect(plaqueWrapper.props.className).not.toContain('absolute');
      expect(plaqueWrapper.props.className).not.toContain('bottom-4');
    });
  });

  describe('Circular Navigation & Dismissal Controls', () => {
    it('should render dedicated Previous, Next, and Close buttons with correct aria-labels', () => {
      const markup = renderToStaticMarkup(React.createElement(ArtworkLightboxModal, defaultProps));

      expect(markup).toContain('aria-label="Previous artwork"');
      expect(markup).toContain('aria-label="Next artwork"');
      expect(markup).toContain('aria-label="Close modal"');
    });

    it('should render circular counter indicator displaying current index and total count', () => {
      const markup = renderToStaticMarkup(
        React.createElement(ArtworkLightboxModal, {
          ...defaultProps,
          currentIndex: 1,
          totalCount: 5,
        })
      );

      expect(markup).toMatch(/2\s*\/\s*5/);
    });

    it('should safely display single-item subset without index errors', () => {
      const markup = renderToStaticMarkup(
        React.createElement(ArtworkLightboxModal, {
          ...defaultProps,
          currentIndex: 0,
          totalCount: 1,
        })
      );

      expect(markup).toMatch(/1\s*\/\s*1/);
    });

    it('should trigger onPrev when Previous button is clicked', () => {
      const onPrev = vi.fn();
      const tree = renderAndCapture({ ...defaultProps, onPrev });
      const buttons = findElements(tree, (el) => el.type === 'button');
      const prevBtn = buttons.find((btn) => btn.props['aria-label'] === 'Previous artwork');
      expect(prevBtn).toBeDefined();
      prevBtn?.props.onClick?.({ preventDefault: () => {} });
      expect(onPrev).toHaveBeenCalledTimes(1);
    });

    it('should trigger onNext when Next button is clicked', () => {
      const onNext = vi.fn();
      const tree = renderAndCapture({ ...defaultProps, onNext });
      const buttons = findElements(tree, (el) => el.type === 'button');
      const nextBtn = buttons.find((btn) => btn.props['aria-label'] === 'Next artwork');
      expect(nextBtn).toBeDefined();
      nextBtn?.props.onClick?.({ preventDefault: () => {} });
      expect(onNext).toHaveBeenCalledTimes(1);
    });

    it('should trigger onClose when Close button is clicked', () => {
      const onClose = vi.fn();
      const tree = renderAndCapture({ ...defaultProps, onClose });
      const buttons = findElements(tree, (el) => el.type === 'button');
      const closeBtn = buttons.find((btn) => btn.props['aria-label'] === 'Close modal');
      expect(closeBtn).toBeDefined();
      closeBtn?.props.onClick?.({ preventDefault: () => {} });
      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it('should trigger onClose when clicking the dark backdrop outside modal content', () => {
      const onClose = vi.fn();
      const tree = renderAndCapture({ ...defaultProps, onClose });
      const backdrop = findElements(tree, (el) => el.props['data-testid'] === 'lightbox-backdrop')[0];
      expect(backdrop).toBeDefined();
      const sentinel = {};
      backdrop.props.onClick({ target: sentinel, currentTarget: sentinel });
      expect(onClose).toHaveBeenCalledTimes(1);
    });

    it('should NOT trigger onClose when clicking inside the artwork content', () => {
      const onClose = vi.fn();
      const tree = renderAndCapture({ ...defaultProps, onClose });
      const content = findElements(tree, (el) => el.props['data-testid'] === 'lightbox-content')[0];
      expect(content).toBeDefined();
      const stopPropagation = vi.fn();
      content.props.onClick({ stopPropagation });
      expect(stopPropagation).toHaveBeenCalledTimes(1);
      expect(onClose).not.toHaveBeenCalled();
    });

    it('should bind onError event handler on artwork image element for fallback handling', () => {
      const tree = renderAndCapture({ ...defaultProps });
      const images = findElements(tree, (el) => el.type === 'img');
      expect(images.length).toBe(1);
      expect(images[0].props.onError).toBeDefined();
    });
  });

  describe('Document Scroll Lock, Keyboard Navigation & Focus Trapping Verification', () => {
    it('should implement document body scroll lock with overflow-hidden on mount and clean restoration', () => {
      expect(componentSource).toContain("document.body.classList.add('overflow-hidden')");
      expect(componentSource).toContain("document.body.classList.remove('overflow-hidden')");
    });

    it('should bind keydown listeners for ArrowLeft, ArrowRight, and Escape shortcuts', () => {
      expect(componentSource).toContain("addEventListener('keydown'");
      expect(componentSource).toContain("removeEventListener('keydown'");
      expect(componentSource).toContain("e.key === 'Escape'");
      expect(componentSource).toContain("e.key === 'ArrowLeft'");
      expect(componentSource).toContain("e.key === 'ArrowRight'");
    });

    it('should implement accessible Tab and Shift+Tab focus trapping within modal elements', () => {
      expect(componentSource).toContain("e.key === 'Tab'");
      expect(componentSource).toContain('querySelectorAll');
      expect(componentSource).toContain('shiftKey');
    });

    it('should persist curatorial plaque collapsed state through onToggleCollapse binding', () => {
      const onTogglePlaque = vi.fn();
      const markupExpanded = renderToStaticMarkup(
        React.createElement(ArtworkLightboxModal, {
          ...defaultProps,
          isPlaqueCollapsed: false,
          onTogglePlaque,
        })
      );
      expect(markupExpanded).toContain('aria-label="Collapse curatorial plaque"');

      const markupCollapsed = renderToStaticMarkup(
        React.createElement(ArtworkLightboxModal, {
          ...defaultProps,
          isPlaqueCollapsed: true,
          onTogglePlaque,
        })
      );
      expect(markupCollapsed).toContain('aria-label="Expand curatorial plaque"');
    });
  });
});
