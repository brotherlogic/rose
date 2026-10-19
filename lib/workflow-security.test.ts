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
  env?: Record<string, string>;
}

interface WorkflowJob {
  if?: string;
  'runs-on'?: string;
  steps?: WorkflowStep[];
}

interface WorkflowDefinition {
  name?: string;
  jobs?: Record<string, WorkflowJob>;
}

describe('GitHub Actions Workflow Security Guards', () => {
  const rootDir = path.resolve(__dirname, '..');
  const geminiReviewPath = path.join(rootDir, '.github', 'workflows', 'gemini-review.yml');
  const automergePath = path.join(rootDir, '.github', 'workflows', 'automerge.yml');
  const assignReviewerPath = path.join(rootDir, '.github', 'workflows', 'assign-reviewer.yml');

  describe('Gemini Review Workflow (.github/workflows/gemini-review.yml)', () => {
    it('should restrict comment trigger to OWNER, COLLABORATOR, or MEMBER author_association', () => {
      const content = fs.readFileSync(geminiReviewPath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      const job = parsed.jobs?.['gemini-review'];
      expect(job).toBeDefined();
      expect(job?.if).toBeDefined();
      expect(job?.if).toContain('github.event.issue.pull_request');
      expect(job?.if).toContain("contains(github.event.comment.body, '/gemini-review')");
      expect(job?.if).toContain('github.event.comment.author_association');
      expect(job?.if).toContain('OWNER');
      expect(job?.if).toContain('COLLABORATOR');
      expect(job?.if).toContain('MEMBER');
    });
  });

  describe('Assign Reviewer Workflow (.github/workflows/assign-reviewer.yml)', () => {
    it('should restrict workflow_run execution to trusted actors (brotherlogic or brotherlogic-automation)', () => {
      const content = fs.readFileSync(assignReviewerPath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      const job = parsed.jobs?.['assign'];
      expect(job).toBeDefined();
      expect(job?.if).toBeDefined();
      expect(job?.if).toContain("github.event.workflow_run.conclusion == 'success'");
      expect(job?.if).toContain('github.event.workflow_run.actor.login');
      expect(job?.if).toContain('brotherlogic');
      expect(job?.if).toContain('brotherlogic-automation');
    });
  });

  describe('Auto Merge Workflow (.github/workflows/automerge.yml)', () => {
    it('should have a job-level condition restricting to trusted actors or collaborator associations', () => {
      const content = fs.readFileSync(automergePath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      const job = parsed.jobs?.['enable-automerge'];
      expect(job).toBeDefined();
      expect(job?.if).toBeDefined();
      expect(job?.if).toContain('brotherlogic');
      expect(job?.if).toContain('brotherlogic-automation');
    });

    it('should verify PR author and headRepositoryOwner via gh pr view before enabling auto-merge', () => {
      const content = fs.readFileSync(automergePath, 'utf-8');
      expect(content).toContain('gh pr view');
      expect(content).toContain('headRepositoryOwner');
      expect(content).toContain('brotherlogic');
      expect(content).toContain('brotherlogic-automation');
    });

    it('should execute gh pr merge when PR author and headRepositoryOwner are trusted', () => {
      const content = fs.readFileSync(automergePath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      const step = parsed.jobs?.['enable-automerge']?.steps?.[0];
      expect(step?.run).toBeDefined();

      // Replace GitHub expression placeholders for pull_request event test
      const script = (step?.run || '')
        .replace(/\$\{\{\s*github\.event_name\s*\}\}/g, 'pull_request')
        .replace(/\$\{\{\s*github\.event\.pull_request\.number\s*\}\}/g, '42')
        .replace(/\$\{\{\s*github\.repository\s*\}\}/g, 'brotherlogic/rose');

      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'automerge-trusted-'));
      try {
        const traceFile = path.join(tmpDir, 'gh-trace.log');
        fs.writeFileSync(
          path.join(tmpDir, 'gh'),
          `#!/bin/sh
echo "gh $*" >> "${traceFile}"
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
  printf '{"author":{"login":"brotherlogic-automation"},"headRepositoryOwner":{"login":"brotherlogic"}}\\n'
  exit 0
fi
exit 0
`,
          { mode: 0o755 }
        );

        cp.execSync(script, {
          env: { ...process.env, PATH: `${tmpDir}:${process.env.PATH}` },
          shell: '/bin/bash',
        });

        const trace = fs.readFileSync(traceFile, 'utf-8');
        expect(trace).toContain('gh pr view 42');
        expect(trace).toContain('gh pr merge 42 --repo brotherlogic/rose --auto --squash');
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it('should refuse to run gh pr merge when PR author is untrusted', () => {
      const content = fs.readFileSync(automergePath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      const step = parsed.jobs?.['enable-automerge']?.steps?.[0];
      expect(step?.run).toBeDefined();

      const script = (step?.run || '')
        .replace(/\$\{\{\s*github\.event_name\s*\}\}/g, 'workflow_run')
        .replace(/\$\{\{\s*github\.event\.workflow_run\.conclusion\s*\}\}/g, 'success')
        .replace(/\$\{\{\s*github\.repository\s*\}\}/g, 'brotherlogic/rose');

      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'automerge-untrusted-author-'));
      try {
        const traceFile = path.join(tmpDir, 'gh-trace.log');
        fs.writeFileSync(
          path.join(tmpDir, 'gh'),
          `#!/bin/sh
echo "gh $*" >> "${traceFile}"
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
  printf '{"author":{"login":"external-attacker"},"headRepositoryOwner":{"login":"brotherlogic"}}\\n'
  exit 0
fi
exit 0
`,
          { mode: 0o755 }
        );

        cp.execSync(script, {
          env: {
            ...process.env,
            PATH: `${tmpDir}:${process.env.PATH}`,
            PR_PAYLOAD: JSON.stringify([{ number: 99 }]),
          },
          shell: '/bin/bash',
        });

        const trace = fs.readFileSync(traceFile, 'utf-8');
        expect(trace).toContain('gh pr view 99');
        expect(trace).not.toContain('gh pr merge');
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it('should refuse to run gh pr merge when headRepositoryOwner is an external fork', () => {
      const content = fs.readFileSync(automergePath, 'utf-8');
      const parsed = yaml.load(content) as WorkflowDefinition;
      const step = parsed.jobs?.['enable-automerge']?.steps?.[0];
      expect(step?.run).toBeDefined();

      const script = (step?.run || '')
        .replace(/\$\{\{\s*github\.event_name\s*\}\}/g, 'pull_request')
        .replace(/\$\{\{\s*github\.event\.pull_request\.number\s*\}\}/g, '101')
        .replace(/\$\{\{\s*github\.repository\s*\}\}/g, 'brotherlogic/rose');

      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'automerge-untrusted-fork-'));
      try {
        const traceFile = path.join(tmpDir, 'gh-trace.log');
        fs.writeFileSync(
          path.join(tmpDir, 'gh'),
          `#!/bin/sh
echo "gh $*" >> "${traceFile}"
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
  printf '{"author":{"login":"brotherlogic"},"headRepositoryOwner":{"login":"external-fork"}}\\n'
  exit 0
fi
exit 0
`,
          { mode: 0o755 }
        );

        cp.execSync(script, {
          env: { ...process.env, PATH: `${tmpDir}:${process.env.PATH}` },
          shell: '/bin/bash',
        });

        const trace = fs.readFileSync(traceFile, 'utf-8');
        expect(trace).toContain('gh pr view 101');
        expect(trace).not.toContain('gh pr merge');
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });
  });
});
