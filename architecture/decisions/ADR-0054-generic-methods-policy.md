<!-- SPDX-License-Identifier: GPL-2.0-only -->

# ADR-0054 — Generic methods: when they earn their keep

**Status:** Accepted (2026-08-27). Extends [ADR-0001](ADR-0001-go-language.md)'s "small set of
generic containers, not deep template metaprogramming" policy to cover Go 1.27's generic
methods, without changing that policy's underlying appetite.

## Context

Go 1.27 lets a method declare its own type parameters (`func (r *Rand) N[Int intType](Int)
Int` in the stdlib's own `math/rand/v2` is the reference example). Before 1.27, only
package-scope functions could be generic — a non-generic receiver type whose operation
needed its own type parameter had no choice but a free function taking the receiver as an
explicit first argument (`Emit(bus, phase, event)` instead of `bus.Emit(phase, event)`).

This ADR records the outcome of evaluating every plausible candidate in the codebase for
this specific new capability, as a deliberate extension of ADR-0001's low-generics appetite:
the bar is real duplication removed or a genuine receiver-owns-this-operation fit, not
"could technically become a method."

## Decision

Adopt generic methods where **either**:

1. A non-generic, stateful receiver type has a free-function helper that takes the receiver
   as its first parameter purely because a generic method wasn't available — converting it
   to a method makes the operation read as what it already conceptually is (an operation
   the receiver performs), with no other design change.
2. Genuinely duplicated logic across two or more types can be collapsed into one generic
   method, and the collapse doesn't force an artificial abstraction onto types whose
   underlying operations differ in more than the substituted type.

Do **not** convert a free generic function to a method merely because Go now allows it. If
the function doesn't use the receiver's state, or the "receiver" is arbitrary/interchangeable
at the call site, it stays a free function — a method implies "this operates on my state,"
and a method that doesn't earn that implication misleads readers.

## Candidates evaluated

- **`event.Bus.Subscribe`/`Emit`/`EmitContext`** (`event/bus.go`) — **ADOPTED.** These were
  free functions (`Subscribe[E Event](b *Bus, p Phase, h Handler[E])`, etc.) taking `*Bus` as
  their first argument purely because `Bus` (non-generic, stateful — holds the subscription
  map and a monotonic sequence counter) couldn't carry a generic method before 1.27. Converted
  to `(b *Bus) Subscribe[E Event](...)` etc. The ~176 existing call sites
  (`event.Emit(bus, ...)`) across ~90 files are untouched: the free functions are now thin
  wrappers delegating to the methods (`func Emit[E Event](b *Bus, p Phase, e E) Outcome {
  return b.Emit(p, e) }`), matching `math/rand/v2`'s own precedent of keeping the top-level
  `N[Int](r, n)` alongside the new `r.N[Int](n)` method rather than forcing a migration. New
  code should prefer `bus.Emit(...)`/`bus.Subscribe(...)` directly.
- **`linetype.dashCursor.walkEdge`** (`model/linetype/dash.go`) — **ADOPTED.** `walkEdge[P
  dashable[P]](c *dashCursor, segs [][2]P, p, q P) [][2]P` mutates `c`'s own pattern-position
  state (via `c.advance`) on every call — it is exactly "an operation the cursor performs,"
  parameterized per call by point dimensionality (2D/3D). Converted to `(c *dashCursor)
  walkEdge[P dashable[P]](segs [][2]P, p, q P) [][2]P`; the single call site in
  `dashPolyline` reads as `cur.walkEdge(segs, pts[i-1], pts[i])` instead of `walkEdge(&cur,
  segs, pts[i-1], pts[i])`.
- **`renderer.LightDistribution`/`EnvironmentDistribution`** (`renderer/light_sampling.go`,
  `renderer/environment_sampling.go`) — **REJECTED.** Both do CDF-based importance sampling,
  but that similarity is superficial on closer reading: `LightDistribution.Sample` is a 1D
  discrete pick over a fixed item list returning `(index, item, pdf)`; `EnvironmentDistribution
  .Sample` is a 2D continuous pick over a pixel grid (row marginal CDF + per-row conditional
  CDF) returning `(direction, pdf)` via a coordinate transform with its own solid-angle
  Jacobian. The only literally-shared logic is a one-line binary-search-a-CDF idiom already
  expressed via `sort.Search`/`sort.SearchFloat64s` — too thin to justify a shared generic
  type, and forcing the two into one abstraction would actively hurt readability for a
  one-line saving (CLAUDE.md's no-code-duplication rule doesn't apply to logic that isn't
  actually duplicated).
- **`addin/router/typed.go`'s ten handler-adapter functions** (`typed`, `typedCtx`,
  `ctxQuery`, `typedPart`, `partQuery`, `typedAssembly`, `assemblyQuery`, `typedHolder`,
  `holderQuery`, `projectAll`) — **REJECTED.** These are combinators that build a
  `handlerFunc` value (e.g. `typedPart(getX)`), evaluated *before* any `Router` is in scope —
  the call site is `r.readOnly(wire.MethodX, typedPart(getX))`. None of them read or use a
  `Router`'s state; there is no receiver to hang a method off without inventing one. Router
  methods here would misleadingly imply these adapters depend on router identity.
- **`kernel/topo.buildKeyIndex`/`scanByKey`** (`kernel/topo/key_index.go`) — **REJECTED.**
  Both are pure functions of their `items []T` argument, called from three sites inside
  `*Body`'s own methods (`b.cachedEdges`/`cachedFaces`/`cachedVertices`), each with a
  different concrete `T`. `Body` itself is non-generic, so nothing is lost by leaving these
  as free functions — a method would need the caller to pass `b`'s own field back into `b`'s
  method (`b.buildKeyIndex(b.cachedEdges)`), which is not more readable than today's
  `buildKeyIndex(b.cachedEdges)` and ties a stateless pure function to `*Body` for no reason.

## Consequences

- New concurrency/dispatch types with the `Bus` shape (a non-generic hub whose per-call
  operation needs its own type parameter) should default to generic methods, not the
  free-function-plus-explicit-receiver-argument workaround this ADR retired for `event.Bus`.
- A generic free function stays the right shape whenever the "receiver" is either stateless,
  interchangeable, or not actually part of the call — most of this codebase's existing
  generics (per ADR-0001's small deliberate set) fall in exactly that bucket and are
  unaffected by this ADR.
- This is not a mandate to audit every existing free generic function for method-conversion
  potential — the bar above is the filter for new code and for reviewing this ADR's five
  evaluated candidates going forward, not a backlog item.
