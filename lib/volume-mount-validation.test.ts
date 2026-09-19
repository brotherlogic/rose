import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';

describe('Volume Mount & Security Context Validation (#59)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const pvcManifestPath = path.join(rootDir, 'deploy', 'rose-data-pvc.yaml');
  const secretManifestPath = path.join(rootDir, 'deploy', 'rose-gcp-sa-secret.yaml');
  const smokeTestScriptPath = path.join(rootDir, 'scripts', 'smoke-test-volume-permissions.sh');

  describe('Cross-Link Infrastructure Tracking Issues', () => {
    it('should cross-reference brotherlogic/prod#968 and parent issues in deploy/rose-data-pvc.yaml', () => {
      const content = fs.readFileSync(pvcManifestPath, 'utf-8');
      expect(content).toContain('brotherlogic/prod#968');
      expect(content).toContain('brotherlogic/rose#29');
      expect(content).toContain('brotherlogic/rose#55');
      expect(content).toContain('brotherlogic/rose#56');
    });

    it('should cross-reference brotherlogic/prod#969 and parent issues in deploy/rose-gcp-sa-secret.yaml', () => {
      const content = fs.readFileSync(secretManifestPath, 'utf-8');
      expect(content).toContain('brotherlogic/prod#969');
      expect(content).toContain('brotherlogic/rose#29');
      expect(content).toContain('brotherlogic/rose#55');
      expect(content).toContain('brotherlogic/rose#56');
    });
  });

  describe('Security Context & Permissions Contract', () => {
    it('should align PVC fsGroup with syncer container nonroot user 65532', () => {
      const pvcContent = fs.readFileSync(pvcManifestPath, 'utf-8');
      expect(pvcContent).toContain('fsGroup: 65532');

      const dockerfilePath = path.join(rootDir, 'Dockerfile.syncer');
      const dockerfileContent = fs.readFileSync(dockerfilePath, 'utf-8');
      expect(dockerfileContent).toMatch(/USER\s+(nonroot:nonroot|65532:65532)/);
      expect(dockerfileContent).toContain('WORKDIR /data');
    });

    it('should specify read-only contract and credentials path in GCP Secret manifest', () => {
      const secretContent = fs.readFileSync(secretManifestPath, 'utf-8');
      expect(secretContent).toContain('/etc/secrets/gcp/credentials.json');
      expect(secretContent).toContain('readOnly: true');
      expect(secretContent).toContain('GOOGLE_APPLICATION_CREDENTIALS=/etc/secrets/gcp/credentials.json');
    });
  });

  describe('Smoke Test Script for Volume Permissions', () => {
    it('should provide an executable smoke test script in scripts/smoke-test-volume-permissions.sh', () => {
      expect(fs.existsSync(smokeTestScriptPath)).toBe(true);
      const stats = fs.statSync(smokeTestScriptPath);
      // Ensure file has execute permissions
      expect(stats.mode & 0o111).toBeGreaterThan(0);
    });

    it('should implement nonroot UID 65532 chown and .sync-state.json verification in smoke test script', () => {
      const scriptContent = fs.readFileSync(smokeTestScriptPath, 'utf-8');
      expect(scriptContent).toContain('65532:65532');
      expect(scriptContent).toContain('.sync-state.json');
      expect(scriptContent).toContain('/data');
    });
  });
});
