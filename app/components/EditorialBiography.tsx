import React from 'react';

interface Movement {
  period: string;
  title: string;
  subtitle: string;
  description: string;
  lore: string;
}

const MOVEMENTS: Movement[] = [
  {
    period: 'I',
    title: 'The Crayon Period',
    subtitle: 'Primitive marks & linear wax explorations',
    description:
      'Characterized by raw, uninhibited wax pigmentation applied directly to virgin substrates. The artist initiates an exploratory spatial dialogue through primitive marks and linear wax explorations, charting boundaries of permissible expression.',
    lore: 'Foundational tactile works investigating surface friction and naive iconography.',
  },
  {
    period: 'II',
    title: 'Domestic Destructionism',
    subtitle: 'Entropy, spills & structural disruption',
    description:
      'A radical departure from static mark-making towards dynamic spatial intervention. Embracing entropy, intentional spills, and structural disruption, this movement interrogates domestic order and the fragile permanence of domestic materials.',
    lore: 'Calculated unravelings that elevate liquid accidents into permanent curatorial statements.',
  },
  {
    period: 'III',
    title: 'Kinetic Scribblism',
    subtitle: 'Frantic gestures & chaotic linework',
    description:
      'Defined by high-velocity motor impulse, frantic gestures, and dense, chaotic linework. In this phase, the surface becomes a frenetic seismograph of emotional immediacy and unbounded physical cadence.',
    lore: 'A feverish rejection of geometric precision in pursuit of pure gestural velocity.',
  },
  {
    period: 'IV',
    title: 'Found Object Assemblage',
    subtitle: 'Sculptural household arrangements',
    description:
      'Repurposing utilitarian domestic remnants into sculptural household arrangements. Found cutlery, textiles, and packaging elements undergo spatial recontextualization, subverting their intended domestic functions.',
    lore: 'Three-dimensional readymades that explore gravity, balance, and spatial tension.',
  },
  {
    period: 'V',
    title: 'Monochrome Nihilism',
    subtitle: 'Minimalist compositions, blurs, pocket shots & void studies',
    description:
      'A culminating minimalist asceticism featuring austere minimalist compositions, atmospheric blurs, accidental pocket shots, and contemplative void studies. Visual information is radically subdued in favor of negative space and existential silence.',
    lore: 'An evocative meditation on sensory reduction, lens occlusion, and the abyss of the frame.',
  },
];

export default function EditorialBiography() {
  return (
    <section
      id="biography"
      aria-labelledby="biography-title"
      className="w-full py-16 md:py-24 border-b border-[var(--border-subtle)] bg-[var(--bg-primary)] transition-colors"
    >
      <div className="container mx-auto px-4 md:px-8 max-w-6xl">
        <header className="mb-12">
          <p className="text-xs uppercase tracking-widest text-[var(--text-secondary)] mb-2">
            Historical Movements Taxonomy
          </p>
          <h2
            id="biography-title"
            className="text-2xl md:text-3xl font-[family-name:var(--font-serif)] font-normal text-[var(--text-primary)] tracking-wide"
          >
            Chronology of Form &amp; Ideology
          </h2>
          <p className="text-sm text-[var(--text-secondary)] mt-3 max-w-2xl leading-relaxed">
            The canonical evolution of Rose Seraphine Tucker traces five distinct artistic movements, moving from foundational linear wax explorations to radical monochrome void studies.
          </p>
        </header>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 md:gap-8">
          {MOVEMENTS.map((movement) => (
            <article
              key={movement.title}
              className="group p-6 md:p-8 bg-[var(--bg-secondary)] border border-[var(--border-subtle)] hover:border-[var(--border-focus)] transition-colors flex flex-col justify-between"
            >
              <div>
                <div className="flex items-baseline justify-between mb-4">
                  <span className="text-xs font-mono text-[var(--text-secondary)] uppercase tracking-widest">
                    Phase {movement.period}
                  </span>
                  <span className="text-xs uppercase tracking-wider text-[var(--text-secondary)]">
                    Canonical
                  </span>
                </div>
                <h3 className="text-xl font-[family-name:var(--font-serif)] text-[var(--text-primary)] mb-2 group-hover:underline">
                  {movement.title}
                </h3>
                <p className="text-xs uppercase tracking-wider text-[var(--text-secondary)] mb-4 pb-3 border-b border-[var(--border-subtle)]">
                  {movement.subtitle}
                </p>
                <p className="text-sm leading-relaxed text-[var(--text-secondary)] mb-6">
                  {movement.description}
                </p>
              </div>

              <div className="pt-4 border-t border-[var(--border-subtle)] text-xs text-[var(--text-secondary)] italic">
                <span className="font-sans font-medium uppercase not-italic text-[10px] tracking-wider block text-[var(--text-primary)] mb-1">
                  Curatorial Caption
                </span>
                {movement.lore}
              </div>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}
