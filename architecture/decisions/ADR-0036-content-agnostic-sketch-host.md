# ADR-0036 — The sketch environment hosts on a content-agnostic `sketchHost` interface (part or assembly)

**Status:** Accepted (2026-06-14) · **Builds on / refines:**
[ADR-0035](ADR-0035-assembly-machining-features.md) (assembly machining features — the consumer
of assembly-authored sketches). **Touches:** the sketch environment (`app/sketch_env.go`), the
assembly definition (`model/compdef/assembly.go` — `Sketches`/`Parameters`/`Recompute`
accessors), and the Sketch ribbon (`app/commands_sketch.go`).

## Context

The sketch environment — create-2D-sketch → edit → finish — was written against the **part**: it
added the sketch to the active `PartComponentDefinition`, shared the part's parameter DAG, and
recomputed the part on finish. M11-F08 needs the *same* flow on an **assembly**: an assembly
extrude/revolve/sweep is driven by a sketch authored in the assembly (#788), and that sketch must
share the assembly's parameters and recompute the assembly on finish.

The options were: (a) duplicate the create/finish/recompute flow for assemblies; (b) make the
session reach into each content type with a type switch; or (c) name the small capability the
sketch environment actually needs and let both content types satisfy it.

## Decision

The session's sketch environment depends on a **`sketchHost` interface**, not on a concrete
content type:

```go
type sketchHost interface {
    Sketches() *sketch.Sketches      // the collection a new sketch is added to
    Parameters() *param.Parameters   // the DAG a sketch's dimension expressions resolve in
    Recompute()                      // re-run the program so finishing a sketch is seen downstream
}
```

`activeSketchHost(s)` resolves the active document's `Content()` to a `sketchHost`, erroring when
there is no active document or its content hosts no sketches. `CreateSketch` / `FinishSketch` /
`CanCreateSketch` are written entirely against the interface, so they open, finish, and recompute
on a part or an assembly **without knowing which**. The part definition already had these methods;
the assembly definition gained `Sketches()`, `Parameters()`, and a `Recompute()` alias for its
feature recompute. This mirrors the content-agnostic seam the `addin/router` already uses.

## Consequences

- **One sketch environment, two hosts, no duplication.** The create/finish/recompute logic lives
  once. Adding a third sketch-hosting content type (were one to appear) is implementing three
  methods, not forking the flow.

- **No new coupling.** `app` already imports `model/sketch` and `model/param`; the interface is
  expressed in those, so the sketch environment does **not** import `compdef` to name the part or
  assembly type. The dependency points the right way (content satisfies an app-owned interface).

- **The part path is unchanged — and must stay verified.** Generalizing a part-only flow risks a
  silent part regression, so the part sketch tests remain the safety gate: they must stay green
  alongside the new assembly-sketch tests (they did).

- **The interface is intentionally minimal.** It names only what the environment needs. A content
  type that owns sketches but, say, cannot recompute cheaply would still have to satisfy
  `Recompute()`; if that ever bites, the interface splits rather than the callers branching.
