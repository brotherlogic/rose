#!/usr/bin/env bash
set -euo pipefail

# Infrastructure Manifest Verification Script
# Reference: brotherlogic/rose#67

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy"

echo "=== Verifying Rose Infrastructure Manifests ==="

REQUIRED_FILES=(
  "${DEPLOY_DIR}/rose-gallery-deploy.yaml"
  "${DEPLOY_DIR}/rose-ingress.yaml"
  "${DEPLOY_DIR}/rose-images.yaml"
)

for file in "${REQUIRED_FILES[@]}"; do
  if [ ! -f "$file" ]; then
    echo "ERROR: Missing required manifest file: $file" >&2
    exit 1
  fi
  echo "Found manifest: $(basename "$file")"
done

# Check cross-referencing in manifests
echo "Verifying cross-referencing comments..."
grep -q "brotherlogic/prod#974" "${DEPLOY_DIR}/rose-gallery-deploy.yaml" || (echo "Missing prod#974 ref" && exit 1)
grep -q "brotherlogic/prod#975" "${DEPLOY_DIR}/rose-ingress.yaml" || (echo "Missing prod#975 ref" && exit 1)
grep -q "brotherlogic/prod#975" "${DEPLOY_DIR}/rose-images.yaml" || (echo "Missing prod#975 ref" && exit 1)

for file in "${REQUIRED_FILES[@]}"; do
  grep -q "brotherlogic/rose#30" "$file" || (echo "Missing #30 ref in $file" && exit 1)
  grep -q "brotherlogic/rose#63" "$file" || (echo "Missing #63 ref in $file" && exit 1)
  grep -q "brotherlogic/rose#64" "$file" || (echo "Missing #64 ref in $file" && exit 1)
  grep -q "brotherlogic/rose#67" "$file" || (echo "Missing #67 ref in $file" && exit 1)
done

echo "Verifying namespace consistency..."
grep -q "namespace: rose" "${DEPLOY_DIR}/rose-gallery-deploy.yaml" || (echo "Missing rose namespace" && exit 1)
grep -q "namespace: rose" "${DEPLOY_DIR}/rose-ingress.yaml" || (echo "Missing rose namespace" && exit 1)
grep -q "namespace: flux-system" "${DEPLOY_DIR}/rose-images.yaml" || (echo "Missing flux-system namespace" && exit 1)

echo "Verifying host and port bindings..."
grep -q "roseseraphinetucker.com" "${DEPLOY_DIR}/rose-ingress.yaml" || (echo "Missing target host binding" && exit 1)
grep -q "port: 80" "${DEPLOY_DIR}/rose-gallery-deploy.yaml" || (echo "Missing port 80 binding" && exit 1)
grep -q "port: 80" "${DEPLOY_DIR}/rose-ingress.yaml" || (echo "Missing port 80 in ingress" && exit 1)

echo "=== All infrastructure manifests verified successfully ==="
