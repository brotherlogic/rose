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
        artworks.push(artwork);
      } catch (error) {
        console.error(`Error reading or decoding ${protoFile}:`, error);
      }
    }
  }

  return artworks;
}
