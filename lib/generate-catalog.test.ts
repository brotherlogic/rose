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
      is_representative: true,
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
      is_representative: false,
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
      is_representative: false,
    });
  });

  it('should designate is_representative: true on the single highest-scoring artwork using dominant canonical movement and completeness weights', () => {
    // 2 works in Kinetic Scribblism (dominant canonical movement) vs 1 in Monochrome Nihilism
    const workDominantComplete = gallery.Artwork.create({
      id: 'art-dom-complete',
      title: 'Velocity in Wax',
      description: 'Energetic marks across heavy stock.',
      themeId: 'Kinetic Scribblism',
      timestamp: 1000,
      imagePath: '/data/images/art-dom-complete.jpg',
      thumbnailPath: '/data/thumbnails/art-dom-complete.webp',
      medium: 'Wax crayon on paper',
    });

    const workDominantIncomplete = gallery.Artwork.create({
      id: 'art-dom-incomplete',
      title: 'Untitled',
      description: '',
      themeId: 'Kinetic Scribblism',
      timestamp: 5000,
      imagePath: '/placeholder.svg',
      thumbnailPath: '/placeholder.svg',
      medium: 'Mixed Media',
    });

    const workOtherComplete = gallery.Artwork.create({
      id: 'art-other-complete',
      title: 'Void Study IV',
      description: 'Dense graphite coverage.',
      themeId: 'Monochrome Nihilism',
      timestamp: 9000,
      imagePath: '/data/images/art-other-complete.jpg',
      thumbnailPath: '/data/thumbnails/art-other-complete.webp',
      medium: 'Graphite on board',
    });

    for (const art of [workDominantComplete, workDominantIncomplete, workOtherComplete]) {
      fs.writeFileSync(
        path.join(testDir, `${art.id}.proto.bin`),
        Buffer.from(gallery.Artwork.encode(art).finish())
      );
    }

    const outputPath = generateCatalog(testDir);
    const catalog: CatalogArtwork[] = JSON.parse(fs.readFileSync(outputPath, 'utf8'));

    expect(catalog).toHaveLength(3);
    const rep = catalog.filter((a) => a.is_representative);
    expect(rep).toHaveLength(1);
    expect(rep[0].id).toBe('art-dom-complete');
    expect(catalog.find((a) => a.id === 'art-dom-incomplete')?.is_representative).toBe(false);
    expect(catalog.find((a) => a.id === 'art-other-complete')?.is_representative).toBe(false);
  });

  it('should break ties deterministically by earliest canonical movement index, positive timestamp descending, and id ascending', () => {
    // 1 work in The Crayon Period (canonical index 0) and 1 work in Domestic Destructionism (canonical index 1)
    // Both have identical completeness and identical positive timestamps; The Crayon Period wins dominant movement tie-breaker.
    const artCrayon = gallery.Artwork.create({
      id: 'art-z',
      title: 'Red Spiral',
      description: 'Early wax composition',
      themeId: 'The Crayon Period',
      timestamp: 2000,
      imagePath: '/data/images/art-z.jpg',
      thumbnailPath: '/data/thumbnails/art-z.webp',
      medium: 'Crayon',
    });

    const artDestruction = gallery.Artwork.create({
      id: 'art-a',
      title: 'Torn Wallpaper',
      description: 'Hallway intervention',
      themeId: 'Domestic Destructionism',
      timestamp: 2000,
      imagePath: '/data/images/art-a.jpg',
      thumbnailPath: '/data/thumbnails/art-a.webp',
      medium: 'Wallpaper and paste',
    });

    fs.writeFileSync(
      path.join(testDir, 'art-z.proto.bin'),
      Buffer.from(gallery.Artwork.encode(artCrayon).finish())
    );
    fs.writeFileSync(
      path.join(testDir, 'art-a.proto.bin'),
      Buffer.from(gallery.Artwork.encode(artDestruction).finish())
    );

    let outputPath = generateCatalog(testDir);
    let catalog: CatalogArtwork[] = JSON.parse(fs.readFileSync(outputPath, 'utf8'));

    // Deterministic sort tie-breaker by id ascending when timestamps are equal
    expect(catalog.map((a) => a.id)).toEqual(['art-a', 'art-z']);
    // art-z wins representative designation because The Crayon Period is earlier in canonical chronology
    expect(catalog.find((a) => a.id === 'art-z')?.is_representative).toBe(true);
    expect(catalog.find((a) => a.id === 'art-a')?.is_representative).toBe(false);

    // Now test tie-breaking when scores and timestamps are equal (id ascending) vs <= 0 timestamp
    const tieDir = fs.mkdtempSync(path.join(os.tmpdir(), 'rose-catalog-tie-'));
    try {
      const artZeroTs = gallery.Artwork.create({
        id: 'art-0',
        title: 'Untitled',
        themeId: 'Early Works',
        timestamp: 0,
        medium: 'Mixed Media',
      });
      const artPositiveB = gallery.Artwork.create({
        id: 'art-b',
        title: 'Untitled',
        themeId: 'Early Works',
        timestamp: 1500,
        medium: 'Mixed Media',
      });
      const artPositiveA = gallery.Artwork.create({
        id: 'art-a',
        title: 'Untitled',
        themeId: 'Early Works',
        timestamp: 1500,
        medium: 'Mixed Media',
      });

      for (const art of [artZeroTs, artPositiveB, artPositiveA]) {
        fs.writeFileSync(
          path.join(tieDir, `${art.id}.proto.bin`),
          Buffer.from(gallery.Artwork.encode(art).finish())
        );
      }

      outputPath = generateCatalog(tieDir);
      catalog = JSON.parse(fs.readFileSync(outputPath, 'utf8'));

      expect(catalog.map((a) => a.id)).toEqual(['art-a', 'art-b', 'art-0']);
      expect(catalog.find((a) => a.id === 'art-a')?.is_representative).toBe(true);
      expect(catalog.find((a) => a.id === 'art-b')?.is_representative).toBe(false);
      expect(catalog.find((a) => a.id === 'art-0')?.is_representative).toBe(false);
    } finally {
      fs.rmSync(tieDir, { recursive: true, force: true });
    }
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
