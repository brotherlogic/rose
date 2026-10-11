'use client';

import React from 'react';

export interface ExhibitionFilterBarProps {
  movements: { name: string; count: number }[];
  totalCount: number;
  selectedMovement: string | null;
  sortDirection: 'desc' | 'asc';
  onSelectMovement: (movement: string | null) => void;
  onToggleSort: (sort: 'desc' | 'asc') => void;
}

export default function ExhibitionFilterBar({
  movements,
  totalCount,
  selectedMovement,
  sortDirection,
  onSelectMovement,
  onToggleSort,
}: ExhibitionFilterBarProps) {
  const isAllWorksActive = selectedMovement === null;

  const handleAllWorksClick = () => {
    onSelectMovement(null);
  };

  const handleMovementClick = (movementName: string) => {
    if (selectedMovement === movementName) {
      onSelectMovement(null);
    } else {
      onSelectMovement(movementName);
    }
  };

  const handleSortToggle = () => {
    onToggleSort(sortDirection === 'desc' ? 'asc' : 'desc');
  };

  return (
    <nav
      aria-label="Exhibition period filtering"
      role="tablist"
      className="w-full flex flex-col md:flex-row md:items-center justify-between gap-4 py-4 border-b border-[var(--border-subtle)]"
    >
      {/* Horizontal scroll container with hidden scrollbars for mobile viewports */}
      <div className="flex items-center gap-2 overflow-x-auto scrollbar-none [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden py-1">
        {/* All Works Pill */}
        <button
          type="button"
          role="tab"
          aria-selected={isAllWorksActive}
          onClick={handleAllWorksClick}
          className={`shrink-0 px-3.5 py-1.5 rounded-full text-xs font-mono uppercase tracking-wider border transition-colors cursor-pointer ${
            isAllWorksActive
              ? 'border-[var(--text-primary)] text-[var(--text-primary)] bg-[var(--bg-secondary)]'
              : 'border-[var(--border-subtle)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:border-[var(--border-focus)] bg-transparent'
          }`}
        >
          All Works ({totalCount})
        </button>

        {/* Movement Pills */}
        {movements.map((movement) => {
          const isActive = selectedMovement === movement.name;
          return (
            <button
              key={movement.name}
              type="button"
              role="tab"
              aria-selected={isActive}
              onClick={() => handleMovementClick(movement.name)}
              className={`shrink-0 px-3.5 py-1.5 rounded-full text-xs font-mono uppercase tracking-wider border transition-colors cursor-pointer ${
                isActive
                  ? 'border-[var(--text-primary)] text-[var(--text-primary)] bg-[var(--bg-secondary)]'
                  : 'border-[var(--border-subtle)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:border-[var(--border-focus)] bg-transparent'
              }`}
            >
              {movement.name} ({movement.count})
            </button>
          );
        })}
      </div>

      {/* Chronological Sort Toggle */}
      <div className="shrink-0 flex items-center">
        <button
          type="button"
          onClick={handleSortToggle}
          aria-label={`Sort chronological order: currently ${sortDirection === 'desc' ? 'Newest First' : 'Earliest First'}`}
          className="inline-flex items-center gap-2 px-3 py-1.5 text-xs font-mono uppercase tracking-wider text-[var(--text-secondary)] hover:text-[var(--text-primary)] border border-[var(--border-subtle)] hover:border-[var(--border-focus)] transition-colors cursor-pointer rounded"
        >
          <span className="text-[10px] text-[var(--text-secondary)]" aria-hidden="true">
            {sortDirection === 'desc' ? '↓' : '↑'}
          </span>
          <span>{sortDirection === 'desc' ? 'Newest First' : 'Earliest First'}</span>
        </button>
      </div>
    </nav>
  );
}
