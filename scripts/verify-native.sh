#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NATIVE_DIR="$ROOT_DIR/internal/native"

status=0
for platform in linux_amd64 linux_arm64 darwin_arm64; do
  if [[ ! -f "$NATIVE_DIR/$platform/libgovm_boxlite_bridge.a" ]]; then
    echo "missing: $NATIVE_DIR/$platform/libgovm_boxlite_bridge.a"
    status=1
  fi
done

if [[ $status -ne 0 ]]; then
  echo "native artifacts are incomplete"
  exit 1
fi

echo "native artifacts look complete"
