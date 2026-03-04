#!/usr/bin/env bash
set -euo pipefail

# Sync prebuilt native artifacts from a local boxlite checkout into govm/internal/native.
# Expected source tree: <boxlite>/sdks/go/internal/native/<platform>

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC_BOXLITE_DIR="${1:-}"

if [[ -z "$SRC_BOXLITE_DIR" ]]; then
  echo "usage: $0 <path-to-boxlite-root>" >&2
  exit 1
fi

DST_NATIVE="$ROOT_DIR/internal/native"

SRC_NATIVE="$SRC_BOXLITE_DIR/sdks/go/internal/native"

for platform in linux_amd64 linux_arm64 darwin_arm64; do
  mkdir -p "$DST_NATIVE/$platform"
done

if [[ -d "$SRC_NATIVE" ]]; then
  for platform in linux_amd64 linux_arm64 darwin_arm64; do
    if [[ -d "$SRC_NATIVE/$platform" ]]; then
      rsync -av --delete "$SRC_NATIVE/$platform/" "$DST_NATIVE/$platform/"
    fi
  done
  echo "native artifacts synced from: $SRC_NATIVE"
  echo "native artifacts synced to: $DST_NATIVE"
  exit 0
fi

echo "source native dir not found: $SRC_NATIVE"
echo "fallback: syncing whatever native deps exist under boxlite/target for current host platform"

goos="$(go env GOOS)"
goarch="$(go env GOARCH)"
platform="${goos}_${goarch}"
dst="$DST_NATIVE/$platform"
mkdir -p "$dst"

# Best-effort sync for static deps built by boxlite; may be partial.
find "$SRC_BOXLITE_DIR/target" -type f \( -name "libgvproxy.a" -o -name "libkrun.a" -o -name "libkrun*.dylib" -o -name "libgvproxy*.dylib" -o -name "libkrun*.so*" -o -name "libgvproxy*.so*" \) 2>/dev/null | while read -r f; do
  cp -f "$f" "$dst/"
done

echo "fallback native deps copied to: $dst"
