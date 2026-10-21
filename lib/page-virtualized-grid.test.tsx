import { describe, it, expect, vi, beforeEach } from 'vitest';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';

const mockGetArtworks = vi.fn();
vi.mock('@/lib/nfs', () => ({
  getArtworks: () => mockGetArtworks(),
}));

import Home from '../app/page';

describe('Homepage CuratedGalleryCollection Integration Suite (#278)', () => {
  beforeEach(() => {
    mockGetArtworks.mockReset();
  });

  it('should render CuratedGalleryCollection within the #collection section preserving editorial sections', () => {
    mockGetArtworks.mockReturnValue([
      {
        id: 'art-001',
        title: 'Avant-Garde Construct No. 1',
        medium: 'Oil on Unstretched Canvas',
        theme: 'The Crayon Period',
        thumbnailPath: '/data/thumbnails/art-001.webp',
        imagePath: '/data/images/art-001.jpg',
        timestamp: 1712707200,
      },
    ]);

    const markup = renderToStaticMarkup(React.createElement(Home));

    // Verify preservation of editorial sections and featured representative artwork ordering
    const statementIdx = markup.indexOf('id="statement"');
    const featuredIdx = markup.indexOf('id="featured-artwork"');
    const biographyIdx = markup.indexOf('id="biography"');
    const collectionIdx = markup.indexOf('id="collection"');

    expect(statementIdx).toBeGreaterThan(-1);
    expect(featuredIdx).toBeGreaterThan(statementIdx);
    expect(biographyIdx).toBeGreaterThan(featuredIdx);
    expect(collectionIdx).toBeGreaterThan(biographyIdx);

    expect(markup).toContain('Curatorial Statement');
    expect(markup).toContain('Historical Movements Taxonomy');

    // Verify collection section exists
    expect(markup).toContain('id="collection"');
    expect(markup).toContain('The Permanent Collection');

    // Verify CuratedGalleryCollection (filter bar & grid) integration retains representative artwork
    const collectionMarkup = markup.slice(collectionIdx);
    expect(collectionMarkup).toContain('aria-label="Exhibition period filtering"');
    expect(collectionMarkup).toContain('All Works (1)');
    expect(collectionMarkup).toContain('The Crayon Period (1)');

    // Verify ArtworkCard rendering inside the collection
    expect(collectionMarkup).toContain('Avant-Garde Construct No. 1');
    expect(collectionMarkup).toContain('Oil on Unstretched Canvas');
    expect(collectionMarkup).toContain('uppercase');
    expect(collectionMarkup).toContain('aspect-square');
    expect(collectionMarkup).toContain('/data/thumbnails/art-001.webp');
  });

  it('should delegate empty state handling and omit #featured-artwork when artworks is empty', () => {
    mockGetArtworks.mockReturnValue([]);

    const markup = renderToStaticMarkup(React.createElement(Home));

    expect(markup).not.toContain('id="featured-artwork"');
    expect(markup).toContain('id="collection"');
    expect(markup).toContain('No Works Currently Cataloged');
    expect(markup).toContain('Waiting for archival syncer ingestion');
  });
});
