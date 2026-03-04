#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STRICT="${STRICT:-0}"

platforms=(linux_amd64 linux_arm64 darwin_arm64)
missing_native=()
missing_runtime=()

for p in "${platforms[@]}"; do
  native="$ROOT_DIR/internal/native/$p/libgovm_boxlite_bridge.a"
  runtime_dir="$ROOT_DIR/internal/runtimeassets/runtime/$p"

  if [[ ! -f "$native" ]]; then
    missing_native+=("$p")
  fi

  if [[ ! -d "$runtime_dir" ]]; then
    missing_runtime+=("$p")
    continue
  fi

  for req in boxlite-shim boxlite-guest; do
    if [[ ! -f "$runtime_dir/$req" ]]; then
      missing_runtime+=("$p:$req")
    fi
  done
done

echo "expected platforms: ${platforms[*]}"
if [[ ${#missing_native[@]} -eq 0 ]]; then
  echo "native assets: ok"
else
  echo "native assets missing: ${missing_native[*]}"
fi

if [[ ${#missing_runtime[@]} -eq 0 ]]; then
  echo "runtime assets: ok"
else
  echo "runtime assets missing: ${missing_runtime[*]}"
fi

if [[ "$STRICT" == "1" ]] && ([[ ${#missing_native[@]} -gt 0 ]] || [[ ${#missing_runtime[@]} -gt 0 ]]); then
  exit 2
fi
