import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
// @ts-expect-error js-yaml types
import * as yaml from 'js-yaml';

interface K8sResource {
  apiVersion?: string;
  kind?: string;
  metadata?: {
    name?: string;
    namespace?: string;
    labels?: Record<string, string>;
  };
  spec?: {
    type?: string;
    selector?: Record<string, string> | { matchLabels?: Record<string, string> };
    ports?: Array<{ name: string; port: number; targetPort: number }>;
    replicas?: number;
    strategy?: {
      type?: string;
      rollingUpdate?: {
        maxSurge?: number;
        maxUnavailable?: number;
      };
    };
    template?: {
      metadata?: {
        labels?: Record<string, string>;
      };
      spec?: {
        containers?: Array<{
          name: string;
          image: string;
          ports?: Array<{ containerPort: number }>;
          livenessProbe?: {
            httpGet?: { path: string; port: number };
            initialDelaySeconds?: number;
            periodSeconds?: number;
          };
          readinessProbe?: {
            httpGet?: { path: string; port: number };
            initialDelaySeconds?: number;
            periodSeconds?: number;
          };
          resources?: {
            requests?: { cpu: string; memory: string };
            limits?: { cpu: string; memory: string };
          };
        }>;
      };
    };
  };
}

describe('Rose Web Gallery Deployment and Service Manifests Turnup Specification (#65)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const manifestPath = path.join(rootDir, 'deploy', 'rose-gallery-deploy.yaml');

  it('should have a dedicated deployment manifest file deploy/rose-gallery-deploy.yaml', () => {
    expect(fs.existsSync(manifestPath)).toBe(true);
  });

  describe('Manifest Schema and Values', () => {
    it('should contain valid YAML with Namespace, Service, and Deployment manifests', () => {
      const content = fs.readFileSync(manifestPath, 'utf-8');
      const documents = yaml.loadAll(content) as K8sResource[];

      expect(documents).toHaveLength(3);

      const [namespaceDoc, serviceDoc, deploymentDoc] = documents;

      // Namespace assertions
      expect(namespaceDoc.apiVersion).toBe('v1');
      expect(namespaceDoc.kind).toBe('Namespace');
      expect(namespaceDoc.metadata?.name).toBe('rose');

      // Service assertions
      expect(serviceDoc.apiVersion).toBe('v1');
      expect(serviceDoc.kind).toBe('Service');
      expect(serviceDoc.metadata?.name).toBe('rose-gallery');
      expect(serviceDoc.metadata?.namespace).toBe('rose');
      expect(serviceDoc.metadata?.labels?.app).toBe('rose-gallery');
      expect(serviceDoc.spec.type).toBe('ClusterIP');
      expect(serviceDoc.spec.selector?.app).toBe('rose-gallery');
      expect(serviceDoc.spec.ports).toEqual([
        {
          name: 'http',
          port: 80,
          targetPort: 80,
        },
      ]);

      // Deployment assertions
      expect(deploymentDoc.apiVersion).toBe('apps/v1');
      expect(deploymentDoc.kind).toBe('Deployment');
      expect(deploymentDoc.metadata?.name).toBe('rose-gallery');
      expect(deploymentDoc.metadata?.namespace).toBe('rose');
      expect(deploymentDoc.spec.replicas).toBe(1);
      expect(deploymentDoc.spec.selector?.matchLabels?.app).toBe('rose-gallery');
      expect(deploymentDoc.spec.strategy?.type).toBe('RollingUpdate');
      expect(deploymentDoc.spec.strategy?.rollingUpdate?.maxSurge).toBe(1);
      expect(deploymentDoc.spec.strategy?.rollingUpdate?.maxUnavailable).toBe(0);

      const container = deploymentDoc.spec.template.spec.containers[0];
      expect(container.name).toBe('rose-gallery');
      expect(container.image).toContain('ghcr.io/brotherlogic/rose-gallery:0.8.0');
      expect(container.ports).toEqual([{ containerPort: 80 }]);

      expect(container.livenessProbe).toEqual({
        httpGet: {
          path: '/',
          port: 80,
        },
        initialDelaySeconds: 5,
        periodSeconds: 10,
      });

      expect(container.readinessProbe).toEqual({
        httpGet: {
          path: '/',
          port: 80,
        },
        initialDelaySeconds: 3,
        periodSeconds: 5,
      });

      expect(container.resources).toEqual({
        requests: {
          cpu: '50m',
          memory: '64Mi',
        },
        limits: {
          cpu: '200m',
          memory: '256Mi',
        },
      });
    });
  });

  describe('Turnup Contract Specifications & References', () => {
    it('should cross-reference brotherlogic/prod#974 and parent issues in deploy/rose-gallery-deploy.yaml', () => {
      const content = fs.readFileSync(manifestPath, 'utf-8');
      expect(content).toContain('brotherlogic/prod#974');
      expect(content).toContain('brotherlogic/rose#30');
      expect(content).toContain('brotherlogic/rose#63');
      expect(content).toContain('brotherlogic/rose#64');
      expect(content).toContain('brotherlogic/rose#65');
    });

    it('should include the Flux image policy comment marker for automated updates', () => {
      const content = fs.readFileSync(manifestPath, 'utf-8');
      expect(content).toContain('{"$imagepolicy": "flux-system:rose-gallery"}');
    });
  });
});
