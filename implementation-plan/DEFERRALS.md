# Deferral Ledger

_Generated 2026-06-04 during the audit (see REPORT.md §6). This is the
canonical, code-derived list of where "done" means "scaffolded": every
`NotYetImplemented`, every `ErrDeferred` passthrough, and the in-code `follow-up` /
`phase B/C/D` / `deferred` comments in non-test source._

How to read the **G-impact** column (per CONVENTIONS.md "Status model"):

- **G⬜** — the feature's primary case produces **no geometry** here (hard blocker on its
  Geometry axis).
- **partial** — the common case works; this is a missing sub-case / quality refinement.
- **n/a** — infrastructural (bootstrap) or a comment only.

Regenerate with:
```
grep -rn 'NotYetImplemented\|ErrDeferred' --include=*.go . | grep -v _test.go
grep -rni 'follow-up\|phase [bcd]\|later refinement\|deferred' --include=*.go . | grep -v _test.go
```

---

## A. Hard `NotYetImplemented` (returns an error — feature is unbuilt)

| Site | What is missing | Feature / PBI | G-impact |
|------|-----------------|---------------|:--------:|
| ~~`model/feature/sketched_features.go:238`~~ | **Rib** generation **RESOLVED 2026-06-04** — thickened-band wall extruded by a finite Depth + combine; tested (wall vol=8, L-path solid). **Full DoD 2026-06-04**: `RibFeatures.Add`, serialize round-trip, `RibTool` + `Create.Rib` ribbon + generic dialog + e2e. "To-next" part-bounding is the only refinement. | RibFeature / PBI-096 | ~~G⬜~~ → **G✅ U✅** |
| `kernel/ops/validate.go:67` | **Tolerant Sew** (stitch near-coincident topology) | PBI-084 | **G⬜** (stitch only welds exact-coincident) |
| `kernel/ops/surface_edit.go` | **Trim** — **multi-face DONE 2026-06-05 (K5)** (clip each planar face + weld; folded sheets/quilts work). Remaining: trimming a body with any **curved** face (NURBS surface–surface trim). | TrimFeature / PBI-111 | planar (single+multi-face) ✅; curved ⬜ |
| `kernel/ops/surface_edit.go` | **Surface offset** — **multi-face (coplanar) DONE 2026-06-05 (K5)** (translate each face + weld). Remaining: **folded** multi-face (intersect adjacent offset planes) and **curved** faces (parallel NURBS surface). | SurfaceOffsetFeature / PBI-112 | planar single + coplanar-multi ✅; folded/curved ⬜ |
| ~~`kernel/geom/transform.go:36,60`~~ | ~~Transform of NURBS curves/surfaces~~ **RESOLVED 2026-06-04 (K2)** — control-point transform, exact for affine; weights/knots/degree invariant. Other exotic curve types (Polyline/EllipticalArc/Helix3d) still NYI. | body transform / pattern / mirror / move (PBI-190/191) | ~~G⬜~~ → **G✅ for NURBS** |
| `cmd/oblikovati/main.go:19` | windowed runtime bootstrap via this entry | PBI-001 | n/a (the `head` binary is the real entry) |

## B. `ErrDeferred` passthrough (inputs resolve, body returned **unchanged**)

These read as health **Warning**, not Sick, so they look "ok" in the tree but change
no geometry.

| Site | Feature | PBI | G-impact |
|------|---------|-----|:--------:|
| `model/feature/dressup.go:134` | Dress-up deferred path (the deferred dress-up features that route through here) | M09 F02/F03 | **G⬜** for any dress-up still on this path |
| `model/feature/hole_boss.go` (defers boss) | **Boss** | PBI-103 | **G⬜** |
| `model/feature/surface_edit.go` (ExtendFeature) | **Extend** — **planar DONE 2026-06-05 (K5)**: `ops.ExtendByEdge` grows a planar surface along a boundary edge; `ExtendTool` + `Surface.Extend` ribbon. Remaining: multi-face / curved-surface extend (NURBS). | PBI-111 | ~~G⬜~~ → **G✅ (planar)** |
| `model/feature/surface.go:156` | **RuledTangent** / **RuledPerpendicular** ruled-surface modes | PBI-109 | **G⬜** for those modes (RuledNormal works) |
| `model/feature/surface_edit.go:103` | direct surface edit deferred path | M10 | **G⬜** for that op |
| `model/feature/modify.go:81` | **Split** — Split Solid / Trim Solid by a work plane **DONE 2026-06-05** (`SplitSolidFeature` + `ops.SplitSolidByPlane` + `SplitTool` + ribbon + serialize + e2e). The deferred `SplitFeature` (face-edit shape) remains as **Split Face** (split faces by a plane/curve), still G⬜. | PBI-106 | ~~G⬜~~ → **G✅ (solid)**; Split Face still ⬜ |

