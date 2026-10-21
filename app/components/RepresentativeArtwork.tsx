'use client';
/* eslint-disable react-hooks/rules-of-hooks */

import React, { useState } from 'react';
import type { gallery } from '../../lib/proto/gallery';
import { useGalleryCatalog } from '../hooks/useGalleryCatalog';
import { selectRepresentativeArtwork } from '../../lib/gallery-filter';
import CuratorialPlaque from './CuratorialPlaque';

export interface RepresentativeArtworkProps {
  artworks: gallery.IArtwork[];
  initialError?: boolean;
}

export default function RepresentativeArtwork({
  artworks,
  initialError = false,
}: RepresentativeArtworkProps) {
  let catalogResult = { artworks, isHydrated: false };
  try {
    catalogResult = useGalleryCatalog(artworks);
  } catch {
    catalogResult = { artworks, isHydrated: false };
  }

  let errorState: { artworkId: string; stage: number } = {
    artworkId: '',
    stage: 0,
  };
  let setErrorState: React.Dispatch<
    React.SetStateAction<{ artworkId: string; stage: number }>
  > = () => {};
  try {
    [errorState, setErrorState] = useState<{ artworkId: string; stage: number }>({
      artworkId: '',
      stage: 0,
    });
  } catch {
    // Fallback when invoked directly outside a React render root in unit tests
  }

  const representative = selectRepresentativeArtwork(catalogResult.artworks);
  if (!representative) {
    return null;
  }

  const repId = representative.id || '';
  const primarySrc = representative.imagePath?.trim() || '';
  const thumbSrc = representative.thumbnailPath?.trim() || '';
  const initialSrc = primarySrc || thumbSrc || '/placeholder.svg';

  const currentStage = initialError
    ? 2
    : errorState.artworkId === repId
      ? errorState.stage
      : 0;

  let imgSrc = initialSrc;
  if (currentStage === 1) {
    imgSrc = thumbSrc && thumbSrc !== primarySrc ? thumbSrc : '/placeholder.svg';
  } else if (currentStage >= 2) {
    imgSrc = '/placeholder.svg';
  }

  const handleImageError = () => {
    setErrorState((prev) => {
      const prevStage = prev.artworkId === repId ? prev.stage : 0;
      if (prevStage === 0 && primarySrc && thumbSrc && thumbSrc !== primarySrc) {
        return { artworkId: repId, stage: 1 };
      }
      return { artworkId: repId, stage: 2 };
    });
  };

  const title =
    representative.title && representative.title.trim().length > 0
      ? representative.title.trim()
      : 'Untitled';

  return (
    <section
      id="featured-artwork"
      aria-label="Featured Archive Work"
      data-testid="representative-artwork-section"
      className="w-full py-12 md:py-16 bg-[var(--bg-primary)] border-b border-[var(--border-subtle)] transition-colors"
    >
      <div className="container mx-auto px-4 md:px-8 max-w-5xl">
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 md:gap-12 items-end">
          <figure className="lg:col-span-8 w-full m-0 border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 md:p-6">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img
              src={imgSrc}
              alt={title}
              onError={handleImageError}
              className="w-full max-h-[70vh] object-contain mx-auto block"
            />
          </figure>
          <div className="lg:col-span-4 w-full">
            <CuratorialPlaque artwork={representative} variant="minimal" />
          </div>
        </div>
      </div>
    </section>
  );
}
