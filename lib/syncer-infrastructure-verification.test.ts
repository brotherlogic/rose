import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
// @ts-expect-error js-yaml types
import * as yaml from 'js-yaml';

interface ManifestDoc {
  apiVersion?: string;
  kind?: string;
  metadata?: {
    name?: string;
    namespace?: string;
    annotations?: Record<string, string>;
    labels?: Record<string, string>;
  };
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  spec?: any;
}

describe('Syncer CronJob Infrastructure Verification (#75)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const cronJobYamlPath = path.join(rootDir, 'deploy', 'rose-syncer-cronjob.yaml');
  const alertsYamlPath = path.join(rootDir, 'deploy', 'rose-alerts.yaml');
  const verifyScriptPath = path.join(rootDir, 'scripts', 'verify-syncer-infrastructure-manifests.sh');

  describe('Deliverable 1: Cross-Link Infrastructure Tracking Issues', () => {
    it('should cross-reference brotherlogic/prod#976 and parent issues in deploy/rose-syncer-cronjob.yaml', () => {
      const content = fs.readFileSync(cronJobYamlPath, 'utf-8');
      expect(content).toContain('brotherlogic/prod#976');
      expect(content).toContain('brotherlogic/rose#31');
      expect(content).toContain('brotherlogic/rose#71');
      expect(content).toContain('brotherlogic/rose#72');
      expect(content).toContain('brotherlogic/rose#73');
      expect(content).toContain('brotherlogic/rose#75');
    });

    it('should cross-reference brotherlogic/prod#977 and parent issues in deploy/rose-alerts.yaml', () => {
      const content = fs.readFileSync(alertsYamlPath, 'utf-8');
      expect(content).toContain('brotherlogic/prod#977');
      expect(content).toContain('brotherlogic/rose#31');
      expect(content).toContain('brotherlogic/rose#71');
      expect(content).toContain('brotherlogic/rose#72');
      expect(content).toContain('brotherlogic/rose#74');
      expect(content).toContain('brotherlogic/rose#75');
    });
  });

  describe('Deliverable 2: Manifest Schema & Specification Verification', () => {
    it('should verify rose-syncer-cronjob.yaml conforms to Kubernetes batch/v1 CronJob schema', () => {
      const content = fs.readFileSync(cronJobYamlPath, 'utf-8');
      const cronjob = yaml.load(content) as ManifestDoc;

      expect(cronjob).toBeDefined();
      expect(cronjob.apiVersion).toBe('batch/v1');
      expect(cronjob.kind).toBe('CronJob');
      expect(cronjob.metadata?.name).toBe('rose-syncer');
      expect(cronjob.metadata?.namespace).toBe('rose');
      expect(cronjob.metadata?.labels?.['app.kubernetes.io/name']).toBe('rose-syncer');
      expect(cronjob.metadata?.labels?.['app.kubernetes.io/part-of']).toBe('rose');

      const spec = cronjob.spec;
      expect(spec.schedule).toBe('0 4 * * *');
      expect(spec.concurrencyPolicy).toBe('Forbid');
      expect(spec.startingDeadlineSeconds).toBe(3600);
      expect(spec.successfulJobsHistoryLimit).toBe(3);
      expect(spec.failedJobsHistoryLimit).toBe(5);

      const jobSpec = spec.jobTemplate?.spec;
      expect(jobSpec.activeDeadlineSeconds).toBe(72000);
      expect(jobSpec.backoffLimit).toBe(2);

      const podSpec = jobSpec.template?.spec;
      expect(podSpec.restartPolicy).toBe('OnFailure');
      expect(podSpec.securityContext?.runAsUser).toBe(65532);
      expect(podSpec.securityContext?.runAsGroup).toBe(65532);
      expect(podSpec.securityContext?.fsGroup).toBe(65532);

      expect(podSpec.containers).toHaveLength(1);
      const container = podSpec.containers[0];
      expect(container.name).toBe('rose-syncer');
      expect(container.image).toContain('ghcr.io/brotherlogic/rose-syncer:0.5.0');
      expect(container.imagePullPolicy).toBe('IfNotPresent');

      // Environment variables
      const envMap = Object.fromEntries(
        container.env.map((e: { name: string; value: string }) => [e.name, e.value])
      );
      expect(envMap['STORAGE_PATH']).toBe('/data');
      expect(envMap['GOOGLE_APPLICATION_CREDENTIALS']).toBe('/etc/secrets/gcp/credentials.json');

      // Resources
      expect(container.resources?.requests?.cpu).toBe('100m');
      expect(container.resources?.requests?.memory).toBe('128Mi');
      expect(container.resources?.limits?.cpu).toBe('500m');
      expect(container.resources?.limits?.memory).toBe('512Mi');

      // Volume mounts
      expect(container.volumeMounts).toEqual([
        {
          name: 'rose-data',
          mountPath: '/data',
        },
        {
          name: 'rose-gcp-sa',
          mountPath: '/etc/secrets/gcp/credentials.json',
          subPath: 'credentials.json',
          readOnly: true,
        },
      ]);

      // Volumes
      expect(podSpec.volumes).toEqual([
        {
          name: 'rose-data',
          persistentVolumeClaim: {
            claimName: 'rose-data-pvc',
          },
        },
        {
          name: 'rose-gcp-sa',
          secret: {
            secretName: 'rose-gcp-sa-secret',
          },
        },
      ]);
    });

    it('should verify rose-alerts.yaml conforms to monitoring.coreos.com/v1 PrometheusRule schema', () => {
      const content = fs.readFileSync(alertsYamlPath, 'utf-8');
      const rule = yaml.load(content) as ManifestDoc;

      expect(rule).toBeDefined();
      expect(rule.apiVersion).toBe('monitoring.coreos.com/v1');
      expect(rule.kind).toBe('PrometheusRule');
      expect(rule.metadata?.name).toBe('rose-syncer-alerts');
      expect(rule.metadata?.namespace).toBe('rose');
      expect(rule.metadata?.labels?.['role']).toBe('alert-rules');
      expect(rule.metadata?.labels?.['app.kubernetes.io/name']).toBe('rose-syncer');
      expect(rule.metadata?.labels?.['app.kubernetes.io/part-of']).toBe('rose');

      const group = rule.spec?.groups?.[0];
      expect(group.name).toBe('rose-syncer.rules');
      expect(group.rules).toHaveLength(1);

      const alert = group.rules[0];
      expect(alert.alert).toBe('RoseSyncerPersistentFailures');
      expect(alert.for).toBe('15m');
      expect(alert.labels?.severity).toBe('critical');
      expect(alert.labels?.team).toBe('brotherlogic');
      expect(alert.annotations?.summary).toBe('Rose syncer has experienced 5 consecutive failures');
      expect(alert.annotations?.description).toContain('has failed 5 consecutive executions');

      // Normalized expression check
      const normalizedExpr = alert.expr.replace(/\s+/g, ' ').trim();
      expect(normalizedExpr).toContain('kube_job_failed_total{job_name=~"rose-syncer-.*", namespace="rose"}[5d]');
      expect(normalizedExpr).toContain('>= 5');
    });

    it('should ensure namespace and naming consistency across syncer manifests', () => {
      const cronjobDoc = yaml.load(fs.readFileSync(cronJobYamlPath, 'utf-8')) as ManifestDoc;
      const alertsDoc = yaml.load(fs.readFileSync(alertsYamlPath, 'utf-8')) as ManifestDoc;

      expect(cronjobDoc.metadata?.namespace).toBe('rose');
      expect(alertsDoc.metadata?.namespace).toBe('rose');

      const alertExpr = alertsDoc.spec?.groups?.[0]?.rules?.[0]?.expr || '';
      expect(alertExpr).toContain(`job_name=~"${cronjobDoc.metadata?.name}-.*"`);
      expect(alertExpr).toContain(`namespace="${cronjobDoc.metadata?.namespace}"`);
    });
  });

  describe('Deliverable 3: Verification Script Executability', () => {
    it('should have an executable verification script in scripts/verify-syncer-infrastructure-manifests.sh', () => {
      expect(fs.existsSync(verifyScriptPath)).toBe(true);
      const stats = fs.statSync(verifyScriptPath);
      expect(stats.mode & 0o111).toBeGreaterThan(0);
    });
  });
});
