import { describe, it, expect } from 'vitest';
import {
  normalizeTheme,
  aggregateMovements,
  filterAndSortArtworks,
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
});
