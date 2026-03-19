#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BRIDGE_DIR="$ROOT_DIR/rust-bridge"
TARGET_PLATFORM="${TARGET_PLATFORM:-$(go env GOOS)_$(go env GOARCH)}"
BOXLITE_REPO_DIR="${BOXLITE_REPO_DIR:-}"
BOXLITE_NATIVE_LIB_DIR="${BOXLITE_NATIVE_LIB_DIR:-}"
STUB_MODE="${BOXLITE_DEPS_STUB:-1}"
WORK_BRIDGE_DIR="$BRIDGE_DIR"
patch_config_file=""
bridge_copy_dir=""

PROFILE="${1:-release}"
if [[ "$PROFILE" != "release" && "$PROFILE" != "debug" ]]; then
  echo "usage: $0 [release|debug]" >&2
  exit 1
fi

cleanup_temp_files() {
  if [[ -n "${patch_config_file:-}" && -f "${patch_config_file:-}" ]]; then
    rm -f "$patch_config_file"
  fi
  if [[ -n "${bridge_copy_dir:-}" && -d "${bridge_copy_dir:-}" ]]; then
    rm -rf "$bridge_copy_dir"
  fi
}

if [[ -n "$BOXLITE_REPO_DIR" ]]; then
  bridge_copy_dir="$(mktemp -d)"
  trap cleanup_temp_files EXIT
  mkdir -p "$bridge_copy_dir/src"
  cp -f "$BRIDGE_DIR/Cargo.toml" "$BRIDGE_DIR/Cargo.lock" "$bridge_copy_dir/"
  cp -a "$BRIDGE_DIR/src/." "$bridge_copy_dir/src/"
  WORK_BRIDGE_DIR="$bridge_copy_dir"
fi

STUB_LIB_DIR="$WORK_BRIDGE_DIR/.stub-native-libs"

pushd "$WORK_BRIDGE_DIR" >/dev/null
if [[ "$STUB_MODE" == "1" ]]; then
  mkdir -p "$STUB_LIB_DIR"
  cat > "$STUB_LIB_DIR/empty.c" <<'C'
int govm_stub_symbol(void) { return 0; }
C
  cc -c "$STUB_LIB_DIR/empty.c" -o "$STUB_LIB_DIR/empty.o"
  ar rcs "$STUB_LIB_DIR/libgvproxy.a" "$STUB_LIB_DIR/empty.o"
  ar rcs "$STUB_LIB_DIR/libkrun.a" "$STUB_LIB_DIR/empty.o"
fi

base_rustflags="${RUSTFLAGS:-}"
if [[ "$STUB_MODE" == "1" ]]; then
  export RUSTFLAGS="${base_rustflags} -Lnative=$STUB_LIB_DIR"
else
  native_link_dir="$BOXLITE_NATIVE_LIB_DIR"
  if [[ -z "$native_link_dir" ]]; then
    native_link_dir="$ROOT_DIR/internal/native/$TARGET_PLATFORM"
  fi
  export RUSTFLAGS="${base_rustflags}"
  if [[ -n "$native_link_dir" ]]; then
    export RUSTFLAGS="${RUSTFLAGS} -Lnative=$native_link_dir"
  fi
fi

strip_archive() {
  if ! command -v strip >/dev/null; then
    return
  fi
  if [[ -f "$1" ]]; then
    strip --strip-unneeded "$1" 2>/dev/null || true
  fi
}

cargo_cmd=(cargo)

if [[ -n "$BOXLITE_REPO_DIR" ]]; then
  if [[ ! -f "$BOXLITE_REPO_DIR/boxlite/Cargo.toml" ]]; then
    echo "missing boxlite crate: $BOXLITE_REPO_DIR/boxlite/Cargo.toml" >&2
    exit 1
  fi
  if [[ ! -f "$BOXLITE_REPO_DIR/ffi/Cargo.toml" ]]; then
    echo "missing boxlite-ffi crate: $BOXLITE_REPO_DIR/ffi/Cargo.toml" >&2
    exit 1
  fi

  patch_config_file="$(mktemp)"
  cat > "$patch_config_file" <<EOF
[patch."https://github.com/boxlite-ai/boxlite"]
boxlite = { path = "$BOXLITE_REPO_DIR/boxlite" }
boxlite-ffi = { path = "$BOXLITE_REPO_DIR/ffi" }
EOF
  cargo_cmd+=(--config "$patch_config_file")
fi

if [[ "$PROFILE" == "release" ]]; then
  export RUSTFLAGS="${RUSTFLAGS:-} -C strip=symbols"
  cargo_cmd+=(build --release)
  BOXLITE_DEPS_STUB="$STUB_MODE" "${cargo_cmd[@]}"
  build_output="$WORK_BRIDGE_DIR/target/release/libgovm_boxlite_bridge.a"
  dest_output="$BRIDGE_DIR/target/release/libgovm_boxlite_bridge.a"
else
  cargo_cmd+=(build)
  BOXLITE_DEPS_STUB="$STUB_MODE" "${cargo_cmd[@]}"
  build_output="$WORK_BRIDGE_DIR/target/debug/libgovm_boxlite_bridge.a"
  dest_output="$BRIDGE_DIR/target/debug/libgovm_boxlite_bridge.a"
fi

mkdir -p "$(dirname "$dest_output")"
if [[ "$build_output" != "$dest_output" ]]; then
  cp -f "$build_output" "$dest_output"
fi
echo "built: $dest_output"
strip_archive "$dest_output"
popd >/dev/null
