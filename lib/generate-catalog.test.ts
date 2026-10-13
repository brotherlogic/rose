import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';
import { execSync } from 'child_process';
import { gallery } from './proto/gallery';
import { generateCatalog, parseArgs, resolveStoragePath, CatalogArtwork } from '../scripts/generate-catalog';

describe('Catalog Generation', () => {
  let testDir: string;

  beforeEach(() => {
    testDir = fs.mkdtempSync(path.join(os.tmpdir(), 'rose-catalog-test-'));
  });

  afterEach(() => {
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true, force: true });
    }
  });

  it('should resolve storage path from CLI flag --storage-path, STORAGE_PATH env, or default to /data', () => {
    expect(parseArgs(['--storage-path', '/custom/path'])).toEqual({ storagePath: '/custom/path' });
    expect(parseArgs(['--storage-path=/custom/path2'])).toEqual({ storagePath: '/custom/path2' });

    delete process.env.STORAGE_PATH;
    expect(resolveStoragePath([])).toBe('/data');

    process.env.STORAGE_PATH = '/env/storage';
    expect(resolveStoragePath([])).toBe('/env/storage');

    expect(resolveStoragePath(['--storage-path', '/cli/storage'])).toBe('/cli/storage');
    delete process.env.STORAGE_PATH;
  });

  it('should write an empty array [] to catalog.json when storage directory is empty', () => {
    const outputPath = generateCatalog(testDir);
    expect(outputPath).toBe(path.join(testDir, 'catalog.json'));
    expect(fs.existsSync(outputPath)).toBe(true);

    const content = JSON.parse(fs.readFileSync(outputPath, 'utf8'));
    expect(content).toEqual([]);
  });

  it('should write an empty array [] when storage directory contains no *.proto.bin files', () => {
    fs.writeFileSync(path.join(testDir, 'some-file.txt'), 'hello world');
    fs.writeFileSync(path.join(testDir, 'image.jpg'), 'fake-image');

    const outputPath = generateCatalog(testDir);
    const content = JSON.parse(fs.readFileSync(outputPath, 'utf8'));
    expect(content).toEqual([]);
  });

  it('should discover all *.proto.bin files, decode them, normalize fields, and sort descending by timestamp', () => {
    const artOld = gallery.Artwork.create({
      id: 'art-old',
      title: 'Old Artwork',
      description: 'First piece created',
      themeId: 'vintage',
      timestamp: 1000,
      imagePath: '/data/images/art-old.jpg',
      thumbnailPath: '/data/thumbnails/art-old.webp',
      medium: 'Oil on canvas',
    });

    const artNew = gallery.Artwork.create({
      id: 'art-new',
      title: 'New Artwork',
      description: 'Latest modern piece',
      themeId: 'contemporary',
      timestamp: 3000,
      imagePath: '/data/images/art-new.jpg',
      thumbnailPath: '/data/thumbnails/art-new.webp',
      medium: 'Digital projection',
    });

    const artMid = gallery.Artwork.create({
      id: 'art-mid',
      title: 'Mid Artwork',
      description: 'Middle era piece',
      themeId: 'impressionism',
      timestamp: 2000,
      imagePath: '/data/images/art-mid.jpg',
      thumbnailPath: '/data/thumbnails/art-mid.webp',
      medium: 'Acrylic',
    });

    fs.writeFileSync(
      path.join(testDir, 'art-old.proto.bin'),
      Buffer.from(gallery.Artwork.encode(artOld).finish())
    );
    fs.writeFileSync(
      path.join(testDir, 'art-new.proto.bin'),
      Buffer.from(gallery.Artwork.encode(artNew).finish())
    );
    fs.writeFileSync(
      path.join(testDir, 'art-mid.proto.bin'),
      Buffer.from(gallery.Artwork.encode(artMid).finish())
    );

    const outputPath = generateCatalog(testDir);
    const catalog = JSON.parse(fs.readFileSync(outputPath, 'utf8'));

    expect(catalog).toHaveLength(3);
    // Verify sorting descending by timestamp (newest first)
    expect(catalog.map((a: CatalogArtwork) => a.id)).toEqual(['art-new', 'art-mid', 'art-old']);

    // Verify extracted and normalized fields
    expect(catalog[0]).toEqual({
      id: 'art-new',
      title: 'New Artwork',
      description: 'Latest modern piece',
      theme_id: 'contemporary',
      timestamp: 3000,
      image_path: '/data/images/art-new.jpg',
      thumbnail_path: '/data/thumbnails/art-new.webp',
      medium: 'Digital projection',
    });

    expect(catalog[1]).toEqual({
      id: 'art-mid',
      title: 'Mid Artwork',
      description: 'Middle era piece',
      theme_id: 'impressionism',
      timestamp: 2000,
      image_path: '/data/images/art-mid.jpg',
      thumbnail_path: '/data/thumbnails/art-mid.webp',
      medium: 'Acrylic',
    });

    expect(catalog[2]).toEqual({
      id: 'art-old',
      title: 'Old Artwork',
      description: 'First piece created',
      theme_id: 'vintage',
      timestamp: 1000,
      image_path: '/data/images/art-old.jpg',
      thumbnail_path: '/data/thumbnails/art-old.webp',
      medium: 'Oil on canvas',
    });
  });

  it('should atomically write catalog.json leaving no leftover temp files', () => {
    const art = gallery.Artwork.create({
      id: 'art-1',
      title: 'Sample Piece',
      timestamp: 12345,
    });
    fs.writeFileSync(
      path.join(testDir, 'art-1.proto.bin'),
      Buffer.from(gallery.Artwork.encode(art).finish())
    );

    generateCatalog(testDir);

    const files = fs.readdirSync(testDir);
    expect(files).toContain('catalog.json');
    expect(files.filter(f => f.includes('.tmp'))).toHaveLength(0);
  });

  it('should execute successfully via CLI command with --storage-path flag', () => {
    const art = gallery.Artwork.create({
      id: 'cli-art',
      title: 'CLI Artwork',
      timestamp: 5000,
    });
    fs.writeFileSync(
      path.join(testDir, 'cli-art.proto.bin'),
      Buffer.from(gallery.Artwork.encode(art).finish())
    );

    const output = execSync(
      `npx tsx scripts/generate-catalog.ts --storage-path "${testDir}"`,
      { encoding: 'utf8', cwd: process.cwd() }
    );
    expect(output).toContain('Generated catalog at');

    const catalogPath = path.join(testDir, 'catalog.json');
    expect(fs.existsSync(catalogPath)).toBe(true);
    const catalog = JSON.parse(fs.readFileSync(catalogPath, 'utf8'));
    expect(catalog).toHaveLength(1);
    expect(catalog[0].id).toBe('cli-art');
  });

  it('package.json should define catalog:generate script pointing to tsx scripts/generate-catalog.ts', () => {
    const pkg = JSON.parse(fs.readFileSync(path.join(process.cwd(), 'package.json'), 'utf8'));
    expect(pkg.scripts['catalog:generate']).toBe('tsx scripts/generate-catalog.ts');
  });
});
