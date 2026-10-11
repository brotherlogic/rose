'use client';

import React from 'react';

export interface ExhibitionEmptyStateProps {
  movementName: string;
  onReset: () => void;
}

export default function ExhibitionEmptyState({
  movementName,
  onReset,
}: ExhibitionEmptyStateProps) {
  return (
    <div
      role="status"
      className="w-full py-16 px-6 text-center border border-dashed border-[var(--border-subtle)] bg-[var(--bg-secondary)]/50 rounded-lg flex flex-col items-center justify-center gap-4 my-8"
    >
      <div className="flex flex-col gap-1 max-w-md">
        <h3 className="font-[family-name:var(--font-serif)] text-lg md:text-xl text-[var(--text-primary)]">
          {movementName}
        </h3>
        <p className="font-sans text-sm text-[var(--text-secondary)]">
          No works from this movement are currently on public exhibition.
        </p>
      </div>

      <button
        type="button"
        onClick={onReset}
        className="px-4 py-2 mt-2 text-xs font-mono uppercase tracking-widest text-[var(--text-primary)] border border-[var(--text-primary)] hover:bg-[var(--text-primary)] hover:text-[var(--bg-primary)] transition-colors cursor-pointer"
      >
        Return to All Works
      </button>
    </div>
  );
}
