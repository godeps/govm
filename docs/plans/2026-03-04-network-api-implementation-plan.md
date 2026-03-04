# Network API Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a complete `govm` network API surface with secure defaults and bridge mapping for currently supported upstream fields.

**Architecture:** Implement network config types/validation/defaults in `pkg/client`, pass effective config through `internal/binding` JSON into `rust-bridge`, and map supported parts (`network`, `ports`, `advanced.security.network_enabled`) into `boxlite::runtime::options::BoxOptions`.

**Tech Stack:** Go (client+binding), Rust (FFI bridge), boxlite runtime options JSON mapping, Go test.

---

### Task 1: Add failing tests for network API behavior

**Files:**
- Create: `pkg/client/network_test.go`
- Modify: `pkg/client/client_test.go`

**Step 1: Write failing test**
- Add tests for:
  - strict profile defaults
  - invalid port ranges/protocol
  - runtime defaults + per-box override merge
  - unsupported configuration rejection policy

**Step 2: Run test to verify it fails**
Run: `go test ./pkg/client -run Network -count=1`
Expected: FAIL due to missing types/functions.

**Step 3: Write minimal implementation**
- Add just enough types/funcs to satisfy compiler and basic behavior.

**Step 4: Run test to verify it passes**
Run: `go test ./pkg/client -run Network -count=1`
Expected: PASS.

### Task 2: Implement public API + validation/defaulting

**Files:**
- Create: `pkg/client/network.go`
- Modify: `pkg/client/types.go`
- Modify: `pkg/client/client.go`

**Step 1: Write failing test**
- Add edge-case tests for nil/empty configs and profile fallback.

**Step 2: Run test to verify it fails**
Run: `go test ./pkg/client -run Network -count=1`
Expected: FAIL.

**Step 3: Write minimal implementation**
- Add network types, validation error values, profile builders.
- Add `RuntimeOptions.NetworkDefaults` and `BoxOptions.Network`.
- Merge effective config in `CreateBox`.

**Step 4: Run test to verify it passes**
Run: `go test ./pkg/client -run Network -count=1`
Expected: PASS.

### Task 3: Extend binding and bridge mapping

**Files:**
- Modify: `internal/binding/binding_native.go`
- Modify: `internal/binding/binding_stub.go`
- Modify: `rust-bridge/src/lib.rs`

**Step 1: Write failing test**
- Add Go-side translation test (via mock runtime provider capture).

**Step 2: Run test to verify it fails**
Run: `go test ./pkg/client -run CreateBox -count=1`
Expected: FAIL due to missing fields in binding options.

**Step 3: Write minimal implementation**
- Add binding JSON fields: network mode, port forwards, macOS sandbox network toggle.
- Update rust bridge deserialization and mapping into `RuntimeBoxOptions`.

**Step 4: Run test to verify it passes**
Run: `go test ./pkg/client -run CreateBox -count=1`
Expected: PASS.

### Task 4: Docs and examples

**Files:**
- Modify: `README.md`
- Modify: `examples/all-api/main.go`
- Optionally create: `examples/network/main.go`

**Step 1: Write failing test**
- (Doc/examples, no code-level failing test required.)

**Step 2: Minimal implementation**
- Add network config usage sample.

**Step 3: Verify**
Run: `go run ./examples/all-api` (optional native env)
Expected: compiles and demonstrates API path.

### Task 5: Full verification

**Files:**
- N/A

**Step 1: Run tests**
Run: `go test ./...`
Expected: PASS.

**Step 2: Native checks**
Run: `make test-native`
Expected: PASS (environment permitting).

**Step 3: Commit**
```bash
git add docs/plans pkg/client internal/binding rust-bridge README.md examples
git commit -m "feat: add network api with validation and bridge mapping"
```
