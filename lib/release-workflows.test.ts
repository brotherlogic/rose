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
  on?: {
    push?: {
      branches?: string[];
      'paths-ignore'?: string[];
      tags?: string[];
    };
  };
  permissions?: {
    contents?: string;
    packages?: string;
  };
  jobs?: Record<string, WorkflowJob>;
}

describe('Release Automation Workflows', () => {
  const rootDir = path.resolve(__dirname, '..');
  const tagWorkflowPath = path.join(rootDir, '.github', 'workflows', 'tag.yml');
  const uploadWorkflowPath = path.join(rootDir, '.github', 'workflows', 'upload.yml');

  describe('Semantic Version Tagging Workflow (.github/workflows/tag.yml)', () => {
    it('should exist', () => {
      expect(fs.existsSync(tagWorkflowPath)).toBe(true);
    });

    it('should be valid YAML syntax', () => {
      const content = fs.readFileSync(tagWorkflowPath, 'utf-8');
      const parsed = yaml.load(content);
      expect(parsed).toBeDefined();
      expect(typeof parsed).toBe('object');
    });

    it('should trigger on push to main branch with paths-ignore for markdown files', () => {
      const content = fs.readFileSync(tagWorkflowPath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      expect(parsed.on).toBeDefined();
      expect(parsed.on?.push).toBeDefined();
      expect(parsed.on?.push?.branches).toContain('main');
      expect(parsed.on?.push?.['paths-ignore']).toContain('**.md');
    });

    it('should checkout code using actions/checkout@v4 with fetch-depth 0', () => {
      const content = fs.readFileSync(tagWorkflowPath, 'utf-8');
      expect(content).toMatch(/uses:\s*actions\/checkout@v4/);
      expect(content).toMatch(/fetch-depth:\s*['"]?0['"]?/);
    });

    it('should use anothrNick/github-tag-action@1.55.0 with PERSONAL_TOKEN and WITH_V', () => {
      const content = fs.readFileSync(tagWorkflowPath, 'utf-8');
      expect(content).toMatch(/uses:\s*anothrNick\/github-tag-action@1\.55\.0/);
      expect(content).toMatch(/GITHUB_TOKEN:\s*\$\{\{\s*secrets\.PERSONAL_TOKEN\s*\}\}/);
      expect(content).toMatch(/WITH_V:\s*true/);
    });
  });

  describe('Release GHCR Publishing Workflow (.github/workflows/upload.yml)', () => {
    it('should exist', () => {
      expect(fs.existsSync(uploadWorkflowPath)).toBe(true);
    });

    it('should be valid YAML syntax', () => {
      const content = fs.readFileSync(uploadWorkflowPath, 'utf-8');
      const parsed = yaml.load(content);
      expect(parsed).toBeDefined();
      expect(typeof parsed).toBe('object');
    });

    it('should trigger on push of release tags', () => {
      const content = fs.readFileSync(uploadWorkflowPath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      expect(parsed.on).toBeDefined();
      expect(parsed.on?.push).toBeDefined();
      expect(parsed.on?.push?.tags).toContain('*');
    });

    it('should have required permissions (contents: read, packages: write)', () => {
      const content = fs.readFileSync(uploadWorkflowPath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      const job = parsed.jobs ? Object.values(parsed.jobs)[0] : undefined;
      const permissions = parsed.permissions || job?.permissions;
      expect(permissions).toBeDefined();
      expect(permissions?.contents).toBe('read');
      expect(permissions?.packages).toBe('write');
    });

    it('should configure checkout, QEMU, and Buildx actions', () => {
      const content = fs.readFileSync(uploadWorkflowPath, 'utf-8');
      expect(content).toMatch(/uses:\s*actions\/checkout@v4/);
      expect(content).toMatch(/uses:\s*docker\/setup-qemu-action@v3/);
      expect(content).toMatch(/uses:\s*docker\/setup-buildx-action@v3/);
    });

    it('should extract Docker metadata targeting ghcr.io/brotherlogic/rose-gallery', () => {
      const content = fs.readFileSync(uploadWorkflowPath, 'utf-8');
      expect(content).toMatch(/uses:\s*docker\/metadata-action@v5/);
      expect(content).toContain('ghcr.io/brotherlogic/rose-gallery');
    });

    it('should log in to GHCR with docker/login-action@v3 and GITHUB_TOKEN', () => {
      const content = fs.readFileSync(uploadWorkflowPath, 'utf-8');
      expect(content).toMatch(/uses:\s*docker\/login-action@v3/);
      expect(content).toMatch(/registry:\s*ghcr\.io/);
      expect(content).toMatch(/password:\s*\$\{\{\s*secrets\.GITHUB_TOKEN\s*\}\}/);
    });

    it('should build and push multi-architecture image (linux/amd64 and linux/arm64)', () => {
      const content = fs.readFileSync(uploadWorkflowPath, 'utf-8');
      expect(content).toMatch(/uses:\s*docker\/build-push-action@v5/);
      expect(content).toMatch(/push:\s*true/);
      expect(content).toMatch(/platforms:\s*.*linux\/amd64.*linux\/arm64|platforms:\s*.*linux\/arm64.*linux\/amd64/);
    });
  });
});
