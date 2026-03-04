# Govm Network API Design

**Date:** 2026-03-04

## Goal
Provide a complete, backward-compatible network control API in `govm` for `govm + boxlite` that supports secure defaults, explicit allow rules, and platform-aware degradation without modifying upstream `boxlite` repository code.

## Constraints
- `other/boxlite` is read-only from this project perspective.
- Existing `CreateBox` flows must remain backward compatible.
- Existing offline image flow remains unchanged.
- Current upstream runtime supports:
  - network mode at box options level (`Isolated`)
  - port mappings (`ports`)
  - macOS sandbox network toggle (`advanced.security.network_enabled`)
- Fine-grained egress ACLs (CIDR/domain) are not first-class in upstream runtime options yet.

## API Direction
Use a hybrid control model in `govm` API:
- Stable public API expresses full policy intent (mode/policy/rules/proxy/dns/limits).
- Runtime v1 implementation maps supported subset to upstream fields now.
- Unsupported parts are validated and surfaced as explicit `unsupported` errors/warnings, not silently ignored.

## Profiles
- `strict` (default): deny-by-default intent, explicit allow required.
- `balanced`: common internet defaults (80/443/DNS) intent.
- `open`: allow-all intent.

## Platform Semantics
- Linux: support mode + port forwarding now; advanced ACL intent accepted but not enforced in-runtime yet.
- macOS: additionally map network on/off via sandbox `network_enabled`.
- Windows: API available but native backend support limited; return `ErrNetworkUnsupportedPlatform` where applicable.

## Non-goals (v1)
- In-guest transparent eBPF firewalling.
- Dynamic host firewall orchestration.
- Runtime policy hot-reload guarantees across all platforms.

## Deliverables
1. New public network API types in `pkg/client`.
2. Validation/defaulting/merge logic in `pkg/client`.
3. Binding and Rust bridge option extension for supported fields.
4. Example updates and docs.
5. Tests for validation and option translation.
