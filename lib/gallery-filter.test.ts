import { describe, it, expect } from 'vitest';
import {
  normalizeTheme,
  aggregateMovements,
  filterAndSortArtworks,
  scoreArtwork,
  selectRepresentativeArtwork,
  selectTopCanonicalMovements,
  CatalogArtwork,
} from './gallery-filter';
import { gallery } from './proto/gallery';

describe('Gallery Filter & Thematic Aggregation Engine (lib/gallery-filter.ts)', () => {
  describe('normalizeTheme', () => {
    it('trims leading and trailing whitespace from theme names', () => {
      expect(normalizeTheme('  The Crayon Period  ')).toBe('The Crayon Period');
      expect(normalizeTheme('\tDomestic Destructionism\n')).toBe('Domestic Destructionism');
    });

    it('falls back to canonical "Early Works" for empty, null, undefined, or all-whitespace strings', () => {
      expect(normalizeTheme(null)).toBe('Early Works');
      expect(normalizeTheme(undefined)).toBe('Early Works');
      expect(normalizeTheme('')).toBe('Early Works');
      expect(normalizeTheme('   ')).toBe('Early Works');
      expect(normalizeTheme(' \t \n ')).toBe('Early Works');
    });

    it('preserves valid theme names without altering internal spaces', () => {
      expect(normalizeTheme('Kinetic Scribblism')).toBe('Kinetic Scribblism');
      expect(normalizeTheme('Found Object Assemblage')).toBe('Found Object Assemblage');
      expect(normalizeTheme('Monochrome Nihilism')).toBe('Monochrome Nihilism');
      expect(normalizeTheme('Avant-Garde Experiments')).toBe('Avant-Garde Experiments');
    });
  });

  describe('aggregateMovements', () => {
    it('returns empty array when catalog is empty', () => {
      expect(aggregateMovements([])).toEqual([]);
    });

    it('dynamically counts artworks grouped by normalized movement name', () => {
      const artworks: gallery.IArtwork[] = [
        { id: '1', themeId: 'The Crayon Period' },
        { id: '2', themeId: '  The Crayon Period  ' },
        { id: '3', themeId: 'Kinetic Scribblism' },
        { id: '4', themeId: '' }, // Should normalize to Early Works
        { id: '5', themeId: null }, // Should normalize to Early Works
      ];

      const movements = aggregateMovements(artworks);
      expect(movements).toEqual([
        { name: 'The Crayon Period', count: 2 },
        { name: 'Kinetic Scribblism', count: 1 },
        { name: 'Early Works', count: 2 },
      ]);
    });

    it('sorts movements in canonical chronological order (Phase I-V), then Early Works, then alphabetical unexpected categories', () => {
      const artworks: gallery.IArtwork[] = [
        { id: 'u1', themeId: 'Zebra Period' },
        { id: 'p5', themeId: 'Monochrome Nihilism' },
        { id: 'u2', themeId: 'Abstract Dreams' },
        { id: 'p2', themeId: 'Domestic Destructionism' },
        { id: 'ew', themeId: '' }, // Early Works
        { id: 'p1', themeId: 'The Crayon Period' },
        { id: 'p4', themeId: 'Found Object Assemblage' },
        { id: 'p3', themeId: 'Kinetic Scribblism' },
        { id: 'u3', themeId: 'Cubist Sketches' },
      ];

      const movements = aggregateMovements(artworks);
      const names = movements.map((m) => m.name);

      expect(names).toEqual([
        'The Crayon Period',         // Phase I
        'Domestic Destructionism',   // Phase II
        'Kinetic Scribblism',        // Phase III
        'Found Object Assemblage',   // Phase IV
        'Monochrome Nihilism',       // Phase V
        'Early Works',               // Canonical fallback
        'Abstract Dreams',           // Alphabetical unexpected 1
        'Cubist Sketches',           // Alphabetical unexpected 2
        'Zebra Period',              // Alphabetical unexpected 3
      ]);
    });

    it('supports artworks specifying theme via theme or themeId property without mutating original array', () => {
      const artworks = [
        { id: '1', theme: 'The Crayon Period' } as gallery.IArtwork,
        { id: '2', themeId: 'The Crayon Period' } as gallery.IArtwork,
      ];
      const clone = [...artworks];

      const result = aggregateMovements(artworks);
      expect(result).toEqual([{ name: 'The Crayon Period', count: 2 }]);
      expect(artworks).toEqual(clone);
    });
  });

  describe('filterAndSortArtworks', () => {
    const mockArtworks: gallery.IArtwork[] = [
      { id: 'art-b', themeId: 'The Crayon Period', timestamp: 200 },
      { id: 'art-a', themeId: 'The Crayon Period', timestamp: 200 },
      { id: 'art-c', themeId: 'The Crayon Period', timestamp: 500 },
      { id: 'art-zero', themeId: 'The Crayon Period', timestamp: 0 },
      { id: 'art-null', themeId: 'The Crayon Period', timestamp: null },
      { id: 'art-d', themeId: 'Domestic Destructionism', timestamp: 300 },
      { id: 'art-early', themeId: '', timestamp: 150 }, // Early Works
    ];

    it('returns all artworks when selectedMovement is null, undefined, or "All Works"', () => {
      expect(filterAndSortArtworks(mockArtworks, null, 'desc')).toHaveLength(mockArtworks.length);
      expect(filterAndSortArtworks(mockArtworks, undefined as unknown as string, 'desc')).toHaveLength(mockArtworks.length);
      expect(filterAndSortArtworks(mockArtworks, 'All Works', 'desc')).toHaveLength(mockArtworks.length);
      expect(filterAndSortArtworks(mockArtworks, '  All Works  ', 'desc')).toHaveLength(mockArtworks.length);
    });

    it('accurately filters artworks matching selectedMovement (including normalized Early Works)', () => {
      const crayonArtworks = filterAndSortArtworks(mockArtworks, 'The Crayon Period', 'desc');
      expect(crayonArtworks).toHaveLength(5);
      expect(crayonArtworks.every((a) => a.themeId === 'The Crayon Period')).toBe(true);

      const earlyWorks = filterAndSortArtworks(mockArtworks, 'Early Works', 'desc');
      expect(earlyWorks).toHaveLength(1);
      expect(earlyWorks[0].id).toBe('art-early');
    });

    it('sorts descending by timestamp (newest first)', () => {
      const filtered = filterAndSortArtworks(mockArtworks, 'The Crayon Period', 'desc');
      // art-c (500), art-a (200, id tie break), art-b (200, id tie break), then art-null / art-zero
      expect(filtered[0].id).toBe('art-c');
      expect(filtered[1].id).toBe('art-a');
      expect(filtered[2].id).toBe('art-b');
    });

    it('sorts ascending by timestamp (earliest first)', () => {
      const filtered = filterAndSortArtworks(mockArtworks, 'The Crayon Period', 'asc');
      // Valid timestamps ascending: art-a (200, id tie break), art-b (200), art-c (500), then zero/null at end
      expect(filtered[0].id).toBe('art-a');
      expect(filtered[1].id).toBe('art-b');
      expect(filtered[2].id).toBe('art-c');
    });

    it('places zero or missing timestamps at the end in both desc and asc sort modes', () => {
      const desc = filterAndSortArtworks(mockArtworks, 'The Crayon Period', 'desc');
      const asc = filterAndSortArtworks(mockArtworks, 'The Crayon Period', 'asc');

      // The last 2 items in both should be art-null and art-zero, tie-broken by id ('art-null' < 'art-zero')
      expect(desc.slice(-2).map((a) => a.id)).toEqual(['art-null', 'art-zero']);
      expect(asc.slice(-2).map((a) => a.id)).toEqual(['art-null', 'art-zero']);
    });

    it('deterministically breaks ties by artwork id when timestamps are equal', () => {
      const items: gallery.IArtwork[] = [
        { id: 'artwork-z', themeId: 'Kinetic Scribblism', timestamp: 100 },
        { id: 'artwork-m', themeId: 'Kinetic Scribblism', timestamp: 100 },
        { id: 'artwork-a', themeId: 'Kinetic Scribblism', timestamp: 100 },
      ];

      const desc = filterAndSortArtworks(items, 'Kinetic Scribblism', 'desc');
      expect(desc.map((a) => a.id)).toEqual(['artwork-a', 'artwork-m', 'artwork-z']);

      const asc = filterAndSortArtworks(items, 'Kinetic Scribblism', 'asc');
      expect(asc.map((a) => a.id)).toEqual(['artwork-a', 'artwork-m', 'artwork-z']);
    });

    it('supports Long or toNumber() timestamp objects', () => {
      const items: gallery.IArtwork[] = [
        { id: 'art-long-2', themeId: 'The Crayon Period', timestamp: { toNumber: () => 2000 } as unknown as number },
        { id: 'art-long-1', themeId: 'The Crayon Period', timestamp: { toNumber: () => 1000 } as unknown as number },
      ];

      const desc = filterAndSortArtworks(items, 'The Crayon Period', 'desc');
      expect(desc.map((a) => a.id)).toEqual(['art-long-2', 'art-long-1']);

      const asc = filterAndSortArtworks(items, 'The Crayon Period', 'asc');
      expect(asc.map((a) => a.id)).toEqual(['art-long-1', 'art-long-2']);
    });

    it('does not mutate the original artworks array', () => {
      const original = [...mockArtworks];
      filterAndSortArtworks(mockArtworks, 'The Crayon Period', 'desc');
      expect(mockArtworks).toEqual(original);
    });
  });

  describe('scoreArtwork', () => {
    it('computes the full 130-point score for a fully curated artwork in the dominant canonical movement', () => {
      const artwork: gallery.IArtwork = {
        id: 'art-full',
        title: 'Crimson Horizon',
        description: 'An exploration of wax density.',
        themeId: 'The Crayon Period',
        medium: 'Wax crayon on paper',
        imagePath: '/data/images/art-full.jpg',
        thumbnailPath: '/data/thumbnails/art-full.webp',
        timestamp: 1700000000,
      };

      expect(scoreArtwork(artwork, 'The Crayon Period')).toBe(130);
    });

    it('computes individual point weights and returns 0 for unannotated/default placeholder fields', () => {
      const placeholder: gallery.IArtwork = {
        id: 'art-placeholder',
        title: '  untitled  ',
        description: '   ',
        themeId: '  Early Works ',
        medium: ' Mixed Media ',
        imagePath: '/placeholder.svg',
        thumbnailPath: '/placeholder.svg',
      };

      expect(scoreArtwork(placeholder, 'The Crayon Period')).toBe(0);

      // Non-dominant non-fallback theme (+25), valid title (+20), missing description (0), valid medium (+15), missing thumbnail (0)
      const partial: gallery.IArtwork = {
        id: 'art-partial',
        title: 'Study #4',
        description: '',
        themeId: 'Kinetic Scribblism',
        medium: 'Graphite on wall',
        imagePath: '/data/images/art-partial.jpg',
        thumbnailPath: '/placeholder.svg',
      };

      expect(scoreArtwork(partial, 'The Crayon Period')).toBe(25 + 20 + 15);
    });
  });

  describe('selectRepresentativeArtwork', () => {
    it('returns null when artworks is empty', () => {
      expect(selectRepresentativeArtwork([])).toBeNull();
      expect(selectRepresentativeArtwork(null as unknown as gallery.IArtwork[])).toBeNull();
    });

    it('returns the single flagged artwork when isRepresentative or is_representative is true', () => {
      const artworks: CatalogArtwork[] = [
        {
          id: 'high-score',
          title: 'Masterpiece',
          description: 'Full curatorial note.',
          themeId: 'The Crayon Period',
          medium: 'Crayon on canvas',
          imagePath: '/data/images/high-score.jpg',
          thumbnailPath: '/data/thumbnails/high-score.webp',
          timestamp: 500,
        },
        {
          id: 'flagged-camel',
          title: 'Untitled',
          isRepresentative: true,
          timestamp: 100,
        },
      ];

      expect(selectRepresentativeArtwork(artworks)?.id).toBe('flagged-camel');

      const snakeFlagged: CatalogArtwork[] = [
        artworks[0],
        {
          id: 'flagged-snake',
          title: 'Untitled',
          is_representative: true,
          timestamp: 100,
        },
      ];

      expect(selectRepresentativeArtwork(snakeFlagged)?.id).toBe('flagged-snake');
    });

    it('breaks ties deterministically when multiple entries are flagged', () => {
      const artworks: CatalogArtwork[] = [
        {
          id: 'flagged-b',
          title: 'Curated B',
          description: 'Note B',
          themeId: 'The Crayon Period',
          medium: 'Oil pastel',
          imagePath: '/data/images/b.jpg',
          thumbnailPath: '/data/thumbnails/b.webp',
          timestamp: 300,
          isRepresentative: true,
        },
        {
          id: 'flagged-a',
          title: 'Curated A',
          description: 'Note A',
          themeId: 'The Crayon Period',
          medium: 'Oil pastel',
          imagePath: '/data/images/a.jpg',
          thumbnailPath: '/data/thumbnails/a.webp',
          timestamp: 300,
          is_representative: true,
        },
        {
          id: 'flagged-lower-score',
          title: 'Untitled',
          timestamp: 999,
          isRepresentative: true,
        },
      ];

      // flagged-a and flagged-b tie on score (130) and timestamp (300); 'flagged-a' wins by id ASC
      expect(selectRepresentativeArtwork(artworks)?.id).toBe('flagged-a');
    });

    it('computes the fallback score and breaks ties deterministically when no entries are flagged', () => {
      const artworks: gallery.IArtwork[] = [
        // 2 works in Kinetic Scribblism (makes Kinetic Scribblism the dominant canonical movement)
        {
          id: 'ks-1',
          title: 'Kinetic I',
          description: 'Movement piece 1',
          themeId: 'Kinetic Scribblism',
          medium: 'Ink on board',
          imagePath: '/data/images/ks-1.jpg',
          thumbnailPath: '/data/thumbnails/ks-1.webp',
          timestamp: 200,
        },
        {
          id: 'ks-2',
          title: 'Kinetic II',
          description: 'Movement piece 2',
          themeId: 'Kinetic Scribblism',
          medium: 'Ink on board',
          imagePath: '/data/images/ks-2.jpg',
          thumbnailPath: '/data/thumbnails/ks-2.webp',
          timestamp: 400,
        },
        // 1 work in The Crayon Period (equally curated, but not in dominant movement -> 90 pts vs 130 pts)
        {
          id: 'cp-1',
          title: 'Crayon I',
          description: 'Crayon piece 1',
          themeId: 'The Crayon Period',
          medium: 'Wax on paper',
          imagePath: '/data/images/cp-1.jpg',
          thumbnailPath: '/data/thumbnails/cp-1.webp',
          timestamp: 900,
        },
      ];

      // ks-1 and ks-2 both score 130; ks-2 has newer timestamp (400 > 200)
      expect(selectRepresentativeArtwork(artworks)?.id).toBe('ks-2');
    });

    it('breaks dominant canonical movement count ties by earliest index in CANONICAL_MOVEMENTS and handles unannotated archives', () => {
      const tieMovements: gallery.IArtwork[] = [
        {
          id: 'mono-1',
          title: 'Void',
          description: 'Darkness.',
          themeId: 'Monochrome Nihilism',
          medium: 'Charcoal',
          imagePath: '/data/images/mono-1.jpg',
          thumbnailPath: '/data/thumbnails/mono-1.webp',
          timestamp: 500,
        },
        {
          id: 'crayon-1',
          title: 'Dawn',
          description: 'Wax.',
          themeId: 'The Crayon Period',
          medium: 'Crayon',
          imagePath: '/data/images/crayon-1.jpg',
          thumbnailPath: '/data/thumbnails/crayon-1.webp',
          timestamp: 100,
        },
      ];

      // Both canonical movements have count 1; 'The Crayon Period' is index 0 in CANONICAL_MOVEMENTS so crayon-1 gets +40
      expect(selectRepresentativeArtwork(tieMovements)?.id).toBe('crayon-1');

      const unannotated: gallery.IArtwork[] = [
        { id: 'un-z', timestamp: 0 },
        { id: 'un-b', timestamp: 250 },
        { id: 'un-a', timestamp: 250 },
      ];

      expect(selectRepresentativeArtwork(unannotated)?.id).toBe('un-a');
    });
  });

  describe('selectTopCanonicalMovements', () => {
    it('returns the first 3 canonical phases in chronological order for empty, null, or undefined artworks', () => {
      const expected = [
        'The Crayon Period',
        'Domestic Destructionism',
        'Kinetic Scribblism',
      ];
      expect(selectTopCanonicalMovements([])).toEqual(expected);
      expect(selectTopCanonicalMovements(null)).toEqual(expected);
      expect(selectTopCanonicalMovements(undefined)).toEqual(expected);
    });

    it('selects the top 3 canonical movements by descending count and returns them in canonical chronological order', () => {
      // Phase V = 4, Phase III = 3, Phase II = 2, Phase I = 1, Phase IV = 0
      const artworks: gallery.IArtwork[] = [
        { id: 'v1', themeId: 'Monochrome Nihilism' },
        { id: 'v2', themeId: 'Monochrome Nihilism' },
        { id: 'v3', themeId: 'Monochrome Nihilism' },
        { id: 'v4', themeId: 'Monochrome Nihilism' },
        { id: 'iii1', themeId: 'Kinetic Scribblism' },
        { id: 'iii2', themeId: 'Kinetic Scribblism' },
        { id: 'iii3', themeId: 'Kinetic Scribblism' },
        { id: 'ii1', themeId: 'Domestic Destructionism' },
        { id: 'ii2', themeId: 'Domestic Destructionism' },
        { id: 'i1', themeId: 'The Crayon Period' },
      ];

      expect(selectTopCanonicalMovements(artworks)).toEqual([
        'Domestic Destructionism',
        'Kinetic Scribblism',
        'Monochrome Nihilism',
      ]);
    });

    it('pads remaining slots with the earliest 0-count canonical phases for sparse collections and returns in chronological order', () => {
      // Only Phase V = 3 and Phase IV = 1 have artworks; remaining slot filled by Phase I (0 count)
      const artworks: gallery.IArtwork[] = [
        { id: 'v1', themeId: 'Monochrome Nihilism' },
        { id: 'v2', themeId: 'Monochrome Nihilism' },
        { id: 'v3', themeId: 'Monochrome Nihilism' },
        { id: 'iv1', themeId: 'Found Object Assemblage' },
      ];

      expect(selectTopCanonicalMovements(artworks)).toEqual([
        'The Crayon Period',
        'Found Object Assemblage',
        'Monochrome Nihilism',
      ]);
    });

    it('breaks ties at the selection boundary deterministically by canonical order (Phase I < II < III < IV < V)', () => {
      // Phases II, III, IV, and V each have 2 artworks, Phase I has 0 -> Phases II, III, IV win over V
      const artworks: gallery.IArtwork[] = [
        { id: 'v1', themeId: 'Monochrome Nihilism' },
        { id: 'v2', themeId: 'Monochrome Nihilism' },
        { id: 'iv1', themeId: 'Found Object Assemblage' },
        { id: 'iv2', themeId: 'Found Object Assemblage' },
        { id: 'iii1', themeId: 'Kinetic Scribblism' },
        { id: 'iii2', themeId: 'Kinetic Scribblism' },
        { id: 'ii1', themeId: 'Domestic Destructionism' },
        { id: 'ii2', themeId: 'Domestic Destructionism' },
      ];

      expect(selectTopCanonicalMovements(artworks)).toEqual([
        'Domestic Destructionism',
        'Kinetic Scribblism',
        'Found Object Assemblage',
      ]);
    });

    it('ignores non-canonical themes and "Early Works" (including missing, empty, or whitespace-only themes)', () => {
      const artworks: gallery.IArtwork[] = [
        { id: 'ew1', themeId: 'Early Works' },
        { id: 'ew2', themeId: '' },
        { id: 'ew3', themeId: '   ' },
        { id: 'ew4', themeId: null },
        { id: 'ew5' },
        { id: 'custom1', themeId: 'Avant-Garde Experiments' },
        { id: 'custom2', themeId: 'Avant-Garde Experiments' },
        { id: 'custom3', themeId: 'Avant-Garde Experiments' },
      ];

      expect(selectTopCanonicalMovements(artworks)).toEqual([
        'The Crayon Period',
        'Domestic Destructionism',
        'Kinetic Scribblism',
      ]);
    });

    it('normalizes and accurately counts both theme and themeId properties with leading/trailing whitespace', () => {
      const artworks: gallery.IArtwork[] = [
        { id: '1', theme: '  Monochrome Nihilism  ' } as gallery.IArtwork,
        { id: '2', themeId: '\tMonochrome Nihilism\n' },
        { id: '3', theme: '  Found Object Assemblage ' } as gallery.IArtwork,
        { id: '4', themeId: ' Found Object Assemblage' },
        { id: '5', theme: ' Kinetic Scribblism ' } as gallery.IArtwork,
      ];

      expect(selectTopCanonicalMovements(artworks)).toEqual([
        'Kinetic Scribblism',
        'Found Object Assemblage',
        'Monochrome Nihilism',
      ]);
    });
  });
});
