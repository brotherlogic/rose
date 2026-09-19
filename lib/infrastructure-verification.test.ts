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
  spec?: any;
}

describe('Web Gallery Production Infrastructure Verification (#67)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const deployYamlPath = path.join(rootDir, 'deploy', 'rose-gallery-deploy.yaml');
  const ingressYamlPath = path.join(rootDir, 'deploy', 'rose-ingress.yaml');
  const imagesYamlPath = path.join(rootDir, 'deploy', 'rose-images.yaml');
  const verifyScriptPath = path.join(rootDir, 'scripts', 'verify-infrastructure-manifests.sh');

  describe('Deliverable 1: Cross-Link Infrastructure Tracking Issues', () => {
    it('should cross-reference brotherlogic/prod#974 and parent issues in deploy/rose-gallery-deploy.yaml', () => {
      const content = fs.readFileSync(deployYamlPath, 'utf-8');
      expect(content).toContain('brotherlogic/prod#974');
      expect(content).toContain('brotherlogic/rose#30');
      expect(content).toContain('brotherlogic/rose#63');
      expect(content).toContain('brotherlogic/rose#64');
      expect(content).toContain('brotherlogic/rose#65');
      expect(content).toContain('brotherlogic/rose#67');
    });

    it('should cross-reference brotherlogic/prod#975 and parent issues in deploy/rose-ingress.yaml', () => {
      const content = fs.readFileSync(ingressYamlPath, 'utf-8');
      expect(content).toContain('brotherlogic/prod#975');
      expect(content).toContain('brotherlogic/rose#30');
      expect(content).toContain('brotherlogic/rose#63');
      expect(content).toContain('brotherlogic/rose#64');
      expect(content).toContain('brotherlogic/rose#66');
      expect(content).toContain('brotherlogic/rose#67');
    });

    it('should cross-reference brotherlogic/prod#975 and parent issues in deploy/rose-images.yaml', () => {
      const content = fs.readFileSync(imagesYamlPath, 'utf-8');
      expect(content).toContain('brotherlogic/prod#975');
      expect(content).toContain('brotherlogic/rose#30');
      expect(content).toContain('brotherlogic/rose#63');
      expect(content).toContain('brotherlogic/rose#64');
      expect(content).toContain('brotherlogic/rose#66');
      expect(content).toContain('brotherlogic/rose#67');
    });
  });

  describe('Deliverable 2: Manifest Schema & Specification Verification', () => {
    it('should verify deploy.yaml conforms to Kubernetes schemas (Namespace, Service, Deployment)', () => {
      const content = fs.readFileSync(deployYamlPath, 'utf-8');
      const docs = yaml.loadAll(content) as ManifestDoc[];
      expect(docs).toHaveLength(3);

      const [ns, svc, deploy] = docs;
      expect(ns.kind).toBe('Namespace');
      expect(ns.apiVersion).toBe('v1');
      expect(ns.metadata?.name).toBe('rose');

      expect(svc.kind).toBe('Service');
      expect(svc.apiVersion).toBe('v1');
      expect(svc.metadata?.name).toBe('rose-gallery');
      expect(svc.metadata?.namespace).toBe('rose');
      expect(svc.spec?.type).toBe('ClusterIP');
      expect(svc.spec?.selector?.app).toBe('rose-gallery');
      expect(svc.spec?.ports).toEqual([
        {
          name: 'http',
          port: 80,
          targetPort: 80,
        },
      ]);

      expect(deploy.kind).toBe('Deployment');
      expect(deploy.apiVersion).toBe('apps/v1');
      expect(deploy.metadata?.name).toBe('rose-gallery');
      expect(deploy.metadata?.namespace).toBe('rose');
      expect(deploy.spec?.replicas).toBe(1);
      expect(deploy.spec?.selector?.matchLabels?.app).toBe('rose-gallery');

      const container = deploy.spec?.template?.spec?.containers?.[0];
      expect(container.name).toBe('rose-gallery');
      expect(container.image).toContain('ghcr.io/brotherlogic/rose-gallery:0.8.0');
      expect(container.ports).toEqual([{ containerPort: 80 }]);
      expect(container.livenessProbe?.httpGet?.port).toBe(80);
      expect(container.readinessProbe?.httpGet?.port).toBe(80);
    });

    it('should verify ingress.yaml conforms to schemas (Certificate, Middleware, IngressRoute)', () => {
      const content = fs.readFileSync(ingressYamlPath, 'utf-8');
      const docs = yaml.loadAll(content) as ManifestDoc[];

      const cert = docs.find((d) => d.kind === 'Certificate');
      expect(cert).toBeDefined();
      expect(cert?.apiVersion).toBe('cert-manager.io/v1');
      expect(cert?.metadata?.namespace).toBe('rose');
      expect(cert?.spec?.commonName).toBe('roseseraphinetucker.com');
      expect(cert?.spec?.dnsNames).toContain('roseseraphinetucker.com');
      expect(cert?.spec?.secretName).toBe('rose-gallery-tls');
      expect(cert?.spec?.issuerRef?.name).toBe('letsencrypt-prod');

      const middleware = docs.find((d) => d.kind === 'Middleware');
      expect(middleware).toBeDefined();
      expect(middleware?.apiVersion).toBe('traefik.io/v1alpha1');
      expect(middleware?.metadata?.namespace).toBe('rose');
      expect(middleware?.metadata?.name).toBe('redirect-to-https');
      expect(middleware?.spec?.redirectScheme?.scheme).toBe('https');
      expect(middleware?.spec?.redirectScheme?.permanent).toBe(true);

      const httpRoute = docs.find(
        (d) => d.kind === 'IngressRoute' && d.metadata?.name === 'rose-gallery-http-redirect'
      );
      expect(httpRoute).toBeDefined();
      expect(httpRoute?.apiVersion).toBe('traefik.io/v1alpha1');
      expect(httpRoute?.metadata?.namespace).toBe('rose');
      expect(httpRoute?.spec?.routes?.[0]?.match).toBe('Host(`roseseraphinetucker.com`)');
      expect(httpRoute?.spec?.routes?.[0]?.services?.[0]?.name).toBe('rose-gallery');
      expect(httpRoute?.spec?.routes?.[0]?.services?.[0]?.port).toBe(80);

      const httpsRoute = docs.find(
        (d) => d.kind === 'IngressRoute' && d.metadata?.name === 'rose-gallery-ingress'
      );
      expect(httpsRoute).toBeDefined();
      expect(httpsRoute?.apiVersion).toBe('traefik.io/v1alpha1');
      expect(httpsRoute?.metadata?.namespace).toBe('rose');
      expect(httpsRoute?.spec?.routes?.[0]?.match).toBe('Host(`roseseraphinetucker.com`)');
      expect(httpsRoute?.spec?.routes?.[0]?.services?.[0]?.name).toBe('rose-gallery');
      expect(httpsRoute?.spec?.routes?.[0]?.services?.[0]?.port).toBe(80);
      expect(httpsRoute?.spec?.tls?.secretName).toBe('rose-gallery-tls');
    });

    it('should verify images.yaml conforms to Flux schemas (ImageRepository, ImagePolicy, ImageUpdateAutomation)', () => {
      const content = fs.readFileSync(imagesYamlPath, 'utf-8');
      const docs = yaml.loadAll(content) as ManifestDoc[];

      const imageRepo = docs.find((d) => d.kind === 'ImageRepository');
      expect(imageRepo).toBeDefined();
      expect(imageRepo?.apiVersion).toBe('image.toolkit.fluxcd.io/v1');
      expect(imageRepo?.metadata?.name).toBe('rose-gallery');
      expect(imageRepo?.metadata?.namespace).toBe('flux-system');
      expect(imageRepo?.spec?.image).toBe('ghcr.io/brotherlogic/rose-gallery');

      const imagePolicy = docs.find((d) => d.kind === 'ImagePolicy');
      expect(imagePolicy).toBeDefined();
      expect(imagePolicy?.apiVersion).toBe('image.toolkit.fluxcd.io/v1');
      expect(imagePolicy?.metadata?.name).toBe('rose-gallery');
      expect(imagePolicy?.metadata?.namespace).toBe('flux-system');
      expect(imagePolicy?.spec?.imageRepositoryRef?.name).toBe('rose-gallery');
      expect(imagePolicy?.spec?.policy?.semver?.range).toBe('0.x.0');

      const imageUpdate = docs.find((d) => d.kind === 'ImageUpdateAutomation');
      expect(imageUpdate).toBeDefined();
      expect(imageUpdate?.apiVersion).toBe('image.toolkit.fluxcd.io/v1');
      expect(imageUpdate?.metadata?.name).toBe('rose-gallery');
      expect(imageUpdate?.metadata?.namespace).toBe('flux-system');
      expect(imageUpdate?.spec?.sourceRef?.name).toBe('flux-system');
      expect(imageUpdate?.spec?.update?.path).toBe('./projects/rose/');
      expect(imageUpdate?.spec?.update?.strategy).toBe('Setters');
    });

    it('should ensure namespace, port, and host binding consistency across all manifests', () => {
      const deployDocs = yaml.loadAll(fs.readFileSync(deployYamlPath, 'utf-8')) as ManifestDoc[];
      const ingressDocs = yaml.loadAll(fs.readFileSync(ingressYamlPath, 'utf-8')) as ManifestDoc[];

      // Namespace consistency for application runtime resources
      const runtimeDocs = [...deployDocs.filter((d) => d.kind !== 'Namespace'), ...ingressDocs];
      for (const doc of runtimeDocs) {
        expect(doc.metadata?.namespace).toBe('rose');
      }

      // Port binding consistency (port 80)
      const service = deployDocs.find((d) => d.kind === 'Service');
      const deployment = deployDocs.find((d) => d.kind === 'Deployment');
      const ingressRoutes = ingressDocs.filter((d) => d.kind === 'IngressRoute');

      expect(service?.spec?.ports?.[0]?.port).toBe(80);
      expect(service?.spec?.ports?.[0]?.targetPort).toBe(80);
      expect(deployment?.spec?.template?.spec?.containers?.[0]?.ports?.[0]?.containerPort).toBe(80);

      for (const ir of ingressRoutes) {
        expect(ir.spec?.routes?.[0]?.services?.[0]?.port).toBe(80);
        expect(ir.spec?.routes?.[0]?.services?.[0]?.name).toBe('rose-gallery');
      }

      // Target host binding consistency (roseseraphinetucker.com)
      const cert = ingressDocs.find((d) => d.kind === 'Certificate');
      expect(cert?.spec?.commonName).toBe('roseseraphinetucker.com');
      expect(cert?.spec?.dnsNames).toEqual(['roseseraphinetucker.com']);

      for (const ir of ingressRoutes) {
        expect(ir.spec?.routes?.[0]?.match).toBe('Host(`roseseraphinetucker.com`)');
      }

      // TLS Secret consistency
      expect(cert?.spec?.secretName).toBe('rose-gallery-tls');
      const httpsRoute = ingressDocs.find(
        (d) => d.kind === 'IngressRoute' && d.metadata?.name === 'rose-gallery-ingress'
      );
      expect(httpsRoute?.spec?.tls?.secretName).toBe('rose-gallery-tls');
    });
  });

  describe('Verification Script Executability', () => {
    it('should have an executable verification script in scripts/verify-infrastructure-manifests.sh', () => {
      expect(fs.existsSync(verifyScriptPath)).toBe(true);
      const stats = fs.statSync(verifyScriptPath);
      expect(stats.mode & 0o111).toBeGreaterThan(0);
    });
  });
});
