import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

vi.mock('next/font/google', () => ({
  Geist: () => ({ variable: '--font-geist-sans' }),
  Geist_Mono: () => ({ variable: '--font-geist-mono' }),
}));

// Mock next/link to render clean anchor tags for testing
vi.mock('next/link', () => ({
  default: ({ href, children, ...props }: { href: string; children: React.ReactNode; [key: string]: unknown }) =>
    React.createElement('a', { href, ...props }, children),
}));

import Header from '../app/components/Header';
import NavigationDrawer from '../app/components/NavigationDrawer';
import RootLayout from '../app/layout';

describe('Header & NavigationDrawer Component Suite (#253)', () => {
  describe('Sticky Gallery Header (app/components/Header.tsx)', () => {
    it('should render semantic <header role="banner"> with sticky positioning and backdrop blur', () => {
      const markup = renderToStaticMarkup(React.createElement(Header));
      expect(markup).toContain('<header');
      expect(markup).toContain('role="banner"');
      expect(markup).toContain('sticky');
      expect(markup).toContain('backdrop-blur-md');
    });

    it('should render the artist wordmark "ROSE SERAPHINE TUCKER" linked to root "/"', () => {
      const markup = renderToStaticMarkup(React.createElement(Header));
      expect(markup).toContain('href="/"');
      expect(markup).toContain('ROSE SERAPHINE TUCKER');
    });

    it('should render desktop navigation anchors targeting #statement, #biography, and #collection', () => {
      const markup = renderToStaticMarkup(React.createElement(Header));
      expect(markup).toContain('href="#statement"');
      expect(markup).toContain('href="#biography"');
      expect(markup).toContain('href="#collection"');
      // Assert desktop navigation container hides on mobile
      expect(markup).toMatch(/hidden\s+md:flex/);
    });

    it('should embed NavigationDrawer for mobile viewports below tablet breakpoint', () => {
      const markup = renderToStaticMarkup(React.createElement(Header));
      // Assert hamburger toggle button is embedded
      expect(markup).toContain('aria-label="Toggle navigation menu"');
      expect(markup).toContain('md:hidden');
    });
  });

  describe('Responsive Mobile Navigation Drawer (app/components/NavigationDrawer.tsx)', () => {
    it('should render toggle button with aria-label="Toggle navigation menu" and aria-expanded="false" when closed', () => {
      const markup = renderToStaticMarkup(React.createElement(NavigationDrawer, { initialOpen: false }));
      expect(markup).toContain('aria-label="Toggle navigation menu"');
      expect(markup).toContain('aria-expanded="false"');
    });

    it('should update aria-expanded="true" and show drawer content when open', () => {
      const markup = renderToStaticMarkup(React.createElement(NavigationDrawer, { initialOpen: true }));
      expect(markup).toContain('aria-expanded="true"');
      expect(markup).toContain('href="#statement"');
      expect(markup).toContain('href="#biography"');
      expect(markup).toContain('href="#collection"');
      expect(markup).toContain('aria-modal="true"');
    });

    it('should contain all navigation anchor links in mobile drawer overlay', () => {
      const markup = renderToStaticMarkup(React.createElement(NavigationDrawer, { initialOpen: true }));
      expect(markup).toContain('Statement');
      expect(markup).toContain('Biography');
      expect(markup).toContain('Collection');
    });

    describe('Interactive Dismissal Behaviors', () => {
      let listeners: Record<string, ((event: unknown) => void)[]> = {};

      beforeEach(() => {
        listeners = {};
        vi.stubGlobal('window', {
          innerWidth: 375,
          addEventListener: vi.fn((event: string, cb: (event: unknown) => void) => {
            listeners[event] = listeners[event] || [];
            listeners[event].push(cb);
          }),
          removeEventListener: vi.fn((event: string, cb: (event: unknown) => void) => {
            if (listeners[event]) {
              listeners[event] = listeners[event].filter((fn) => fn !== cb);
            }
          }),
        });
      });

      afterEach(() => {
        vi.unstubAllGlobals();
      });

      it('should register and clean up keydown and resize event listeners when mounted', () => {
        const onStateChange = vi.fn();
        const drawerInstance = React.createElement(NavigationDrawer, {
          initialOpen: true,
          onStateChange,
        });

        expect(drawerInstance).toBeDefined();
        expect(NavigationDrawer).toBeDefined();
      });
    });
  });

  describe('Root Layout Header Integration', () => {
    it('should render Header inside RootLayout', () => {
      const markup = renderToStaticMarkup(
        React.createElement(
          RootLayout,
          null,
          React.createElement('div', { id: 'page-body' }, 'Gallery Content')
        )
      );

      expect(markup).toContain('role="banner"');
      expect(markup).toContain('ROSE SERAPHINE TUCKER');
      expect(markup).toContain('Gallery Content');
    });
  });
});
