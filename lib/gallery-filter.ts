import { gallery } from './proto/gallery';

export const CANONICAL_MOVEMENTS = [
  'The Crayon Period',
  'Domestic Destructionism',
  'Kinetic Scribblism',
  'Found Object Assemblage',
  'Monochrome Nihilism',
] as const;

export const FALLBACK_MOVEMENT = 'Early Works';
export const ALL_WORKS_LABEL = 'All Works';

export interface MovementCount {
  name: string;
  count: number;
}

export interface CatalogArtwork extends gallery.IArtwork {
  isRepresentative?: boolean;
  is_representative?: boolean;
}

interface HasToNumber {
  toNumber: () => number;
}

function hasToNumber(val: unknown): val is HasToNumber {
  return (
    typeof val === 'object' &&
    val !== null &&
    'toNumber' in val &&
    typeof (val as { toNumber: unknown }).toNumber === 'function'
  );
}

function getTimestamp(timestamp: unknown): number {
  if (!timestamp) return 0;
  if (typeof timestamp === 'number') return timestamp;
  if (hasToNumber(timestamp)) {
    return timestamp.toNumber();
  }
  return Number(timestamp) || 0;
}

function getArtworkTheme(artwork: gallery.IArtwork): string | null | undefined {
  return (artwork as { theme?: string | null; themeId?: string | null }).theme ?? artwork.themeId;
}

/**
 * Sanitizes and normalizes movement/theme string.
 * Falls back to canonical "Early Works" if empty, null, undefined, or all-whitespace.
 */
export function normalizeTheme(theme?: string | null): string {
  if (typeof theme !== 'string') {
    return FALLBACK_MOVEMENT;
  }
  const trimmed = theme.trim();
  return trimmed.length > 0 ? trimmed : FALLBACK_MOVEMENT;
}

/**
 * Dynamically counts artworks grouped by normalized movement name across the catalog.
 * Movements are sorted in canonical chronology (Phase I-V), followed by "Early Works",
 * and then alphabetically for any unexpected categories.
 */
export function aggregateMovements(artworks: gallery.IArtwork[]): MovementCount[] {
  if (!artworks || artworks.length === 0) {
    return [];
  }

  const counts = new Map<string, number>();

  for (const artwork of artworks) {
    const rawTheme = getArtworkTheme(artwork);
    const normalized = normalizeTheme(rawTheme);
    counts.set(normalized, (counts.get(normalized) || 0) + 1);
  }

  const result: MovementCount[] = Array.from(counts.entries()).map(([name, count]) => ({
    name,
    count,
  }));

  const canonicalIndices = new Map<string, number>(
    CANONICAL_MOVEMENTS.map((name, index) => [name, index])
  );

  result.sort((a, b) => {
    const idxA = canonicalIndices.has(a.name) ? canonicalIndices.get(a.name)! : -1;
    const idxB = canonicalIndices.has(b.name) ? canonicalIndices.get(b.name)! : -1;

    // Phase I-V canonical movements
    if (idxA !== -1 && idxB !== -1) {
      return idxA - idxB;
    }
    if (idxA !== -1) {
      return -1;
    }
    if (idxB !== -1) {
      return 1;
    }

    // Canonical fallback ("Early Works")
    if (a.name === FALLBACK_MOVEMENT && b.name === FALLBACK_MOVEMENT) {
      return 0;
    }
    if (a.name === FALLBACK_MOVEMENT) {
      return -1;
    }
    if (b.name === FALLBACK_MOVEMENT) {
      return 1;
    }

    // Unexpected categories sorted alphabetically
    return a.name.localeCompare(b.name);
  });

  return result;
}

/**
 * Filters and sorts artworks based on selected movement and chronological direction.
 *
 * If selectedMovement is active and not "All Works", filters artworks whose normalized
 * theme matches selectedMovement.
 *
 * Primary sort:
 * - 'desc': newest first (timeB - timeA)
 * - 'asc': earliest first (timeA - timeB)
 * - Missing or zero timestamps (!timestamp || timestamp <= 0) sorted to the end in both modes
 *
 * Secondary sort:
 * - Deterministic tie-breaking by artwork id: (a.id || '').localeCompare(b.id || '')
 */
