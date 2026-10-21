import * as fs from 'fs';
import * as path from 'path';
import { gallery } from '../lib/proto/gallery';

export const CANONICAL_MOVEMENTS = [
  'The Crayon Period',
  'Domestic Destructionism',
  'Kinetic Scribblism',
  'Found Object Assemblage',
  'Monochrome Nihilism',
] as const;

export interface CatalogArtwork {
  id: string;
  title: string;
  description: string;
  theme_id: string;
  timestamp: number;
  image_path: string;
  thumbnail_path: string;
  medium: string;
  is_representative: boolean;
}

export function parseArgs(args: string[]): { storagePath?: string } {
  let storagePath: string | undefined;

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    if (arg === '--storage-path' && i + 1 < args.length) {
      storagePath = args[++i];
    } else if (arg.startsWith('--storage-path=')) {
      storagePath = arg.substring('--storage-path='.length);
    }
  }

  return { storagePath };
}

export function resolveStoragePath(args: string[] = process.argv.slice(2)): string {
  const parsed = parseArgs(args);
  if (parsed.storagePath) {
    return parsed.storagePath;
  }
  if (process.env.STORAGE_PATH) {
    return process.env.STORAGE_PATH;
  }
  return '/data';
}

function getTimestamp(timestamp: unknown): number {
  if (!timestamp) return 0;
  if (typeof timestamp === 'number') return timestamp;
  if (
    typeof timestamp === 'object' &&
    timestamp !== null &&
    'toNumber' in timestamp &&
    typeof (timestamp as { toNumber: unknown }).toNumber === 'function'
  ) {
    return (timestamp as { toNumber: () => number }).toNumber();
  }
  return Number(timestamp) || 0;
}

function scoreCatalogArtwork(a: CatalogArtwork, dominantMovement: string): number {
  let score = 0;
  const theme = a.theme_id.trim();
  const title = a.title.trim();
  const description = a.description.trim();
  const medium = a.medium.trim();
  const imagePath = a.image_path.trim();
  const thumbnailPath = a.thumbnail_path.trim();

  if (dominantMovement !== '' && theme === dominantMovement) {
    score += 40;
  }
  if (theme !== '' && theme.toLowerCase() !== 'early works') {
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

function isBetterRepresentativeCandidate(
  candidate: CatalogArtwork,
  candidateScore: number,
  best: CatalogArtwork,
  bestScore: number
): boolean {
  if (candidateScore !== bestScore) {
    return candidateScore > bestScore;
  }

  const candValidTs = candidate.timestamp > 0;
  const bestValidTs = best.timestamp > 0;
  if (candValidTs !== bestValidTs) {
    return candValidTs;
  }
  if (candValidTs && bestValidTs && candidate.timestamp !== best.timestamp) {
    return candidate.timestamp > best.timestamp;
  }

  return candidate.id < best.id;
}

export function designateRepresentativeArtwork(artworks: CatalogArtwork[]): void {
  if (artworks.length === 0) {
    return;
  }

  const movementCounts = new Map<string, number>();
  for (const a of artworks) {
    const theme = a.theme_id.trim();
    if (theme !== '') {
      movementCounts.set(theme, (movementCounts.get(theme) ?? 0) + 1);
    }
  }

  let dominantMovement = '';
  let maxCount = 0;
  for (const movement of CANONICAL_MOVEMENTS) {
    const count = movementCounts.get(movement) ?? 0;
    if (count > maxCount) {
      maxCount = count;
      dominantMovement = movement;
    }
  }

  let bestIdx = 0;
  let bestScore = scoreCatalogArtwork(artworks[0], dominantMovement);
  artworks[0].is_representative = false;

  for (let i = 1; i < artworks.length; i++) {
    artworks[i].is_representative = false;
    const score = scoreCatalogArtwork(artworks[i], dominantMovement);
    if (isBetterRepresentativeCandidate(artworks[i], score, artworks[bestIdx], bestScore)) {
      bestIdx = i;
      bestScore = score;
    }
  }

  artworks[bestIdx].is_representative = true;
}

export function generateCatalog(storagePath: string): string {
  if (!fs.existsSync(storagePath)) {
    fs.mkdirSync(storagePath, { recursive: true });
  }

  const entries = fs.readdirSync(storagePath);
  const protoFiles = entries.filter((file) => file.endsWith('.proto.bin'));
  const artworks: CatalogArtwork[] = [];

  for (const file of protoFiles) {
    const fullPath = path.join(storagePath, file);
    try {
      const buffer = fs.readFileSync(fullPath);
      const decoded = gallery.Artwork.decode(new Uint8Array(buffer));

      const normalized: CatalogArtwork = {
        id: String(decoded.id ?? ''),
        title: String(decoded.title ?? ''),
        description: String(decoded.description ?? ''),
        theme_id: String(decoded.themeId ?? (decoded as unknown as { theme_id?: string }).theme_id ?? ''),
        timestamp: getTimestamp(decoded.timestamp),
        image_path: String(decoded.imagePath ?? (decoded as unknown as { image_path?: string }).image_path ?? ''),
        thumbnail_path: String(decoded.thumbnailPath ?? (decoded as unknown as { thumbnail_path?: string }).thumbnail_path ?? ''),
        medium: String(decoded.medium ?? ''),
        is_representative: false,
      };

      artworks.push(normalized);
    } catch (err) {
      console.error(`Error reading or decoding protobuf file ${fullPath}:`, err);
    }
  }

  artworks.sort((a, b) => {
    if (b.timestamp !== a.timestamp) {
      return b.timestamp - a.timestamp;
    }
    return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
  });

  designateRepresentativeArtwork(artworks);

  const catalogPath = path.join(storagePath, 'catalog.json');
  const tempPath = path.join(
    storagePath,
    `.catalog.json.tmp.${process.pid}.${Date.now()}.${Math.random().toString(36).substring(2, 8)}`
  );

  fs.writeFileSync(tempPath, JSON.stringify(artworks, null, 2), 'utf8');
  fs.renameSync(tempPath, catalogPath);

  return catalogPath;
}

export function main(): void {
  try {
    const storagePath = resolveStoragePath(process.argv.slice(2));
    const outputPath = generateCatalog(storagePath);
    console.log(`Generated catalog at ${outputPath}`);
  } catch (error) {
    console.error('Failed to generate catalog:', error);
    process.exit(1);
  }
}

const isDirectCli = Boolean(
  process.argv[1] &&
  (process.argv[1].endsWith('generate-catalog.ts') || process.argv[1].endsWith('generate-catalog.js'))
);

if (isDirectCli) {
  main();
}
