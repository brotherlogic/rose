import React from 'react';

export default function Footer() {
  return (
    <footer
      role="contentinfo"
      className="w-full border-t border-[var(--border-subtle)] bg-[var(--bg-primary)] text-[var(--text-secondary)] transition-colors"
    >
      <div className="container mx-auto px-4 md:px-8 py-12 md:py-16">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8 mb-12">
          {/* Institutional Credits */}
          <div>
            <p className="text-xs uppercase tracking-widest text-[var(--text-secondary)] mb-2 font-mono">
              Institutional Credits
            </p>
            <h3 className="text-lg font-[family-name:var(--font-serif)] text-[var(--text-primary)] mb-3">
              Rose Seraphine Tucker Archive
            </h3>
            <p className="text-xs leading-relaxed max-w-sm">
              Established as an austere curatorial repository dedicated to documenting, cataloging, and preserving the developmental oeuvre, spatial disruptions, and tactile investigations of Rose Seraphine Tucker.
            </p>
          </div>

          {/* Curatorial Monograph & Colophon */}
          <div>
            <p className="text-xs uppercase tracking-widest text-[var(--text-secondary)] mb-2 font-mono">
              Archive Colophon
            </p>
            <h3 className="text-lg font-[family-name:var(--font-serif)] text-[var(--text-primary)] mb-3">
              Catalog Architecture
            </h3>
            <p className="text-xs leading-relaxed max-w-sm mb-2">
              Automated archival syncer and distributed ingestion pipeline operating over institutional network storage.
            </p>
            <p className="text-xs leading-relaxed max-w-sm text-[var(--text-secondary)] opacity-80">
              Typeset in Geist Sans &amp; curated editorial serif typography. Design system structured upon White Cube and Dark Luxury minimalist exhibition tokens.
            </p>
          </div>

          {/* Curatorial Preservation & Legal */}
          <div>
            <p className="text-xs uppercase tracking-widest text-[var(--text-secondary)] mb-2 font-mono">
              Curatorial Preservation
            </p>
            <h3 className="text-lg font-[family-name:var(--font-serif)] text-[var(--text-primary)] mb-3">
              Permanent Catalog
            </h3>
            <p className="text-xs leading-relaxed max-w-sm mb-2">
              All compositions, photographic documentation, and historical movement taxonomies are cataloged under curatorial oversight.
            </p>
            <p className="text-xs text-[var(--text-secondary)] font-mono">
              Archive Ref: RST-RETRO-2026
            </p>
          </div>
        </div>

        {/* Bottom Bar: Copyright Notices */}
        <div className="pt-8 border-t border-[var(--border-subtle)] flex flex-col sm:flex-row items-center justify-between text-xs text-[var(--text-secondary)]">
          <p>© 2026 Rose Seraphine Tucker Archive. All rights reserved.</p>
          <p className="mt-2 sm:mt-0 font-mono text-[11px] tracking-wider uppercase">
            Curatorial Retrospective Catalog
          </p>
        </div>
      </div>
    </footer>
  );
}
