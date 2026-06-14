# ADR-0035 — Assembly machining features: occurrence-relative references + a serialized feature program

**Status:** Accepted (2026-06-14) · **Builds on / refines:**
[ADR-0020](ADR-0020-yaml-git-friendly-document-format.md) (the recipe-only `.obk` document) —
this ADR adds *how an assembly's machining features are referenced and serialized*.
**Touches:** the assembly definition (`model/compdef/assembly.go`,
`assembly_features_recompute.go`, `assembly_serialize.go`), the assembly feature kinds
(`model/feature/assembly_*.go` + `assembly_feature_serialize.go`), the lineage-suffix
resolver (`kernel/topo/body.go`, `kernel/topo/lineage.go`), and the assembly tools
(`app/assembly_*_tools.go`).

## Context

M11-F08 adds **machining features authored in the assembly** that cut or modify *placed
component* geometry in place — an extrude/revolve/hole pocket, a chamfer/fillet/move-face on a
participant's edges or faces, a box-cut, or a proxy-cut driven by another occurrence. Each
feature recomputes **per participant**: the same feature applies to every placement of the
component(s) it touches.

Two problems had no obvious answer, both rooted in how a placed component's geometry is built.

**1. Reference keys are not stable across placements.** The assembly recompute transforms each
participant body with a **per-occurrence lineage prefix** — `asmFeatureLineage(i)` prepends
`topo.Tok("assemblyFeature", "occ", i)` to every entity's lineage
(`assembly_features_recompute.go`). So the reference key of "this vertical edge" differs for
occurrence 0 and occurrence 1 (the prefix carries the index), and differs again from the bare
key the edge has on the *component* document. The part dress-up ops we want to reuse
(`feature.chamferEdges`, `ops.FilletEdges`, `ops.MoveFaces`/`RotateFaces`) resolve an edge/face
by **exact** `Body.FindEdgeByKey` — so a key picked at add time is invalid after the next
recompute, and a bare component key matches nothing on a prefixed participant body. A naive
"store the key you picked" breaks the moment the feature recomputes or a second instance is
placed.

**2. The feature program was not persisted.** Occurrences serialized (ADR-0020 recipe), but the
assembly's *feature program* did not, so an assembly with a chamfer reopened — and undid —
without it (#785). Each feature kind holds heterogeneous inputs (a sketch reference, closure-
backed scalars, edge/face references, a tool solid, an occurrence reference), so there was no
single shape to write.

## Decision

**Store component-local lineage *suffixes*, resolve them per participant; serialize the program
as a tagged union over feature kind.**

### Occurrence-relative references (#735)

Because `asmFeatureLineage` only ever **prepends**, a participant entity's lineage key is the
component entity's lineage key with a placement-specific prefix in front — i.e. the component
key is a **suffix** of the participant key at a `/` token boundary. So:

1. A feature stores the **component-local** reference suffix (the component `ReferenceKey` with
   its leading kind byte stripped — `topo.LineageSuffixOf`), never the placement-specific full
   key.
2. At recompute, for each participant body, `Body.EdgeReferenceKeysWithLineageSuffix(suffix)`
   (and the face twin) returns the **full** reference keys of that body whose lineage key equals
   the suffix or ends with `"/" + suffix` (the `/`-boundary guard stops `edge#1` matching
   `edge#10`). Those full keys are handed to the existing exact-match part op unchanged.
3. A participant whose component does not carry the picked edge yields no match and **passes
   through unchanged** (`dressParticipants`).

Suffix matching is robust precisely *because* every placement carries a different prefix: the
shared component suffix is the stable part, so one stored suffix resolves correctly on every
instance — two placements of one component are both machined from a single feature. A
WORLD-picked edge first has its `occurrence:occ#i/` lineage stripped to a component-local suffix
before storage, so the picked key and the stored key live in the same (component) frame.

### Serialized feature program (#785)