## C. Kernel-phase / quality follow-ups (common case works; refinement pending)

| Site | Refinement deferred | Phase |
|------|---------------------|:-----:|
| `kernel/ops/fillet.go:20` | fillet edge-chains, fillet-fillet corner blends, concave edges | B |
| `kernel/ops/thicken.go:16` | thicken of creased/curved sheets | follow-up |
| `kernel/ops/shell.go:16` | outward / both-sides shell (inward only today) | follow-up |
| `kernel/ops/move_face.go:15` | move large enough to collapse/invert a face (retopology) | follow-up |
| `kernel/ops/tessellate_trim.go:23` | general constrained triangulation of trim regions (iso-rectangle bands + **full periodic bands** [K1b 2b, done 2026-06-04] + small boundary patches now work; holes / arbitrary non-rectangular trims remain) | follow-up |
| `kernel/ops/stitch.go:25` | tolerant (gap-band) stitching | D |
| `kernel/brep/boolean.go:39` | booleans where operands **share a face plane** the general way; **and (2026-06-05) PARTIAL PENETRATION / concave faceted-wall crossing** — the 2D arrangement only cuts a face when imprint segments close a loop *on it*, so a tool that pokes part-way in / crosses a re-entrant wall leaves dangling segments → overlap left un-cut → non-manifold / inverted normals. Tracked as **PBI-199**; repro = skipped fan e2e. Dead-end: post-stitch orientation-repair does NOT fix it. | follow-up / **PBI-199** |
| `kernel/brep/boolean_stitch.go` | **face reference-key survival through a boolean** — **DONE 2026-06-04 (K1a)**: result faces carry their source face's lineage (single-piece → exact key; split → child token); tested. *Remaining:* edge/vertex key survival; cross-operand collision when operands share lineage. | follow-up (edges) |
| `model/feature/mold.go:17` | mold part-shaped pocket + silhouette parting (general solid-solid boolean) | C |
| `model/feature/extent.go:41`, `extrude_extent.go:151` | angled/curved extrude trims ("to-next/to-face" curved) | C |
| `model/feature/sweep_loft.go:18` | guide rails, centerline twist, swept *surfaces* | refinement |
| `model/feature/sketched_features.go:19` | curved profile edges, exact analytic/NURBS swept surfaces (revolve/coil) | refinement |
| ~~`model/feature/patterns.go:22`~~ | source-only (vs whole-solid) pattern replication + per-feature provenance — **DONE 2026-06-05 (PBI-191)**: `ToolFeature`/`OperationalFeature` + `Input.SourceTool`; a pattern re-applies the source's cut/join per occurrence (one body, N holes/blades) instead of N copies; all boolean tool features incl. Hole carry the contract; a deferred source (Boss) replicates nothing. | ~~follow-up~~ → **done** |
| `model/feature/swept.go` | ~~twisted/lofted **warped non-planar quad** side faces broke the planar boolean~~ — **DONE 2026-06-05 (PBI-174)**: `sideQuad`/`quadPlanar` triangulate a non-coplanar side quad so `sweptSolid` stays planar-faceted. | done |
| `model/sketch/edit_trim.go:26` | trim/extend against **curves** (lines only today) | follow-up |
| `model/sketch/curvesample_spline*.go` | true NURBS spline fidelity (polyline approximation today) | kernel |
| `model/sketch/path.go:30` | sampling **arc** path segments into the rail polyline | refinement |
| `model/sketch/inference.go:24` | inference **glyph overlay** in the UI | UI |

---

## Summary

- **Functional no-ops (G⬜) marked otherwise-done:** Boss, Split Face,
  RuledTangent/Perpendicular, tolerant Sew, deferred dress-up/surface-edit paths.
  (NURBS body-transform **resolved 2026-06-04, K2**.) **These should not carry a ✅
  until their G axis is green.**
- **Highest-risk quality follow-up:** reference-key survival through a boolean
  (`kernel/brep/boolean_stitch.go:17`) — picks on boolean output (Hole/Combine/Cut)
  don't rebind after an edit, undermining parametric robustness.
- Each row above should become a tracked issue and be linked from the owning PBI's
  **G** axis in PROGRESS.md.
