export default function Home() {
  return (
    <main className="main-content container">
      <header style={{ marginBottom: 'var(--space-xl)' }}>
        <h1 style={{ fontSize: '3rem', fontWeight: 'bold' }}>Artist Portfolio</h1>
        <p style={{ color: 'var(--text-secondary)', marginTop: 'var(--space-sm)' }}>
          Minimalist gallery showcasing automated artwork generation.
        </p>
      </header>
      <section>
        <div className="interactive-element" style={{ padding: 'var(--space-md)', border: '1px solid var(--border-color)', borderRadius: '8px' }}>
          <h2>Gallery Placeholder</h2>
          <p>Photos and Vision AI integration will populate here.</p>
        </div>
      </section>
    </main>
  );
}
