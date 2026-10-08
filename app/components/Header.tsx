import React from 'react';
import Link from 'next/link';
import NavigationDrawer from './NavigationDrawer';

export default function Header() {
  return (
    <header
      role="banner"
      className="sticky top-0 z-50 w-full h-[var(--header-height,64px)] border-b border-[var(--border-subtle)] bg-[var(--bg-primary)]/80 backdrop-blur-md transition-colors"
    >
      <div className="container h-full mx-auto px-4 md:px-8 flex items-center justify-between">
        {/* Artist Wordmark */}
        <Link
          href="/"
          className="text-sm md:text-base font-semibold tracking-widest uppercase text-[var(--text-primary)] hover:opacity-80 transition-opacity"
        >
          ROSE SERAPHINE TUCKER
        </Link>

        {/* Desktop Navigation Anchors */}
        <nav
          className="hidden md:flex items-center space-x-8"
          aria-label="Desktop Navigation"
        >
          <Link
            href="#statement"
            className="text-xs uppercase tracking-widest text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors"
          >
            Statement
          </Link>
          <Link
            href="#biography"
            className="text-xs uppercase tracking-widest text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors"
          >
            Biography
          </Link>
          <Link
            href="#collection"
            className="text-xs uppercase tracking-widest text-[var(--text-secondary)] hover:text-[var(--text-primary)] transition-colors"
          >
            Collection
          </Link>
        </nav>

        {/* Responsive Mobile Navigation Drawer */}
        <NavigationDrawer />
      </div>
    </header>
  );
}
