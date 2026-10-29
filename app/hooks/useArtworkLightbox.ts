'use client';
/* eslint-disable react-hooks/rules-of-hooks */

import { useState, useRef, useCallback, useEffect } from 'react';
import type React from 'react';
import type { gallery } from '@/lib/proto/gallery';

export interface UseArtworkLightboxOptions {
  catalogArtworks?: gallery.IArtwork[];
  isCatalogReady?: boolean;
  onClearMovementFilter?: () => void;
}

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

function buildLightboxUrl(
  artworkId: string | null,
  options?: { clearMovement?: boolean }
): string {
  const params = new URLSearchParams(window.location.search);

  if (options?.clearMovement || params.get('movement') === 'All Works') {
    params.delete('movement');
  }

  if (params.get('sort') === 'desc') {
    params.delete('sort');
  }

  if (artworkId && artworkId.trim() !== '') {
    params.set('artwork', artworkId.trim());
  } else {
    params.delete('artwork');
  }

  const queryString = params.toString();
  return queryString
    ? `${window.location.pathname}?${queryString}`
    : window.location.pathname;
}

/**
 * Client hook for managing fullscreen artwork lightbox modal state:
 * - Active modal index and circular navigation across artworks
 * - Persistent curatorial plaque collapse state
 * - Trigger element reference tracking for accessible focus restoration
 * - Bidirectional ?artwork=<id> URL query parameter and browser history synchronization
 */
export function useArtworkLightbox(
  artworks: gallery.IArtwork[] = [],
  options?: UseArtworkLightboxOptions
): UseArtworkLightboxReturn {
  try {
    const [activeIndex, setActiveIndex] = useState<number | null>(null);
    const [isPlaqueCollapsed, setIsPlaqueCollapsed] = useState<boolean>(false);
    const triggerElementRef = useRef<HTMLElement | null>(null);

    const catalogArtworks = options?.catalogArtworks;
    const isCatalogReady = options?.isCatalogReady;
    const onClearMovementFilter = options?.onClearMovementFilter;
    const onClearMovementFilterRef = useRef(onClearMovementFilter);

    useEffect(() => {
      onClearMovementFilterRef.current = onClearMovementFilter;
    }, [onClearMovementFilter]);

    useEffect(() => {
      if (typeof window === 'undefined') {
        return;
      }

      const syncFromLocation = (isPopState: boolean) => {
        const params = new URLSearchParams(window.location.search);

        if (!params.has('artwork')) {
          if (isPopState) {
            setActiveIndex(null);
            if (
              triggerElementRef.current &&
              typeof triggerElementRef.current.focus === 'function'
            ) {
              triggerElementRef.current.focus();
            }
          }
          return;
        }

        const targetId = (params.get('artwork') ?? '').trim();
        const matchIndex = targetId
          ? artworks.findIndex((a) => a.id === targetId)
          : -1;

        if (matchIndex !== -1) {
          setActiveIndex((prev) => {
            if (prev === null) {
              setIsPlaqueCollapsed(false);
            }
            return matchIndex;
          });
          return;
        }

        if (
          targetId !== '' &&
          catalogArtworks?.some((a) => a.id === targetId)
        ) {
          const nextUrl = buildLightboxUrl(targetId, { clearMovement: true });
          window.history.replaceState(null, '', nextUrl);
          onClearMovementFilterRef.current?.();
          return;
        }

        const ready = isCatalogReady ?? artworks.length > 0;
        if (ready) {
          setActiveIndex(null);
          const cleanUrl = buildLightboxUrl(null);
          window.history.replaceState(null, '', cleanUrl);
        }
      };

      syncFromLocation(false);

      const handlePopState = () => {
        syncFromLocation(true);
      };

      window.addEventListener('popstate', handlePopState);
      return () => {
        window.removeEventListener('popstate', handlePopState);
      };
    }, [artworks, catalogArtworks, isCatalogReady]);

    const openLightbox = useCallback(
      (index: number, triggerEl?: HTMLElement | null) => {
        if (!artworks || artworks.length === 0) {
          return;
        }
        if (index >= 0 && index < artworks.length) {
          setActiveIndex(index);
          triggerElementRef.current = triggerEl ?? null;
          const targetId = artworks[index]?.id?.trim() ?? '';
          if (targetId && typeof window !== 'undefined') {
            const nextUrl = buildLightboxUrl(targetId);
            window.history.pushState(null, '', nextUrl);
          }
        }
      },
      [artworks]
    );

    const closeLightbox = useCallback(() => {
      setActiveIndex(null);
      if (triggerElementRef.current && typeof triggerElementRef.current.focus === 'function') {
        triggerElementRef.current.focus();
      }
      if (typeof window !== 'undefined') {
        const cleanUrl = buildLightboxUrl(null);
        window.history.pushState(null, '', cleanUrl);
      }
    }, []);

    const nextArtwork = useCallback(() => {
      setActiveIndex((prev) => {
        if (prev === null || !artworks || artworks.length === 0) {
          return null;
        }
        const nextIndex = (prev + 1) % artworks.length;
        const targetId = artworks[nextIndex]?.id?.trim() ?? '';
        if (nextIndex !== prev && targetId && typeof window !== 'undefined') {
          const nextUrl = buildLightboxUrl(targetId);
          window.history.replaceState(null, '', nextUrl);
        }
        return nextIndex;
      });
    }, [artworks]);

    const prevArtwork = useCallback(() => {
      setActiveIndex((prev) => {
        if (prev === null || !artworks || artworks.length === 0) {
          return null;
        }
        const nextIndex = (prev - 1 + artworks.length) % artworks.length;
        const targetId = artworks[nextIndex]?.id?.trim() ?? '';
        if (nextIndex !== prev && targetId && typeof window !== 'undefined') {
          const nextUrl = buildLightboxUrl(targetId);
          window.history.replaceState(null, '', nextUrl);
        }
        return nextIndex;
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
  } catch {
    return {
      activeIndex: null,
      isPlaqueCollapsed: false,
      triggerElementRef: { current: null },
      openLightbox: () => {},
      closeLightbox: () => {},
      nextArtwork: () => {},
      prevArtwork: () => {},
      togglePlaque: () => {},
    };
  }
}
