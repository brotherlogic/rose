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
      paths?: string[];
    };
  };
  jobs?: Record<string, WorkflowJob>;
}

describe('PR Validation Workflow (.github/workflows/validate-syncer-pr.yml)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const workflowPath = path.join(rootDir, '.github', 'workflows', 'validate-syncer-pr.yml');

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

    it('should trigger on pull_request against main branch with specific path filters', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      expect(parsed.on).toBeDefined();
      expect(parsed.on?.pull_request).toBeDefined();
      expect(parsed.on?.pull_request?.branches).toContain('main');
      expect(parsed.on?.pull_request?.paths).toEqual(
        expect.arrayContaining([
          'cmd/syncer/**',
          'internal/**',
          'proto/**',
          'Dockerfile.syncer',
          '.github/workflows/validate-syncer-pr.yml',
        ])
      );
    });

    it('should checkout code using actions/checkout@v4', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      expect(content).toMatch(/uses:\s*actions\/checkout@v4/);
    });

    it('should setup Docker Buildx using docker/setup-buildx-action@v3', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      expect(content).toMatch(/uses:\s*docker\/setup-buildx-action@v3/);
    });

    it('should build container image locally with Dockerfile.syncer and github.sha tag', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      expect(content).toMatch(/docker\s+build\s+-f\s+Dockerfile\.syncer\s+-t\s+rose-syncer-test:\$\{\{\s*github\.sha\s*\}\}\s+\./);
    });

    it('should execute smoke-test run against temporary directory mounted to /data', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      expect(content).toMatch(/mktemp\s+-d/);
      expect(content).toMatch(/chmod\s+777/);
      expect(content).toMatch(/docker\s+run\s+--rm\s+-v\s+.*:\/data.*rose-syncer-test:\$\{\{\s*github\.sha\s*\}\}\s+--storage-path\s+\/data/);
      expect(content).toMatch(/test\s+-f\s+.*\.sync-state\.json/);
      expect(content).toMatch(/rm\s+-rf/);
    });

    it('should strictly ensure no container images are pushed during PR validation', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      expect(content).not.toMatch(/docker\s+push/);
      expect(content).not.toMatch(/ghcr\.io/);
    });
  });

  describe('Smoke Test Execution Script', () => {
    it('should execute smoke test and verify sync-state file creation', () => {
      const content = fs.readFileSync(workflowPath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      const job = Object.values(parsed.jobs || {})[0];
      const smokeStep = job?.steps?.find((s) => s.run && s.run.includes('mktemp'));
      expect(smokeStep).toBeDefined();

      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'smoke-test-mock-'));
      try {
        fs.writeFileSync(
          path.join(tmpDir, 'docker'),
          '#!/bin/sh\nfor arg in "$@"; do case "$arg" in *:/data) hostdir="${arg%%:/data}"; touch "$hostdir/.sync-state.json";; esac; done\nexit 0\n',
          { mode: 0o755 }
        );

        const script = (smokeStep?.run || '').replace(/\$\{\{\s*github\.sha\s*\}\}/g, '1234567890abcdef');
        cp.execSync(script, {
          env: { ...process.env, PATH: tmpDir + ':' + process.env.PATH },
          shell: '/bin/bash',
        });
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });
  });
});
