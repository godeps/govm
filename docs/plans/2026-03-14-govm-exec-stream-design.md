# govm ExecStream Design

## Goal

Add a native streaming exec API to `govm` so callers can receive stdout/stderr incrementally during guest command execution, while preserving the existing synchronous `Exec()` API.

## Current State

`pkg/client.Box.Exec()` returns only a final `ExecResult`. However, the native binding layer already receives incremental stdout/stderr through the callback bridge in:

- `internal/binding/exec_bridge.c`
- `internal/binding/exec_native.go`

Today those callbacks are only used to accumulate final output in memory before returning from `Exec()`.

## Decision

Introduce a parallel streaming API instead of changing `Exec()`:

- add `ExecStreamCallbacks` at the `pkg/client` layer
- add `Box.ExecStream(...)`
- extend the internal binding layer so callbacks can be forwarded upward while still collecting final output

## Proposed API

```go
type ExecStreamCallbacks struct {
    OnStdout func(string)
    OnStderr func(string)
}

func (b *Box) ExecStream(command string, opts *ExecOptions, cb ExecStreamCallbacks) (*ExecResult, error)
```

Semantics:

- `OnStdout` and `OnStderr` are invoked incrementally during execution
- the method still returns a final `ExecResult`
- `Exec()` remains unchanged and continues to use the non-streaming path

## Binding Layer Design

The native binding layer will be extended to support an optional Go-side forwarder in addition to the existing collector:

- keep collecting output into `[]string` for the final `ExecResult`
- if external callbacks are provided, invoke them from the same output callback path

This avoids introducing a second native exec primitive and reuses the existing callback bridge.

## Non-Native Behavior

The stub build should expose the same API shape but return `ErrNativeUnavailable`, consistent with current native-only exec behavior.

## Risks

- callback blocking can delay command execution if callers do heavy work inside callbacks
- stdout/stderr relative ordering is only guaranteed per stream, not necessarily as a single merged total order
- callback shape is line-oriented today because the existing bridge already forwards text chunks in that form

## Validation

Add tests for:

- callback forwarding to stdout and stderr
- final `ExecResult` still containing complete output
- no-callback usage preserving compatibility
- stub/native API parity at compile time
