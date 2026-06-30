# ADR-0043 — Generalized provenance naming for all generated topology

**Status:** Accepted — design (2026-06-28). **Builds on / refines:** the M31 topological-naming
work — provenance naming for the planar boolean (`kernel/brep/boolean_provenance.go`,
`boolean_nonmanifold.go`; #1152/#1153/#1155), tiered binding (`model/identity/binding.go`, F06),
versioned encoding (F07) — and [ADR-0027](ADR-0027-curved-face-boolean.md) (curved boolean),
[ADR-0040](ADR-0040-external-geometric-references.md) (external geometric references).
**Touches (when implemented):** `kernel/topo` (a new shared provenance seam), the entity-
generating operations in `kernel/ops/*` and `kernel/brep/*`, and the dress-up resolution in
`model/feature/*`.

## Context

Every realized edge/face/vertex of the running body carries a **lineage reference key** — a kind
byte plus a `Feature:Role#Index` string (`kernel/topo/lineage.go`) — and feature selections
(fillet, chamfer, shell, draft, hole/boss faces, …) persist that key and re-resolve it each
recompute by exact match (`Body.FindEdgeByKey` / `FindFaceByKey`, via `resolveEdges`).

A reference is only as stable as the **name**. The M31 work established the robust rule for the
planar boolean (Kripac 1997): name a generated edge by the **two parent faces whose crossing
produced it**, captured from the ORIGINAL faces (before any split) and disambiguated, when a pair
recurs, by a **transform-invariant geometric characteristic** (position along the parents'
intersection line) rather than a build-order counter. That name survives an unrelated upstream
edit that reorders the stitch — the defining property of a topological name.

**The gap:** that robust scheme was applied **only to the planar boolean**. Every other operation
that generates topology still mints **construction-order ordinal names** whose index is a build
counter — fragile by exactly the mechanism M31 fixed for booleans:

| Site | Generates | Current scheme |
|---|---|---|
| `kernel/ops/assemble_curved.go:106` | fillet + curved-boolean assembly edges | `Tok(tag,"e",len(c.edges))` |
| `kernel/ops/delete_face.go:282` | delete-face stitch edges | `Tok("delface","e",i)` over a map |
| `kernel/ops/csg_body.go:417` | BSP/CSG edges | `Tok(feat,"edge",i)` |
| `kernel/ops/ruled_surface.go`, `wire_offset_corners.go`, `fillet_rim_build.go`, … | ruled/wire/rim edges | per-op ordinals |
| `model/feature/dressup.go` | every fillet/chamfer's result | hardcoded `"fillet"`/`"chamfer"` tag → all features of a kind **share one namespace** |
| `kernel/topo/body.go` `FindEdgeByKey` etc. | resolution | silent **first-match**, no ambiguity guard |

These do not all surface as user bugs today (e.g. `fillet:e#13` is stable in #1536 because the
curved blend's build order happens to be deterministic for that geometry), but each is a latent
wrong-rebind waiting for an upstream edit that reorders construction — the fillet-stack class
(#1494, #1536, #1537).

## Decision

Make provenance naming **the single naming discipline for all generated topology**, generalizing
the boolean's mechanism out of `kernel/brep` into a reusable seam, and retrofitting every
entity-generating operation to it.

### 1. A shared provenance seam (`kernel/topo/provenance`)

Lift the three boolean-private pieces into a small reusable package, behaviour-preserving:

- **`NameByParents(parents []topo.Lineage, rank int) topo.Lineage`** — generalizes
  `intersectionLineage`: a canonical, order-independent concatenation of the parent lineages
  (sorted by `Key()`), separated by a reserved token, with `rank>0` appended when several entities
  share one parent set. The parents are *original* entities, so the name is invariant to later
  subdivision and to build/stitch order.
- **`RankByAxis(entities, parentAxis)`** — generalizes `rankSamePairEdges`/`lineCharacteristic`:
  order entities that share a parent set by a transform-invariant characteristic along a parent-
  derived axis (equivariant under rigid motion), never by a build counter.
- **`Witness` resolution** — generalizes `edgeParents`: assign a built entity its parent set by
  geometric containment (midpoint-on-segment for edges; centroid-in-region for faces) against the
  op's recorded provenance.

The boolean is refactored to consume this seam (its tests stay green), proving the extraction.

### 2. Per-operation provenance capture

Each op records, for every entity it generates, the **generating parent set** — the input
entities that produced it. The parent set is op-specific but always *original* lineages:

| Operation | Generated entity | Generating parent set |
|---|---|---|
| Boolean (planar/curved) | intersection edge | the two crossing faces *(done; curved = Phases 4 + SSI-edge)* |
| **Fillet** | cylinder/torus/sphere blend face | the filleted edge (or its vertex, for a corner blend) |
| **Fillet** | tangent edge | (filleted edge, adjacent original face) |
| **Chamfer** | bevel face / edges | the chamfered edge + adjacent faces |
| **Delete-face** | stitch edge | the bordering surviving faces |
| **CSG/BSP** | cut edge | the two BSP half-space faces that bound it |
| Ruled / wire-offset / rim | rail/ruling edge | the generating section/wire entity |

An entity with no parent (a genuinely new free vertex) keeps an ordinal fallback under the op's
**unique feature tag** (see 3), which the uniqueness guard (4) backstops.

### 3. Per-feature unique lineage tags

`model/feature` passes each feature's **unique name** (`Fillet1`, `Fillet2`, … from
`PartFeatures.UniqueName`) as the lineage tag, replacing the hardcoded `"fillet"`/`"chamfer"`
constants, so two features of the same kind never share a namespace.

### 4. A resolution uniqueness guard

`FindEdgeByKey`/`FindFaceByKey`/`FindVertexByKey` and `resolveEdges`/`resolveFaceSet` detect **more
than one** match for a key and surface a clear error (or route to the F06 tiered binding to
disambiguate), instead of silently returning the first. This is the universal safety net: any
residual naming collision becomes an honest failure, never a wrong-rebind.

## Consequences

- **Buys:** references survive upstream edits across *all* operations, not just booleans; the
  fillet-stack class of bugs (#1494/#1536/#1537) is closed at the root; silent wrong-rebinds become
  honest errors; one naming discipline instead of two.
- **Costs:** each op's geometry builder must thread parent provenance through — a delicate change to
  the kernel's most intricate code (the curved blend). Done op-by-op behind the shared seam, each
  gated by a cross-edit stability test (the name must survive an upstream edit that reorders the
  build). Names get longer (parent concatenations); the F07 versioned encoding already absorbs this.
- **Rejected:** *transform-invariant geometric ordinals everywhere* (sort entities by a canonical
  geometric key) — stable under build/stitch reordering but NOT under geometry-moving edits that
  reorder the sort, so not a true topological name. *Guard-only* — turns wrong-rebinds into errors
  but leaves the names fragile. Both are strictly weaker than provenance and were folded in only as
  the no-parent fallback (geometric ordinal) and the safety net (guard).

## Scope & phasing (tracked as a milestone)

- **P0 — shared seam.** Extract `kernel/topo/provenance`; refactor the boolean onto it (no
  behaviour change). The uniqueness guard (4) and per-feature unique tags (3), which are
  independent of any single op.
- **P1 — fillet.** Thread parent provenance through the rolling-ball blend → `assemble_curved.go`.
  The user's pain point; gated by a cross-edit-stability regression test.
- **P2 — chamfer.** Same seam, bevel provenance.
- **P3 — delete-face, CSG/BSP.** Stitch/cut-edge provenance.
- **P4 — curved boolean.** Bring `curvedbool` onto the shared seam (it shares `assemble_curved`):
  P4 restored survivor identity on the curved path; the **SSI-edge** follow-up then named the new
  surface-intersection curves by their bordering face pair in `curvedStitch`
  (`RelineageByFaceProvenance`), with a geometric rank that disambiguates two intersection branches
  between one face pair (a bigon) by their curve midpoint. So no curved-boolean result keeps a
  `curvedbool:e#N` ordinal.
- **P5 — remaining generators** (ruled, sweep, wire-offset, rim, split, steinmetz, halfspace).
- **P6 — integrate with F06 tiered binding & F07 encoding**; retire the ordinal fallbacks that are
  no longer reachable. Sub-phased so the serialization change is isolated:
  - **P6a** — wire the live dress-up edge resolution (`resolveEdges`) to the tiered binder's
    **ancestral** tier through a `model/feature` adapter; a lost reference with a lone surviving
    parent-sibling heals to a Warning instead of going Sick. No format change (the parent is derived
    from the key). The exact-one P0 guard stays authoritative; only a 0/>1 miss escalates.
  - **P6b** — persist the mint-time **anchor** and enable the **geometric** tier, so several
    surviving siblings are disambiguated by nearness (the binder still refuses a tie). Isolated
    serialization change.
  - **P6c** — retire the now-dead silent-match resolution branches; make `Input.Keys`/`fs.keys` live
    (or remove). The no-parent ordinal fallback and the degraded CSG soup fallback legitimately stay.

Each phase is one or more PRs; each adds a regression test that mutates an upstream feature and
asserts the downstream selection still binds to the same physical entity.
