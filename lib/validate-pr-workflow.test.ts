import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
import * as cp from 'child_process';
import * as os from 'os';
// @ts-expect-error js-yaml types
import * as yaml from 'js-yaml';

interface WorkflowStep {
  name?: string;
  uses?: string;
  run?: string;
}

interface WorkflowJob {
  'runs-on'?: string;
  steps?: WorkflowStep[];
}

interface WorkflowDefinition {
  name?: string;
  on?: {
    pull_request?: {
      branches?: string[];
    };
  };
  jobs?: Record<string, WorkflowJob>;
}

describe('PR Validation Workflow (.github/workflows/validate-pr.yml)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const workflowPath = path.join(rootDir, '.github', 'workflows', 'validate-pr.yml');

  it('should exist', () => {
    expect(fs.existsSync(workflowPath)).toBe(true);
  });

  describe('Workflow Configuration', () => {
    it('should be valid YAML syntax', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      expect(parsed).toBeDefined();
      expect(typeof parsed).toBe('object');
    });

    it('should trigger on pull_request against main branch', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      expect(parsed.on).toBeDefined();
      expect(parsed.on?.pull_request).toBeDefined();
      expect(parsed.on?.pull_request?.branches).toContain('main');
    });

    it('should checkout code using actions/checkout@v4', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      expect(content).toMatch(/uses:\s*actions\/checkout@v4/);
    });

    it('should build container image locally with github.sha tag', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      expect(content).toMatch(/docker\s+build\s+-t\s+rose-gallery-test:\$\{\{\s*github\.sha\s*\}\}\s+\./);
    });

    it('should run container detached on port 8080', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      expect(content).toMatch(/docker\s+run\s+-d\s+--name\s+test-gallery\s+-p\s+8080:80\s+rose-gallery-test:\$\{\{\s*github\.sha\s*\}\}/);
    });

    it('should implement healthcheck polling loop with 15 attempts and 2-second sleep', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      expect(content).toMatch(/15/);
      expect(content).toMatch(/sleep\s+2/);
      expect(content).toMatch(/curl\s+.*http:\/\/localhost:8080\//);
      expect(content).toContain('Artist Portfolio');
    });

    it('should handle cleanup on success and failure', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      expect(content).toMatch(/docker\s+logs\s+test-gallery/);
      expect(content).toMatch(/docker\s+stop\s+test-gallery/);
      expect(content).toMatch(/docker\s+rm\s+test-gallery/);
    });

    it('should strictly ensure no container images are pushed during PR validation', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      expect(content).not.toMatch(/docker\s+push/);
      expect(content).not.toMatch(/ghcr\.io/);
    });
  });

  describe('Healthcheck Script Execution Logic', () => {
    it('should pass on successful healthcheck response and clean up container', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      const job = Object.values(parsed.jobs || {})[0];
      const healthcheckStep = job?.steps?.find((s) => s.run && s.run.includes('curl'));
      expect(healthcheckStep).toBeDefined();

      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'healthcheck-test-success-'));
      try {
        fs.writeFileSync(path.join(tmpDir, 'docker'), '#!/bin/sh\necho docker "$@" >> ' + path.join(tmpDir, 'trace.log') + '\n', { mode: 0o755 });
        fs.writeFileSync(path.join(tmpDir, 'curl'), '#!/bin/sh\necho "Artist Portfolio"\nexit 0\n', { mode: 0o755 });

        cp.execSync(healthcheckStep?.run || '', {
          env: { ...process.env, PATH: tmpDir + ':' + process.env.PATH },
          shell: '/bin/bash',
        });

        const trace = fs.readFileSync(path.join(tmpDir, 'trace.log'), 'utf-8');
        expect(trace).toContain('docker stop test-gallery');
        expect(trace).toContain('docker rm test-gallery');
        expect(trace).not.toContain('docker logs');
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it('should timeout on failed healthcheck, capture docker logs, and clean up container', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      const job = Object.values(parsed.jobs || {})[0];
      const healthcheckStep = job?.steps?.find((s) => s.run && s.run.includes('curl'));
      expect(healthcheckStep).toBeDefined();

      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'healthcheck-test-fail-'));
      try {
        fs.writeFileSync(path.join(tmpDir, 'docker'), '#!/bin/sh\necho docker "$@" >> ' + path.join(tmpDir, 'trace.log') + '\n', { mode: 0o755 });
        fs.writeFileSync(path.join(tmpDir, 'curl'), '#!/bin/sh\nexit 1\n', { mode: 0o755 });
        // Fast sleep for testing loop without waiting 30 seconds
        fs.writeFileSync(path.join(tmpDir, 'sleep'), '#!/bin/sh\nexit 0\n', { mode: 0o755 });

        let failed = false;
        try {
          cp.execSync(healthcheckStep?.run || '', {
            env: { ...process.env, PATH: tmpDir + ':' + process.env.PATH },
            shell: '/bin/bash',
          });
        } catch {
          failed = true;
        }

        expect(failed).toBe(true);
        const trace = fs.readFileSync(path.join(tmpDir, 'trace.log'), 'utf-8');
        expect(trace).toContain('docker logs test-gallery');
        expect(trace).toContain('docker stop test-gallery');
        expect(trace).toContain('docker rm test-gallery');
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });
  });
});
