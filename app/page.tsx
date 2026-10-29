import React, { Suspense } from 'react';
import { getArtworks } from '@/lib/nfs';
import EditorialStatement from './components/EditorialStatement';
import RepresentativeArtwork from './components/RepresentativeArtwork';
import EditorialBiography from './components/EditorialBiography';
import CuratedGalleryCollection from './components/CuratedGalleryCollection';

export default function Home() {
  const artworks = getArtworks();

  return (
    <div className="flex flex-col w-full">
      {/* Curatorial Statement Section (#statement) */}
      <EditorialStatement />

      {/* Featured Representative Artwork Section (#featured-artwork) */}
      <RepresentativeArtwork artworks={artworks} />

      {/* Historical Movements Taxonomy Section (#biography) */}
      <EditorialBiography artworks={artworks} />

      {/* Permanent Collection Gallery Section (#collection) */}
      <section
        id="collection"
        aria-labelledby="collection-title"
        className="w-full py-16 md:py-24 bg-[var(--bg-primary)] transition-colors"
      >
        <div className="container mx-auto px-4 md:px-8 max-w-6xl">
          <header className="mb-12">
            <p className="text-xs uppercase tracking-widest text-[var(--text-secondary)] mb-2 font-mono">
              Archival Catalog
            </p>
            <h2
              id="collection-title"
              className="text-2xl md:text-3xl font-[family-name:var(--font-serif)] font-normal text-[var(--text-primary)] tracking-wide"
            >
              The Permanent Collection
            </h2>
            <p className="text-sm text-[var(--text-secondary)] mt-3 max-w-2xl leading-relaxed">
              Systematically documented compositions and physical artifacts preserved in the retrospective archive.
            </p>
          </header>

          <Suspense fallback={null}>
            <CuratedGalleryCollection artworks={artworks} />
          </Suspense>
        </div>
      </section>
    </div>
  );
}
