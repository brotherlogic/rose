'use client';

import React, { useState, useEffect, useRef } from 'react';
import ArtworkCard from './ArtworkCard';
import { gallery } from '@/lib/proto/gallery';

export interface VirtualizedGalleryGridProps {
  artworks: gallery.IArtwork[];
  columnCount?: number;
  rowHeight?: number;
  gap?: number;
  overscan?: number;
  startIndex?: number;
  endIndex?: number;
  initialScrollTop?: number;
  viewportHeight?: number;
  onSelectArtwork?: (artwork: gallery.IArtwork, index: number, triggerEl: HTMLElement) => void;
}

/**
 * Breakpoint-based column calculation:
 * • < 640px : 1 column
 * • 640px - 1023px : 2 columns
 * • 1024px - 1439px : 3 columns
 * • >= 1440px : 4 columns
 */
export function getColumnCountForWidth(width: number): number {
  if (width < 640) return 1;
  if (width < 1024) return 2;
  if (width < 1440) return 3;
  return 4;
}

/**
 * Total container height formula:
 * totalRows * rowHeight + (totalRows - 1) * gap
 */
export function calculateGridTotalHeight(
  totalArtworks: number,
  columnCount: number,
  rowHeight: number,
  gap: number
): number {
  const totalRows = Math.ceil(totalArtworks / columnCount);
  if (totalRows <= 0) return 0;
  return totalRows * rowHeight + (totalRows - 1) * gap;
}

