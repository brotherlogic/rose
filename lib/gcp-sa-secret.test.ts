import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
// @ts-expect-error js-yaml types
import * as yaml from 'js-yaml';

interface SecretDefinition {
  apiVersion?: string;
  kind?: string;
  metadata?: {
    name?: string;
    namespace?: string;
  };
  type?: string;
  stringData?: Record<string, string>;
}

describe('GCP Service Account Secret Turnup Specification (#58)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const secretManifestPath = path.join(rootDir, 'deploy', 'rose-gcp-sa-secret.yaml');

  it('should have a dedicated deployment manifest file deploy/rose-gcp-sa-secret.yaml', () => {
    expect(fs.existsSync(secretManifestPath)).toBe(true);
  });

  describe('Manifest Schema and Values', () => {
    it('should be valid YAML matching Kubernetes Secret specification', () => {
      const content = fs.readFileSync(secretManifestPath, 'utf-8');
      const parsed = yaml.load(content) as SecretDefinition;

      expect(parsed.apiVersion).toBe('v1');
      expect(parsed.kind).toBe('Secret');
      expect(parsed.metadata?.name).toBe('rose-gcp-sa-secret');
      expect(parsed.metadata?.namespace).toBe('default');
      expect(parsed.type).toBe('Opaque');
      expect(parsed.stringData).toHaveProperty('credentials.json');
    });
  });

  describe('Turnup Contract Specifications', () => {
    it('should document mount path, read-only contract, and GOOGLE_APPLICATION_CREDENTIALS', () => {
      const content = fs.readFileSync(secretManifestPath, 'utf-8');
      expect(content).toContain('/etc/secrets/gcp/credentials.json');
      expect(content).toContain('GOOGLE_APPLICATION_CREDENTIALS=/etc/secrets/gcp/credentials.json');
      expect(content).toContain('brotherlogic/prod#969');
    });

    it('should document required GCP IAM API access and fail-fast requirements', () => {
      const content = fs.readFileSync(secretManifestPath, 'utf-8');
      expect(content).toContain('Google Photos Library API');
      expect(content).toContain('Google Cloud Vision API');
      expect(content).toContain('fail-fast');
    });
  });
});
