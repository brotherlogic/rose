'use client';

import React, { useState, useEffect, useRef, useCallback } from 'react';
import Link from 'next/link';

interface NavigationDrawerProps {
  initialOpen?: boolean;
  onStateChange?: (isOpen: boolean) => void;
}

export default function NavigationDrawer({
  initialOpen = false,
  onStateChange,
}: NavigationDrawerProps) {
  const [isOpen, setIsOpen] = useState(initialOpen);
  const toggleButtonRef = useRef<HTMLButtonElement>(null);
  const drawerRef = useRef<HTMLDivElement>(null);

  const setOpenState = useCallback(
    (newState: boolean) => {
      setIsOpen(newState);
      onStateChange?.(newState);
    },
    [onStateChange]
  );

  const handleToggle = () => {
    setOpenState(!isOpen);
  };

  const handleClose = useCallback(() => {
    setOpenState(false);
    toggleButtonRef.current?.focus();
  }, [setOpenState]);

  // Window resize and Escape key listeners
  useEffect(() => {
    if (typeof window === 'undefined') return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        handleClose();
      }
    };

    const handleResize = () => {
      if (window.innerWidth >= 768) {
        handleClose();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('resize', handleResize);
    };
  }, [handleClose]);

  // Prevent background scrolling when drawer is open
  useEffect(() => {
    if (typeof document === 'undefined') return;
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      if (typeof document !== 'undefined') {
        document.body.style.overflow = '';
      }
    };
  }, [isOpen]);

  return (
    <div className="md:hidden">
      {/* Accessible Hamburger Toggle Button */}
      <button
        ref={toggleButtonRef}
        type="button"
        aria-label="Toggle navigation menu"
        aria-expanded={isOpen ? 'true' : 'false'}
        aria-controls="navigation-drawer"
        onClick={handleToggle}
        className="p-2 text-[var(--text-primary)] hover:opacity-75 focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--border-focus)] transition-opacity"
      >
        <svg
          className="w-6 h-6"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          {isOpen ? (
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M6 18L18 6M6 6l12 12"
            />
          ) : (
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1.5}
              d="M4 6h16M4 12h16M4 18h16"
            />
          )}
        </svg>
      </button>

      {/* Slide-out & Fade Drawer Overlay */}
      <div
        id="navigation-drawer"
        role="dialog"
        aria-modal="true"
        aria-label="Navigation menu"
        className={`fixed inset-0 top-[var(--header-height,64px)] z-40 transition-all duration-300 ease-in-out ${
          isOpen
            ? 'opacity-100 pointer-events-auto visible'
            : 'opacity-0 pointer-events-none invisible'
        }`}
      >
        {/* Backdrop overlay */}
        <div
          onClick={handleClose}
          className="absolute inset-0 bg-black/40 backdrop-blur-sm transition-opacity"
          aria-hidden="true"
        />

        {/* Drawer content panel */}
        <div
          ref={drawerRef}
          className={`relative z-10 w-full max-w-sm h-full bg-[var(--bg-primary)] border-r border-[var(--border-subtle)] p-6 shadow-2xl flex flex-col justify-between transform transition-transform duration-300 ease-in-out ${
            isOpen ? 'translate-x-0' : '-translate-x-full'
          }`}
        >
          <nav className="flex flex-col space-y-6 pt-4" aria-label="Mobile Navigation">
            <Link
              href="#statement"
              onClick={handleClose}
              className="text-lg font-medium tracking-wide uppercase text-[var(--text-primary)] hover:opacity-70 transition-opacity focus:outline-none focus-visible:underline"
            >
              Statement
            </Link>
            <Link
              href="#biography"
              onClick={handleClose}
              className="text-lg font-medium tracking-wide uppercase text-[var(--text-primary)] hover:opacity-70 transition-opacity focus:outline-none focus-visible:underline"
            >
              Biography
            </Link>
            <Link
              href="#collection"
              onClick={handleClose}
              className="text-lg font-medium tracking-wide uppercase text-[var(--text-primary)] hover:opacity-70 transition-opacity focus:outline-none focus-visible:underline"
            >
              Collection
            </Link>
          </nav>

          <div className="pt-8 border-t border-[var(--border-subtle)] text-xs uppercase tracking-widest text-[var(--text-secondary)]">
            Rose Seraphine Tucker
          </div>
        </div>
      </div>
    </div>
  );
}
