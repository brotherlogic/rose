#!/usr/bin/env bash
set -euo pipefail

# Generate Go bindings
protoc --go_out=. --go_opt=paths=source_relative proto/gallery.proto

# Generate TypeScript static module
npx pbjs -t static-module -w commonjs -o lib/proto/gallery.js proto/gallery.proto

# Generate TypeScript definitions
npx pbts -o lib/proto/gallery.d.ts lib/proto/gallery.js
