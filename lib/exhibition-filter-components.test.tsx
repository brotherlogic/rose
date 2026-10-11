import { describe, it, expect, vi } from 'vitest';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import ExhibitionFilterBar from '../app/components/ExhibitionFilterBar';
import ExhibitionEmptyState from '../app/components/ExhibitionEmptyState';

// Helper to recursively find elements by type or predicate in a React element tree
function findElements(node: React.ReactNode, predicate: (el: React.ReactElement) => boolean): React.ReactElement[] {
  const matches: React.ReactElement[] = [];

  function traverse(current: React.ReactNode) {
    if (!current) return;
    if (Array.isArray(current)) {
      current.forEach(traverse);
      return;
    }
    if (React.isValidElement(current)) {
      if (predicate(current)) {
        matches.push(current);
      }
      if (current.props && (current.props as { children?: React.ReactNode }).children) {
        traverse((current.props as { children?: React.ReactNode }).children);
      }
    }
  }

  traverse(node);
  return matches;
}

describe('ExhibitionFilterBar & ExhibitionEmptyState Components Suite (#277)', () => {
  const mockMovements = [
    { name: 'The Crayon Period', count: 24 },
    { name: 'Domestic Destructionism', count: 18 },
    { name: 'Kinetic Scribblism', count: 7 },
  ];
  const totalCount = 49;

  describe('ExhibitionFilterBar Component (app/components/ExhibitionFilterBar.tsx)', () => {
    describe('Semantic Markup & Accessibility', () => {
      it('should render semantic <nav aria-label="Exhibition period filtering" role="tablist">', () => {
        const markup = renderToStaticMarkup(
          React.createElement(ExhibitionFilterBar, {
            movements: mockMovements,
            totalCount,
            selectedMovement: null,
            sortDirection: 'desc',
            onSelectMovement: vi.fn(),
            onToggleSort: vi.fn(),
          })
        );

        expect(markup).toContain('<nav');
        expect(markup).toContain('aria-label="Exhibition period filtering"');
        expect(markup).toContain('role="tablist"');
      });

      it('should render horizontal scroll container with hidden scrollbars for mobile viewports', () => {
        const markup = renderToStaticMarkup(
          React.createElement(ExhibitionFilterBar, {
            movements: mockMovements,
            totalCount,
            selectedMovement: null,
            sortDirection: 'desc',
            onSelectMovement: vi.fn(),
            onToggleSort: vi.fn(),
          })
        );

        expect(markup).toContain('overflow-x-auto');
        expect(markup).toMatch(/scrollbar-none|\[scrollbar-width:none\]|\[-ms-overflow-style:none\]/);
      });
    });

    describe('Filter Pills Rendering & Active State Highlighting', () => {
      it('should render "All Works" as the first pill with total count', () => {
        const markup = renderToStaticMarkup(
          React.createElement(ExhibitionFilterBar, {
            movements: mockMovements,
            totalCount,
            selectedMovement: null,
            sortDirection: 'desc',
            onSelectMovement: vi.fn(),
            onToggleSort: vi.fn(),
          })
        );

        expect(markup).toContain('All Works');
        expect(markup).toContain('49');
      });

      it('should render all movement pills with correct counts and names', () => {
        const markup = renderToStaticMarkup(
          React.createElement(ExhibitionFilterBar, {
            movements: mockMovements,
            totalCount,
            selectedMovement: null,
            sortDirection: 'desc',
            onSelectMovement: vi.fn(),
            onToggleSort: vi.fn(),
          })
        );

        for (const mov of mockMovements) {
          expect(markup).toContain(mov.name);
          expect(markup).toContain(String(mov.count));
        }
      });

      it('should highlight "All Works" pill with active design tokens when selectedMovement is null', () => {
        const markup = renderToStaticMarkup(
          React.createElement(ExhibitionFilterBar, {
            movements: mockMovements,
            totalCount,
            selectedMovement: null,
            sortDirection: 'desc',
            onSelectMovement: vi.fn(),
            onToggleSort: vi.fn(),
          })
        );

        // Active tokens: border-[var(--text-primary)] text-[var(--text-primary)] bg-[var(--bg-secondary)]
        expect(markup).toContain('border-[var(--text-primary)]');
        expect(markup).toContain('text-[var(--text-primary)]');
        expect(markup).toContain('bg-[var(--bg-secondary)]');
      });

      it('should highlight active movement pill when selectedMovement is set and un-highlight others', () => {
        const markup = renderToStaticMarkup(
          React.createElement(ExhibitionFilterBar, {
            movements: mockMovements,
            totalCount,
            selectedMovement: 'Domestic Destructionism',
            sortDirection: 'desc',
            onSelectMovement: vi.fn(),
            onToggleSort: vi.fn(),
          })
        );

        expect(markup).toContain('Domestic Destructionism');
        expect(markup).toContain('aria-selected="true"');
      });
    });

    describe('Pills Callback Invocations', () => {
      it('should invoke onSelectMovement(null) when clicking "All Works" pill', () => {
        const onSelectMovement = vi.fn();
        const tree = ExhibitionFilterBar({
          movements: mockMovements,
          totalCount,
          selectedMovement: 'Domestic Destructionism',
          sortDirection: 'desc',
          onSelectMovement,
          onToggleSort: vi.fn(),
        });

        const buttons = findElements(tree, (el) => el.type === 'button');
        const allWorksBtn = buttons.find((btn) => {
          const content = JSON.stringify(btn.props);
          return content.includes('All Works');
        });

        expect(allWorksBtn).toBeDefined();
        allWorksBtn?.props.onClick?.({ preventDefault: () => {} });
        expect(onSelectMovement).toHaveBeenCalledWith(null);
      });

      it('should invoke onSelectMovement(movementName) when clicking an inactive movement pill', () => {
        const onSelectMovement = vi.fn();
        const tree = ExhibitionFilterBar({
          movements: mockMovements,
          totalCount,
          selectedMovement: null,
          sortDirection: 'desc',
          onSelectMovement,
          onToggleSort: vi.fn(),
        });

        const buttons = findElements(tree, (el) => el.type === 'button');
        const crayonBtn = buttons.find((btn) => {
          const content = JSON.stringify(btn.props);
          return content.includes('The Crayon Period');
        });

        expect(crayonBtn).toBeDefined();
        crayonBtn?.props.onClick?.({ preventDefault: () => {} });
        expect(onSelectMovement).toHaveBeenCalledWith('The Crayon Period');
      });

      it('should invoke onSelectMovement(null) when clicking the currently active movement pill', () => {
        const onSelectMovement = vi.fn();
        const tree = ExhibitionFilterBar({
          movements: mockMovements,
          totalCount,
          selectedMovement: 'The Crayon Period',
          sortDirection: 'desc',
          onSelectMovement,
          onToggleSort: vi.fn(),
        });

        const buttons = findElements(tree, (el) => el.type === 'button');
        const activeBtn = buttons.find((btn) => {
          const content = JSON.stringify(btn.props);
          return content.includes('The Crayon Period');
        });

        expect(activeBtn).toBeDefined();
        activeBtn?.props.onClick?.({ preventDefault: () => {} });
        expect(onSelectMovement).toHaveBeenCalledWith(null);
      });
    });

    describe('Chronological Sort Toggle', () => {
      it('should display "Newest First" and invoke onToggleSort("asc") when sortDirection is "desc"', () => {
        const onToggleSort = vi.fn();
        const tree = ExhibitionFilterBar({
          movements: mockMovements,
          totalCount,
          selectedMovement: null,
          sortDirection: 'desc',
          onSelectMovement: vi.fn(),
          onToggleSort,
        });

        const buttons = findElements(tree, (el) => el.type === 'button');
        const sortBtn = buttons.find((btn) => {
          const content = JSON.stringify(btn.props);
          return content.includes('Newest First');
        });

        expect(sortBtn).toBeDefined();
        expect(sortBtn?.props['aria-label']).toBeDefined();
        sortBtn?.props.onClick?.({ preventDefault: () => {} });
        expect(onToggleSort).toHaveBeenCalledWith('asc');
      });

      it('should display "Earliest First" and invoke onToggleSort("desc") when sortDirection is "asc"', () => {
        const onToggleSort = vi.fn();
        const tree = ExhibitionFilterBar({
          movements: mockMovements,
          totalCount,
          selectedMovement: null,
          sortDirection: 'asc',
          onSelectMovement: vi.fn(),
          onToggleSort,
        });

        const buttons = findElements(tree, (el) => el.type === 'button');
        const sortBtn = buttons.find((btn) => {
          const content = JSON.stringify(btn.props);
          return content.includes('Earliest First');
        });

        expect(sortBtn).toBeDefined();
        sortBtn?.props.onClick?.({ preventDefault: () => {} });
        expect(onToggleSort).toHaveBeenCalledWith('desc');
      });
    });
  });

  describe('ExhibitionEmptyState Component (app/components/ExhibitionEmptyState.tsx)', () => {
    it('should render dignified curatorial message with active movement name', () => {
      const markup = renderToStaticMarkup(
        React.createElement(ExhibitionEmptyState, {
          movementName: 'Kinetic Scribblism',
          onReset: vi.fn(),
        })
      );

      expect(markup).toContain('Kinetic Scribblism');
      expect(markup).toContain('No works from this movement are currently on public exhibition.');
    });

    it('should render quick-action button "Return to All Works" and invoke onReset when clicked', () => {
      const onReset = vi.fn();
      const tree = ExhibitionEmptyState({
        movementName: 'Kinetic Scribblism',
        onReset,
      });

      const buttons = findElements(tree, (el) => el.type === 'button');
      const resetBtn = buttons.find((btn) => {
        const content = JSON.stringify(btn.props);
        return content.includes('Return to All Works');
      });

      expect(resetBtn).toBeDefined();
      resetBtn?.props.onClick?.({ preventDefault: () => {} });
      expect(onReset).toHaveBeenCalledTimes(1);
    });

    it('should be styled consistent with gallery minimalist aesthetic using dashed border', () => {
      const markup = renderToStaticMarkup(
        React.createElement(ExhibitionEmptyState, {
          movementName: 'Domestic Destructionism',
          onReset: vi.fn(),
        })
      );

      expect(markup).toMatch(/border\s+border-dashed\s+border-\[var\(--border-subtle\)\]/);
    });
  });
});
