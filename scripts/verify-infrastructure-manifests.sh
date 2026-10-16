#!/usr/bin/env bash
set -euo pipefail

# Infrastructure Manifest Verification Script
# Reference: brotherlogic/rose#67, brotherlogic/rose#312, brotherlogic/rose#350, brotherlogic/rose#355

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_DIR="${ROOT_DIR}/deploy"
PVC_FILE="${DEPLOY_DIR}/rose-data-pvc.yaml"

echo "=== Verifying Rose Infrastructure Manifests ==="

if [ ! -f "$PVC_FILE" ]; then
  echo "ERROR: Missing required manifest file: $PVC_FILE" >&2
  exit 1
fi
echo "Found manifest: $(basename "$PVC_FILE")"

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
grep -q "brotherlogic/prod#968" "$PVC_FILE" || (echo "Missing prod#968 ref in PVC" && exit 1)
grep -q "brotherlogic/prod#1024" "$PVC_FILE" || (echo "Missing prod#1024 ref in PVC" && exit 1)
grep -q "brotherlogic/rose#312" "$PVC_FILE" || (echo "Missing #312 ref in PVC" && exit 1)
grep -q "brotherlogic/rose#350" "$PVC_FILE" || (echo "Missing #350 ref in PVC" && exit 1)

grep -q "brotherlogic/prod#974" "${DEPLOY_DIR}/rose-gallery-deploy.yaml" || (echo "Missing prod#974 ref" && exit 1)
grep -q "brotherlogic/prod#1026" "${DEPLOY_DIR}/rose-gallery-deploy.yaml" || (echo "Missing prod#1026 ref" && exit 1)
grep -q "brotherlogic/rose#312" "${DEPLOY_DIR}/rose-gallery-deploy.yaml" || (echo "Missing #312 ref in deploy" && exit 1)
grep -q "brotherlogic/rose#350" "${DEPLOY_DIR}/rose-gallery-deploy.yaml" || (echo "Missing #350 ref in deploy" && exit 1)

grep -q "brotherlogic/prod#975" "${DEPLOY_DIR}/rose-ingress.yaml" || (echo "Missing prod#975 ref" && exit 1)
grep -q "brotherlogic/prod#975" "${DEPLOY_DIR}/rose-images.yaml" || (echo "Missing prod#975 ref" && exit 1)

for file in "${REQUIRED_FILES[@]}"; do
  grep -q "brotherlogic/rose#30" "$file" || (echo "Missing #30 ref in $file" && exit 1)
  grep -q "brotherlogic/rose#63" "$file" || (echo "Missing #63 ref in $file" && exit 1)
  grep -q "brotherlogic/rose#64" "$file" || (echo "Missing #64 ref in $file" && exit 1)
  grep -q "brotherlogic/rose#67" "$file" || (echo "Missing #67 ref in $file" && exit 1)
done

echo "Verifying namespace consistency..."
grep -q "namespace: rose" "$PVC_FILE" || (echo "Missing rose namespace in PVC" && exit 1)
grep -q "namespace: rose" "${DEPLOY_DIR}/rose-gallery-deploy.yaml" || (echo "Missing rose namespace" && exit 1)
grep -q "namespace: rose" "${DEPLOY_DIR}/rose-ingress.yaml" || (echo "Missing rose namespace" && exit 1)
grep -q "namespace: flux-system" "${DEPLOY_DIR}/rose-images.yaml" || (echo "Missing flux-system namespace" && exit 1)

echo "Verifying PVC storage and access mode specifications..."
grep -q "ReadWriteMany" "$PVC_FILE" || (echo "Missing ReadWriteMany in PVC" && exit 1)
grep -q "ceph-filesystem" "$PVC_FILE" || (echo "Missing ceph-filesystem in PVC" && exit 1)
grep -q "storage: 50Gi" "$PVC_FILE" || (echo "Missing storage: 50Gi in PVC" && exit 1)

echo "Verifying gallery deployment volume mount and security context..."
grep -q "fsGroup: 65532" "${DEPLOY_DIR}/rose-gallery-deploy.yaml" || (echo "Missing fsGroup: 65532 in deploy" && exit 1)
grep -q "mountPath: /data" "${DEPLOY_DIR}/rose-gallery-deploy.yaml" || (echo "Missing mountPath: /data in deploy" && exit 1)
grep -q "readOnly: true" "${DEPLOY_DIR}/rose-gallery-deploy.yaml" || (echo "Missing readOnly: true in deploy" && exit 1)
grep -q "claimName: rose-data-pvc" "${DEPLOY_DIR}/rose-gallery-deploy.yaml" || (echo "Missing claimName: rose-data-pvc in deploy" && exit 1)

echo "Verifying host and port bindings..."
grep -q "roseseraphinetucker.com" "${DEPLOY_DIR}/rose-ingress.yaml" || (echo "Missing target host binding" && exit 1)
grep -q "port: 80" "${DEPLOY_DIR}/rose-gallery-deploy.yaml" || (echo "Missing port 80 binding" && exit 1)
grep -q "port: 80" "${DEPLOY_DIR}/rose-ingress.yaml" || (echo "Missing port 80 in ingress" && exit 1)

echo "=== All infrastructure manifests verified successfully ==="

