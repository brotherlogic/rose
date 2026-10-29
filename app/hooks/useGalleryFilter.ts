'use client';

import { useState, useEffect, useCallback } from 'react';

export type SortDirection = 'desc' | 'asc';

export interface UseGalleryFilterReturn {
  selectedMovement: string | null;
  sortDirection: SortDirection;
  setMovement: (movement: string | null) => void;
  setSortDirection: (sort: SortDirection) => void;
  resetFilter: () => void;
  clearMovementFilter?: () => void;
}

/**
 * Parses movement and sort query parameters from a URL search string.
 */
function parseUrlParams(search: string): {
  movement: string | null;
  sort: SortDirection;
} {
  try {
    const params = new URLSearchParams(search);
    const m = params.get('movement');
    const movement = m && m.trim() !== '' && m !== 'All Works' ? m : null;
    const s = params.get('sort');
    const sort: SortDirection = s === 'asc' ? 'asc' : 'desc';
    return { movement, sort };
  } catch {
    return { movement: null, sort: 'desc' };
  }
}

/**
 * Client hook for managing exhibition movement filtering and chronological sort state
 * with bidirectional URL search parameter synchronization and browser history support.
 *
 * Ensures full compatibility with Next.js App Router and static export builds.
 */
export function useGalleryFilter(): UseGalleryFilterReturn {
  const [selectedMovement, setSelectedMovement] = useState<string | null>(null);
  const [sortDirection, setSortDirectionState] = useState<SortDirection>('desc');

  // Synchronize state from URL search params on mount and listen to popstate events
  useEffect(() => {
    if (typeof window === 'undefined') return;

    const syncFromLocation = () => {
      const { movement, sort } = parseUrlParams(window.location.search);
      setSelectedMovement(movement);
      setSortDirectionState(sort);
    };

    // Synchronize initial state on mount
    syncFromLocation();

    window.addEventListener('popstate', syncFromLocation);
    return () => {
      window.removeEventListener('popstate', syncFromLocation);
    };
  }, []);

  const setMovement = useCallback((movement: string | null) => {
    setSelectedMovement((prevMovement) => {
      const nextMovement =
        movement === null || movement === 'All Works' || movement === prevMovement
          ? null
          : movement;

      if (typeof window !== 'undefined') {
        const params = new URLSearchParams(window.location.search);
        if (nextMovement && nextMovement !== 'All Works') {
          params.set('movement', nextMovement);
        } else {
          params.delete('movement');
        }

        // Maintain canonical clean URL: omit default sort if desc
        if (params.get('sort') === 'desc') {
          params.delete('sort');
        }

        const queryString = params.toString();
        const cleanUrl = queryString
          ? `${window.location.pathname}?${queryString}`
          : window.location.pathname;

        window.history.pushState(null, '', cleanUrl);
      }

      return nextMovement;
    });
  }, []);

  const setSortDirection = useCallback((sort: SortDirection) => {
    const nextSort: SortDirection = sort === 'asc' ? 'asc' : 'desc';
    setSortDirectionState(nextSort);

    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      if (nextSort === 'asc') {
        params.set('sort', 'asc');
      } else {
        params.delete('sort');
      }

      // Maintain canonical clean URL: omit movement if All Works
      if (params.get('movement') === 'All Works') {
        params.delete('movement');
      }

      const queryString = params.toString();
      const cleanUrl = queryString
        ? `${window.location.pathname}?${queryString}`
        : window.location.pathname;

      window.history.pushState(null, '', cleanUrl);
    }
  }, []);

  const resetFilter = useCallback(() => {
    setSelectedMovement(null);
    setSortDirectionState('desc');

    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      params.delete('movement');
      params.delete('sort');

      const queryString = params.toString();
      const cleanUrl = queryString
        ? `${window.location.pathname}?${queryString}`
        : window.location.pathname;

      window.history.pushState(null, '', cleanUrl);
    }
  }, []);

  const clearMovementFilter = useCallback(() => {
    setSelectedMovement(null);

    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      if (params.has('movement') || params.get('sort') === 'desc') {
        params.delete('movement');
        if (params.get('sort') === 'desc') {
          params.delete('sort');
        }

        const queryString = params.toString();
        const cleanUrl = queryString
          ? `${window.location.pathname}?${queryString}`
          : window.location.pathname;

        window.history.replaceState(null, '', cleanUrl);
      }
    }
  }, []);

  return {
    selectedMovement,
    sortDirection,
    setMovement,
    setSortDirection,
    resetFilter,
    clearMovementFilter,
  };
}
