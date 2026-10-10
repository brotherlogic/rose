# Rose
Automated Gallery Website

## Frontend
The frontend is a static Next.js application built with Vanilla CSS design tokens.
Run `npm run dev` to start the development server.

### Components
- `ArtworkCard`: Museum-grade artwork card featuring fixed aspect-ratio containers (`aspect-square`) with zero Cumulative Layout Shift (CLS), shimmer placeholder skeleton states, lazy-loaded WebP thumbnails, error fallbacks to `/placeholder.svg`, and archival typography.
- `VirtualizedGalleryGrid`: Zero-dependency responsive row-based virtualized windowing grid strictly bounding active mounted DOM nodes (< 50 active items for 1,000+ works) across viewports with overscan buffer, requestAnimationFrame throttling, and museum-grade empty state.

## Syncer
The syncer (`cmd/syncer`) discovers and ingests photos, generates WebP thumbnails, records artwork metadata, and automatically backfills missing thumbnails for previously processed photos.

