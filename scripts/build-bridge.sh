#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BRIDGE_DIR="$ROOT_DIR/rust-bridge"
STUB_MODE="${BOXLITE_DEPS_STUB:-1}"
STUB_LIB_DIR="$BRIDGE_DIR/.stub-native-libs"

PROFILE="${1:-release}"
if [[ "$PROFILE" != "release" && "$PROFILE" != "debug" ]]; then
  echo "usage: $0 [release|debug]" >&2
  exit 1
fi

pushd "$BRIDGE_DIR" >/dev/null
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
fi

if [[ "$PROFILE" == "release" ]]; then
  export RUSTFLAGS="${RUSTFLAGS:-} -C strip=symbols"
  BOXLITE_DEPS_STUB="$STUB_MODE" cargo build --release
  echo "built: $BRIDGE_DIR/target/release/libgovm_boxlite_bridge.a"
else
  BOXLITE_DEPS_STUB="$STUB_MODE" cargo build
  echo "built: $BRIDGE_DIR/target/debug/libgovm_boxlite_bridge.a"
fi
popd >/dev/null
