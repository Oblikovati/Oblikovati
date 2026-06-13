# Modeling 04 — Surfacing & freeform

*Modernizes M10 (surface creation/editing, freeform/sub-D, mesh & mold). These are
more features on the spine (modeling/01–03) plus new kernel demands; architecturally
they introduce **surface bodies** and **sub-D bodies** alongside solids, and lean on
kernel Phases B–D (ADR-0002).*

Almost nothing new is needed at the *architecture* level — surfacing is the feature
engine (modeling/01) over a broader kernel. What it adds: surface bodies as
first-class results, a sub-D body kind, and mesh bodies from import.

## Surface bodies are just bodies

`topo.Body` already distinguishes solid vs. surface (core/03). Surface features
produce open shells; the `Operation` enum's `surface` variant (modeling/02) routes a
feature's output to a surface body instead of a solid. The feature engine, reference
keys, async recompute, inspector — all unchanged.

| Feature | Definition highlights | Kernel op | Phase |
|---|---|---|---|
| **Boundary patch** | boundary `[]Ref` (edges/curves) + per-edge condition (G0/G1/G2) | `ops.BoundaryPatch` | B |
| **Ruled surface** | edge `[]Ref` + direction (normal/tangent/perp) + distance | `ops.Ruled` | B |
| **Sculpt** | bounding surfaces `[]Ref` → enclosed solid/surface region | `ops.Sculpt` | C |
| **Stitch / Knit** | surface `[]Ref` + tolerance (→ solid if closed) | `ops.Stitch` | C/D |
| **Trim / Extend** | surface + cutting/boundary `Ref` | `ops.Trim` / `ops.Extend` | B/C |
| **Offset / Mid-surface** | faces `[]Ref` + thickness; face-pairing for midsurface | `ops.Offset` | B/C |
| **Thicken** | surface/face `Ref` + thickness (→ solid) | `ops.Thicken` | B/C |

- **Conditions** (G0/G1/G2 continuity) are typed fields on the definition →
  reflection inspector exposes them (core/09).
- **Mid-surface** records per-pair thickness (for FEA, M18) as attributes (core/05).
- Open-profile/surface inputs are *allowed* here (rejected only for solids) — the
  feature decides, exactly as modeling/00 noted.

## Freeform (sub-D) is a distinct body kind

Sub-division-surface modeling needs a body representation the B-rep can't carry, so
`kernel/` gains a sibling:

```go
package subd
type Body struct{ cage Cage; faces []Face; edges []Edge; verts []Vertex; creases []Crease }
func (b *Body) ToBRep(tol float64) *topo.Body   // tessellate the limit surface into a B-rep body
```

- A `FreeformFeature` owns a `subd.Body` and **converts to a `topo.Body`** at recompute
  (the limit surface → NURBS/B-rep), so downstream solid features and the renderer
  see an ordinary body. The sub-D edit *operations* (move/scale/crease/subdivide/
  bridge/symmetry on cage selections) are the feature's editable state.
- Editing the cage is direct manipulation in the viewport (overlay/manipulators,
  core/08), committed as commands (core/06) → recompute → re-convert → re-tessellate.
- Sub-D is **independent of the B-rep kernel phases** — it can progress on its own
  timeline; it only needs the conversion-to-B-rep step (a NURBS fit, Phase B).

## Mesh bodies & mold

- **Mesh features** wrap imported tessellated geometry (`topo` gains a `MeshBody`
  with mesh face/edge/vertex). Selectable, convertible, but not parametric B-rep.
  Arrive via the translator framework (iteration 4, M17).
- **Core/cavity** (mold tooling) is a feature that splits a tooling block by a part's
  parting surfaces — `ops.Split` over surface inputs (Phase C/D).

## Why this iteration is light

The architecture decisions that matter (feature engine, reference-keyed inputs,
async recompute, sealed definitions, registry/inspector) were all made in
iteration 2 and apply unchanged. Surfacing's cost is **kernel work** (Phases B–D),
not architecture — which is exactly why ADR-0002's phasing matters: the surfacing
features and their UI exist now; their ops return `NotYetImplemented` health until
the relevant kernel phase lands. Sub-D is the one genuinely new representation, and
it is cleanly isolated in `kernel/subd` behind a `ToBRep` boundary.

## As built (M10, 2026-06)

What shipped, against the plan above (issues #333–#344; exposure pass #697–#705):

- **Boundary patch**: a closed planar profile fills into a one-face trimmed surface
  body; per-loop G0/G1/G2 conditions are carried (vacuously satisfied for an isolated
  planar loop — curved blends are NURBS phase B).
- **Ruled surface**: the normal mode builds a real planar-quad band; tangent /
  perpendicular resolve inputs then defer (Warning).
- **Stitch/Knit**: an exact-coincidence weld (tolerance-grid vertex merge); a closed
  quilt becomes a solid. Tolerant near-gap `Sew` stays phase D. **Sculpt** fills an
  enclosed volume.
- **Trim**: Sutherland–Hodgman half-space clip of planar surfaces (single and
  coplanar multi-face); curved trims stay phase B/C. **Extend** grows a planar
  surface along a boundary edge. **Surface offset** translates planar patches;
  **mid-surface** pairs antiparallel planar faces under a thickness threshold and
  records per-pair thickness (for FEA, M18).
- **Freeform**: `kernel/subd` is the sub-D kernel — control cage + per-edge creases,
  Catmull–Clark `SubdivideN`, `ToBody` to a planar B-rep (closed cage → solid). The
  conversion is per-level subdivision, not the `ToBRep` NURBS limit-surface fit the
  plan sketched (that fit is the remaining phase-B step). Cage editing
  (level/move/crease) is exposed over the wire as `freeform.*`; viewport direct
  manipulation is follow-up UI work.
- **Mesh**: an ASCII STL parses into welded reference geometry (`MeshFeature`) with
  selectable facet topology, placeable from the ribbon and the `mesh` op — there is
  no `topo.MeshBody`; the body-producing import path is the translator framework
  (`ImportedBodyFeature`, M17). **Core/cavity** parts the tooling block by a planar
  parting (axis + position + shrinkage); part-shaped pockets and silhouette parting
  surfaces stay phase C (#653).

Every feature above is reachable four ways: ribbon tool, `features.add` op schema,
`features.get`/`features.edit` scalars, and the serialized `.obk` recipe.
