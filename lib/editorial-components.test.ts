import { describe, it, expect } from 'vitest';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import EditorialStatement from '../app/components/EditorialStatement';
import EditorialBiography from '../app/components/EditorialBiography';

describe('Editorial Components Suite (#254)', () => {
  describe('Curatorial Editorial Statement (app/components/EditorialStatement.tsx)', () => {
    it('should render section with anchor id="statement"', () => {
      const markup = renderToStaticMarkup(React.createElement(EditorialStatement));
      expect(markup).toContain('id="statement"');
      expect(markup).toContain('<section');
    });

    it('should articulate the conceptual thesis (domesticity vs destruction, ephemeral materiality, existential spatiality)', () => {
      const markup = renderToStaticMarkup(React.createElement(EditorialStatement));
      expect(markup).toContain('domesticity');
      expect(markup).toContain('destruction');
      expect(markup).toContain('ephemeral materiality');
      expect(markup).toContain('existential spatiality');
    });

    it('should feature elegant typography styling with serif font and drop-quote visual treatment', () => {
      const markup = renderToStaticMarkup(React.createElement(EditorialStatement));
      expect(markup).toMatch(/var\(--font-serif\)/);
      // Drop-quote / quotation treatment: blockquote element or quote marks
      expect(markup).toMatch(/<blockquote|“|&#x201C;|&ldquo;|drop-quote/);
    });

    it('should render formal curatorial attribution', () => {
      const markup = renderToStaticMarkup(React.createElement(EditorialStatement));
      expect(markup).toMatch(/Curatorial Statement|Curatorial Note|Curator/i);
      expect(markup).toContain('Rose Seraphine Tucker');
    });
  });

  describe('Historical Movements Biography (app/components/EditorialBiography.tsx)', () => {
    it('should render section with anchor id="biography"', () => {
      const markup = renderToStaticMarkup(React.createElement(EditorialBiography));
      expect(markup).toContain('id="biography"');
      expect(markup).toContain('<section');
    });

    it('should render the first 3 canonical movements in chronological order when artworks is omitted or empty', () => {
      const markup = renderToStaticMarkup(React.createElement(EditorialBiography));
      const articleMatches = markup.match(/<article\b/g) ?? [];
      expect(articleMatches).toHaveLength(3);

      expect(markup).toContain('Phase I');
      expect(markup).toContain('The Crayon Period');
      expect(markup).toContain('Phase II');
      expect(markup).toContain('Domestic Destructionism');
      expect(markup).toContain('Phase III');
      expect(markup).toContain('Kinetic Scribblism');

      expect(markup).not.toContain('Found Object Assemblage');
      expect(markup).not.toContain('Monochrome Nihilism');

      const idxI = markup.indexOf('The Crayon Period');
      const idxII = markup.indexOf('Domestic Destructionism');
      const idxIII = markup.indexOf('Kinetic Scribblism');
      expect(idxI).toBeLessThan(idxII);
      expect(idxII).toBeLessThan(idxIII);
    });

    it('should dynamically select the top 3 preponderant canonical phases, break ties by canonical order, and render in canonical chronological order', () => {
      const artworks = [
        { id: '1', theme: 'Monochrome Nihilism' },
        { id: '2', theme: 'Monochrome Nihilism' },
        { id: '3', theme: 'Monochrome Nihilism' },
        { id: '4', themeId: 'Found Object Assemblage' },
        { id: '5', themeId: 'Found Object Assemblage' },
        { id: '6', theme: 'Early Works' },
      ];
      const markup = renderToStaticMarkup(
        React.createElement(EditorialBiography, { artworks })
      );
      const articleMatches = markup.match(/<article\b/g) ?? [];
      expect(articleMatches).toHaveLength(3);

      // Top 2 are Phase V (3) and Phase IV (2); 3rd slot tied at 0 goes to Phase I
      expect(markup).toContain('Phase I');
      expect(markup).toContain('The Crayon Period');
      expect(markup).toContain('Phase IV');
      expect(markup).toContain('Found Object Assemblage');
      expect(markup).toContain('Phase V');
      expect(markup).toContain('Monochrome Nihilism');

      expect(markup).not.toContain('Domestic Destructionism');
      expect(markup).not.toContain('Kinetic Scribblism');

      // Rendered left-to-right in canonical chronological order (Phase I -> Phase IV -> Phase V)
      const idxI = markup.indexOf('The Crayon Period');
      const idxIV = markup.indexOf('Found Object Assemblage');
      const idxV = markup.indexOf('Monochrome Nihilism');
      expect(idxI).toBeLessThan(idxIV);
      expect(idxIV).toBeLessThan(idxV);
    });

    it('should describe the formative aesthetic concepts and curatorial captions across all 5 canonical movements', () => {
      const earlyPhasesMarkup = renderToStaticMarkup(
        React.createElement(EditorialBiography, { artworks: [] })
      );
      expect(earlyPhasesMarkup.match(/<article\b/g) ?? []).toHaveLength(3);
      // Movement 1
      expect(earlyPhasesMarkup).toContain('primitive marks');
      expect(earlyPhasesMarkup).toContain('linear wax explorations');
      // Movement 2
      expect(earlyPhasesMarkup).toContain('entropy');
      expect(earlyPhasesMarkup).toContain('spills');
      expect(earlyPhasesMarkup).toContain('structural disruption');
      // Movement 3
      expect(earlyPhasesMarkup).toContain('frantic gestures');
      expect(earlyPhasesMarkup).toContain('chaotic linework');

      const laterPhasesMarkup = renderToStaticMarkup(
        React.createElement(EditorialBiography, {
          artworks: [
            { id: '1', theme: 'Kinetic Scribblism' },
            { id: '2', theme: 'Found Object Assemblage' },
            { id: '3', theme: 'Monochrome Nihilism' },
          ],
        })
      );
      expect(laterPhasesMarkup.match(/<article\b/g) ?? []).toHaveLength(3);
      // Movement 4
      expect(laterPhasesMarkup).toContain('sculptural household arrangements');
      // Movement 5
      expect(laterPhasesMarkup).toContain('minimalist compositions');
      expect(laterPhasesMarkup).toContain('pocket shots');
      expect(laterPhasesMarkup).toContain('void studies');
    });

    it('should utilize a responsive grid layout with border separators and curatorial captions', () => {
      const markup = renderToStaticMarkup(React.createElement(EditorialBiography));
      expect(markup).toMatch(/grid|display:\s*grid/);
      expect(markup).toMatch(/border|border-subtle|var\(--border-subtle\)/);
    });
  });
});
