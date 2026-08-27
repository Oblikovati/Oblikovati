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

This ADR records the outcome of a systematic sweep for this specific new capability across
both `Oblikovati` and `Oblikovati.API` — every top-level generic function whose first
parameter is a pointer to a concrete (non-generic) type, the exact shape this feature
retires — evaluated as a deliberate extension of ADR-0001's low-generics appetite: the bar
is real duplication removed or a genuine receiver-owns-this-operation fit, not "could
technically become a method." A function whose flagged first parameter turns out to live in
a *different* package from the receiver type is automatically disqualified regardless of
shape — Go methods can only be declared in the receiver type's own package, which generic
methods don't change.

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

## Candidates adopted

- **`event.Bus.Subscribe`/`Emit`/`EmitContext`** (`event/bus.go`) — These were
  free functions (`Subscribe[E Event](b *Bus, p Phase, h Handler[E])`, etc.) taking `*Bus` as
  their first argument purely because `Bus` (non-generic, stateful — holds the subscription
  map and a monotonic sequence counter) couldn't carry a generic method before 1.27. Converted
  to `(b *Bus) Subscribe[E Event](...)` etc. The ~176 existing call sites
  (`event.Emit(bus, ...)`) across ~90 files are untouched: the free functions are now thin
  wrappers delegating to the methods (`func Emit[E Event](b *Bus, p Phase, e E) Outcome {
  return b.Emit(p, e) }`), matching `math/rand/v2`'s own precedent of keeping the top-level
  `N[Int](r, n)` alongside the new `r.N[Int](n)` method rather than forcing a migration. New
  code should prefer `bus.Emit(...)`/`bus.Subscribe(...)` directly.
- **`linetype.dashCursor.walkEdge`** (`model/linetype/dash.go`) — `walkEdge[P
  dashable[P]](c *dashCursor, segs [][2]P, p, q P) [][2]P` mutates `c`'s own pattern-position
  state (via `c.advance`) on every call — it is exactly "an operation the cursor performs,"
  parameterized per call by point dimensionality (2D/3D). Converted to `(c *dashCursor)
  walkEdge[P dashable[P]](segs [][2]P, p, q P) [][2]P`; the single call site in
  `dashPolyline` reads as `cur.walkEdge(segs, pts[i-1], pts[i])` instead of `walkEdge(&cur,
  segs, pts[i-1], pts[i])`.
- **`sketch.cloneMap`'s six `carryXxx` helpers** (`model/sketch/copy_constraints.go`) —
  `cloneMap` already had six non-generic lookup methods (`point`, `line`, `curve`, `arc`,
  `entity`, `ellipse`, `smoothCurve`); `carryPoints[R any](m *cloneMap, ...)` and its five
  siblings sat right next to them as free functions purely because the factory callback's
  return type `R` couldn't be a method-level type parameter before 1.27. Converted all six to
  `(m *cloneMap) carryPoints[R any](...)` etc.; updated their 14 call sites in
  `copy_constraints_registry.go` (`carryPoints(m, a, b, add)` → `m.carryPoints(a, b, add)`) —
  private to this package, so no compatibility wrapper was needed.
- **`app.Session`'s `activeSheetMetalTool`, `runDialogHook`, `cursorMove`**
  (`app/sheet_metal_session.go`, `app/file_ui_hooks.go`, `app/undo.go`) — three separate free
  functions taking `*Session` (the central per-document app-state type) as their first
  argument. `cursorMove`'s own doc comment said it outright: *"Generic (and free, since
  methods cannot be) because the bus dispatches on the event's static type"* — exactly the
  limitation this ADR retires. All three are private to the `app` package (18, 3, and 2 call
  sites respectively, all internal), so — unlike `event.Bus` — converted directly with no
  compatibility wrapper needed: `s.activeSheetMetalTool[*SheetMetalFaceTool]()`,
  `s.runDialogHook(ev)`, `s.cursorMove(d, dh, ev, move)`. `runDialogHook` and `cursorMove`
  also switched their internal `event.Emit(s.bus, ...)` calls to `s.bus.Emit(...)`, the
  method form this ADR's `event.Bus` conversion added.