The program serializes as `AssemblyFeatureData` — a **tagged union**: a `Kind` discriminator (a
named constant per kind) plus the superset of inputs the kinds use, each kind populating only
its own fields. `AssemblyFeatureMarshaler.MarshalAssembly(sketchIndex)` renders a feature to one;
`RestoreAssemblyFeature(data, sketches, occByName)` reconstructs it, switching on `Kind`. Rules
that fall out of the kinds' differing inputs:

- **Sketch-bearing kinds** (extrude/revolve/sweep) reference the assembly's sketch **by index**
  into the assembly's sketch collection — the sketch round-trips once, features cite it.
- **Closure-backed scalars** (distance/radius/angle, kept as `func() float64` so an edit reflows
  them) capture their current value and restore as a constant (`constScalar`); re-opening a
  restored feature for edit is a separate concern (#816).
- **The box-cut persists its tool's axis-aligned corners** (every tool is a `brep.SolidBlock`;
  `RangeBox()` recovers the corners), rebuilt on restore.
- **The proxy-cut persists its source occurrence by name**, rebound through `occByName` *after*
  the occurrences are bound — because restore order is occurrences-then-features, and an
  assembly `RestoreRecipe` is a full replace that re-creates occurrences (so any cached
  occurrence pointer is stale and must be re-fetched, paired with `ResolveReferences`).
- **Edge/face kinds persist the component-local suffixes** from the resolver above — frame-
  independent, so they survive the prefix churn that full keys could not.

Feature-kind discriminators are defined **once** as package constants
(`assembly_feature_serialize.go`), shared by the `Kind()` method, the restore switch, and
`MarshalAssembly`, so the three sites cannot drift.

## Consequences

- **One feature, every placement, stable across recompute.** Storing the suffix (not the picked
  full key) is what makes a chamfer survive the next recompute and apply to instances added
  later — the property a CAD assembly feature must have. The cost is an O(edges) suffix scan per
  participant per feature at recompute; acceptable, and bounded by the participant body size.

- **Existing part ops are reused unchanged.** The kernel dress-up ops stay exact-match and
  part-only; the assembly layer adapts by resolving keys, not by forking the ops. New assembly
  edge/face features are a thin `dressParticipants` wrapper.

- **The assembly round-trips through save/load and undo/redo.** The feature program is now part
  of the recipe, so an assembly reopens and undoes with its machining intact. A latent risk this
  introduces: any per-kind field the marshal writes but the restore drops is silent data loss
  the general round-trip test cannot catch if its fixture uses the field's zero value — exactly
  the flat-corner-chamfer bug (#785, fixed) — so each kind's non-default inputs need an explicit
  pin.

- **A restored feature is built from constants, not its original closures.** Until #816's
  in-session edit threads the live parameter back, a reopened scalar feature is editable only by
  delete-and-re-add. The closure seam is preserved (the scalar is still a `func`), so wiring the
  edit through is additive.

- **Restore order is load-bearing.** Occurrences must bind before features restore (proxy-cut
  rebind, participant resolution), and the assembly `RestoreRecipe` being a full replace means
  the app pairs every restore with `rebindReferences`/`ResolveReferences`. A feature that
  resolves an occurrence at restore time, rather than after the bind, would bind to a stale or
  absent placement.

## Shape

```
# component frame (stored)            # participant frame (resolved at recompute)
edgeSuffix = "edge#3/loop#0/…"        occ0 body: "assemblyFeature:occ#0/…/edge#3/loop#0/…"
                                      occ1 body: "assemblyFeature:occ#1/…/edge#3/loop#0/…"
  EdgeReferenceKeysWithLineageSuffix(edgeSuffix) → the full key on THIS participant → ops.FilletEdges

# serialized program (recipe)
features:
  - kind: assemblyChamfer   edgeSuffixes: [...]  distance: 0.2  flatCorners: false
  - kind: assemblyExtrude   sketchIndex: 0  profileIndex: 1  operation: 0  distance: 2.5
  - kind: assemblyCut       operation: 0  toolMin: [...]  toolMax: [...]
  - kind: assemblyProxyCut  operation: 0  sourceName: "widget:1"   # rebound after occurrences bind
```
