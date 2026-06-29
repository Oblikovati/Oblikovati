# ADR-0044 — Recompute dependency-key invalidation seam

**Status:** Accepted — design (2026-06-29). · **Builds on / refines:**
[ADR-0018](ADR-0018-apache-api-contract-module.md) (the API/implementation split this
follows), the M39 `ParameterHolder` seam (part-or-assembly parameter ownership),
Oblikovati#1414 (incremental parameter recompute) and #1413 (the single parameter-edit
invalidation seam). · **Complements** [ADR-0040](ADR-0040-external-geometric-references.md):
that ADR supplies the *identity* of an external geometric selection (binding geometry →
topology); this ADR supplies the *invalidation* half (which consumers a change dirties).
Together they are the two halves a future cross-part **adaptive reference** needs. ·
**Touches (when implemented):** new `model/depend`, `model/param/tracking.go`,
`model/compdef/{parameter_holder,part,assembly}.go`, `model/feature/engine_param_invalidate.go`.

## Context

A parameter edit must trigger the **minimal correct** geometry rebuild: dirty only the
features a change can reach, reuse cached bodies for the rest (Oblikovati#1414). Today the
part does this with two pieces:

- **Footprint capture** — `param.Parameters.Track` wraps a recompute and records which
  `param.ID`s were read.
- **Consumer attribution** — `PartFeatures.featureAffectedByParams`: a feature is dirty if a
  changed parameter is in its direct `paramReads` or in a consumed 2D sketch's
  `ParameterFootprint`.

Everything else read during recompute — a work-plane offset, a 3D-sketch dimension, a
host-plane closure — is *not* attributable to a single feature, so it is dumped into
`wholesaleParams` and forces a full rebuild. This is the **load-bearing invariant**
(#1403/#1414): **a read through an un-attributed path is never silently skipped — it
conservatively rebuilds everything.** Silent-stale geometry is the worst failure class.

Two pressures converge on this code:

1. **F07 (#1563)** wants to *shrink* the wholesale fallback by attributing the work-plane,
   3D-sketch, and host-plane paths to their consuming features.
2. **Future cross-part adaptive references** (part B geometry driven by part A geometry,
   mediated by the assembly) are the *geometry-level* sibling of Derived Parameter Tables:
   a change in one document must dirty the dependent features in another. This is the same
   footprint-and-attribution problem, one level up and on evaluated B-rep instead of scalars.

The directive is **"set the foundation now so adaptive references just fit later, without a
rewrite."** The trap is reading that as "build the full assembly incremental engine now": an
incremental engine is shaped to *what it can attribute*, and the only cross-part input that
exists today is occurrence placement (which full re-solve already handles). Building it now
means shaping a real engine against imagined requirements — the code most likely to be
rebuilt when real adaptivity lands.

## Decision

The foundation is a **stable invalidation seam + a generalized dependency key + the
wholesale-never-skip invariant** — *not* a built-out assembly incremental engine. A
**wholesale implementation is a valid point on the same contract** ("attribute nothing →
rebuild everything" is the degenerate, always-correct case), so the assembly satisfies the
seam now wholesale, and adaptivity narrows it later from inside the seam.

1. **Generalize the footprint key now (`model/depend`, document-free).** Replace the
   `param.ID`-typed footprint with an opaque comparable value:

   ```go
   type KeyKind uint8
   const ( ParameterKey KeyKind = iota; WorkPlaneKey; SketchKey; ExternalGeometryKey )
   type Key struct { Kind KeyKind; ID uint64 }   // comparable value type
   type Footprint []Key
   ```

   Parameters are `Key{ParameterKey, …}`. `ExternalGeometryKey` is **defined now but
   produced only by the future adaptivity resolver**. This is the one thing expensive to
   retrofit: hard-coding `param.ID` through capture/footprint/attribution would force a new
   key type through the whole engine later — the rewrite we are avoiding. Generalizing while
   only parameters use it is cheap now.

2. **Holder-agnostic attribution (`model/depend.DirtyTail`).** The
   `changed-keys ∩ footprint → earliest dirty consumer → dirty the tail` logic is written
   once over `depend.Key` and reused by part (incremental) and assembly (wholesale).

3. **Generalized holder seam.** `ParameterHolder` gains
   `RecomputeAfterChange(changed []depend.Key)` — the single invalidation entry for parameter
   edits, geometry edits, and future edge kinds (subsuming `RecomputeAfterParameterEdit`,
   keeping #1413's "one seam, no divergence"). It **MAY** rebuild wholesale for any
   unattributable key; it **MUST NOT** silently skip one (the invariant).

4. **Part narrows; assembly stays wholesale.** The part's F07 work attributes the
   work-plane / 3D-sketch / host-plane paths over `depend.Key`. The assembly satisfies
   `RecomputeAfterChange` with a full re-solve (its current behaviour, now behind the named
   seam — exercised, not speculative).

5. **The inter-document dependency graph is owned by orchestration, not the model.** When
   adaptivity lands, `app.Session`/`Workspace` resolves an `ExternalGeometryKey` to a
   `(sourceDocument, topoRefKey)` via the workspace reference graph — the **same
   anticorruption seam** that already resolves Derived-Parameter-Table source documents. The
   model reports "I read key K"; only the orchestrator knows K names another document. The
   key's geometric identity reuses ADR-0040's external geometric reference and ADR-0043
   provenance. `model` stays document-free (`compdef` must not import `app`); no cycle,
   because `ExternalGeometryKey` is opaque to the model.

## Consequences

- **Adaptivity slots in behind a stable seam:** it adds an `ExternalGeometryKey` producer
  (the Session resolver) and, when justified, narrows the assembly body — touching neither
  the key type, the attribution logic, nor the part. The "no rebuild later" goal is met by
  the seam, not by pre-building the engine.
- **Assembly parameter edits stay full re-solve** until a future milestone narrows them.
  Correct today, just not incremental — and the path to incremental is purely additive.
- **One mechanical sweep now:** routing the existing param-id footprint path through
  `depend.Key`. Attribution becomes shared instead of part-private.
- **Safety floor preserved and generalized:** the wholesale-never-skip invariant now lives in
  the shared seam, so every holder — and every future edge kind — inherits it. An adaptive
  reference not yet in the footprint model will conservatively force a re-solve, never be
  silently ignored.

### Why not build the full assembly incremental engine now

It has nothing to attribute (no cross-part edges exist yet) and would be shaped to guessed
adaptivity requirements — maximizing, not minimizing, the chance of a later rewrite. A
wholesale assembly behind the generalized seam gives the same external guarantee (the seam
exists and is exercised) at a fraction of the risk.

### Why not keep `param.ID` and add a parallel geometry path later

Two key types means two capture paths, two attribution routines, and a divergent invalidation
surface — exactly the divergence #1413 consolidated away, and the rebuild this ADR exists to
prevent.

## Scope

This ADR + `model/depend` + the part's F07 narrowing + the assembly satisfying the seam
wholesale are **this change (F07)**. The Session `ExternalGeometryKey` resolver and any
assembly incremental narrowing are the **adaptivity milestone**, and plug into the seam
defined here.
