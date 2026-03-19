#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLATFORM="${1:-}"
BRIDGE_ARCHIVE="${2:-}"
RUNTIME_SRC_DIR="${3:-}"
DEST_ROOT="${4:-$ROOT_DIR}"

if [[ -z "$PLATFORM" || -z "$BRIDGE_ARCHIVE" || -z "$RUNTIME_SRC_DIR" ]]; then
  echo "usage: $0 <platform> <bridge-archive-path> <runtime-dir> [dest-root]" >&2
  exit 1
fi

if [[ ! -f "$BRIDGE_ARCHIVE" ]]; then
  echo "missing bridge archive: $BRIDGE_ARCHIVE" >&2
  exit 1
fi

if [[ ! -d "$RUNTIME_SRC_DIR" ]]; then
  echo "missing runtime dir: $RUNTIME_SRC_DIR" >&2
  exit 1
fi

DEST_NATIVE_DIR="$DEST_ROOT/internal/native/$PLATFORM"
DEST_RUNTIME_DIR="$DEST_ROOT/internal/runtimeassets/runtime/$PLATFORM"

mkdir -p "$DEST_NATIVE_DIR"
rm -rf "$DEST_RUNTIME_DIR"
mkdir -p "$DEST_RUNTIME_DIR"

cp -f "$BRIDGE_ARCHIVE" "$DEST_NATIVE_DIR/libgovm_boxlite_bridge.a"
cp -a "$RUNTIME_SRC_DIR/." "$DEST_RUNTIME_DIR/"

echo "staged native bridge:"
ls -lh "$DEST_NATIVE_DIR/libgovm_boxlite_bridge.a"
echo "staged runtime assets:"
find "$DEST_RUNTIME_DIR" -maxdepth 1 -type f | sort
