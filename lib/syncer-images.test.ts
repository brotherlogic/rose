import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
// @ts-expect-error js-yaml types
import * as yaml from 'js-yaml';

interface ResourceManifest {
  apiVersion?: string;
  kind?: string;
  metadata?: {
    name?: string;
    namespace?: string;
    annotations?: Record<string, string>;
  };
  spec?: {
    image?: string;
    interval?: string;
    imageRepositoryRef?: {
      name?: string;
    };
    policy?: {
      semver?: {
        range?: string;
      };
    };
  };
}

describe('Rose Syncer Flux Image Automation Turnup Specification (#93)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const imagesManifestPath = path.join(rootDir, 'deploy', 'rose-images.yaml');
  const cronJobManifestPath = path.join(rootDir, 'deploy', 'rose-syncer-cronjob.yaml');

  describe('Manifest File Existence & Cross-References', () => {
    it('should have deploy/rose-images.yaml with cross-references to parent issues #79, #85, #92', () => {
      expect(fs.existsSync(imagesManifestPath)).toBe(true);
      const content = fs.readFileSync(imagesManifestPath, 'utf-8');
      expect(content).toContain('brotherlogic/rose#79');
      expect(content).toContain('brotherlogic/rose#85');
      expect(content).toContain('brotherlogic/rose#92');
    });
  });

  describe('Flux CD Image Automation Manifests for rose-syncer', () => {
    it('should declare ImageRepository for rose-syncer matching Flux CD v1 schema', () => {
      const content = fs.readFileSync(imagesManifestPath, 'utf-8');
      const docs = yaml.loadAll(content) as ResourceManifest[];

      const imageRepo = docs.find((d) => d?.kind === 'ImageRepository' && d?.metadata?.name === 'rose-syncer');
      expect(imageRepo).toBeDefined();
      expect(imageRepo?.apiVersion).toBe('image.toolkit.fluxcd.io/v1');
      expect(imageRepo?.metadata?.namespace).toBe('flux-system');
      expect(imageRepo?.spec?.image).toBe('ghcr.io/brotherlogic/rose-syncer');
      expect(imageRepo?.spec?.interval).toBe('1m0s');
    });

    it('should declare ImagePolicy for rose-syncer matching Flux CD v1 schema', () => {
      const content = fs.readFileSync(imagesManifestPath, 'utf-8');
      const docs = yaml.loadAll(content) as ResourceManifest[];

      const imagePolicy = docs.find((d) => d?.kind === 'ImagePolicy' && d?.metadata?.name === 'rose-syncer');
      expect(imagePolicy).toBeDefined();
      expect(imagePolicy?.apiVersion).toBe('image.toolkit.fluxcd.io/v1');
      expect(imagePolicy?.metadata?.namespace).toBe('flux-system');
      expect(imagePolicy?.spec?.imageRepositoryRef?.name).toBe('rose-syncer');
      expect(imagePolicy?.spec?.policy?.semver?.range).toBe('0.x.0');
    });

    it('should maintain existing rose-gallery manifests', () => {
      const content = fs.readFileSync(imagesManifestPath, 'utf-8');
      const docs = yaml.loadAll(content) as ResourceManifest[];

      const galleryRepo = docs.find((d) => d?.kind === 'ImageRepository' && d?.metadata?.name === 'rose-gallery');
      expect(galleryRepo).toBeDefined();
      const galleryPolicy = docs.find((d) => d?.kind === 'ImagePolicy' && d?.metadata?.name === 'rose-gallery');
      expect(galleryPolicy).toBeDefined();
    });

    it('should confirm alignment with setter annotation in deploy/rose-syncer-cronjob.yaml', () => {
      const cronJobContent = fs.readFileSync(cronJobManifestPath, 'utf-8');
      expect(cronJobContent).toContain('{"$imagepolicy": "flux-system:rose-syncer"}');
    });
  });
});
