'use client';

import { useState, useRef, useCallback } from 'react';
import type React from 'react';
import type { gallery } from '@/lib/proto/gallery';

export interface UseArtworkLightboxReturn {
  activeIndex: number | null;
  isPlaqueCollapsed: boolean;
  triggerElementRef: React.RefObject<HTMLElement | null>;
  openLightbox: (index: number, triggerEl?: HTMLElement | null) => void;
  closeLightbox: () => void;
  nextArtwork: () => void;
  prevArtwork: () => void;
  togglePlaque: () => void;
}

/**
 * Client hook for managing fullscreen artwork lightbox modal state:
 * - Active modal index and circular navigation across artworks
 * - Persistent curatorial plaque collapse state
 * - Trigger element reference tracking for accessible focus restoration
 * - Pure in-memory UI state strictly preserving canonical URL search params
 */
export function useArtworkLightbox(artworks: gallery.IArtwork[] = []): UseArtworkLightboxReturn {
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const [isPlaqueCollapsed, setIsPlaqueCollapsed] = useState<boolean>(false);
  const triggerElementRef = useRef<HTMLElement | null>(null);

  const openLightbox = useCallback(
    (index: number, triggerEl?: HTMLElement | null) => {
      if (!artworks || artworks.length === 0) {
        return;
      }
      if (index >= 0 && index < artworks.length) {
        setActiveIndex(index);
        triggerElementRef.current = triggerEl ?? null;
      }
    },
    [artworks]
  );

  const closeLightbox = useCallback(() => {
    setActiveIndex(null);
    if (triggerElementRef.current && typeof triggerElementRef.current.focus === 'function') {
      triggerElementRef.current.focus();
    }
  }, []);

  const nextArtwork = useCallback(() => {
    setActiveIndex((prev) => {
      if (prev === null || !artworks || artworks.length === 0) {
        return null;
      }
      return (prev + 1) % artworks.length;
    });
  }, [artworks]);

  const prevArtwork = useCallback(() => {
    setActiveIndex((prev) => {
      if (prev === null || !artworks || artworks.length === 0) {
        return null;
      }
      return (prev - 1 + artworks.length) % artworks.length;
    });
  }, [artworks]);

  const togglePlaque = useCallback(() => {
    setIsPlaqueCollapsed((prev) => !prev);
  }, []);

  return {
    activeIndex,
    isPlaqueCollapsed,
    triggerElementRef,
    openLightbox,
    closeLightbox,
    nextArtwork,
    prevArtwork,
    togglePlaque,
  };
}
