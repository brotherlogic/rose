# Project Rose: Milestones & Visual Verification Plan

This document outlines the phased milestones for Project Rose. Each milestone defines explicit functional deliverables, Prometheus metrics, and **visual confirmation criteria** to ensure progress is tangible and observable rather than just code residing in the repository.

Refer to [INTENT.md](file:///workspaces/rose/INTENT.md) for the high-level project vision, architecture, and design principles.

---

## Milestone Summary

| Milestone | Focus Area | Primary Deliverable | Visual Confirmation Mechanism |
| :--- | :--- | :--- | :--- |
| **M1** | Ingestion & Storage | Shared album scraper, thumbnail generator, local storage | **Grafana Ingestion Panel** showing $N$ photos downloaded, thumbnails generated, and storage bytes. |
| **M2** | AI Vision Annotation | Local Llama OpenAI-compatible vision client, pretentious curatorial critique, proto serialization | **Grafana AI Curation Panel** showing annotation counts, latency, and movement distribution chart. |
| **M3** | Artist Web Gallery | Next.js gallery for Rose Seraphine Tucker, virtualized grid (>1,000 works), lightbox plaques | **Live Web Gallery UI** displaying the white-cube museum aesthetic, artist bio, filtered exhibitions, and lightbox. |
| **M4** | Automation & Production | Kubernetes CronJob setup, automated catalog refresh, unified monitoring | **End-to-End Pipeline Dashboard** in Grafana showing hands-free sync execution when photos are added. |

---

## Milestone 1: Photo Ingestion & Storage Pipeline

### Objective
Automate the discovery and download of photos from a shared Google Photos album link into persistent shared storage (NFS), generate web-optimized WebP thumbnails, maintain an incremental sync state, and emit Prometheus metrics.

### Deliverables
1. **Google Photos Ingestion Engine (`internal/photos`)**:
   - Implement `PhotoService.FetchPhotos` to scrape/parse the shared Google Photos album link and return media download URLs.
   - Stream and persist full-resolution images to `/data/images/{id}.jpg`.
2. **Thumbnail Generation Engine (`internal/storage`)**:
   - Downscale images to generate lightweight, web-optimized thumbnails (e.g., max 600px width/height in WebP format) at `/data/thumbnails/{id}.webp`.
3. **Sync State Ledger**:
   - Record processed photo IDs in `/data/.sync-state.json` to ensure subsequent runs are incremental and idempotent.
4. **Prometheus Exporter (`internal/metrics`)**:
   - `rose_syncer_photos_discovered_total`: Total count of photos discovered in the shared album.
   - `rose_syncer_photos_downloaded_total`: Count of newly downloaded images.
   - `rose_syncer_thumbnails_generated_total`: Count of WebP thumbnails created.
   - `rose_syncer_storage_bytes`: Disk usage of the storage volume.
5. **Grafana Dashboard Definition**:
   - Provide `dashboards/rose-syncer.json` with visual gauges and timeseries for ingestion progress.

### Visual Confirmation Criteria
- [ ] **Grafana Dashboard Ingestion View**:
  - The "Photos Ingested" single-stat panel displays $N > 0$ (progressing toward all 1,000+ album photos).
  - The "Thumbnails Generated" counter matches the downloaded count.
  - The "Storage Volume" panel graphs storage growth.
- [ ] **Local Storage Inspection**:
  - Viewing `/data/thumbnails/` shows generated WebP thumbnails for all downloaded photos.

---

## Milestone 2: AI Vision Annotation & Thematic Taxonomy

### Objective
Integrate a local Llama multimodal/vision model via an OpenAI-compatible endpoint to analyze downloaded images, generating ultra-pretentious art titles, medium descriptions, philosophical critiques, and thematic movements serialized into `proto/gallery.proto`.

### Deliverables
1. **Multimodal Vision Service (`internal/vision`)**:
   - Implement `VisionService.AnalyzeImage` using standard HTTP POST to `http://<llama-host>:<port>/v1/chat/completions`.
   - Send base64-encoded thumbnails/images with structured system prompts commanding an ultra-pretentious contemporary art critic persona.
   - Extract structured JSON containing:
     - `title`: Avant-garde, deadpan serious title (e.g., *"Deconstructed Entropy No. 4"*).
     - `medium`: Inventive description of toddler/child materials (e.g., *"Wax pigment on reclaimed cardboard"*).
     - `description`: Solemn philosophical and existential critique.
     - `theme`: Artistic movement or thematic period (e.g., *"The Crayon Period"*, *"Domestic Destructionism"*).
2. **Schema & Serialization (`proto/gallery.proto` & `internal/storage`)**:
   - Ensure `Artwork` proto includes `medium`, `title`, `description`, `theme_id`, `timestamp`, and image paths.
   - Write serialized binary protos to `/data/{id}.proto.bin`.
3. **Prometheus Metrics**:
   - `rose_syncer_photos_annotated_total`: Total count of artworks successfully analyzed.
   - `rose_syncer_annotation_duration_seconds`: Histogram/summary of AI vision inference time.
   - `rose_syncer_annotation_errors_total`: Count of inference failures.
   - `rose_syncer_themes_total`: Count of distinct thematic movements identified.
4. **Grafana Dashboard Curation View**:
   - Add panels to `dashboards/rose-syncer.json` for AI annotation velocity, error rate, and a pie/bar chart showing the breakdown of artistic movements.

### Visual Confirmation Criteria
- [ ] **Grafana AI Curation Panels**:
  - "Artworks Annotated" gauge matches the downloaded photo count.
  - "Artistic Movements" pie chart visually displays categorized periods (e.g., "The Crayon Period", "Kinetic Scribblism").
  - "Annotation Latency" panel displays stable response times with zero fatal errors.
- [ ] **Curatorial Output Verification**:
  - Inspection of sample `.proto.bin` files confirms deadpan high-brow critiques and creative medium labels.

---

## Milestone 3: High-Scale Artist Gallery Website

### Objective
Deliver a responsive, modern artist gallery website for **Rose Seraphine Tucker**, built with Next.js and styled with a minimalist "white cube / dark luxury" aesthetic, engineered to display > 1,000 artworks fluidly with virtualized scrolling, exhibition filters, and a curatorial lightbox.

### Deliverables
1. **Artist Identity & Editorial Framing**:
   - Dedicated header and artist bio/statement presenting Rose Seraphine Tucker as a groundbreaking conceptual artist.
2. **High-Performance Gallery Grid (`app/page.tsx`)**:
   - Virtualized or progressive lazy-loaded grid capable of smoothly browsing 1,000+ items without DOM bloat or performance degradation.
   - Responsive layout loading `/data/thumbnails/{id}.webp` for instantaneous grid rendering.
3. **Exhibition / Period Filter**:
   - Interactive filtering by artistic movement/period (e.g., "The Crayon Period", "Domestic Destructionism") and chronology.
4. **Artwork Plaque & Lightbox Modal**:
   - Fullscreen/modal artwork detail view displaying high-res image, museum-style curatorial plaque, title, year, medium, and philosophical critique.
5. **Static Export Integration**:
   - Compatible with `output: 'export'` served via Nginx in Docker.

### Visual Confirmation Criteria
- [ ] **Live Browser Demonstration**:
  - Visiting the website in a browser displays the refined Rose Seraphine Tucker gallery branding.
  - Scrolling through all 1,000+ artworks maintains smooth 60fps rendering with lazy-loaded thumbnails.
  - Clicking on an artistic movement filter immediately updates the gallery grid to show only works from that period.
  - Clicking any artwork opens the lightbox, displaying the high-res artwork accompanied by its pretentious curatorial plaque.

---

## Milestone 4: Cron Syncer Automation & Production Deployment

### Objective
Deploy `rose-syncer` as an automated scheduled CronJob in the Kubernetes cluster (`brotherlogic/prod`), automate website catalog refreshes when new artworks arrive, and deliver a unified end-to-end monitoring dashboard.

### Deliverables
1. **Kubernetes CronJob Manifests**:
   - Configure scheduled execution for `Dockerfile.syncer` in `brotherlogic/prod` with shared NFS volume mounts and resource limits.
   - Expose Prometheus metrics for scraping by the cluster Prometheus instance.
2. **End-to-End Refresh Pipeline**:
   - Ensure the gallery frontend automatically detects or receives updated metadata when syncer completes a run without manual container rebuilds.
3. **Complete Unified Grafana Dashboard**:
   - Finalized production Grafana dashboard monitoring the entire pipeline:
     - Album Sync Status & Discovery Count
     - Local Storage Allocation (Originals + Thumbnails)
     - AI Vision Inference Metrics & Movement Distribution
     - Web Gallery Health & Total Works Cataloged

### Visual Confirmation Criteria
- [ ] **Hands-Free Sync Verification**:
  - Adding a new photo to the Google Photos shared album results in the new artwork appearing in the web gallery on the subsequent cron cycle without developer intervention.
- [ ] **Unified Grafana Pipeline Dashboard**:
  - A single Grafana dashboard in `brotherlogic/prod` displaying 100% green health across Ingestion, AI Curation, Storage, and Web Catalog status.
