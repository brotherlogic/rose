import React from 'react';

export default function EditorialStatement() {
  return (
    <section
      id="statement"
      aria-labelledby="statement-title"
      className="w-full py-16 md:py-24 border-b border-[var(--border-subtle)] bg-[var(--bg-primary)] transition-colors"
    >
      <div className="container mx-auto px-4 md:px-8 max-w-4xl">
        <header className="mb-8">
          <p className="text-xs uppercase tracking-widest text-[var(--text-secondary)] mb-2">
            Curatorial Statement
          </p>
          <h2
            id="statement-title"
            className="text-2xl md:text-3xl font-[family-name:var(--font-serif)] font-normal text-[var(--text-primary)] tracking-wide"
          >
            The Dialectic of Domesticity and Destruction
          </h2>
        </header>

        <blockquote className="relative my-8 pl-6 md:pl-10 border-l-2 border-[var(--border-focus)]">
          <span
            className="absolute -top-6 -left-3 text-6xl font-[family-name:var(--font-serif)] text-[var(--text-secondary)] opacity-20 select-none pointer-events-none drop-quote"
            aria-hidden="true"
          >
            “
          </span>
          <p className="text-lg md:text-xl font-[family-name:var(--font-serif)] italic leading-relaxed text-[var(--text-primary)] mb-6">
            In interrogating the everyday living space, the oeuvre of Rose Seraphine Tucker operates at the fragile intersection of domesticity vs. destruction. Through an uncompromising commitment to ephemeral materiality and existential spatiality, mundane domestic artifacts are transformed into resonant ontological inquiries.
          </p>
          <p className="text-sm md:text-base leading-relaxed text-[var(--text-secondary)] mb-6">
            Each composition confronts entropy not as an end-state, but as an active collaborator. From accidental spills to calculated structural disruptions, the works challenge institutional sanctity by locating sublime transcendence within household chaos and temporal decay.
          </p>
          <footer className="pt-4 border-t border-[var(--border-subtle)] flex flex-col sm:flex-row sm:items-center sm:justify-between text-xs tracking-wider uppercase text-[var(--text-secondary)]">
            <cite className="font-semibold text-[var(--text-primary)] not-italic">
              Rose Seraphine Tucker Retrospective Archive
            </cite>
            <span className="mt-1 sm:mt-0">Curatorial Notes &amp; Monograph Catalog</span>
          </footer>
        </blockquote>
      </div>
    </section>
  );
}
