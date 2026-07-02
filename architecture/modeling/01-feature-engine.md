# Modeling 01 — Feature-history engine

*Modernizes M07-F04 (part content container) + M08-F01 (feature history engine).
Implements ADR-0010. The heart of the modeler — "model = evaluated program"
(parametric-cad §0, §2) on the Go stack.*

## The content container

```go
package compdef
type PartComponentDefinition struct {
    features  []feature.Feature   // THE ordered program (history)
    eop       int                 // end-of-part / rollback marker (index)
    params    *param.Graph        // the variable layer (core/04)
    sketches  Collection[*sketch.Sketch]
    bodies    []*topo.Body        // CACHE: result of the last evaluation
    version   uint64              // ModelGeometryVersion (bumps every edit)
}
```

(Reference identity needs no field here: keys are `identity.RefKey` values stored
inside the feature/sketch definitions, rebound against each entity's `Lineage`
after recompute — core/05.)

`bodies` is a *cache* of evaluating `features[:eop]`, never the source of truth
(parametric-cad §0). The truth is the feature list + parameters + sketches — that is
what persists (core/05).

## The Definition → Add → Feature triangle, in Go

Every feature is three things (parametric-cad §3), now plain Go:

```go
type Definition interface{ Kind() feature.Kind }      // the editable recipe (a POD struct)

type Feature interface {
    Name() string
    Definition() Definition
    SetDefinition(Definition) error                   // round-trip edit → dirty → recompute
    Health() Health
    Suppressed() bool
    recompute(ctx, in Inputs) (Result, error)         // PURE; called by the engine
}

// the owning typed collection is the only constructor (the seam — parametric-cad §13)
func (f *PartFeatures) Add(def Definition) (Feature, error)   // wraps Add in a command (core/06)
```

- The `Definition` is an inert struct of plain-old-data + `param.Quantity` + `Ref`
  (ADR-0010). It is simultaneously: the create payload, the editable record, the
  serialized form, the reflection-inspector source (core/09), and the gRPC `Edit`
  message — **one representation, five uses.**
- `feature.Definition()` returns it; `SetDefinition` round-trips → the feature
  dirties and recomputes. Construction is only via `PartFeatures.Add`, never `new` —
  the collection hooks history/identity/undo.

## Inputs are reference keys, resolved at recompute (ADR-0010)

A feature holds `Ref`s, not pointers, to the topology/sketches it consumes:

```go
type Ref struct{ key identity.RefKey; ctx *OccurrenceContext }
type Inputs struct {
    bodies  []*topo.Body                  // the running state from prior features
    resolve func(Ref) (topo.Entity, bool) // resolves against CURRENT topology
    params  func(param.ParamID) param.Quantity
}
```

At the top of `recompute`, each `Ref` is resolved. **Resolved → proceed. Lost →
`Health.Sick`, poison dependents, do not abort the rebuild** (parametric-cad §7).
This is the make-or-break seam: keys derive from `topo.Lineage` (core/03), so "the
same" face re-resolves after an upstream change and a vanished face resolves to
nothing → re-selection, not a crash.

## Recompute: rollback-replay, async, pure (ADR-0007 + ADR-0010)

```
on input change:
  dirty   := paramOrTopoChange → DAG.DirtyClosure(features)     # same DAG as core/04
  first   := earliest dirty feature in history order
  snapshot:= immutable {params, sketches, refKeys, bodies[:before(first)]}
  enqueue job(ctx, snapshot):                                   # worker pool (core/00)
      bodies := snapshot.prefixBodies                           # reuse clean prefix
      for f in features[first:eop]:
          if f.Suppressed || suppressedByCondition(f): f.health=Suppressed; continue
          in  := Inputs{bodies, resolve(refKeys, bodies), params}
          res, err := f.recompute(ctx, in)                      # calls kernel ops (core/03)
          if err != nil { f.health = Sick; poison(dependents); continue }  # never abort all
          bodies = res.bodies ; recordLineage(res)              # advance running state
      tessellate(changed bodies)                                # parallel
      return bodies, tess
  # at a frame boundary (core/00 phase 5): swap bodies + meshes, bump version, fire "recomputed"
```

Properties:
- **Clean prefix reused** — only the dirty tail replays (not the whole model).
- **Independent branches parallel** — unconnected feature chains / separate bodies
  replay concurrently (the DAG parallelism, like transforms in realtime-3d §3).
- **Cancellable** — a newer edit cancels the in-flight job via `ctx`; stale results
  are dropped (ADR-0007). The view keeps showing last-good geometry meanwhile.
- **Pure** — inputs in, bodies out, no global state → unit-testable headless and
  reusable verbatim as the gRPC recompute path (dogfood, core/07).

## Health, suppression, reorder, rollback

- **Health** (`ok|warning|sick|suppressed`) is per-feature state, surfaced in the
  browser (core/09) — never an exception (parametric-cad §2).
- **Suppression**: a flag, plus **conditional suppression** driven by an expression
  (`when d0 < 5mm`) evaluated through the parameter DAG (core/04) — suppression is
  part of the parametric program, not a UI toggle.
- **Reorder/rename** mutate the history list (via commands); reorder past a
  dependency is rejected using the DAG. Rename is id-stable (refs unaffected).
- **End-of-part marker** (`eop`) defines how far the program evaluates; moving it is
  mid-history editing (`SetEndOfPart(before)`), re-running replay to that point.

## Tessellation hand-off to the view

After recompute, changed bodies are tessellated (kernel, core/03) to `float32`
meshes + edge polylines and **swapped into the renderer's mesh cache keyed by the
body's reference key** (core/08). Every scene entity referencing that body updates
with no per-entity sync — the cache is the one swap point. Assembly reuse falls out:
one body mesh, many occurrence transforms (the flyweight, parametric-cad §5).

## Net mapping from COM

| COM | Here |
|---|---|
| `PartFeatures` + `ExtrudeFeatures.Add(def)` | `PartFeatures.Add(Definition)` wrapped in a command |
| `Feature.Definition` get/set | same, but `Definition` is a POD struct (5 uses) |
| synchronous rollback/replay recompute | rollback-replay on the job pool, cancellable (ADR-0007) |
| `HealthStatusEnum` via `Type` checks | typed `Health` field |
| `SetSuppressionCondition(param,cmp,expr)` | expression in the param DAG |
| `GetReferenceKey`/`BindKeyToObject` on inputs | `Ref` resolved at recompute (ADR-0010) |
| `ModelGeometryVersion` string | `version uint64` bumped per edit |
