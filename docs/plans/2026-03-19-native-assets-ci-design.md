# Govm Native Assets CI Design

**Goal**

Add a manual GitHub Actions workflow that builds `govm` native bridge and runtime assets for `linux_amd64`, `linux_arm64`, and `darwin_arm64` from `boxlite` source, stages them into this repository, validates them, and commits them back to the triggering branch.

**Constraints**

- Trigger mode is `workflow_dispatch` only.
- The workflow must build from `boxlite` source, not download release archives.
- The resulting assets must continue to live under `internal/native/<platform>` and `internal/runtimeassets/runtime/<platform>`.
- The workflow must avoid self-trigger loops when it pushes back to the branch.

**Approaches**

1. Build everything inline in one job.
Tradeoff: simple to read, but slow and awkward for cross-platform collection.

2. Matrix build per platform, upload artifacts, aggregate in a final commit job.
Tradeoff: slightly more YAML, but clean separation and the best fit for multi-platform assets.

3. Split into separate reusable workflows.
Tradeoff: reusable, but unnecessary complexity for the current repo size.

**Recommendation**

Use approach 2.

**Architecture**

- A platform matrix job runs on native GitHub runners for:
  - `ubuntu-latest` -> `linux_amd64`
  - `ubuntu-24.04-arm` -> `linux_arm64`
  - `macos-15` -> `darwin_arm64`
- Each platform job:
  - checks out `govm`
  - checks out `boxlite` with submodules
  - installs Go and Rust
  - runs `boxlite` build setup and runtime build
  - builds `govm` bridge against the local `boxlite` checkout
  - stages files into a temporary `internal/native/...` and `internal/runtimeassets/runtime/...` layout
  - uploads the staged tree as a workflow artifact
- A final aggregation job:
  - downloads all staged artifacts
  - writes them into the repo checkout
  - runs strict asset validation
  - commits and pushes only if there is a diff

**Key Repo Changes**

- Make `scripts/build-bridge.sh` support a local `boxlite` checkout override instead of hardcoding `internal/native/linux_amd64`.
- Add a small staging script so CI can consistently assemble platform assets.
- Replace the placeholder native workflow with the full matrix build-and-commit flow.
- Document the manual workflow in the README.

**Error Handling**

- Fail fast if the local `boxlite` checkout path does not exist.
- Fail fast if expected outputs (`libgovm_boxlite_bridge.a`, `boxlite-shim`, `boxlite-guest`) are missing.
- Use strict validation before commit so incomplete platform sets are never pushed.

**Verification**

- Run `STRICT=1 make platform-check` in the aggregation job.
- Run a local smoke check for script behavior in stub/native-path scenarios where practical.
