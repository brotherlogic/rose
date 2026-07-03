import { describe, it, expect, vi, beforeEach } from 'vitest';
import { getArtworks, getArtworkPaths } from './nfs';
import * as fs from 'fs';
import * as path from 'path';
import { gallery } from './proto/gallery';

vi.mock('fs');

describe('NFS Data Fetching', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    process.env.NFS_PATH = '/mock/nfs/path';
  });

  it('should return artwork paths from .sync-state.json', () => {
    const mockSyncState = { 'photo-1': true, 'photo-2': true };
    vi.mocked(fs.existsSync).mockReturnValue(true);
    vi.mocked(fs.readFileSync).mockReturnValue(JSON.stringify(mockSyncState));

    const paths = getArtworkPaths();
    expect(paths).toEqual([{ id: 'photo-1' }, { id: 'photo-2' }]);
  });

  it('should return decoded Artworks from .proto.bin files', () => {
    const mockSyncState = { 'photo-1': true };
    vi.mocked(fs.existsSync).mockReturnValue(true);
    
    const mockArtwork = gallery.Artwork.create({
      id: 'photo-1',
      title: 'Test Photo',
      description: 'A test photo',
      imagePath: '/mock/nfs/path/photo-1.jpg',
    });
    // convert Uint8Array to Buffer for the mock
    const buffer = Buffer.from(gallery.Artwork.encode(mockArtwork).finish());

    vi.mocked(fs.readFileSync).mockImplementation((filePath) => {
      if (filePath === path.join('/mock/nfs/path', '.sync-state.json')) {
        return JSON.stringify(mockSyncState);
      }
      if (filePath === path.join('/mock/nfs/path', 'photo-1.proto.bin')) {
        return buffer;
      }
      throw new Error(`File not found: ${filePath}`);
    });

    const artworks = getArtworks();
    expect(artworks).toHaveLength(1);
    expect(artworks[0].id).toBe('photo-1');
    expect(artworks[0].title).toBe('Test Photo');
  });
});
