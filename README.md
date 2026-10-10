# Rose
Automated Gallery Website

## Frontend
The frontend is a static Next.js application built with Vanilla CSS design tokens.
Run `npm run dev` to start the development server.

### Components
- `ArtworkCard`: Museum-grade artwork card featuring fixed aspect-ratio containers (`aspect-square`) with zero Cumulative Layout Shift (CLS), shimmer placeholder skeleton states, lazy-loaded WebP thumbnails, error fallbacks to `/placeholder.svg`, and archival typography.

## Syncer
The syncer (`cmd/syncer`) discovers and ingests photos, generates WebP thumbnails, records artwork metadata, and automatically backfills missing thumbnails for previously processed photos.

