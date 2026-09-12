import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';

describe('Background Syncer Containerization Configuration (Dockerfile.syncer)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const dockerfilePath = path.join(rootDir, 'Dockerfile.syncer');

  it('should exist in the repository root', () => {
    expect(fs.existsSync(dockerfilePath)).toBe(true);
  });

  describe('Builder Stage', () => {
    it('should use golang:1.25-alpine as the builder base image', () => {
      const content = fs.readFileSync(dockerfilePath, 'utf-8');
      expect(content).toMatch(/FROM\s+golang:1\.25-alpine\s+AS\s+builder/i);
    });

    it('should install build prerequisites ca-certificates', () => {
      const content = fs.readFileSync(dockerfilePath, 'utf-8');
      expect(content).toMatch(/apk\s+add\s+.*ca-certificates/);
    });

    it('should copy go module definitions and source directories', () => {
      const content = fs.readFileSync(dockerfilePath, 'utf-8');
      expect(content).toMatch(/COPY\s+go\.mod\s+go\.sum/);
      expect(content).toMatch(/COPY\s+proto/);
      expect(content).toMatch(/COPY\s+internal/);
      expect(content).toMatch(/COPY\s+cmd/);
    });

    it('should compile static binary without cgo and with trimpath and stripped symbols', () => {
      const content = fs.readFileSync(dockerfilePath, 'utf-8');
      expect(content).toMatch(/CGO_ENABLED=0/);
      expect(content).toMatch(/GOOS=linux/);
      expect(content).toMatch(/go\s+build/);
      expect(content).toMatch(/-trimpath/);
      expect(content).toMatch(/-ldflags=["']-s\s+-w["']/);
      expect(content).toMatch(/-o\s+\/bin\/syncer/);
      expect(content).toMatch(/\.\/cmd\/syncer/);
    });
  });

  describe('Runtime Stage', () => {
    it('should use gcr.io/distroless/static:nonroot as runtime base image', () => {
      const content = fs.readFileSync(dockerfilePath, 'utf-8');
      expect(content).toMatch(/FROM\s+gcr\.io\/distroless\/static:nonroot/i);
    });

    it('should set working directory to /data', () => {
      const content = fs.readFileSync(dockerfilePath, 'utf-8');
      expect(content).toMatch(/WORKDIR\s+\/data/);
    });

    it('should copy syncer binary from builder stage', () => {
      const content = fs.readFileSync(dockerfilePath, 'utf-8');
      expect(content).toMatch(/COPY\s+--from=builder\s+\/bin\/syncer\s+\/syncer/);
    });

    it('should execute as nonroot user', () => {
      const content = fs.readFileSync(dockerfilePath, 'utf-8');
      expect(content).toMatch(/USER\s+(nonroot:nonroot|65532:65532)/);
    });

    it('should configure ENTRYPOINT to ["/syncer"]', () => {
      const content = fs.readFileSync(dockerfilePath, 'utf-8');
      expect(content).toMatch(/ENTRYPOINT\s+\[\s*"\/syncer"\s*\]/);
    });
  });
});
