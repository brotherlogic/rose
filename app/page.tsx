import React from 'react';
import { getArtworks } from '@/lib/nfs';
import EditorialStatement from './components/EditorialStatement';
import EditorialBiography from './components/EditorialBiography';

export default function Home() {
  const artworks = getArtworks();

  return (
    <div className="flex flex-col w-full">
      {/* Curatorial Statement Section (#statement) */}
      <EditorialStatement />

      {/* Historical Movements Taxonomy Section (#biography) */}
      <EditorialBiography />

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

          {artworks.length === 0 ? (
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
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 md:gap-8">
              {artworks.map((artwork) => (
                <article
                  key={artwork.id}
                  className="group p-6 bg-[var(--bg-secondary)] border border-[var(--border-subtle)] hover:border-[var(--border-focus)] transition-colors flex flex-col justify-between"
                >
                  <div>
                    <h3 className="text-lg font-[family-name:var(--font-serif)] text-[var(--text-primary)] mb-2 group-hover:underline">
                      {artwork.title}
                    </h3>
                    {artwork.description && (
                      <p className="text-sm text-[var(--text-secondary)] leading-relaxed mb-4">
                        {artwork.description}
                      </p>
                    )}
                  </div>
                  {artwork.imagePath && (
                    <div className="mt-4 overflow-hidden border border-[var(--border-subtle)]">
                      {/* eslint-disable-next-line @next/next/no-img-element */}
                      <img
                        src={artwork.imagePath}
                        alt={artwork.title || 'Artwork'}
                        className="w-full object-cover"
                      />
                    </div>
                  )}
                </article>
              ))}
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
