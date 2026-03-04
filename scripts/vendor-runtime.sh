#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC_RUNTIME_DIR="${1:-}"
PLATFORM="${2:-$(go env GOOS)_$(go env GOARCH)}"
DST_DIR="$ROOT_DIR/internal/runtimeassets/runtime/$PLATFORM"

if [[ -z "$SRC_RUNTIME_DIR" ]]; then
  echo "usage: $0 <boxlite-runtime-dir> [platform]" >&2
  exit 1
fi

mkdir -p "$DST_DIR"
rm -f "$DST_DIR"/boxlite-shim \
      "$DST_DIR"/boxlite-guest \
      "$DST_DIR"/mke2fs \
      "$DST_DIR"/debugfs \
      "$DST_DIR"/bwrap \
      "$DST_DIR"/libkrun* \
      "$DST_DIR"/libgvproxy*

for f in boxlite-shim boxlite-guest mke2fs debugfs; do
  if [[ -f "$SRC_RUNTIME_DIR/$f" ]]; then
    cp -f "$SRC_RUNTIME_DIR/$f" "$DST_DIR/$f"
  fi
done
if [[ -f "$SRC_RUNTIME_DIR/bwrap" ]]; then
  cp -f "$SRC_RUNTIME_DIR/bwrap" "$DST_DIR/bwrap"
fi
for lib in "$SRC_RUNTIME_DIR"/libkrunfw* "$SRC_RUNTIME_DIR"/libgvproxy* "$SRC_RUNTIME_DIR"/libkrun*; do
  if [[ -f "$lib" ]]; then
    cp -f "$lib" "$DST_DIR/"
  fi
done

echo "runtime assets synced to: $DST_DIR"
ls -lh "$DST_DIR"
