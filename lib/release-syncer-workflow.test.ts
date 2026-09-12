import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
// @ts-expect-error js-yaml types
import * as yaml from 'js-yaml';

interface WorkflowJob {
  permissions?: {
    contents?: string;
    packages?: string;
  };
}

interface WorkflowDefinition {
  name?: string;
  on?: {
    push?: {
      tags?: string[];
    };
  };
  permissions?: {
    contents?: string;
    packages?: string;
  };
  jobs?: Record<string, WorkflowJob>;
}

describe('Background Syncer Release Workflow (.github/workflows/upload-syncer.yml)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const uploadSyncerWorkflowPath = path.join(rootDir, '.github', 'workflows', 'upload-syncer.yml');

  it('should exist in .github/workflows/upload-syncer.yml', () => {
    expect(fs.existsSync(uploadSyncerWorkflowPath)).toBe(true);
  });

  describe('Workflow Configuration', () => {
    it('should be valid YAML syntax', () => {
      const content = fs.readFileSync(uploadSyncerWorkflowPath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      expect(parsed).toBeDefined();
      expect(typeof parsed).toBe('object');
    });

    it('should trigger on push of tags', () => {
      const content = fs.readFileSync(uploadSyncerWorkflowPath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      expect(parsed.on).toBeDefined();
      expect(parsed.on?.push).toBeDefined();
      expect(parsed.on?.push?.tags).toContain('*');
    });

    it('should configure required permissions (contents: read, packages: write)', () => {
      const content = fs.readFileSync(uploadSyncerWorkflowPath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      const job = parsed.jobs ? Object.values(parsed.jobs)[0] : undefined;
      const permissions = parsed.permissions || job?.permissions;
      expect(permissions).toBeDefined();
      expect(permissions?.contents).toBe('read');
      expect(permissions?.packages).toBe('write');
    });

    it('should configure actions/checkout@v4 step', () => {
      const content = fs.readFileSync(uploadSyncerWorkflowPath, 'utf-8');
      expect(content).toMatch(/uses:\s*actions\/checkout@v4/);
    });

    it('should setup QEMU and Docker Buildx', () => {
      const content = fs.readFileSync(uploadSyncerWorkflowPath, 'utf-8');
      expect(content).toMatch(/uses:\s*docker\/setup-qemu-action@v3/);
      expect(content).toMatch(/uses:\s*docker\/setup-buildx-action@v3/);
    });

    it('should extract Docker metadata targeting ghcr.io/brotherlogic/rose-syncer', () => {
      const content = fs.readFileSync(uploadSyncerWorkflowPath, 'utf-8');
      expect(content).toMatch(/uses:\s*docker\/metadata-action@v5/);
      expect(content).toContain('ghcr.io/brotherlogic/rose-syncer');
    });

    it('should authenticate to GHCR using docker/login-action@v3', () => {
      const content = fs.readFileSync(uploadSyncerWorkflowPath, 'utf-8');
      expect(content).toMatch(/uses:\s*docker\/login-action@v3/);
      expect(content).toMatch(/registry:\s*ghcr\.io/);
      expect(content).toMatch(/password:\s*\$\{\{\s*secrets\.GITHUB_TOKEN\s*\}\}/);
    });

    it('should build and push multi-architecture image using Dockerfile.syncer', () => {
      const content = fs.readFileSync(uploadSyncerWorkflowPath, 'utf-8');
      expect(content).toMatch(/uses:\s*docker\/build-push-action@v5/);
      expect(content).toMatch(/file:\s*Dockerfile\.syncer/);
      expect(content).toMatch(/push:\s*true/);
      expect(content).toMatch(/tags:\s*\$\{\{\s*steps\.meta\.outputs\.tags\s*\}\}/);
      expect(content).toMatch(/platforms:\s*.*linux\/amd64.*linux\/arm64|platforms:\s*.*linux\/arm64.*linux\/amd64/);
    });
  });
});
