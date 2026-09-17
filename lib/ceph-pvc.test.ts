import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
// @ts-expect-error js-yaml types
import * as yaml from 'js-yaml';

interface PVCDefinition {
  apiVersion?: string;
  kind?: string;
  metadata?: {
    name?: string;
    namespace?: string;
  };
  spec?: {
    accessModes?: string[];
    storageClassName?: string;
    resources?: {
      requests?: {
        storage?: string;
      };
    };
  };
}

describe('Ceph PersistentVolumeClaim Turnup Specification (#57)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const pvcManifestPath = path.join(rootDir, 'deploy', 'rose-data-pvc.yaml');

  it('should have a dedicated deployment manifest file deploy/rose-data-pvc.yaml', () => {
    expect(fs.existsSync(pvcManifestPath)).toBe(true);
  });

  describe('Manifest Schema and Values', () => {
    it('should be valid YAML matching Kubernetes PersistentVolumeClaim specification', () => {
      const content = fs.readFileSync(pvcManifestPath, 'utf-8');
      const parsed = yaml.load(content) as PVCDefinition;

      expect(parsed.apiVersion).toBe('v1');
      expect(parsed.kind).toBe('PersistentVolumeClaim');
      expect(parsed.metadata?.name).toBe('rose-data-pvc');
      expect(parsed.metadata?.namespace).toBe('default');
      expect(parsed.spec?.accessModes).toContain('ReadWriteOnce');
      expect(parsed.spec?.storageClassName).toBe('rook-ceph-block');
      expect(parsed.spec?.resources?.requests?.storage).toBe('20Gi');
    });
  });

  describe('Turnup Contract Specifications', () => {
    it('should document container mount path and nonroot fsGroup contract matching syncer container', () => {
      const content = fs.readFileSync(pvcManifestPath, 'utf-8');
      expect(content).toContain('mountPath: /data');
      expect(content).toContain('fsGroup: 65532');
      expect(content).toContain('brotherlogic/prod#968');
    });
  });
});
