'use client';

import React from 'react';
import type { gallery } from '../../lib/proto/gallery';
import { useGalleryFilter } from '../hooks/useGalleryFilter';
import { aggregateMovements, filterAndSortArtworks } from '../../lib/gallery-filter';
import ExhibitionFilterBar from './ExhibitionFilterBar';
import ExhibitionEmptyState from './ExhibitionEmptyState';
import VirtualizedGalleryGrid from './VirtualizedGalleryGrid';

export interface CuratedGalleryCollectionProps {
  artworks: gallery.IArtwork[];
}

export default function CuratedGalleryCollection({
  artworks,
}: CuratedGalleryCollectionProps) {
  const totalCount = artworks.length;
  const {
    selectedMovement,
    sortDirection,
    setMovement,
    setSortDirection,
    resetFilter,
  } = useGalleryFilter();

  if (totalCount === 0) {
    return (
      <div data-testid="curated-gallery-collection" className="w-full">
        <VirtualizedGalleryGrid artworks={artworks} />
      </div>
    );
  }

  const movements = aggregateMovements(artworks);
  const filteredArtworks = filterAndSortArtworks(artworks, selectedMovement, sortDirection);

  return (
    <div data-testid="curated-gallery-collection" className="flex flex-col gap-8 w-full">
      <ExhibitionFilterBar
        movements={movements}
        totalCount={totalCount}
        selectedMovement={selectedMovement}
        sortDirection={sortDirection}
        onSelectMovement={setMovement}
        onToggleSort={setSortDirection}
      />

      {filteredArtworks.length === 0 ? (
        <ExhibitionEmptyState
          movementName={selectedMovement ?? ''}
          onReset={resetFilter}
        />
      ) : (
        <VirtualizedGalleryGrid artworks={filteredArtworks} />
      )}
    </div>
  );
}
