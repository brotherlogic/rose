import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
// @ts-expect-error js-yaml types
import * as yaml from 'js-yaml';

interface CronJobResource {
  apiVersion?: string;
  kind?: string;
  metadata?: {
    name?: string;
    namespace?: string;
    labels?: Record<string, string>;
  };
  spec?: {
    schedule?: string;
    concurrencyPolicy?: string;
    startingDeadlineSeconds?: number;
    successfulJobsHistoryLimit?: number;
    failedJobsHistoryLimit?: number;
    jobTemplate?: {
      metadata?: {
        labels?: Record<string, string>;
      };
      spec?: {
        activeDeadlineSeconds?: number;
        backoffLimit?: number;
        template?: {
          metadata?: {
            labels?: Record<string, string>;
          };
          spec?: {
            restartPolicy?: string;
            securityContext?: {
              runAsUser?: number;
              runAsGroup?: number;
              fsGroup?: number;
            };
            containers?: Array<{
              name: string;
              image: string;
              imagePullPolicy?: string;
              env?: Array<{ name: string; value: string }>;
              resources?: {
                requests?: { cpu: string; memory: string };
                limits?: { cpu: string; memory: string };
              };
              volumeMounts?: Array<{
                name: string;
                mountPath: string;
                subPath?: string;
                readOnly?: boolean;
              }>;
            }>;
            volumes?: Array<{
              name: string;
              persistentVolumeClaim?: { claimName: string };
              secret?: { secretName: string };
            }>;
          };
        };
      };
    };
  };
}

describe('Rose Syncer CronJob Manifest Turnup Specification (#73)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const cronJobManifestPath = path.join(rootDir, 'deploy', 'rose-syncer-cronjob.yaml');

  it('should have a dedicated CronJob manifest file deploy/rose-syncer-cronjob.yaml', () => {
    expect(fs.existsSync(cronJobManifestPath)).toBe(true);
  });

  describe('Manifest Schema and Values', () => {
    it('should be valid YAML matching Kubernetes CronJob specification', () => {
      const content = fs.readFileSync(cronJobManifestPath, 'utf-8');
      const cronJob = yaml.load(content) as CronJobResource;

      expect(cronJob.apiVersion).toBe('batch/v1');
      expect(cronJob.kind).toBe('CronJob');
      expect(cronJob.metadata?.name).toBe('rose-syncer');
      expect(cronJob.metadata?.namespace).toBe('rose');
      expect(cronJob.metadata?.labels).toEqual({
        'app.kubernetes.io/name': 'rose-syncer',
        'app.kubernetes.io/part-of': 'rose',
      });

      expect(cronJob.spec?.schedule).toBe('0 4 * * *');
      expect(cronJob.spec?.concurrencyPolicy).toBe('Forbid');
      expect(cronJob.spec?.startingDeadlineSeconds).toBe(3600);
      expect(cronJob.spec?.successfulJobsHistoryLimit).toBe(3);
      expect(cronJob.spec?.failedJobsHistoryLimit).toBe(5);

      const jobSpec = cronJob.spec?.jobTemplate?.spec;
      expect(jobSpec?.activeDeadlineSeconds).toBe(72000);
      expect(jobSpec?.backoffLimit).toBe(2);

      const podSpec = jobSpec?.template?.spec;
      expect(podSpec?.restartPolicy).toBe('OnFailure');
      expect(podSpec?.securityContext).toEqual({
        runAsUser: 65532,
        runAsGroup: 65532,
        fsGroup: 65532,
      });

      expect(podSpec?.containers).toHaveLength(1);
      const container = podSpec!.containers![0];
      expect(container.name).toBe('rose-syncer');
      expect(container.image).toContain('ghcr.io/brotherlogic/rose-syncer:0.5.0');
      expect(container.imagePullPolicy).toBe('IfNotPresent');

      expect(container.env).toEqual([
        { name: 'STORAGE_PATH', value: '/data' },
        { name: 'GOOGLE_APPLICATION_CREDENTIALS', value: '/etc/secrets/gcp/credentials.json' },
      ]);

      expect(container.resources).toEqual({
        requests: { cpu: '100m', memory: '128Mi' },
        limits: { cpu: '500m', memory: '512Mi' },
      });

      expect(container.volumeMounts).toEqual([
        { name: 'rose-data', mountPath: '/data' },
        {
          name: 'rose-gcp-sa',
          mountPath: '/etc/secrets/gcp/credentials.json',
          subPath: 'credentials.json',
          readOnly: true,
        },
      ]);

      expect(podSpec?.volumes).toEqual([
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
  });

  describe('Turnup Contract Specifications & References', () => {
    it('should cross-reference brotherlogic/prod#976 and parent Rose issues in deploy/rose-syncer-cronjob.yaml', () => {
      const content = fs.readFileSync(cronJobManifestPath, 'utf-8');
      expect(content).toContain('brotherlogic/prod#976');
      expect(content).toContain('brotherlogic/rose#31');
      expect(content).toContain('brotherlogic/rose#71');
      expect(content).toContain('brotherlogic/rose#72');
      expect(content).toContain('brotherlogic/rose#73');
    });

    it('should include the Flux image policy comment marker for automated updates', () => {
      const content = fs.readFileSync(cronJobManifestPath, 'utf-8');
      expect(content).toContain('{"$imagepolicy": "flux-system:rose-syncer"}');
    });
  });
});
