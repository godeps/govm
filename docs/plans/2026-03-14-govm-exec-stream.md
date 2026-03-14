# govm ExecStream Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a native streaming command execution API to `govm` that forwards stdout/stderr incrementally while preserving the existing synchronous `Exec()` behavior.

**Architecture:** Reuse the existing native exec callback bridge instead of inventing a second transport. Add a new `ExecStream()` path in `pkg/client`, plumb optional forward callbacks through `internal/binding`, and keep collecting complete output for the final `ExecResult`. The old `Exec()` remains source-compatible and behavior-compatible.

**Tech Stack:** Go, cgo native bindings, existing `pkg/client` and `internal/binding` test suites.

---

### Task 1: Add failing client-layer tests for streaming callbacks

**Files:**
- Modify: `pkg/client/box_test.go`
- Modify: `pkg/client/binding_mock_test.go`

**Step 1: Write the failing test**

Add tests that verify:
- `Box.ExecStream()` forwards stdout callbacks
- `Box.ExecStream()` forwards stderr callbacks
- final `ExecResult` still contains complete stdout/stderr

Use the existing mock provider path rather than native integration for the first red test.

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/client -run TestBoxExecStream -count=1`
Expected: FAIL because `ExecStream()` and related callback plumbing do not exist yet.

**Step 3: Write minimal implementation**

Add new callback types and mock plumbing just enough to make the client tests compile and pass.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/client -run TestBoxExecStream -count=1`
Expected: PASS

### Task 2: Extend binding interfaces with streaming callbacks

**Files:**
- Modify: `pkg/client/client.go`
- Modify: `pkg/client/box.go`
- Modify: `pkg/client/types.go`
- Modify: `internal/binding/binding_native.go`
- Modify: `internal/binding/binding_stub.go`
- Modify: `pkg/client/binding_mock_test.go`

**Step 1: Write the failing compile/test change**

Update the provider interface to include a streaming-capable exec entrypoint, then run tests to confirm the build breaks until all implementations are updated.

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/client -count=1`
Expected: FAIL because the binding/provider implementations are incomplete.

**Step 3: Write minimal implementation**

Introduce:
- `ExecStreamCallbacks`
- `ExecWithCallbacks` (binding-side naming if needed) or equivalent
- `Box.ExecStream(...)`

Keep `Box.Exec(...)` unchanged externally.

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/client -count=1`
Expected: PASS

### Task 3: Implement native callback forwarding

**Files:**
- Modify: `internal/binding/exec_native.go`
- Modify: `internal/binding/exec_bridge.c`

**Step 1: Write the failing test**

Add a focused test around the callback collector/forwarder behavior if practical; if not practical at package level, use `pkg/client` tests plus native integration in Task 4 as the proving layer.

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/client -run TestBoxExecStream -count=1`
Expected: FAIL if callbacks are still not forwarded.

**Step 3: Write minimal implementation**

Update the native exec callback path so it:
- always appends to final stdout/stderr collectors
- conditionally calls external Go callbacks when supplied

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/client -run TestBoxExecStream -count=1`
Expected: PASS

### Task 4: Add native integration coverage

**Files:**
- Modify: `pkg/client/native_integration_test.go`

**Step 1: Write the failing integration test**

Add a native-only test that:
- starts a box
- runs `ExecStream("/bin/sh", ...)`
- emits both stdout and stderr
- asserts callbacks observed both streams
- asserts final `ExecResult` still contains both streams and correct exit code

**Step 2: Run test to verify it fails**

Run: `GOVM_E2E=1 go test -tags govm_native ./pkg/client -run TestNativeExecStreamE2E -count=1 -v`
Expected: FAIL before native forwarding is complete.

**Step 3: Write minimal implementation**

Make only the native callback plumbing changes required for the integration test to pass.

**Step 4: Run test to verify it passes**

Run: `GOVM_E2E=1 go test -tags govm_native ./pkg/client -run TestNativeExecStreamE2E -count=1 -v`
Expected: PASS

### Task 5: Verify compatibility and document the new API

**Files:**
- Modify: `README.md`
- Modify: `examples/basic/main.go` or another example only if a minimal streaming example adds clear value

**Step 1: Run regression tests**

Run: `go test ./pkg/client -count=1`
Expected: PASS

**Step 2: Run native regression tests**

Run: `GOVM_E2E=1 go test -tags govm_native ./pkg/client -run 'TestNativeLifecycleE2E|TestNativeMountE2E|TestNativeExecStreamE2E' -count=1 -v`
Expected: PASS

**Step 3: Update docs**

Add a short example of `ExecStream()` usage to `README.md`, keeping the change minimal and scoped to the new API.

**Step 4: Re-run verification**

Run:
- `go test ./pkg/client -count=1`
- `GOVM_E2E=1 go test -tags govm_native ./pkg/client -run TestNativeExecStreamE2E -count=1 -v`

Expected: PASS
