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

    it('should present all 5 canonical artistic movements in the taxonomy', () => {
      const markup = renderToStaticMarkup(React.createElement(EditorialBiography));
      expect(markup).toContain('The Crayon Period');
      expect(markup).toContain('Domestic Destructionism');
      expect(markup).toContain('Kinetic Scribblism');
      expect(markup).toContain('Found Object Assemblage');
      expect(markup).toContain('Monochrome Nihilism');
    });

    it('should describe the formative aesthetic concepts for each movement', () => {
      const markup = renderToStaticMarkup(React.createElement(EditorialBiography));
      // Movement 1
      expect(markup).toContain('primitive marks');
      expect(markup).toContain('linear wax explorations');
      // Movement 2
      expect(markup).toContain('entropy');
      expect(markup).toContain('spills');
      expect(markup).toContain('structural disruption');
      // Movement 3
      expect(markup).toContain('frantic gestures');
      expect(markup).toContain('chaotic linework');
      // Movement 4
      expect(markup).toContain('sculptural household arrangements');
      // Movement 5
      expect(markup).toContain('minimalist compositions');
      expect(markup).toContain('pocket shots');
      expect(markup).toContain('void studies');
    });

    it('should utilize a responsive grid layout with border separators and curatorial captions', () => {
      const markup = renderToStaticMarkup(React.createElement(EditorialBiography));
      expect(markup).toMatch(/grid|display:\s*grid/);
      expect(markup).toMatch(/border|border-subtle|var\(--border-subtle\)/);
    });
  });
});
