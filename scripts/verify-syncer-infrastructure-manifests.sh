#!/usr/bin/env bash
set -euo pipefail

# Syncer Infrastructure Manifest Verification Script
# Reference: brotherlogic/rose#75

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy"

echo "=== Verifying Rose Syncer Infrastructure Manifests ==="

REQUIRED_FILES=(
  "${DEPLOY_DIR}/rose-syncer-cronjob.yaml"
  "${DEPLOY_DIR}/rose-alerts.yaml"
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
grep -q "brotherlogic/prod#976" "${DEPLOY_DIR}/rose-syncer-cronjob.yaml" || (echo "Missing prod#976 ref" && exit 1)
grep -q "brotherlogic/prod#977" "${DEPLOY_DIR}/rose-alerts.yaml" || (echo "Missing prod#977 ref" && exit 1)

for file in "${REQUIRED_FILES[@]}"; do
  grep -q "brotherlogic/rose#31" "$file" || (echo "Missing #31 ref in $file" && exit 1)
  grep -q "brotherlogic/rose#71" "$file" || (echo "Missing #71 ref in $file" && exit 1)
  grep -q "brotherlogic/rose#72" "$file" || (echo "Missing #72 ref in $file" && exit 1)
  grep -q "brotherlogic/rose#75" "$file" || (echo "Missing #75 ref in $file" && exit 1)
done

grep -q "brotherlogic/rose#73" "${DEPLOY_DIR}/rose-syncer-cronjob.yaml" || (echo "Missing #73 ref in cronjob" && exit 1)
grep -q "brotherlogic/rose#74" "${DEPLOY_DIR}/rose-alerts.yaml" || (echo "Missing #74 ref in alerts" && exit 1)

echo "Verifying namespace consistency..."
grep -q "namespace: rose" "${DEPLOY_DIR}/rose-syncer-cronjob.yaml" || (echo "Missing rose namespace in cronjob" && exit 1)
grep -q "namespace: rose" "${DEPLOY_DIR}/rose-alerts.yaml" || (echo "Missing rose namespace in alerts" && exit 1)

echo "Verifying volume mounts and security context..."
grep -q "rose-data-pvc" "${DEPLOY_DIR}/rose-syncer-cronjob.yaml" || (echo "Missing rose-data-pvc" && exit 1)
grep -q "rose-gcp-sa-secret" "${DEPLOY_DIR}/rose-syncer-cronjob.yaml" || (echo "Missing rose-gcp-sa-secret" && exit 1)
grep -q "fsGroup: 65532" "${DEPLOY_DIR}/rose-syncer-cronjob.yaml" || (echo "Missing fsGroup: 65532" && exit 1)

echo "Verifying PrometheusRule alert expr..."
grep -q "rose-syncer-.*" "${DEPLOY_DIR}/rose-alerts.yaml" || (echo "Missing alert expr job name regex" && exit 1)

echo "=== All syncer infrastructure manifests verified successfully ==="
