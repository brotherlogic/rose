import { describe, it, expect } from 'vitest';
import * as fs from 'fs';
import * as path from 'path';
// @ts-expect-error js-yaml types
import * as yaml from 'js-yaml';

interface ResourceManifest {
  apiVersion?: string;
  kind?: string;
  metadata?: {
    name?: string;
    namespace?: string;
    annotations?: Record<string, string>;
  };
  spec?: {
    commonName?: string;
    secretName?: string;
    dnsNames?: string[];
    issuerRef?: {
      name?: string;
      kind?: string;
    };
    redirectScheme?: {
      scheme?: string;
      permanent?: boolean;
    };
    entryPoints?: string[];
    routes?: Array<{
      match?: string;
      middlewares?: Array<{ name?: string }>;
      services?: Array<{ name?: string; port?: number }>;
    }>;
    tls?: {
      secretName?: string;
    };
    image?: string;
    interval?: string;
    imageRepositoryRef?: {
      name?: string;
    };
    policy?: {
      semver?: {
        range?: string;
      };
    };
    sourceRef?: {
      name?: string;
      kind?: string;
    };
    git?: {
      checkout?: {
        ref?: {
          branch?: string;
        };
      };
    };
    update?: {
      path?: string;
      strategy?: string;
    };
  };
}

