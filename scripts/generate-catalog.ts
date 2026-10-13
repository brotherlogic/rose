import * as fs from 'fs';
import * as path from 'path';
import { gallery } from '../lib/proto/gallery';

export interface CatalogArtwork {
  id: string;
  title: string;
  description: string;
  theme_id: string;
  timestamp: number;
  image_path: string;
  thumbnail_path: string;
  medium: string;
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
      };

      artworks.push(normalized);
    } catch (err) {
      console.error(`Error reading or decoding protobuf file ${fullPath}:`, err);
    }
  }

  artworks.sort((a, b) => b.timestamp - a.timestamp);

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
