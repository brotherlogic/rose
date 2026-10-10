import * as fs from 'fs';
import * as path from 'path';
import { gallery } from './proto/gallery';

function getNfsPath(): string {
  return process.env.NFS_PATH || '/tmp/nfs_mock';
}

export function getArtworkPaths(): { id: string }[] {
  const nfsPath = getNfsPath();
  const syncStateFile = path.join(nfsPath, '.sync-state.json');
  
  if (!fs.existsSync(syncStateFile)) {
    return [];
  }

  try {
    const rawData = fs.readFileSync(syncStateFile, 'utf8');
    const syncState = JSON.parse(rawData);
    
    return Object.keys(syncState)
      .filter((id) => syncState[id] === true)
      .map((id) => ({ id }));
  } catch (error) {
    console.error('Error reading sync state:', error);
    return [];
  }
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

export function getArtworks(): gallery.IArtwork[] {
  const nfsPath = getNfsPath();
  const paths = getArtworkPaths();
  const artworks: gallery.IArtwork[] = [];

  for (const p of paths) {
    const protoFile = path.join(nfsPath, `${p.id}.proto.bin`);
    if (fs.existsSync(protoFile)) {
      try {
        const buffer = fs.readFileSync(protoFile);
        const artwork = gallery.Artwork.decode(new Uint8Array(buffer));

        artwork.title = artwork.title?.trim() || 'Untitled';
        artwork.medium = artwork.medium?.trim() || 'Mixed Media';
        artwork.thumbnailPath = artwork.thumbnailPath || artwork.imagePath || '/placeholder.svg';

        artworks.push(artwork);
      } catch (error) {
        console.error(`Error reading or decoding ${protoFile}:`, error);
      }
    }
  }

  artworks.sort((a, b) => {
    const timeA = getTimestamp(a.timestamp);
    const timeB = getTimestamp(b.timestamp);
    return timeB - timeA;
  });

  return artworks;
}

