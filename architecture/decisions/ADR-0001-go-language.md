# ADR-0001 — Go as the implementation language

**Status:** accepted · **Context:** modernization of an Inventor-class MCAD app.

## Decision

Implement the application in **Go** (1.27+), targeting Linux, macOS, Windows.

## Why

- **Cross-platform by default.** One toolchain, `GOOS`/`GOARCH` cross-compilation,
  no per-platform project files. The original COM/C# stack is Windows-bound.
- **Concurrency is first-class.** Goroutines + channels + a worker pool map
  directly onto the parallelism CAD needs: parallel feature recompute of
  independent branches, parallel tessellation, parallel transform recompute
  (realtime-3d skill §3, §13).
- **Simple, fast builds and a strong static type system** that lets us delete the
  COM ceremony (variants, dual interfaces, RTTI) in favor of compile-time types
  and generics.
- **Good GPU/FFI story** for a Vulkan renderer (cgo or purego, see ADR-0008).

## Costs / mitigations

- **GC pauses** threaten a 60fps loop → enforce near-zero steady-state allocation,
  pool transient per-frame data, and run heavy recompute *off* the frame loop
  (ADR-0007). (realtime-3d §13)
- **Generics are newer/less expressive** than C++ templates → the kernel leans on
  `float64` concrete types and a small set of generic containers, not deep
  template metaprogramming. [ADR-0054](ADR-0054-generic-methods-policy.md) extends
  this appetite to Go 1.27's generic methods: adopted only where a non-generic
  receiver's own free-function helper exists purely for lack of the feature, or
  where it collapses genuinely duplicated logic — not merely because it's now possible.
- **No exceptions** → errors as typed values (realtime-3d §15); modeling failures
  are *health state*, not panics (parametric-cad skill).

## Consequences

Everything else in this folder assumes idiomatic Go: the mediator instead of a
singleton, interfaces only where justified, build tags for platform/feature
gating, structured logging, `NotYetImplemented(issueID)` stubs instead of `TODO`.
