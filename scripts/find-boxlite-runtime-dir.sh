#!/usr/bin/env bash
set -euo pipefail

TARGET_ROOT="${1:-}"

if [[ -z "$TARGET_ROOT" ]]; then
  echo "usage: $0 <boxlite-target-dir>" >&2
  exit 1
fi

if [[ ! -d "$TARGET_ROOT" ]]; then
  echo "missing target dir: $TARGET_ROOT" >&2
  exit 1
fi

mapfile -t candidates < <(find "$TARGET_ROOT" -path '*/out/runtime' -type d | sort)
WORKSPACE_ROOT="$(dirname "$TARGET_ROOT")"

for runtime_dir in "${candidates[@]}"; do
  if [[ -f "$runtime_dir/boxlite-shim" && -f "$runtime_dir/boxlite-guest" ]]; then
    echo "$runtime_dir"
    exit 0
  fi
done

if [[ -f "$WORKSPACE_ROOT/boxlite-shim" && -f "$WORKSPACE_ROOT/boxlite-guest" ]]; then
  echo "$WORKSPACE_ROOT"
  exit 0
fi

if [[ ${#candidates[@]} -eq 0 ]]; then
  echo "runtime dir not found under $TARGET_ROOT or its workspace root" >&2
  exit 1
fi

echo "no runtime dir with boxlite-shim and boxlite-guest found under $TARGET_ROOT" >&2
exit 1