- **`client.Client.call`** (`Oblikovati.API/client/client.go`) — the Apache-2.0 API client's
  single most-used internal helper: every typed wire method funnels through
  `call[Resp any](c *Client, method string, req any) (Resp, error)`, ~641 call sites across
  101 files. Its own doc comment: *"Go has no generic methods, hence a package-level function
  taking the `*Client`"* — the textbook case. The existing non-generic 3-arg helper (used
  directly by ~12 void/no-typed-response call sites, e.g. deletes) was renamed `invoke` to
  free up the name; the new `(c *Client) call[Resp any](method string, req any) (Resp,
  error)` method now holds it, calling `c.invoke(method, req, &r)`. The package-level `call`
  free function is kept as a one-line wrapper (`return c.call[Resp](method, req)`) for the
  641 existing call sites; new client methods should prefer `c.call[Resp](method, req)`
  directly, matching the doc comment's now-updated example.

## Candidates rejected

- **`renderer.LightDistribution`/`EnvironmentDistribution`** (`renderer/light_sampling.go`,
  `renderer/environment_sampling.go`) — both do CDF-based importance sampling,
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
  `holderQuery`, `projectAll`) — these are combinators that build a
  `handlerFunc` value (e.g. `typedPart(getX)`), evaluated *before* any `Router` is in scope —
  the call site is `r.readOnly(wire.MethodX, typedPart(getX))`. None of them read or use a
  `Router`'s state; there is no receiver to hang a method off without inventing one. Router
  methods here would misleadingly imply these adapters depend on router identity.
- **`kernel/topo.buildKeyIndex`/`scanByKey`** (`kernel/topo/key_index.go`) —
  both are pure functions of their `items []T` argument, called from three sites inside
  `*Body`'s own methods (`b.cachedEdges`/`cachedFaces`/`cachedVertices`), each with a
  different concrete `T`. `Body` itself is non-generic, so nothing is lost by leaving these
  as free functions — a method would need the caller to pass `b`'s own field back into `b`'s
  method (`b.buildKeyIndex(b.cachedEdges)`), which is not more readable than today's
  `buildKeyIndex(b.cachedEdges)` and ties a stateless pure function to `*Body` for no reason.
- **Disqualified by package boundary** (a distinct, larger rejection class the sweep
  surfaced): `addin/opregistry.decodeFeatureArgs`, `addin/router.commitAssemblyProgramChange`,
  `addin/router.twoRefArgs` (all take `*app.Session`); `model/compdef.raiseBefore`,
  `model/doc.vetoed` (both take `*event.Bus`); the SolidWorks translator's
  `applyToLinePairs`/`applyToCurvePairs` (take `*sketch.Sketch`). Every one of these lives in a
  *different* package from the type it takes as first argument — Go methods can only be
  declared in the receiver type's own package, a rule generic methods don't relax. These stay
  free functions regardless of how well they'd otherwise fit the pattern.
- **Constraint-narrowing over an already-generic type's own parameter**:
  `model/proxy/geometry.go`'s `RangeBoxInContext[E Boxed](p Proxy[E])`,
  `PointInContext[E Pointed]`, `DirectionInContext[E Directed]`. `Proxy[E any]` is itself
  generic with an unconstrained `E`; these functions need a *narrower* constraint (`Boxed`,
  `Pointed`, `Directed`) than the type declares, for one operation each. That is not what Go
  1.27 generic methods add — a method's parameter list can't re-constrain a receiver's own
  already-bound type parameter — so these were never candidates, independent of any
  duplication judgment.

## Consequences

- New concurrency/dispatch/state-hub types with the `Bus`/`Session`/`Client` shape (a
  non-generic type whose per-call operation needs its own type parameter) should default to
  generic methods, not the free-function-plus-explicit-receiver-argument workaround this ADR
  retired for the five types above.
- A generic free function stays the right shape whenever the "receiver" is either stateless,
  interchangeable, cross-package from its own type, or not actually part of the call — most
  of this codebase's existing generics (per ADR-0001's small deliberate set) fall in exactly
  that bucket and are unaffected by this ADR.
- This sweep already covered every top-level generic function with a concrete-pointer first
  parameter in both repos (via `grep -rnE '^func [A-Za-z_]+\[[A-Za-z_]+ '`, cross-checked by
  hand); it is not a mandate to re-run that audit on every future free generic function — the
  bar above is the filter for new code and for revisiting this ADR's evaluated candidates,
  not a standing backlog item.
