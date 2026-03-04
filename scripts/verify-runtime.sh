#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLATFORM="${1:-$(go env GOOS)_$(go env GOARCH)}"
DIR="$ROOT_DIR/internal/runtimeassets/runtime/$PLATFORM"

if [[ ! -d "$DIR" ]]; then
  echo "missing runtime dir: $DIR"
  exit 1
fi

status=0
for f in boxlite-shim boxlite-guest; do
  if [[ ! -f "$DIR/$f" ]]; then
    echo "missing: $DIR/$f"
    status=1
  elif [[ ! -x "$DIR/$f" ]]; then
    echo "not executable: $DIR/$f"
    status=1
  fi
done

for f in mke2fs debugfs; do
  if [[ ! -f "$DIR/$f" ]]; then
    echo "warning: optional runtime tool missing (will fallback to host if available): $DIR/$f"
  elif [[ ! -x "$DIR/$f" ]]; then
    echo "warning: optional runtime tool is not executable: $DIR/$f"
  fi
done

case "$PLATFORM" in
  linux_amd64|linux_arm64)
    if [[ ! -f "$DIR/libkrunfw.so.5" ]]; then
      echo "missing required firmware library: $DIR/libkrunfw.so.5"
      status=1
    fi
    ;;
  darwin_arm64)
    if [[ ! -f "$DIR/libkrunfw.5.dylib" ]]; then
      echo "missing required firmware library: $DIR/libkrunfw.5.dylib"
      status=1
    fi
    ;;
esac

if [[ $status -ne 0 ]]; then
  exit 1
fi

echo "runtime assets verified for $PLATFORM: $DIR"
