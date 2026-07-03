import { getArtworks } from '@/lib/nfs';

export default function Home() {
  const artworks = getArtworks();
  
  return (
    <main className="main-content container">
      <header style={{ marginBottom: 'var(--space-xl)' }}>
        <h1 style={{ fontSize: '3rem', fontWeight: 'bold' }}>Artist Portfolio</h1>
        <p style={{ color: 'var(--text-secondary)', marginTop: 'var(--space-sm)' }}>
          Minimalist gallery showcasing automated artwork generation.
        </p>
      </header>
      <section style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: 'var(--space-md)' }}>
        {artworks.length === 0 ? (
          <div className="interactive-element" style={{ padding: 'var(--space-md)', border: '1px solid var(--border-color)', borderRadius: '8px' }}>
            <h2>No Artworks Found</h2>
            <p>Waiting for the backend job to fetch photos and generate metadata.</p>
          </div>
        ) : (
          artworks.map((artwork) => (
            <div key={artwork.id} className="interactive-element" style={{ padding: 'var(--space-md)', border: '1px solid var(--border-color)', borderRadius: '8px' }}>
              <h2 style={{ fontSize: '1.5rem', marginBottom: 'var(--space-xs)' }}>{artwork.title}</h2>
              <p style={{ color: 'var(--text-secondary)', marginBottom: 'var(--space-sm)' }}>{artwork.description}</p>
              {artwork.imagePath && (
                <div style={{ marginTop: 'var(--space-sm)' }}>
                  <img src={artwork.imagePath} alt={artwork.title || 'Artwork'} style={{ width: '100%', borderRadius: '4px', objectFit: 'cover' }} />
                </div>
              )}
            </div>
          ))
        )}
      </section>
    </main>
  );
}
