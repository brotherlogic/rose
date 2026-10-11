import { describe, it, expect, vi, beforeEach } from 'vitest';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

const mockGetArtworks = vi.fn();
vi.mock('@/lib/nfs', () => ({
  getArtworks: () => mockGetArtworks(),
}));

import Home from '../app/page';

describe('Homepage VirtualizedGalleryGrid Integration Suite (#266)', () => {
  beforeEach(() => {
    mockGetArtworks.mockReset();
  });

  it('should render VirtualizedGalleryGrid within the #collection section preserving editorial sections', () => {
    mockGetArtworks.mockReturnValue([
      {
        id: 'art-001',
        title: 'Avant-Garde Construct No. 1',
        medium: 'Oil on Unstretched Canvas',
        thumbnailPath: '/data/thumbnails/art-001.webp',
        imagePath: '/data/images/art-001.jpg',
        timestamp: 1712707200,
      },
    ]);

    const markup = renderToStaticMarkup(React.createElement(Home));

    // Verify preservation of editorial sections
    expect(markup).toContain('id="statement"');
    expect(markup).toContain('Curatorial Statement');
    expect(markup).toContain('id="biography"');
    expect(markup).toContain('Historical Movements Taxonomy');

    // Verify collection section exists
    expect(markup).toContain('id="collection"');
    expect(markup).toContain('The Permanent Collection');

    // Verify VirtualizedGalleryGrid container and ArtworkCard rendering
    expect(markup).toContain('Avant-Garde Construct No. 1');
    expect(markup).toContain('Oil on Unstretched Canvas');
    expect(markup).toContain('uppercase');
    expect(markup).toContain('aspect-square');
    expect(markup).toContain('/data/thumbnails/art-001.webp');
  });

  it('should delegate empty state handling to VirtualizedGalleryGrid when artworks is empty', () => {
    mockGetArtworks.mockReturnValue([]);

    const markup = renderToStaticMarkup(React.createElement(Home));

    expect(markup).toContain('id="collection"');
    expect(markup).toContain('No Works Currently Cataloged');
    expect(markup).toContain('Waiting for archival syncer ingestion');
  });
});
