'use client';
/* eslint-disable react-hooks/set-state-in-effect */

import React, { useEffect, useRef, useState } from 'react';
import type { gallery } from '@/lib/proto/gallery';
import CuratorialPlaque from './CuratorialPlaque';

export interface ArtworkLightboxModalProps {
  artwork: gallery.IArtwork;
  currentIndex: number;
  totalCount: number;
  isPlaqueCollapsed: boolean;
  onTogglePlaque: () => void;
  onNext: () => void;
  onPrev: () => void;
  onClose: () => void;
  initialError?: boolean;
}

export default function ArtworkLightboxModal({
  artwork,
  currentIndex,
  totalCount,
  isPlaqueCollapsed,
  onTogglePlaque,
  onNext,
  onPrev,
  onClose,
  initialError = false,
}: ArtworkLightboxModalProps) {
  const modalRef = useRef<HTMLDivElement>(null);

  // Compute initial source with fallback hierarchy: imagePath -> thumbnailPath -> placeholder
  const getInitialSource = () => {
    if (initialError) return '/placeholder.svg';
    return artwork.imagePath || artwork.thumbnailPath || '/placeholder.svg';
  };

  const [imgSrc, setImgSrc] = useState<string>(getInitialSource);
  const [imageLoaded, setImageLoaded] = useState<boolean>(false);

  // Update image source when artwork changes during navigation
  useEffect(() => {
    if (initialError) {
      setImgSrc('/placeholder.svg');
    } else {
      setImgSrc(artwork.imagePath || artwork.thumbnailPath || '/placeholder.svg');
    }
    setImageLoaded(false);
  }, [artwork.id, artwork.imagePath, artwork.thumbnailPath, initialError]);

  // Document body scroll lock
  useEffect(() => {
    if (typeof document !== 'undefined' && document.body) {
      document.body.classList.add('overflow-hidden');
    }
    return () => {
      if (typeof document !== 'undefined' && document.body) {
        document.body.classList.remove('overflow-hidden');
      }
    };
  }, []);

  // Keyboard navigation & accessibility event listeners
  useEffect(() => {
    if (typeof window === 'undefined') return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        onClose();
        return;
      }

      if (e.key === 'ArrowLeft') {
        e.preventDefault();
        onPrev();
        return;
      }

      if (e.key === 'ArrowRight') {
        e.preventDefault();
        onNext();
        return;
      }

      if (e.key === 'Tab' && modalRef.current) {
        const focusableElements = modalRef.current.querySelectorAll<HTMLElement>(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );

        if (focusableElements.length === 0) return;

        const firstElement = focusableElements[0];
        const lastElement = focusableElements[focusableElements.length - 1];

        if (e.shiftKey) {
          if (document.activeElement === firstElement) {
            e.preventDefault();
            lastElement.focus();
          }
        } else {
          if (document.activeElement === lastElement) {
            e.preventDefault();
            firstElement.focus();
          }
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [onClose, onNext, onPrev]);

  // Auto-focus the close button or modal container on mount
  useEffect(() => {
    if (modalRef.current) {
      const closeButton = modalRef.current.querySelector<HTMLElement>('button[aria-label="Close modal"]');
      if (closeButton) {
        closeButton.focus();
      } else {
        modalRef.current.focus();
      }
    }
  }, []);

  const handleImageError = () => {
    if (artwork.imagePath && imgSrc === artwork.imagePath && artwork.thumbnailPath) {
      setImgSrc(artwork.thumbnailPath);
    } else {
      setImgSrc('/placeholder.svg');
    }
    setImageLoaded(true);
  };

  const title = artwork.title || 'Untitled';
  const displayIndex = totalCount > 0 ? currentIndex + 1 : 0;

  return (
    <div
      ref={modalRef}
      role="dialog"
      aria-modal="true"
      aria-labelledby="curatorial-plaque-title"
      data-testid="lightbox-backdrop"
      onClick={(e) => {
        if (e.target === e.currentTarget) {
          onClose();
        }
      }}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/90 backdrop-blur-sm p-4 md:p-8 select-none"
    >
      {/* Top Header Bar: Counter & Close Button */}
      <div className="absolute top-4 right-4 left-4 z-20 flex items-center justify-between pointer-events-none">
        <div className="pointer-events-auto px-3 py-1.5 rounded-full bg-[var(--bg-primary)]/80 backdrop-blur-md border border-[var(--border-subtle)] text-[var(--text-secondary)] font-mono text-xs uppercase tracking-wider">
          {displayIndex} / {totalCount}
        </div>

        <button
          type="button"
          aria-label="Close modal"
          onClick={onClose}
          className="pointer-events-auto p-2 rounded-full bg-[var(--bg-primary)]/80 backdrop-blur-md border border-[var(--border-subtle)] hover:border-[var(--border-focus)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors cursor-pointer shadow-lg"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 20 20"
            fill="currentColor"
            className="w-5 h-5"
            aria-hidden="true"
          >
            <path
              d="M6.28 5.22a.75.75 0 0 0-1.06 1.06L8.94 10l-3.72 3.72a.75.75 0 1 0 1.06 1.06L10 11.06l3.72 3.72a.75.75 0 1 0 1.06-1.06L11.06 10l3.72-3.72a.75.75 0 0 0-1.06-1.06L10 8.94 6.28 5.22Z"
            />
          </svg>
        </button>
      </div>

      {/* Navigation Controls: Previous Button */}
      <button
        type="button"
        aria-label="Previous artwork"
        onClick={onPrev}
        className="absolute left-4 top-1/2 -translate-y-1/2 z-20 p-2.5 rounded-full bg-[var(--bg-primary)]/80 backdrop-blur-md border border-[var(--border-subtle)] hover:border-[var(--border-focus)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-all cursor-pointer shadow-xl hidden md:flex items-center justify-center hover:scale-105 active:scale-95"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          className="w-6 h-6"
          aria-hidden="true"
        >
          <path
            fillRule="evenodd"
            d="M11.78 5.22a.75.75 0 0 1 0 1.06L8.06 10l3.72 3.72a.75.75 0 1 1-1.06 1.06l-4.25-4.25a.75.75 0 0 1 0-1.06l4.25-4.25a.75.75 0 0 1 1.06 0Z"
            clipRule="evenodd"
          />
        </svg>
      </button>

      {/* Navigation Controls: Next Button */}
      <button
        type="button"
        aria-label="Next artwork"
        onClick={onNext}
        className="absolute right-4 top-1/2 -translate-y-1/2 z-20 p-2.5 rounded-full bg-[var(--bg-primary)]/80 backdrop-blur-md border border-[var(--border-subtle)] hover:border-[var(--border-focus)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-all cursor-pointer shadow-xl hidden md:flex items-center justify-center hover:scale-105 active:scale-95"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
          className="w-6 h-6"
          aria-hidden="true"
        >
          <path
            fillRule="evenodd"
            d="M8.22 5.22a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.75.75 0 0 1-1.06-1.06L11.94 10 8.22 6.28a.75.75 0 0 1 0-1.06Z"
            clipRule="evenodd"
          />
        </svg>
      </button>

      {/* Main Content Area */}
      <div
        data-testid="lightbox-content"
        onClick={(e) => e.stopPropagation()}
        className="relative flex flex-col items-center justify-center max-h-[85vh] max-w-[90vw]"
      >
        {/* High-Resolution Artwork Presentation */}
        <div className="relative flex items-center justify-center overflow-hidden rounded-md shadow-2xl">
          {/* Skeleton Shimmer Loading State */}
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
            onLoad={() => setImageLoaded(true)}
            onError={handleImageError}
            className="object-contain max-h-[85vh] max-w-[90vw] select-none transition-opacity duration-300"
          />
        </div>

        {/* Embedded Museum Curatorial Plaque */}
        <div className="absolute bottom-4 left-4 right-4 md:left-auto md:right-4 z-30 pointer-events-auto">
          <CuratorialPlaque
            artwork={artwork}
            isCollapsed={isPlaqueCollapsed}
            onToggleCollapse={onTogglePlaque}
          />
        </div>
      </div>
    </div>
  );
}
