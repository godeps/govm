# Native Assets CI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a manual GitHub Actions workflow that builds cross-platform native assets from `boxlite` source and commits them back to the current branch.

**Architecture:** Use a matrix workflow to build each platform on a native runner, upload staged repo-shaped artifacts, and use a final aggregation job to validate and push any resulting asset updates. Update the bridge build script so CI can bind to a local `boxlite` checkout instead of only using vendored native libraries.

**Tech Stack:** GitHub Actions, Bash, Go, Rust, Cargo, git

---

### Task 1: Make the bridge build script CI-friendly

**Files:**
- Modify: `scripts/build-bridge.sh`

**Step 1: Define the expected behavior**

The script must:
- keep stub mode working as today
- allow `BOXLITE_REPO_DIR=/path/to/boxlite` to patch `boxlite` and `boxlite-ffi` to local paths
- stop hardcoding `internal/native/linux_amd64` for non-stub builds

**Step 2: Run a local baseline command**

Run: `./scripts/build-bridge.sh release`
Expected: release bridge builds successfully in stub mode.

**Step 3: Implement the minimal script changes**

Add:
- local path validation
- temporary Cargo patch config generation
- conditional `cargo build --config <tempfile>` invocation

**Step 4: Re-run the baseline command**

Run: `./scripts/build-bridge.sh release`
Expected: release bridge still builds successfully in stub mode.

### Task 2: Add a CI staging helper

**Files:**
- Create: `scripts/stage-platform-assets.sh`

**Step 1: Define the contract**

Inputs:
- `<platform>`
- `<bridge-archive-path>`
- `<runtime-dir>`
- optional destination root

Outputs:
- `internal/native/<platform>/libgovm_boxlite_bridge.a`
- `internal/runtimeassets/runtime/<platform>/...`

**Step 2: Verify failure mode first**

Run: `./scripts/stage-platform-assets.sh linux_amd64 /no/such/file /no/such/runtime`
Expected: command fails with a missing-path error.

**Step 3: Implement the script**

Add path validation, copy logic, executable bit preservation, and a short staged-file summary.

**Step 4: Verify success path**

Run: `tmpdir=$(mktemp -d) && ./scripts/stage-platform-assets.sh linux_amd64 rust-bridge/target/release/libgovm_boxlite_bridge.a internal/runtimeassets/runtime/linux_amd64 "$tmpdir"`
Expected: staged files appear under `$tmpdir/internal/...`.

### Task 3: Replace the placeholder native workflow

**Files:**
- Modify: `.github/workflows/build-native.yml`

**Step 1: Remove the one-step placeholder flow**

The current workflow only builds the stub bridge on Ubuntu and is not sufficient.

**Step 2: Implement the matrix workflow**

Add:
- `workflow_dispatch`
- platform matrix
- `actions/checkout` for both repos
- Go/Rust setup
- `boxlite` source build steps
- staged artifact upload
- aggregation + strict verification + commit/push

**Step 3: Validate YAML locally**

Run: `sed -n '1,260p' .github/workflows/build-native.yml`
Expected: workflow structure is coherent and references existing scripts and paths.

### Task 4: Document the manual workflow

**Files:**
- Modify: `README.md`

**Step 1: Add the operator flow**

Document:
- manual trigger
- what the workflow builds
- where assets land
- that it commits back to the triggering branch only when files changed

**Step 2: Verify docs placement**

Run: `rg -n "workflow_dispatch|build-native|internal/native|internal/runtimeassets/runtime" README.md .github/workflows/build-native.yml`
Expected: the README and workflow use the same terminology.

### Task 5: Final verification

**Files:**
- Verify all changed files

**Step 1: Run local script smoke checks**

Run:
- `./scripts/build-bridge.sh release`
- `tmpdir=$(mktemp -d) && ./scripts/stage-platform-assets.sh linux_amd64 rust-bridge/target/release/libgovm_boxlite_bridge.a internal/runtimeassets/runtime/linux_amd64 "$tmpdir"`

Expected: both commands exit successfully.

**Step 2: Run repo validation**

Run:
- `make platform-check`
- `git diff --stat`

Expected:
- platform check reflects current repo contents
- diff matches the intended workflow/scripts/docs changes

**Step 3: Summarize remaining risk**

Call out that full source builds for all platforms are validated in GitHub Actions, not fully reproducible in the current local host environment.
