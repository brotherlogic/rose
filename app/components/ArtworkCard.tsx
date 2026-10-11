'use client';

import React, { useState } from 'react';
import { gallery } from '@/lib/proto/gallery';

export interface ArtworkCardProps {
  artwork: gallery.IArtwork;
  initialError?: boolean;
  onSelect?: (artwork: gallery.IArtwork, triggerEl: HTMLElement) => void;
}

export default function ArtworkCard({ artwork, initialError = false, onSelect }: ArtworkCardProps) {
  const [imageLoaded, setImageLoaded] = useState(initialError);
  const initialSrc = initialError
    ? '/placeholder.svg'
    : artwork.thumbnailPath || artwork.imagePath || '/placeholder.svg';
  const [imgSrc, setImgSrc] = useState<string>(initialSrc);

  const title = artwork.title || 'Untitled';
  const medium = artwork.medium || 'Mixed Media';

  const handleSelect = (triggerEl: HTMLElement) => {
    if (onSelect) {
      onSelect(artwork, triggerEl);
    }
  };

  return (
    <article className="group flex flex-col w-full">
      {/* Interactive trigger button wrapping image container */}
      <button
        type="button"
        aria-haspopup="dialog"
        aria-label="View artwork in fullscreen"
        onClick={(e) => handleSelect(e.currentTarget)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            handleSelect(e.currentTarget);
          }
        }}
        className="w-full text-left p-0 border-0 bg-transparent cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--border-focus)] rounded-sm"
      >
        {/* Fixed aspect ratio container ensuring zero Cumulative Layout Shift (CLS) */}
        <div className="relative aspect-square w-full overflow-hidden border border-[var(--border-subtle)] bg-[var(--bg-secondary)] transition-colors group-hover:border-[var(--border-focus)]">
          {/* Background skeleton shimmer placeholder while image loads */}
          {!imageLoaded && (
            <div
              className="absolute inset-0 animate-pulse bg-gradient-to-r from-[var(--bg-secondary)] via-[var(--border-subtle)] to-[var(--bg-secondary)]"
              aria-hidden="true"
            />
          )}

          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={imgSrc}
            alt={title}
            loading="lazy"
            onLoad={() => setImageLoaded(true)}
            onError={() => {
              if (imgSrc !== '/placeholder.svg') {
                setImgSrc('/placeholder.svg');
              }
              setImageLoaded(true);
            }}
            className={`w-full h-full object-cover transition-opacity duration-500 ease-in-out ${
              imageLoaded ? 'opacity-100' : 'opacity-0'
            }`}
          />
        </div>
      </button>

      {/* Archival typography */}
      <div className="pt-3 pb-2 flex flex-col gap-1">
        <h3
          className="font-[family-name:var(--font-serif)] text-base truncate text-[var(--text-primary)]"
          title={title}
        >
          {title}
        </h3>
        <p
          className="font-mono text-xs text-[var(--text-secondary)] uppercase tracking-wider truncate"
          title={medium}
        >
          {medium}
        </p>
      </div>
    </article>
  );
}