export function filterAndSortArtworks(
  artworks: gallery.IArtwork[],
  selectedMovement: string | null | undefined,
  sortDirection: 'desc' | 'asc' = 'desc'
): gallery.IArtwork[] {
  if (!artworks) {
    return [];
  }

  const trimmedFilter = selectedMovement ? selectedMovement.trim() : null;
  const isAllWorks = !trimmedFilter || trimmedFilter === ALL_WORKS_LABEL;

  const filtered = isAllWorks
    ? [...artworks]
    : artworks.filter((artwork) => {
        const theme = normalizeTheme(getArtworkTheme(artwork));
        return theme === trimmedFilter;
      });

  filtered.sort((a, b) => {
    const timeA = getTimestamp(a.timestamp);
    const timeB = getTimestamp(b.timestamp);

    const validA = timeA > 0;
    const validB = timeB > 0;

    // Artworks with missing or zero timestamps (!timestamp || timestamp <= 0)
    // are sorted to the end in both modes.
    if (!validA && validB) {
      return 1;
    }
    if (validA && !validB) {
      return -1;
    }
    if (!validA && !validB) {
      return (a.id || '').localeCompare(b.id || '');
    }

    // Primary sort for valid timestamps
    const primary = sortDirection === 'asc' ? timeA - timeB : timeB - timeA;
    if (primary !== 0) {
      return primary;
    }

    // Deterministic tie-breaking by artwork ID
    return (a.id || '').localeCompare(b.id || '');
  });

  return filtered;
}

/**
 * Computes the deterministic weighted additive curatorial score for an artwork
 * matching the backend catalog generation heuristic.
 */
export function scoreArtwork(
  artwork: gallery.IArtwork,
  dominantCanonicalMovement: string
): number {
  let score = 0;
  const rawTheme = (getArtworkTheme(artwork) || '').trim();
  const title = (artwork.title || '').trim();
  const description = (artwork.description || '').trim();
  const medium = (artwork.medium || '').trim();
  const imagePath = (artwork.imagePath || '').trim();
  const thumbnailPath = (artwork.thumbnailPath || '').trim();

  if (dominantCanonicalMovement !== '' && rawTheme === dominantCanonicalMovement) {
    score += 40;
  }
  if (rawTheme !== '' && rawTheme.toLowerCase() !== 'early works') {
    score += 25;
  }
  if (title !== '' && title.toLowerCase() !== 'untitled') {
    score += 20;
  }
  if (description !== '') {
    score += 20;
  }
  if (medium !== '' && medium.toLowerCase() !== 'mixed media') {
    score += 15;
  }
  if (
    imagePath !== '' &&
    imagePath !== '/placeholder.svg' &&
    thumbnailPath !== '' &&
    thumbnailPath !== '/placeholder.svg'
  ) {
    score += 10;
  }

  return score;
}

/**
 * Selects the single representative artwork for the archive.
 * - Returns null when artworks is empty.
 * - Honors a single flagged artwork (`isRepresentative: true` or `is_representative: true`).
 * - Falls back to deterministic weighted additive scoring (with chronological/ID tie-breaking)
 *   when zero or multiple artworks are flagged.
 */
export function selectRepresentativeArtwork(
  artworks: gallery.IArtwork[]
): gallery.IArtwork | null {
  if (!artworks || artworks.length === 0) {
    return null;
  }

  const movementCounts = new Map<string, number>();
  for (const artwork of artworks) {
    const rawTheme = (getArtworkTheme(artwork) || '').trim();
    if (rawTheme) {
      movementCounts.set(rawTheme, (movementCounts.get(rawTheme) || 0) + 1);
    }
  }

  let dominantCanonicalMovement = '';
  let maxCount = 0;
  for (const movement of CANONICAL_MOVEMENTS) {
    const count = movementCounts.get(movement) || 0;
    if (count > maxCount) {
      maxCount = count;
      dominantCanonicalMovement = movement;
    }
  }

  const flagged = artworks.filter(
    (a) =>
      (a as CatalogArtwork).isRepresentative === true ||
      (a as CatalogArtwork).is_representative === true
  );

  if (flagged.length === 1) {
    return flagged[0];
  }

  const candidates = flagged.length > 0 ? [...flagged] : [...artworks];

  candidates.sort((a, b) => {
    const scoreA = scoreArtwork(a, dominantCanonicalMovement);
    const scoreB = scoreArtwork(b, dominantCanonicalMovement);
    if (scoreA !== scoreB) {
      return scoreB - scoreA;
    }

    const timeA = getTimestamp(a.timestamp);
    const timeB = getTimestamp(b.timestamp);
    const validA = timeA > 0;
    const validB = timeB > 0;

    if (!validA && validB) {
      return 1;
    }
    if (validA && !validB) {
      return -1;
    }
    if (validA && validB && timeA !== timeB) {
      return timeB - timeA;
    }

    return (a.id || '').localeCompare(b.id || '');
  });

  return candidates[0] || null;
}
