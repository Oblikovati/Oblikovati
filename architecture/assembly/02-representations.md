# Assembly 02 — Representations & model states

*Modernizes M12-F04 (design-view / positional / level-of-detail representations,
model states). Representations are **named override layers** over the assembly —
captured, activated, and edited through commands/events; no new engine.*

The three representation families are orthogonal override sets the user captures and
switches between. Each is a named layer that, when active, overrides aspects of the
base assembly without mutating it.

```go
package assembly
type Representations struct {
    design   Collection[*DesignViewRep]   // visibility / appearance / selection / camera
    position Collection[*PositionalRep]    // constraint & joint value overrides
    lod      Collection[*LevelOfDetailRep] // component suppression (performance)
    active   activeReps
}
```

## Design-view representations — visual override layer

```go
type DesignViewRep struct {
    name        string
    visibility  map[OccurrencePath]bool    // keyed by occurrence path (doc 00)
    appearance  map[OccurrencePath]Appearance
    sectioning  []SectionPlane
    camera      *scene.CameraState
}
```

- Overrides are keyed by **occurrence path** (doc 00) — stable instance identity, not
  pointers — so a captured view survives recompute and reload.
- Activating a rep applies its overrides to the scene (core/08): hidden occurrences
  drop from the render queue, appearance overrides swap material instances (caches,
  core/08), section planes add a clipping pass. The base model is untouched.
- Appearance precedence (occurrence override > rep override > definition material) is
  the same source-resolution the part appearance system uses (M16) — one rule.

## Positional representations — constraint/joint value overrides

```go
type PositionalRep struct {
    name      string
    overrides map[ConstraintID]param.Quantity   // alternate offset/angle/driven values
    flexible  map[OccurrencePath]bool            // subassembly flexibility per rep
}
```

A positional rep stores **alternate values** for constraint/joint parameters
(core/04) — e.g. a piston at top-dead-center vs bottom. Activating it sets those
parameter overrides and triggers an **async position re-solve** (doc 01, ADR-0007),
warm-started from the current configuration. It is "a named set of parameter
overrides + a solve," nothing more.

## Level-of-detail representations — suppression for performance

```go
type LODRep struct {
    name       string
    suppressed map[OccurrencePath]bool   // which occurrences/subassemblies are unloaded
}
```

LOD reps suppress occurrences to cut memory/recompute on huge assemblies. A
suppressed occurrence's definition need not even be loaded (the document reference
stub stays unopened — core/05). This is the performance lever for large-assembly
work, and it composes with GPU instancing (doc 00): fewer occurrences → fewer
transforms in the instanced draw.

## Model states unify the three

A **model state** is a named tuple `(designView, positional, lod)` selecting one of
each — the single switch users actually flip. Switching a model state is a command
(core/06) that activates the three layers together and fires representation events
(`ModelStateEvents`/`RepresentationEvents` in COM → typed events, core/06).

```go
type ModelState struct{ name string; design, position, lod string }   // names into the collections
func (c *Content) Activate(ms *ModelState) Command   // undoable; fires before/after events
```

## Why representations need no new architecture

Every piece already exists:
- **Overrides keyed by occurrence path** — stable identity from doc 00.
- **Appearance/visibility** apply to the **scene/render-queue** (core/08), not the
  model — the base assembly is immutable, so reps are cheap and non-destructive.
- **Positional re-solve** is the doc-01 solver + async recompute (ADR-0007).
- **LOD suppression** rides the document-reference lazy-load (core/05).
- **Activation** is a command; switches fire **typed events** (core/06).

Representations are thus a thin, declarative layer — exactly what the parametric-cad
skill's "overrides live on the occurrence/representation; shared truth on the
definition" (§5) prescribes, realized as override maps over a base that is never
mutated.

## Net mapping from COM

| COM | Here |
|---|---|
| `DesignViewRepresentation` | `DesignViewRep` override maps (visibility/appearance/section/camera) |
| `PositionalRepresentation` | `PositionalRep` constraint/joint parameter overrides + re-solve |
| `LevelOfDetailRepresentation` | `LODRep` suppression set (+ lazy-unload via core/05) |
| model states | `ModelState` tuple selecting one of each; activated via command |
| `RepresentationEvents`/`ModelStateEvents` | typed events (core/06) |