export default function VirtualizedGalleryGrid({
  artworks,
  columnCount: propColumnCount,
  rowHeight = 460,
  gap = 24,
  overscan = 2,
  startIndex: propStartIndex,
  endIndex: propEndIndex,
  initialScrollTop = 0,
  viewportHeight: propViewportHeight,
  onSelectArtwork,
}: VirtualizedGalleryGridProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  // Responsive column count calculation
  const [columnCount, setColumnCount] = useState<number>(() => {
    if (propColumnCount !== undefined) return propColumnCount;
    if (typeof window !== 'undefined') {
      return getColumnCountForWidth(window.innerWidth);
    }
    return 4;
  });

  const activeColumnCount = propColumnCount !== undefined ? propColumnCount : columnCount;
  const totalRows = Math.ceil(artworks.length / activeColumnCount);

  // Windowing indices calculation
  const [windowRange, setWindowRange] = useState<{ start: number; end: number }>(() => {
    if (propStartIndex !== undefined && propEndIndex !== undefined) {
      return { start: propStartIndex, end: propEndIndex };
    }
    const vpHeight = propViewportHeight ?? (typeof window !== 'undefined' ? window.innerHeight : 800);
    const effectiveRowHeight = rowHeight + gap;
    const visibleStartRow = Math.floor(initialScrollTop / effectiveRowHeight);
    const visibleEndRow = Math.ceil((initialScrollTop + vpHeight) / effectiveRowHeight);
    return {
      start: Math.max(0, visibleStartRow - overscan),
      end: Math.min(totalRows, visibleEndRow + overscan),
    };
  });

  // Observe container / window width changes to adjust columns dynamically
  useEffect(() => {
    if (propColumnCount !== undefined) return;
    if (typeof window === 'undefined') return;

    const handleWidth = (width: number) => {
      if (width > 0) {
        setColumnCount(getColumnCountForWidth(width));
      }
    };

    if (typeof ResizeObserver !== 'undefined' && containerRef.current) {
      const resizeObserver = new ResizeObserver((entries) => {
        for (const entry of entries) {
          const width = entry.contentRect.width;
          handleWidth(width);
        }
      });
      resizeObserver.observe(containerRef.current);
      return () => resizeObserver.disconnect();
    } else {
      const onResize = () => handleWidth(window.innerWidth);
      window.addEventListener('resize', onResize);
      return () => window.removeEventListener('resize', onResize);
    }
  }, [propColumnCount]);

  // Track viewport scroll position and container position with requestAnimationFrame throttling
  useEffect(() => {
    if (propStartIndex !== undefined && propEndIndex !== undefined) return;
    if (typeof window === 'undefined') return;

    let rafId: number | null = null;

    const updateWindowSlice = () => {
      if (!containerRef.current) return;
      const rect = containerRef.current.getBoundingClientRect();
      const vpHeight = propViewportHeight ?? window.innerHeight;
      const scrollTop = Math.max(0, -rect.top);

      const effectiveRowHeight = rowHeight + gap;
      const visibleStartRow = Math.floor(scrollTop / effectiveRowHeight);
      const visibleEndRow = Math.ceil((scrollTop + vpHeight) / effectiveRowHeight);

      const newStart = Math.max(0, visibleStartRow - overscan);
      const newEnd = Math.min(totalRows, visibleEndRow + overscan);

      setWindowRange({ start: newStart, end: newEnd });
    };

    const handleScroll = () => {
      if (rafId !== null) return;
      rafId = window.requestAnimationFrame(() => {
        updateWindowSlice();
        rafId = null;
      });
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    window.addEventListener('resize', handleScroll);

    updateWindowSlice();

    return () => {
      window.removeEventListener('scroll', handleScroll);
      window.removeEventListener('resize', handleScroll);
      if (rafId !== null) {
        window.cancelAnimationFrame(rafId);
      }
    };
  }, [totalRows, rowHeight, gap, overscan, propStartIndex, propEndIndex, propViewportHeight]);

  // Empty state handling
  if (artworks.length === 0) {
    return (
      <div className="p-12 md:p-16 border border-[var(--border-subtle)] bg-[var(--bg-secondary)] text-center transition-colors">
        <p className="text-xs uppercase tracking-widest font-mono text-[var(--text-secondary)] mb-3">
          Archival Status
        </p>
        <h3 className="text-xl md:text-2xl font-[family-name:var(--font-serif)] text-[var(--text-primary)] mb-3">
          No Works Currently Cataloged
        </h3>
        <p className="text-sm text-[var(--text-secondary)] max-w-md mx-auto leading-relaxed">
          Waiting for archival syncer ingestion. Cataloged works will appear here automatically following automated synchronization.
        </p>
      </div>
    );
  }

  const effectiveStartIndex = propStartIndex !== undefined ? propStartIndex : windowRange.start;
  const effectiveEndIndex = propEndIndex !== undefined ? propEndIndex : windowRange.end;

  const totalHeight = calculateGridTotalHeight(artworks.length, activeColumnCount, rowHeight, gap);
  const startOffset = effectiveStartIndex * (rowHeight + gap);

  // Partition artworks into rows
  const visibleRows: gallery.IArtwork[][] = [];
  for (let r = effectiveStartIndex; r < effectiveEndIndex; r++) {
    const startItemIdx = r * activeColumnCount;
    const endItemIdx = Math.min(artworks.length, startItemIdx + activeColumnCount);
    if (startItemIdx < artworks.length) {
      visibleRows.push(artworks.slice(startItemIdx, endItemIdx));
    }
  }

  return (
    <div
      ref={containerRef}
      className="relative w-full"
      style={{ height: `${totalHeight}px` }}
    >
      <div
        style={{
          transform: `translateY(${startOffset}px)`,
          position: 'absolute',
          top: 0,
          left: 0,
          width: '100%',
        }}
        className="flex flex-col"
      >
        {visibleRows.map((row, index) => {
          const rowIndex = effectiveStartIndex + index;
          return (
            <div
              key={rowIndex}
              data-row-index={rowIndex}
              style={{
                marginBottom: index < visibleRows.length - 1 ? `${gap}px` : 0,
              }}
              className={`grid gap-6 md:gap-8 ${
                activeColumnCount === 1
                  ? 'grid-cols-1'
                  : activeColumnCount === 2
                  ? 'grid-cols-2'
                  : activeColumnCount === 3
                  ? 'grid-cols-3'
                  : 'grid-cols-4'
              }`}
            >
              {row.map((artwork, colIndex) => {
                const itemIndex = rowIndex * activeColumnCount + colIndex;
                return (
                  <ArtworkCard
                    key={artwork.id}
                    artwork={artwork}
                    onSelect={
                      onSelectArtwork
                        ? (art, triggerEl) => onSelectArtwork(art, itemIndex, triggerEl)
                        : undefined
                    }
                  />
                );
              })}
            </div>
          );
        })}
      </div>
    </div>
  );
}
