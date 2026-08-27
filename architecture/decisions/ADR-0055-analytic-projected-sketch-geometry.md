<!-- SPDX-License-Identifier: GPL-2.0-only -->

# ADR-0055 — Projected sketch geometry is an analytic reference, never a sampled polyline

**Status:** Proposed (2026-08-27). Refines the projection seam introduced in M07
([`model/sketch/projection.go`](../../model/sketch/projection.go),
[`model/compdef/reference_source.go`](../../model/compdef/reference_source.go)); relies on the
ADR-0043 topological-naming reference keys for the source handle. Drives the fix for the
`piston-head.opd` file-size and faceting defects (see Context).

## Context

Projecting a model edge into a sketch ("Project Geometry" / include reference geometry) currently
**samples the source edge into a 17-point polyline** and stores that polyline. Evidence, from the
real demo part `Oblikovati.Demos/S54 Engine/piston-head.opd` (3 sketches, 3 extrudes):

- The file is **282 KB**; `sketches:` is **98.5%** of it. **71% of the whole file (202 KB) is bare
  coordinate numbers** — 146 `projectedCurve` entities each storing a fixed **34-float** (`coords:`)
  polyline. The only real input per curve is its `source:` field, a topological-name reference to the
  model edge (e.g. `"\x02Extrusion2:top-edge#0"`); the `coords` are a derived, discretized copy.
- A fresh headless recompute of that part yields a body with **333 planar faces and 0 analytic
  cylinders** — fully faceted. The projected profiles are consumed as polylines, so the extruded
  walls are faceted before any boolean runs. This is why the #2167 cocylindrical seam persists on
  the real part even though the boolean-level fix is in place: the walls were never analytic.

The discretization is a single seam. `EdgeRefSource` holds the exact `geom.Curve3`
(`reference_source.go:49`) but the `CurveSource` interface only exposes `SamplePoints() []Point3`
(`projection.go:26`); the curve is sampled at `referenceSampleSteps = 16`, the points are serialized
as `coords`, reloaded verbatim, and rejected by the extrude's analytic path (`rawSegmentOf` handles
only `LineKind`/`ArcKind`), so the profile falls to the faceted `buildPrism`. **One seam causes both
the file bloat and the faceting.**

Three independent problems follow from the polyline representation, all raised by the user:

1. **File size** grows with sampling density, not with design intent.
2. **Offset breaks**: offsetting a polyline yields partial/looped/self-intersecting results
   (Patrikalakis & Maekawa, *Shape Interrogation for CAD/M*, Ch. 11); the offset of a line is a line
   and of an arc a concentric arc — clean only if the curve stays analytic.
3. **Faceting** propagates into every downstream solid (extrude, boolean, mass properties, export).

### What every reference kernel does instead (deep dive)

A cross-kernel + literature survey (OCCT, SHAPER, FreeCAD, solvespace source read locally; Parasolid,
ACIS, Inventor from documentation) is unanimous: **a projection is a reference to the source edge plus
an analytically-recomputed 2D curve of the matching type; sampled points are never stored.**

| Kernel/app | Representation of a projected curve |
| --- | --- |
| **OCCT** | `GeomProjLib::ProjectOnPlane(curve, plane, dir, keepParam=false)` → an analytic `Geom_Curve` (type + parameters + trim), by an affine map of the defining data. |
| **SHAPER** | `SketchPlugin_Projection`: an external-edge selection reference + a generated concrete `Line/Circle/Arc/Ellipse/EllipticArc/BSpline` sketch entity; recomputed each rebuild; OCCT-backed. |
| **FreeCAD** | `ExternalGeometry` (link + subelement = the reference) distinct from `ExternalGeo` (the derived **analytic** curve); topological-name stability; `Missing`/`Frozen`/`Detached` flags. |
| **solvespace** | entities are always analytic (circle = centre + radius param, arc = centre + endpoints); the `.slvs` file stores only handles/params — never points. |
| **Inventor / Fusion** | associative **reference geometry** of the same curve type as the source, updated when the model changes; "Project Cut Edges" is the *only* baked, non-associative mode. |

**The projection math** is an affine (parallel) map onto the sketch plane, so every CAD curve class is
closed under it and only the defining data transforms (Piegl & Tiller, *The NURBS Book* — affine
invariance of control points):

