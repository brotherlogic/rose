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

  it('should decode thumbnailPath from .proto.bin files', () => {
    const mockSyncState = { 'photo-1': true };
    vi.mocked(fs.existsSync).mockReturnValue(true);

    const mockArtwork = gallery.Artwork.create({
      id: 'photo-1',
      title: 'Test Photo',
      description: 'A test photo',
      imagePath: '/mock/nfs/path/photo-1.jpg',
      thumbnailPath: 'thumbnails/photo-1.webp',
    });
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
    expect(artworks[0].thumbnailPath).toBe('thumbnails/photo-1.webp');
  });

  it('should handle legacy .proto.bin files without thumbnailPath', () => {
    const mockSyncState = { 'photo-legacy': true };
    vi.mocked(fs.existsSync).mockReturnValue(true);

    const mockArtwork = gallery.Artwork.create({
      id: 'photo-legacy',
      title: 'Legacy Photo',
      description: 'A legacy photo without thumbnailPath',
      imagePath: '/mock/nfs/path/photo-legacy.jpg',
    });
    const buffer = Buffer.from(gallery.Artwork.encode(mockArtwork).finish());

    vi.mocked(fs.readFileSync).mockImplementation((filePath) => {
      if (filePath === path.join('/mock/nfs/path', '.sync-state.json')) {
        return JSON.stringify(mockSyncState);
      }
      if (filePath === path.join('/mock/nfs/path', 'photo-legacy.proto.bin')) {
        return buffer;
      }
      throw new Error(`File not found: ${filePath}`);
    });

    const artworks = getArtworks();
    expect(artworks).toHaveLength(1);
    expect(artworks[0].thumbnailPath).toBe('/mock/nfs/path/photo-legacy.jpg');
  });

  it('should decode all proto fields properly (id, title, description, medium, themeId, thumbnailPath, imagePath, timestamp)', () => {
    const mockSyncState = { 'artwork-1': true };
    vi.mocked(fs.existsSync).mockReturnValue(true);

    const mockArtwork = gallery.Artwork.create({
      id: 'artwork-1',
      title: 'Avant-garde Composition',
      description: 'Oil on textured canvas',
      themeId: 'modernism',
      timestamp: 1712000000,
      imagePath: '/mock/nfs/path/artwork-1.webp',
      thumbnailPath: '/mock/nfs/path/thumbnails/artwork-1.webp',
      medium: 'Oil on Canvas',
    });
    const buffer = Buffer.from(gallery.Artwork.encode(mockArtwork).finish());

    vi.mocked(fs.readFileSync).mockImplementation((filePath) => {
      if (filePath === path.join('/mock/nfs/path', '.sync-state.json')) {
        return JSON.stringify(mockSyncState);
      }
      if (filePath === path.join('/mock/nfs/path', 'artwork-1.proto.bin')) {
        return buffer;
      }
      throw new Error(`File not found: ${filePath}`);
    });

    const artworks = getArtworks();
    expect(artworks).toHaveLength(1);
    const art = artworks[0];
    expect(art.id).toBe('artwork-1');
    expect(art.title).toBe('Avant-garde Composition');
    expect(art.description).toBe('Oil on textured canvas');
    expect(art.themeId).toBe('modernism');
    expect(Number(art.timestamp)).toBe(1712000000);
    expect(art.imagePath).toBe('/mock/nfs/path/artwork-1.webp');
    expect(art.thumbnailPath).toBe('/mock/nfs/path/thumbnails/artwork-1.webp');
    expect(art.medium).toBe('Oil on Canvas');
  });

  it('should sort artworks reverse-chronologically by timestamp descending (newest first)', () => {
    const mockSyncState = { 'item-old': true, 'item-new': true, 'item-mid': true };
    vi.mocked(fs.existsSync).mockReturnValue(true);

    const artOld = gallery.Artwork.create({ id: 'item-old', title: 'Old Artwork', timestamp: 1000 });
    const artNew = gallery.Artwork.create({ id: 'item-new', title: 'New Artwork', timestamp: 3000 });
    const artMid = gallery.Artwork.create({ id: 'item-mid', title: 'Mid Artwork', timestamp: 2000 });

    const buffers: Record<string, Buffer> = {
      [path.join('/mock/nfs/path', 'item-old.proto.bin')]: Buffer.from(gallery.Artwork.encode(artOld).finish()),
      [path.join('/mock/nfs/path', 'item-new.proto.bin')]: Buffer.from(gallery.Artwork.encode(artNew).finish()),
      [path.join('/mock/nfs/path', 'item-mid.proto.bin')]: Buffer.from(gallery.Artwork.encode(artMid).finish()),
    };

    vi.mocked(fs.readFileSync).mockImplementation((filePath) => {
      if (filePath === path.join('/mock/nfs/path', '.sync-state.json')) {
        return JSON.stringify(mockSyncState);
      }
      if (typeof filePath === 'string' && buffers[filePath]) {
        return buffers[filePath];
      }
      throw new Error(`File not found: ${filePath}`);
    });

    const artworks = getArtworks();
    expect(artworks).toHaveLength(3);
    expect(artworks.map((a) => a.id)).toEqual(['item-new', 'item-mid', 'item-old']);
  });

  it('should fall back to Untitled and Mixed Media when title or medium are missing or empty', () => {
    const mockSyncState = { 'item-empty': true, 'item-whitespace': true };
    vi.mocked(fs.existsSync).mockReturnValue(true);

    const artEmpty = gallery.Artwork.create({ id: 'item-empty', title: '', medium: '' });
    const artWhitespace = gallery.Artwork.create({ id: 'item-whitespace', title: '   ', medium: '   ' });

    const buffers: Record<string, Buffer> = {
      [path.join('/mock/nfs/path', 'item-empty.proto.bin')]: Buffer.from(gallery.Artwork.encode(artEmpty).finish()),
      [path.join('/mock/nfs/path', 'item-whitespace.proto.bin')]: Buffer.from(gallery.Artwork.encode(artWhitespace).finish()),
    };

    vi.mocked(fs.readFileSync).mockImplementation((filePath) => {
      if (filePath === path.join('/mock/nfs/path', '.sync-state.json')) {
        return JSON.stringify(mockSyncState);
      }
      if (typeof filePath === 'string' && buffers[filePath]) {
        return buffers[filePath];
      }
      throw new Error(`File not found: ${filePath}`);
    });

    const artworks = getArtworks();
    expect(artworks).toHaveLength(2);
    expect(artworks.find(a => a.id === 'item-empty')?.title).toBe('Untitled');
    expect(artworks.find(a => a.id === 'item-empty')?.medium).toBe('Mixed Media');
    expect(artworks.find(a => a.id === 'item-whitespace')?.title).toBe('Untitled');
    expect(artworks.find(a => a.id === 'item-whitespace')?.medium).toBe('Mixed Media');
  });

  it('should fall back missing thumbnailPath to imagePath or /placeholder.svg', () => {
    const mockSyncState = { 'item-image-only': true, 'item-no-image': true };
    vi.mocked(fs.existsSync).mockReturnValue(true);

    const artImageOnly = gallery.Artwork.create({
      id: 'item-image-only',
      imagePath: '/mock/nfs/path/image-only.jpg',
      thumbnailPath: '',
    });
    const artNoImage = gallery.Artwork.create({
      id: 'item-no-image',
      imagePath: '',
      thumbnailPath: '',
    });

    const buffers: Record<string, Buffer> = {
      [path.join('/mock/nfs/path', 'item-image-only.proto.bin')]: Buffer.from(gallery.Artwork.encode(artImageOnly).finish()),
      [path.join('/mock/nfs/path', 'item-no-image.proto.bin')]: Buffer.from(gallery.Artwork.encode(artNoImage).finish()),
    };

    vi.mocked(fs.readFileSync).mockImplementation((filePath) => {
      if (filePath === path.join('/mock/nfs/path', '.sync-state.json')) {
        return JSON.stringify(mockSyncState);
      }
      if (typeof filePath === 'string' && buffers[filePath]) {
        return buffers[filePath];
      }
      throw new Error(`File not found: ${filePath}`);
    });

    const artworks = getArtworks();
    expect(artworks).toHaveLength(2);
    expect(artworks.find(a => a.id === 'item-image-only')?.thumbnailPath).toBe('/mock/nfs/path/image-only.jpg');
    expect(artworks.find(a => a.id === 'item-no-image')?.thumbnailPath).toBe('/placeholder.svg');
  });

  it('should skip corrupted or unreadable .proto.bin files safely without failing', () => {
    const mockSyncState = { 'valid-1': true, 'corrupt-1': true };
    vi.mocked(fs.existsSync).mockReturnValue(true);

    const validArt = gallery.Artwork.create({ id: 'valid-1', title: 'Valid One' });
    const validBuffer = Buffer.from(gallery.Artwork.encode(validArt).finish());
    const corruptBuffer = Buffer.from([0xff, 0xff, 0xff, 0xff]);

    vi.mocked(fs.readFileSync).mockImplementation((filePath) => {
      if (filePath === path.join('/mock/nfs/path', '.sync-state.json')) {
        return JSON.stringify(mockSyncState);
      }
      if (filePath === path.join('/mock/nfs/path', 'valid-1.proto.bin')) {
        return validBuffer;
      }
      if (filePath === path.join('/mock/nfs/path', 'corrupt-1.proto.bin')) {
        return corruptBuffer;
      }
      throw new Error(`File not found: ${filePath}`);
    });

    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const artworks = getArtworks();
    expect(artworks).toHaveLength(1);
    expect(artworks[0].id).toBe('valid-1');
    expect(consoleSpy).toHaveBeenCalled();
    consoleSpy.mockRestore();
  });
});
