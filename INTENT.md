# Project Rose: Intent & High-Level Architecture

## 1. Project Vision

**Project Rose** is a bespoke digital retrospective and modern art gallery celebrating the artistic oeuvre of **Rose Seraphine Tucker**.

While the underlying works are personal childhood photos and drawings, the site presents them with complete deadpan sincerity as the output of an uncompromising, avant-garde contemporary artist. Every piece is treated as a profound cultural artifact, complete with philosophical critique, curatorial classifications, and exhibition-grade presentation.

The project is designed to scale to **1,000+ pieces**, maintaining high performance, automated synchronization, and rich accessibility across the entire body of work.

---

## 2. Core Pillars & Capabilities

```
+---------------------------+      +---------------------------+      +---------------------------+
|    Google Photos Album    |      |    Local Llama Vision     |      |    Next.js + Nginx        |
|    (Shared Album Link)    |      |    (OpenAI-compat API)    |      |    (Modern Gallery)       |
+-------------+-------------+      +-------------+-------------+      +-------------+-------------+
              |                                  |                                  ^
              v                                  v                                  |
    +-------------------------------------------------------------------------------+
    |                         Shared Storage / NFS Volume                           |
    |  - Original Images & WebP Thumbnails                                          |
    |  - Artwork Metadata Protobufs (*.proto.bin)                                   |
    |  - Sync State & Curatorial Index (.sync-state.json)                           |
    +-------------------------------------------------------------------------------+
```

### (a) Photo Ingestion & Storage (`rose-syncer`)
- **Source**: Automated extraction from a Google Photos shared album link, bypassing cloud API read limitations.
- **Execution Model**: Runs as a scheduled cron job to continuously discover and ingest newly added photos.
- **Storage Strategy**:
  - Persists high-resolution originals to shared local storage (NFS mount).
  - Automatically generates web-optimized thumbnails (e.g., compressed WebP) for responsive, fast gallery browsing.
  - Maintains an incremental sync state ledger (`.sync-state.json`) ensuring idempotent, duplicate-free processing.

### (b) AI Annotation & Curatorial Analysis
- **Inference Engine**: Local Llama instance with multimodal/vision capabilities (e.g., LLaVA, Llama 3.2 Vision), accessed via standard OpenAI-compatible `/v1/chat/completions` endpoint using base64 encoded images.
- **Voice & Tone**: Ultra-pretentious contemporary art critic. Every piece receives:
  - **Avant-garde Title**: E.g., *"Deconstructed Entropy No. 4"*, *"Ode to the Spilled Milk"*.
  - **Medium Specification**: Creative, serious descriptions of everyday child materials (e.g., *"Wax pigment on reclaimed cardboard"*, *"Organic polymer paste on synthetic carpet"*).
  - **Curatorial Critique**: Solemn philosophical analysis exploring tension, existential spatiality, and post-modern domesticity.
  - **Movement / Period Classification**: Assignments to evolving thematic periods (e.g., *"The Crayon Period"*, *"Domestic Destructionism"*, *"Kinetic Scribblism"*).
- **Extensible Grouping**: A flexible taxonomy system that adapts and synthesizes movements into cohesive exhibitions as the collection grows past 1,000+ works.
- **Metadata Serialization**: Outputs standardized Protocol Buffer definitions (`gallery.Artwork`, `gallery.Theme`) stored alongside each asset.

### (c) Modern Artist Gallery Experience (`rose-web`)
- **Identity & Aesthetics**: High-end minimalist gallery aesthetic (white cube / dark luxury typography and layout) celebrating the artist persona of Rose Seraphine Tucker.
- **Key Sections**:
  - **Artist Statement & Biography**: Authoritative, serious statement outlining the artist's conceptual philosophy.
  - **Permanent Collection & Exhibitions**: Dynamic filtering and exploration by artistic period, medium, and chronology.
  - **High-Performance Gallery**: Virtualized, lazy-loaded grid engineered to handle 1,000+ artworks fluidly without browser stutter.
  - **Artwork Detail View (Lightbox)**: Full-resolution viewing with plaque-style curatorial notes, medium, date, and critique.
- **Delivery Architecture**: Next.js static export deployed in a lightweight Nginx container, reading from the shared NFS volume and updated on sync cycles.

---

## 3. High-Level System Architecture & Data Contract

### Data Contracts
- **Protobuf Schema (`proto/gallery.proto`)**: Defines the canonical schema for `Artwork` and `Theme` entities, bridging the Go syncer and TypeScript web frontend.
- **Media Assets**:
  - Full Resolution: `/data/images/{id}.{ext}`
  - Thumbnails: `/data/thumbnails/{id}.webp`
  - Metadata: `/data/{id}.proto.bin`
  - Synchronization Index: `/data/.sync-state.json`

### Infrastructure
- **Syncer Container (`Dockerfile.syncer`)**: Go multi-stage build running periodically via cron.
- **Web Container (`Dockerfile`)**: Next.js static export served via Nginx.
- **CI/CD**: GitHub Actions workflows validating PRs and publishing tagged container images to GHCR.