- line → **line**
- circle → **circle** when its plane is parallel to the sketch (the common case: including a
  cylinder's rim onto a parallel face), **ellipse** in general (`a = r`, `b = r·cosθ`), line when
  edge-on;
- ellipse / parabola / hyperbola → the **same conic**;
- NURBS/Bézier → **same-degree** curve (project control points; weights unchanged);
- only genuinely free-form or edge-on-degenerate sources fall back to a spline — the same policy as
  Parasolid (`straight`/`ellipse`/`bcurve`) and ACIS (`straight`/`ellipse`/`intcurve`).

## Decision

**Projected sketch geometry is analytic reference geometry, defined by (a) a topological reference to
the source edge and (b) the analytic 2D curve type + parameters obtained by projecting the source's
analytic curve onto the sketch plane. The sampled polyline is never the representation and is never
serialized — it exists only as a transient render/hit-test cache derived from the analytic curve.**

Concretely — and folding in a curve-calculation duplication audit (see "Unify, do not duplicate"):

1. **One canonical analytic projection in `geom`.** Introduce
   `geom.ProjectCurveToPlane(pl Plane, c Curve3) (Curve2, bool)`, dispatching on the existing
   `geom.Kind()` discriminator (not a new fit): `CurveLineSegment`/`CurveLine`→`Line2d`,
   `CurveCircle`→`Circle2d` (parallel) or `EllipseFull2d` (oblique, phase 2), `CurveArc`→`Arc2d`
   (parallel) or `EllipticalArc2d` (phase 2), else `false`. This is the type-preserving projection the
   feature needs, and it **removes the `analytic → polyline → analytic` round-trip** entirely.
2. **The seam carries the analytic `geom.Curve3`.** `sketch.CurveSource` gains
   `SourceCurve() (geom.Curve3, bool)`. This does not violate the M07 seam discipline: that discipline
   keeps the sketch free of the **B-rep topology kernel** (`kernel/topo`, still 0 imports in
   `model/sketch`), NOT of the **geometry kernel** (`kernel/geom`, already imported by 14 sketch
   files). `EdgeRefSource.SourceCurve` returns `edge.Geometry()` directly; the sketch never touches
   topology. `SamplePoints` stays only as the fallback for a source with no single analytic curve
   (a spline, or a multi-edge silhouette loop).
3. **`ProjectedCurve` stores a `geom.Curve2`, not points.** `ProjectedCurve.Update()` calls
   `SourceCurve()` + `geom.ProjectCurveToPlane`; the analytic 2D curve is the representation. The
   sampled polyline becomes a transient render/hit-test cache derived from it. **Delete**
   `fitProjectedShape`, `fitCircleThrough`, `collinearPolyline`, `allOnCircle`, `arcSpan`
   (`projected_shape.go`) — they exist only to undo the seam's information loss.
4. **Persistence stores the reference + the analytic 2D curve type, not `coords`.** The serializer
   writes `source`/`sourceKind` and the compact analytic descriptor (kind + params — a circle is
   centre+radius, ~3 floats vs 34). On load the analytic curve is available immediately and is
   refreshed analytically on the next recompute after the source rebinds
   (`compdef.rebindSketchProjections`). `coords` is removed from new saves; the decoder still reads it
   from legacy files.
5. **Collapse the duplicated samplers.** Replace the three fixed-16 domain-walks
   (`sampleReferenceCurve`, `sampleUseCurve`/`sampleWireCurve`, router `appendUseSamples`) with one
   `geom.SampleCurve3(c Curve3, n int) []Point3`. After (2) this is only the fallback path, but all
   three route through the one helper.
6. **Downstream consumes the analytic curve.** The extrude analytic path recognises a projected
   circle/arc/line (via its `geom.Curve2` `Kind()`), so a projected rim extrudes to a real
   `geom.Cylinder`. Offset, constraints and region tracing read the same analytic form. The extrude's
   existing sketch-entity classifier (`rawSegmentOf`/`circleLoop`) is NOT folded into `geom` — it
   operates on authored `sketch.Entity` and is a distinct, legitimate axis.
7. **Associativity unchanged.** Projected geometry stays driven/reference; `BreakLink` converts it to
   an editable native curve; a future explicit non-associative "snapshot" mode is the only path that
   ever bakes geometry (the "Project Cut Edges" analog).

### Unify, do not duplicate (audit)

A codebase audit found the projection round-trip is one instance of a recurrent pattern: **3 true
copies** of the fixed-16 curve→polyline walk, **3 independent** point-fit circle/line classifiers
(sketch/drawing/sweep), and a canonical `geom.Kind()` discriminator **bypassed** by two concrete-type
switches (`wrapCurve3`/`wrapConic3`). This ADR's change must therefore be a **net reduction** in
duplication — introducing `geom.ProjectCurveToPlane` + `geom.SampleCurve3` while **deleting**
`fitProjectedShape` and consolidating the three samplers. Adjacent cleanups (route the two wrap
switches through `Kind()`; extract a shared `geom.FitCircleFromPoints` for the residual statistical
fitters, keeping each caller's accept/reject policy local) are done opportunistically, not required
for phase 1. Do NOT merge the adaptive tessellation sampler (`kernel/ops/edge_discretize.go`,
`tessellate.go`) — it is a distinct watertight/tolerance contract.

**Persistence policy: reference + compact analytic descriptor (no derived polyline).** This is the
FreeCAD "definition + analytic cache" model with an analytic — not tessellated — cache, so the file
is small *and* the geometry survives a load before the source rebinds. A pure reference-only store
(rebuild entirely on load) is a valid stricter variant; we keep the compact analytic descriptor so a
lost/frozen source still shows its last analytic shape.

## Phased plan

- **Phase 1 (line/arc/circle) — LANDED.** `geom.ProjectCurveToPlane` + `geom.SampleCurve3` (the
  three fixed-16 samplers consolidated); `CurveSource.SourceCurve()` on the single-edge
  `EdgeRefSource`; `ProjectedCurve` carries a `geom.Curve2`; persistence stores the analytic
  descriptor and drops `coords` (legacy `coords` still read); the extrude and offset consume the
  analytic curve. Verified on the piston-head demo: all 146 projected curves recompute analytic,
  document 282 KB → 125 KB (2.26×). **Dedup complete:** offset now reads the analytic `geom.Curve2`
  directly (arrangement already used `RenderPolyline`), so `ProjectedCurve` drops its `shape` field
  and `projectedShape` + the fit-from-points machinery (`fitProjectedShape`, `fitCircleThrough`,
  `allOnCircle`, `collinearPolyline`, `arcSpan`) are **deleted** — one analytic form (the
  `geom.Curve2`) is consumed by extrude, offset, arrangement and serialization, no re-fit, no second
  representation. A non-analytic projection (oblique conic, multi-edge silhouette loop) still
  offsets/renders as a polyline.
- **Phase 2 (oblique conics).** Map an oblique projected circle to the existing `EllipticalArc`
  entity (`entity_kind.go`), so oblique projections are analytic too; extend the extrude/offset paths
  to ellipse segments.
- **Phase 3 (concrete reference entities).** Migrate the `ProjectedCurve` wrapper to concrete
  reference-flagged `Line`/`Circle`/`Arc`/`EllipticalArc`/spline entities (the SHAPER/Inventor model),
  so projected geometry participates uniformly in constraints, offset and region tracing with no
  wrapper special-casing.
- **Phase 4 (free-form).** Project NURBS edges to same-degree sketch splines (control-point
  projection); reserve a sampled fallback only as an explicit non-associative snapshot.

## Consequences

- Files store design intent, not sampling: the piston-head sketch data drops from ~278 KB toward a
  few KB (146 curves × 34 floats → 146 × a handful).
- The persistent #2167 seam on the real part is resolved as a **side effect** of analytic profiles —
  projected rims extrude to real cylinders.
- Offsets, fillets and constraints on projected geometry become exact.
- Existing `.opd` files with `coords` still load (the decoder keeps reading `coords` as the fallback
  path and re-derives the analytic form on recompute); new saves omit `coords`.
- Risk is contained to the projection seam and the extrude analytic dispatch; the boolean/kernel
  layers are untouched.

## Alternatives considered and rejected

- **Keep sampling but raise density / compress `coords`.** Rejected: still faceted, still breaks
  offset, still scales with sampling not intent — treats the symptom.
- **Fit the analytic shape from the sampled points at save time** (today's `fitProjectedShape`).
  Rejected as the *source of truth*: it is lossy, needs the points persisted, and cannot recover an
  oblique ellipse; the source's exact `geom.Curve3` is already in hand and should define the curve.
- **Pure reference-only persistence (store nothing but the source ref, rebuild on load).** Deferred:
  clean, but a document with a deleted/renamed source would show nothing until repaired; the compact
  analytic descriptor keeps a last-known analytic shape at negligible cost.
