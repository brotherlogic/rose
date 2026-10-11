'use client';

import React from 'react';
import { gallery } from '@/lib/proto/gallery';

export interface CuratorialPlaqueProps {
  artwork: gallery.IArtwork;
  isCollapsed: boolean;
  onToggleCollapse: () => void;
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

function formatCreationYear(timestamp?: unknown): string {
  if (timestamp === null || timestamp === undefined) {
    return 'Undated';
  }

  let tsNum: number;
  if (hasToNumber(timestamp)) {
    tsNum = timestamp.toNumber();
  } else {
    tsNum = Number(timestamp);
  }

  if (!tsNum || isNaN(tsNum) || tsNum <= 0) {
    return 'Undated';
  }

  // Handle epoch in seconds vs milliseconds
  const dateMs = tsNum < 1e11 ? tsNum * 1000 : tsNum;
  const date = new Date(dateMs);

  if (isNaN(date.getTime())) {
    return 'Undated';
  }

  return String(date.getUTCFullYear());
}

export default function CuratorialPlaque({
  artwork,
  isCollapsed,
  onToggleCollapse,
}: CuratorialPlaqueProps) {
  const title = artwork.title && artwork.title.trim().length > 0 ? artwork.title.trim() : 'Untitled';
  const year = formatCreationYear(artwork.timestamp);
  const medium =
    artwork.medium && artwork.medium.trim().length > 0 ? artwork.medium.trim() : 'Mixed Media';
  const movement =
    artwork.themeId && artwork.themeId.trim().length > 0
      ? artwork.themeId.trim()
      : 'Early Works';
  const critique =
    artwork.description && artwork.description.trim().length > 0
      ? artwork.description.trim()
      : 'Catalog description pending curatorial annotation.';

  if (isCollapsed) {
    return (
      <aside
        aria-label="Museum curatorial plaque"
        className="relative inline-flex items-center transition-all duration-300 ease-in-out"
      >
        <button
          type="button"
          onClick={onToggleCollapse}
          aria-expanded={false}
          aria-label="Expand curatorial plaque"
          className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-[var(--bg-primary)]/80 backdrop-blur-md border border-[var(--border-subtle)] hover:border-[var(--border-focus)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors cursor-pointer text-xs font-mono uppercase tracking-wider shadow-lg"
        >
          <span
            id="curatorial-plaque-title"
            className="font-[family-name:var(--font-serif)] italic text-sm text-[var(--text-primary)] capitalize"
          >
            {title}
          </span>
          <span className="text-[var(--text-secondary)]" aria-hidden="true">
            ·
          </span>
          <span>{year}</span>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 20 20"
            fill="currentColor"
            className="w-4 h-4 shrink-0 text-[var(--text-secondary)]"
            aria-hidden="true"
          >
            <path
              fillRule="evenodd"
              d="M14.78 11.78a.75.75 0 0 1-1.06 0L10 8.06l-3.72 3.72a.75.75 0 0 1-1.06-1.06l4.25-4.25a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06Z"
              clipRule="evenodd"
            />
          </svg>
        </button>
      </aside>
    );
  }

  return (
    <aside
      aria-label="Museum curatorial plaque"
      className="relative w-full max-w-xl bg-[var(--bg-primary)]/90 backdrop-blur-md border border-[var(--border-subtle)] p-5 md:p-6 rounded-lg shadow-2xl transition-all duration-300 ease-in-out text-[var(--text-primary)]"
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex flex-col gap-1 min-w-0">
          <div className="flex flex-wrap items-baseline gap-2">
            <h2
              id="curatorial-plaque-title"
              className="font-[family-name:var(--font-serif)] italic text-lg md:text-xl text-[var(--text-primary)]"
            >
              {title}
            </h2>
            <span className="font-mono text-xs md:text-sm text-[var(--text-secondary)]">
              ({year})
            </span>
          </div>
          <p className="font-mono text-xs uppercase tracking-wider text-[var(--text-secondary)]">
            {medium}
          </p>
        </div>

        <button
          type="button"
          onClick={onToggleCollapse}
          aria-expanded={true}
          aria-label="Collapse curatorial plaque"
          className="shrink-0 p-1.5 rounded text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-secondary)] border border-[var(--border-subtle)] transition-colors cursor-pointer"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 20 20"
            fill="currentColor"
            className="w-4 h-4"
            aria-hidden="true"
          >
            <path
              fillRule="evenodd"
              d="M5.22 8.22a.75.75 0 0 1 1.06 0L10 11.94l3.72-3.72a.75.75 0 1 1 1.06 1.06l-4.25 4.25a.75.75 0 0 1-1.06 0L5.22 9.28a.75.75 0 0 1 0-1.06Z"
              clipRule="evenodd"
            />
          </svg>
        </button>
      </div>

      <div className="mt-3">
        <span className="inline-block font-mono text-xs uppercase tracking-wider border border-[var(--border-subtle)] bg-[var(--bg-secondary)] text-[var(--text-secondary)] px-2.5 py-0.5 rounded-full">
          {movement}
        </span>
      </div>

      <div className="mt-4 pt-3 border-t border-[var(--border-subtle)]/60">
        <p className="text-sm md:text-base text-[var(--text-primary)] font-serif leading-relaxed italic">
          {critique}
        </p>
      </div>
    </aside>
  );
}
