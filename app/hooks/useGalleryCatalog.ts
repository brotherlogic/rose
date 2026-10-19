'use client';

import { useState, useEffect } from 'react';
import type { gallery } from '../../lib/proto/gallery';

export const CATALOG_ENDPOINT = '/data/catalog.json';

export interface UseGalleryCatalogResult {
  artworks: gallery.IArtwork[];
  isHydrated: boolean;
}

interface HasToNumber {
  toNumber: () => number;
}

function hasToNumber(val: unknown): val is HasToNumber {
  return (
    typeof val === 'object' &&
    val !== null &&
    'toNumber' in val &&
    typeof (val as HasToNumber).toNumber === 'function'
  );
}

function coerceTimestamp(rawTimestamp: unknown): number {
  if (rawTimestamp === null || rawTimestamp === undefined || rawTimestamp === '') {
    return 0;
  }
  if (typeof rawTimestamp === 'number') {
    return Number.isFinite(rawTimestamp) ? rawTimestamp : 0;
  }
  if (hasToNumber(rawTimestamp)) {
    const converted = rawTimestamp.toNumber();
    return Number.isFinite(converted) ? converted : 0;
  }
  const num = Number(rawTimestamp);
  return Number.isFinite(num) ? num : 0;
}

/**
 * Normalizes media asset paths from catalog JSON payloads.
 * - Prefixes relative `images/...` and `thumbnails/...` paths with `/data/`.
 * - Normalizes bare `<id>.jpg` image filenames to `/data/images/<id>.jpg`
 *   and bare `<id>.webp` thumbnail filenames to `/data/thumbnails/<id>.webp`.
 * - Preserves paths already starting with `/` or `http(s)://`.
 */
export function normalizeMediaPath(rawPath: unknown): string {
  if (typeof rawPath !== 'string') {
    return '';
  }
  const trimmed = rawPath.trim();
  if (!trimmed) {
    return '';
  }
  if (trimmed.startsWith('images/') || trimmed.startsWith('thumbnails/')) {
    return `/data/${trimmed}`;
  }
  if (
    trimmed.startsWith('/') ||
    trimmed.startsWith('http://') ||
    trimmed.startsWith('https://')
  ) {
    return trimmed;
  }
  if (!trimmed.includes('/')) {
    const lower = trimmed.toLowerCase();
    if (lower.endsWith('.jpg') || lower.endsWith('.jpeg')) {
      return `/data/images/${trimmed}`;
    }
    if (lower.endsWith('.webp')) {
      return `/data/thumbnails/${trimmed}`;
    }
  }
  return trimmed;
}

/**
 * Normalizes a raw catalog JSON entry (supporting both snake_case and camelCase keys)
 * into a canonical `gallery.IArtwork` object with curatorial defaults consistent with `lib/nfs.ts`.
 * Returns `null` if the entry is not an object or lacks a non-empty string `id`.
 */
export function normalizeCatalogArtwork(raw: unknown): gallery.IArtwork | null {
  if (typeof raw !== 'object' || raw === null || Array.isArray(raw)) {
    return null;
  }

  const item = raw as Record<string, unknown>;
  const id = typeof item.id === 'string' ? item.id.trim() : '';
  if (!id) {
    return null;
  }

  const rawTitle = typeof item.title === 'string' ? item.title : '';
  const rawMedium = typeof item.medium === 'string' ? item.medium : '';
  const rawDescription = typeof item.description === 'string' ? item.description : '';

  const rawThemeId =
    typeof item.themeId === 'string' && item.themeId.trim() !== ''
      ? item.themeId
      : typeof item.theme_id === 'string' && item.theme_id.trim() !== ''
        ? item.theme_id
        : typeof item.theme === 'string'
          ? item.theme
          : '';

  const rawImagePath =
    typeof item.imagePath === 'string' && item.imagePath.trim() !== ''
      ? item.imagePath
      : typeof item.image_path === 'string'
        ? item.image_path
        : '';

  const rawThumbnailPath =
    typeof item.thumbnailPath === 'string' && item.thumbnailPath.trim() !== ''
      ? item.thumbnailPath
      : typeof item.thumbnail_path === 'string'
        ? item.thumbnail_path
        : '';

  const imagePath = normalizeMediaPath(rawImagePath);
  const normalizedThumbnail = normalizeMediaPath(rawThumbnailPath);
  const thumbnailPath = normalizedThumbnail || imagePath || '/placeholder.svg';

  return {
    id,
    title: rawTitle.trim() || 'Untitled',
    medium: rawMedium.trim() || 'Mixed Media',
    description: rawDescription,
    themeId: rawThemeId.trim() || '',
    timestamp: coerceTimestamp(item.timestamp),
    imagePath,
    thumbnailPath,
  };
}

/**
 * Client-side catalog hydration hook.
 * Initializes synchronously with `initialArtworks` for zero-CLS SSR/SSG rendering,
 * fetches `/data/catalog.json` on client mount, and completely replaces the baseline
 * when a valid non-empty catalog array is returned.
 */
export function useGalleryCatalog(
  initialArtworks: gallery.IArtwork[]
): UseGalleryCatalogResult {
  const [artworks, setArtworks] = useState<gallery.IArtwork[]>(initialArtworks);
  const [isHydrated, setIsHydrated] = useState<boolean>(false);

  useEffect(() => {
    if (typeof fetch !== 'function') {
      return;
    }

    const controller = new AbortController();
    let isCancelled = false;

    async function fetchCatalog() {
      try {
        const response = await fetch(CATALOG_ENDPOINT, {
          cache: 'no-store',
          signal: controller.signal,
        });

        if (!response || !response.ok) {
          return;
        }

        const payload: unknown = await response.json();
        if (!Array.isArray(payload)) {
          return;
        }

        const validArtworks: gallery.IArtwork[] = [];
        for (const entry of payload) {
          const normalized = normalizeCatalogArtwork(entry);
          if (normalized !== null) {
            validArtworks.push(normalized);
          }
        }

        if (!isCancelled && validArtworks.length > 0) {
          setArtworks(validArtworks);
          setIsHydrated(true);
        }
      } catch {
        // Gracefully ignore AbortError, network failures, or malformed JSON
        // and retain the static initialArtworks baseline.
      }
    }

    void fetchCatalog();

    return () => {
      isCancelled = true;
      controller.abort();
    };
  }, []);

  return {
    artworks: isHydrated ? artworks : initialArtworks,
    isHydrated,
  };
}
