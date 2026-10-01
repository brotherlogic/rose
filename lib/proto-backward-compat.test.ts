import { describe, it, expect } from 'vitest';
import { gallery } from './proto/gallery';

describe('gallery.Artwork TypeScript Protobuf Compatibility', () => {
  it('should encode and decode medium correctly', () => {
    const original = gallery.Artwork.create({
      id: 'art-1',
      title: 'Oil on Canvas',
      medium: 'Huile sur toile, rehauts d’or',
    });
    const buffer = gallery.Artwork.encode(original).finish();
    const decoded = gallery.Artwork.decode(buffer);
    expect(decoded.medium).toBe('Huile sur toile, rehauts d’or');
  });

  it('should decode legacy payload without medium gracefully', () => {
    const legacy = gallery.Artwork.create({
      id: 'legacy-1',
      title: 'Legacy Artwork',
    });
    const buffer = gallery.Artwork.encode(legacy).finish();
    const decoded = gallery.Artwork.decode(buffer);
    expect(decoded.medium).toBe('');
  });

  it('should fail cleanly on corrupt binary input', () => {
    const corrupt = new Uint8Array([0xff, 0xff, 0xff, 0xff]);
    expect(() => gallery.Artwork.decode(corrupt)).toThrow();
  });
});
