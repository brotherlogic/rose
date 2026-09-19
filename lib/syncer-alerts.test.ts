import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
// @ts-expect-error js-yaml types
import * as yaml from 'js-yaml';

interface PrometheusRuleDocument {
  apiVersion?: string;
  kind?: string;
  metadata?: {
    name?: string;
    namespace?: string;
    labels?: Record<string, string>;
  };
  spec?: {
    groups?: Array<{
      name: string;
      rules: Array<{
        alert: string;
        expr: string;
        for?: string;
        labels?: Record<string, string>;
        annotations?: Record<string, string>;
      }>;
    }>;
  };
}

describe('Rose Syncer Prometheus Alerting Rule Specification (#74)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const manifestPath = path.join(rootDir, 'deploy', 'rose-alerts.yaml');

  describe('Manifest File Existence', () => {
    it('should have a dedicated alerting rule manifest file deploy/rose-alerts.yaml', () => {
      expect(fs.existsSync(manifestPath)).toBe(true);
    });
  });

  describe('PrometheusRule Schema and Specifications', () => {
    it('should contain a valid PrometheusRule manifest conforming to monitoring.coreos.com/v1', () => {
      const content = fs.readFileSync(manifestPath, 'utf-8');
      const doc = yaml.load(content) as PrometheusRuleDocument;

      expect(doc).toBeDefined();
      expect(doc.apiVersion).toBe('monitoring.coreos.com/v1');
      expect(doc.kind).toBe('PrometheusRule');
      expect(doc.metadata?.name).toBe('rose-syncer-alerts');
      expect(doc.metadata?.namespace).toBe('rose');
      expect(doc.metadata?.labels).toEqual({
        role: 'alert-rules',
        'app.kubernetes.io/name': 'rose-syncer',
        'app.kubernetes.io/part-of': 'rose',
      });
    });

    it('should configure group rose-syncer.rules with RoseSyncerPersistentFailures alert', () => {
      const content = fs.readFileSync(manifestPath, 'utf-8');
      const doc = yaml.load(content) as PrometheusRuleDocument;

      const group = doc.spec?.groups?.find((g) => g.name === 'rose-syncer.rules');
      expect(group).toBeDefined();

      const rule = group?.rules.find((r) => r.alert === 'RoseSyncerPersistentFailures');
      expect(rule).toBeDefined();

      // Normalize whitespace for query comparison
      const normalizedExpr = rule?.expr.replace(/\s+/g, ' ').trim();
      expect(normalizedExpr).toBe(
        'sum by (cronjob) ( increase(kube_job_failed_total{job_name=~"rose-syncer-.*", namespace="rose"}[5d]) ) >= 5'
      );

      expect(rule?.for).toBe('15m');
      expect(rule?.labels).toEqual({
        severity: 'critical',
        team: 'brotherlogic',
      });
      expect(rule?.annotations?.summary).toBe('Rose syncer has experienced 5 consecutive failures');
      expect(rule?.annotations?.description).toContain(
        "The background artwork syncer CronJob 'rose-syncer' in namespace 'rose' has failed 5 consecutive executions"
      );
    });
  });

  describe('Turnup Contract Specifications & References', () => {
    it('should cross-reference brotherlogic/prod#977 and parent issues in deploy/rose-alerts.yaml', () => {
      const content = fs.readFileSync(manifestPath, 'utf-8');
      expect(content).toContain('brotherlogic/prod#977');
      expect(content).toContain('brotherlogic/rose#31');
      expect(content).toContain('brotherlogic/rose#71');
      expect(content).toContain('brotherlogic/rose#72');
      expect(content).toContain('brotherlogic/rose#74');
    });
  });
});
