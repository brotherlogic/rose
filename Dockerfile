# Stage 1: Builder
FROM node:22-alpine AS builder

WORKDIR /app

# Install dependencies
COPY package*.json ./
RUN npm ci

# Copy application source files
COPY . .

# Generate static export in out/
RUN npm run build

# Stage 2: Runner
FROM nginx:alpine

# Create scaffolding for NFS media volume mounts
RUN mkdir -p /data/images /data/thumbnails

# Copy static export bundle from builder
COPY --from=builder /app/out /usr/share/nginx/html

# Copy custom NGINX configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose port 80
EXPOSE 80

# Native healthcheck querying local Nginx /healthz endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost/healthz || exit 1

# Run NGINX in foreground
CMD ["nginx", "-g", "daemon off;"]