describe('Ingress Routing, TLS & Flux Automation Turnup Specification (#66)', () => {
  const rootDir = path.resolve(__dirname, '..');
  const ingressManifestPath = path.join(rootDir, 'deploy', 'rose-ingress.yaml');
  const imagesManifestPath = path.join(rootDir, 'deploy', 'rose-images.yaml');

  describe('Manifest Files Existence', () => {
    it('should have deploy/rose-ingress.yaml', () => {
      expect(fs.existsSync(ingressManifestPath)).toBe(true);
    });

    it('should have deploy/rose-images.yaml', () => {
      expect(fs.existsSync(imagesManifestPath)).toBe(true);
    });
  });

  describe('Ingress & TLS Manifest Specification', () => {
    it('should contain valid Certificate, Middleware, and IngressRoute manifests', () => {
      const content = fs.readFileSync(ingressManifestPath, 'utf-8');
      const docs = yaml.loadAll(content) as ResourceManifest[];

      const cert = docs.find((d) => d?.kind === 'Certificate');
      expect(cert).toBeDefined();
      expect(cert?.apiVersion).toBe('cert-manager.io/v1');
      expect(cert?.metadata?.name).toBe('rose-gallery-tls');
      expect(cert?.metadata?.namespace).toBe('rose');
      expect(cert?.spec?.commonName).toBe('roseseraphinetucker.com');
      expect(cert?.spec?.secretName).toBe('rose-gallery-tls');
      expect(cert?.spec?.dnsNames).toContain('roseseraphinetucker.com');
      expect(cert?.spec?.issuerRef?.name).toBe('letsencrypt-prod');
      expect(cert?.spec?.issuerRef?.kind).toBe('ClusterIssuer');

      const middleware = docs.find((d) => d?.kind === 'Middleware');
      expect(middleware).toBeDefined();
      expect(middleware?.apiVersion).toBe('traefik.io/v1alpha1');
      expect(middleware?.metadata?.name).toBe('redirect-to-https');
      expect(middleware?.metadata?.namespace).toBe('rose');
      expect(middleware?.spec?.redirectScheme?.scheme).toBe('https');
      expect(middleware?.spec?.redirectScheme?.permanent).toBe(true);

      const httpRoute = docs.find(
        (d) => d?.kind === 'IngressRoute' && d?.metadata?.name === 'rose-gallery-http-redirect'
      );
      expect(httpRoute).toBeDefined();
      expect(httpRoute?.apiVersion).toBe('traefik.io/v1alpha1');
      expect(httpRoute?.metadata?.namespace).toBe('rose');
      expect(httpRoute?.metadata?.annotations?.['kubernetes.io/ingress.class']).toBe('traefik-external');
      expect(httpRoute?.spec?.entryPoints).toContain('web');
      expect(httpRoute?.spec?.routes?.[0]?.match).toBe('Host(`roseseraphinetucker.com`)');
      expect(httpRoute?.spec?.routes?.[0]?.middlewares?.[0]?.name).toBe('redirect-to-https');
      expect(httpRoute?.spec?.routes?.[0]?.services?.[0]?.name).toBe('rose-gallery');
      expect(httpRoute?.spec?.routes?.[0]?.services?.[0]?.port).toBe(80);

      const httpsRoute = docs.find(
        (d) => d?.kind === 'IngressRoute' && d?.metadata?.name === 'rose-gallery-ingress'
      );
      expect(httpsRoute).toBeDefined();
      expect(httpsRoute?.apiVersion).toBe('traefik.io/v1alpha1');
      expect(httpsRoute?.metadata?.namespace).toBe('rose');
      expect(httpsRoute?.metadata?.annotations?.['kubernetes.io/ingress.class']).toBe('traefik-external');
      expect(httpsRoute?.metadata?.annotations?.['cert-manager.io/cluster-issuer']).toBe('letsencrypt-prod');
      expect(httpsRoute?.spec?.entryPoints).toContain('websecure');
      expect(httpsRoute?.spec?.routes?.[0]?.match).toBe('Host(`roseseraphinetucker.com`)');
      expect(httpsRoute?.spec?.routes?.[0]?.services?.[0]?.name).toBe('rose-gallery');
      expect(httpsRoute?.spec?.routes?.[0]?.services?.[0]?.port).toBe(80);
      expect(httpsRoute?.spec?.tls?.secretName).toBe('rose-gallery-tls');
    });

    it('should cross-reference brotherlogic/prod#975 and parent Rose specifications', () => {
      const content = fs.readFileSync(ingressManifestPath, 'utf-8');
      expect(content).toContain('brotherlogic/prod#975');
      expect(content).toContain('brotherlogic/rose#30');
      expect(content).toContain('brotherlogic/rose#63');
      expect(content).toContain('brotherlogic/rose#64');
      expect(content).toContain('brotherlogic/rose#66');
    });
  });

  describe('Flux GitOps Automation Manifest Specification', () => {
    it('should contain valid ImageRepository, ImagePolicy, and ImageUpdateAutomation manifests', () => {
      const content = fs.readFileSync(imagesManifestPath, 'utf-8');
      const docs = yaml.loadAll(content) as ResourceManifest[];

      const imageRepo = docs.find((d) => d?.kind === 'ImageRepository');
      expect(imageRepo).toBeDefined();
      expect(imageRepo?.apiVersion).toBe('image.toolkit.fluxcd.io/v1');
      expect(imageRepo?.metadata?.name).toBe('rose-gallery');
      expect(imageRepo?.metadata?.namespace).toBe('flux-system');
      expect(imageRepo?.spec?.image).toBe('ghcr.io/brotherlogic/rose-gallery');
      expect(imageRepo?.spec?.interval).toBe('1m0s');

      const imagePolicy = docs.find((d) => d?.kind === 'ImagePolicy');
      expect(imagePolicy).toBeDefined();
      expect(imagePolicy?.apiVersion).toBe('image.toolkit.fluxcd.io/v1');
      expect(imagePolicy?.metadata?.name).toBe('rose-gallery');
      expect(imagePolicy?.metadata?.namespace).toBe('flux-system');
      expect(imagePolicy?.spec?.imageRepositoryRef?.name).toBe('rose-gallery');
      expect(imagePolicy?.spec?.policy?.semver?.range).toBe('0.x.0');

      const imageUpdate = docs.find((d) => d?.kind === 'ImageUpdateAutomation');
      expect(imageUpdate).toBeDefined();
      expect(imageUpdate?.apiVersion).toBe('image.toolkit.fluxcd.io/v1');
      expect(imageUpdate?.metadata?.name).toBe('rose-gallery');
      expect(imageUpdate?.metadata?.namespace).toBe('flux-system');
      expect(imageUpdate?.spec?.interval).toBe('1m0s');
      expect(imageUpdate?.spec?.sourceRef?.name).toBe('flux-system');
      expect(imageUpdate?.spec?.sourceRef?.kind).toBe('GitRepository');
      expect(imageUpdate?.spec?.git?.checkout?.ref?.branch).toBe('main');
      expect(imageUpdate?.spec?.update?.path).toBe('./projects/rose/');
      expect(imageUpdate?.spec?.update?.strategy).toBe('Setters');
    });

    it('should cross-reference brotherlogic/prod#975 and parent Rose specifications', () => {
      const content = fs.readFileSync(imagesManifestPath, 'utf-8');
      expect(content).toContain('brotherlogic/prod#975');
      expect(content).toContain('brotherlogic/rose#30');
      expect(content).toContain('brotherlogic/rose#63');
      expect(content).toContain('brotherlogic/rose#64');
      expect(content).toContain('brotherlogic/rose#66');
    });
  });
});
