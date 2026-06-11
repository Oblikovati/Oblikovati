# Implementation log (historical)

> Migrated verbatim (trademark-scrubbed) from `implementation-plan/PROGRESS.md` when
> roadmap tracking moved to GitHub milestones/issues. Frozen as a historical record —
> current status lives on the issue tracker.


Live tracker for building Oblikovati against [the roadmap](README.md). Updated as
each PBI lands. Status legend: ⬜ not started · 🟦 in progress · ✅ done (green in CI).

> **⚠ Audited 2026-06-04 (see REPORT.md).** The milestone table below
> was reconciled against the actual code. It had been wrong in both directions —
> under-reporting shipped work (M16/M18/M19/M23) and over-stating completion against the
> Definition of Done (M05/M09/M10/M21). Per CONVENTIONS.md "Status model", a feature is
> **Done** only when **Model + Geometry + UI/e2e** are all green; ✅ here now means all
> three, 🟦 means partial. Several rows previously ✅ are model-complete only.

> **▶ NEXT PRIORITY (set 2026-06-09): analytic-cylinder feature faces — [#129](https://github.com/Oblikovati/Oblikovati/issues/129).**
> The revolve/extrude feature builders pre-facet circular profiles, so a feature-modeled "cylinder"
> is a 64-gon prism of planar faces, not an analytic `Cylinder`. Consequence: `thread` can't attach
> (`face is not cylindrical`) and chamfer/fillet on round rims go non-manifold ([#127](https://github.com/Oblikovati/Oblikovati/issues/127)).
> **Fix:** emit analytic `Cylinder`/`Cone`/`Torus` faces when a generating profile edge is a
> circle/arc/axis-parallel line (M07/M20). One change unblocks `thread` + curved-face chamfer/fillet.
> Executable spec pinned by `Oblikovati.AddIns/oblikovati-mcp-bridge/bridge/nopscad_threadedspacer_test.go`
> (goes red the moment it lands). Surfaced by the NopSCADlib re-modelling integration suite.

- **Code root:** `/source` (Go module `oblikovati`, go 1.22).
- **"Done" means:** acceptance criteria green via `make ci` (build + vet + lint +
  `CGO_ENABLED=0 go test ./...` + race), per [CONVENTIONS.md](CONVENTIONS.md).
- **Architecture mapping:** plan milestones map onto the modernized Go packages in
  `../architecture/core/01-module-layout.md`. M00 (COM interop) is deleted in the
  Go design; implementation begins at M01.

## Milestone status

| ID | Milestone | Status | Notes |
|----|-----------|:------:|-------|
| M00 | Platform Foundation & Interop | ✅ n/a | Deleted in Go architecture; replaced by `build/` foundation + ADR-0015. |
| M01 | Math & Transient Geometry | ✅ | All 4 features done: `math/` (linear algebra + boxes) + `kernel/geom/` (curves, surfaces, NURBS, queries). |
| M02 | Units, Parameters & Expressions | ✅ | All 4 features done → `model/param` (units, expression engine, parameter model, dependency graph). |
| M03 | Documents, Persistence & Identity | ✅ | All 6 features done → `model/doc` (Document/Workspace/refs/FileManager), `persistence` (.obk), `model/identity` (reference keys), `model/health`, `model/attr` (attributes + iProperties). |
| M04 | Transactions, Undo & Events | ✅ | All 4 features done → `command` (undo/redo/transactions/checkpoints) + `event` (typed bus) + `model/doc` core event sets & `ChangeManager`. |
| M05 | UI, Commands, Add-ins | 🟦 | **[audit: ✅→🟦 — framework done; DoD UI exists for only 14/44 feature types, see REPORT.md §5]** Headless logic (pure-Go, 93.7%): command + interaction/selection framework, **end-to-end Extrude via synthetic clicks → solid**, ribbon/browser models, add-in platform (**sample add-in → ribbon button → interactive command** = exit criterion), client/preview graphics + null backend. **cgo head DONE** (`source/head`, separate module): vendored Dear ImGui 1.92.8 + GLFW + **Vulkan 1.3** — Go drives the frame loop and ImGui chrome (menu/ribbon/browser from the live Session) + a **3D Vulkan viewport** (offscreen color+depth, lit-triangle/flat-line pipelines) rendering `renderer.DrawList` of the extruded box. Runs with **zero Vulkan validation errors**. ADR-0004/0005 honored. |
| M06 | Sketching & Constraint Solver | ✅ | All 6 features → `model/sketch`: sketches, entities, geometric/dimensional/3D constraints, inference, Newton/LM solver + DOF, profiles & paths. **Headless first (M05/M16 deferred).** |
| M07 | B-Rep Kernel & Topology | ✅ | All 4 features → `kernel/topo` + `math/predicate` + `kernel/ops` (tessellation/validation/Phase-A booleans) + `model/compdef` (PartComponentDefinition). General intersecting booleans + tolerant sew deferred to kernel phase C/D. |
| M08 | Part: Sketched & Work Features | ✅ | All 4 features → `model/feature` (recompute engine, datums/UCS, extrude solid generator + sketched-feature defs, derived/base features). Revolve+sweep/loft/coil generation deferred to kernel phase A/B. |
| M09 | Part: Dress-up & Pattern | 🟦 | **[audit: ✅→🟦 — dress-up geometry landed in M20, but patterns/mirror/combine/face-edits have NO ribbon/dialog UI (U⬜) and boss/split still defer geometry (G⬜); see REPORT.md §5/§6]** All 4 features → `model/feature` (dress-up, hole/boss, patterns/mirror, modify/direct). Combine real (Phase-A booleans); fillet/hole/face-edit geometry deferred (phase B/C) with reference-key input resolution + Warning/Sick health; pattern element bookkeeping real. |
| M10 | Surfacing & Freeform | 🟦 | **[audit: ✅→🟦 — model layer done but ZERO ribbon/dialog UI (U⬜) for any surfacing feature, and several ops defer geometry (extend, ruled-tangent/perp, curved trim/offset = G⬜); see REPORT.md §5/§6]** All 4 features done → `model/feature` (patch/ruled/sculpt/stitch/knit; trim/surface-offset/mid-surface; **sub-D freeform**; **mesh import + mold core/cavity**) + `kernel/ops` (`Stitch`, `TrimByPlane`, `OffsetSurface`, `MidSurfaces`) + new **`kernel/subd`** (Catmull–Clark). |
| M11 | Assembly & Instancing | ⬜ | |
| M12 | Assembly Constraints/Joints | ⬜ | |
| M13 | Sheet Metal | ⬜ | |
| M14 | Drawing & Documentation | ⬜ | |
| M15 | Design Automation | ⬜ | |
| M16 | Visualization & Presentation | 🟦 | **[audit: ⬜→🟦]** Lighting/IBL/shadows shipped: `app/lighting.go`, `renderer/{environment,lighting,lighting_styles}.go`, `View.*` ribbon (HDR/shadows). Presentations/animation not started. |
| M17 | Interoperability & Translation | ⬜ | Only ASCII-STL parse (in `model/feature/mesh.go`). No translator framework / STEP / IGES / DWG. |
| M18 | Analysis & Simulation | 🟦 | **[audit: ⬜→🟦]** `kernel/ops/massprops.go` (mass properties) exists. Measure/interference/FEA/dynamic/tolerance not started. |
| M19 | Materials & Appearances | 🟦 | **[audit: added — was absent from this table]** Real: `model/material`, `app/materials*.go`, head Materials/Appearance/Preferences windows. `_milestone.md` exists but **PBI files to backfill**. Image textures + GGX/IBL out of scope. |
| M20 | Feature Completion & Geometry Parity | 🟦 | Consolidated drive of every remaining the reference platform `*Feature` to real geometry: kernel enablers (intersecting booleans, swept surfaces, fillet, local face ops, body transform) + sheet-metal + plastic + misc model features. |
| M23 | Renderer Display-Mode Parity & Realistic PBR | 🟦 | **[audit: ⬜→🟦]** `renderer/visualstyle.go` + `View.{Realistic,Watercolor,Monochrome,Illustration,TechnicalIllustration,ShadedWithHiddenEdges,WireframeWithHiddenEdges,...}` ribbon commands shipped. Depth of software-PBR/NPR/hidden-line vs full parity unverified. Original scope: Brings the viewport to full the reference platform `DisplayModeEnum` parity (PBIs 300–311): the public `DisplayMode` contract + Visual Style gallery (F01), a **software PBR** Realistic mode (GGX + IBL + tone mapping, no hardware RT — F02), real-time **viewport hidden-line** removal feeding modes 8707/8711/8712 (F03), and an **NPR framework** + the four stylized modes Monochrome/Watercolor/Illustration/Technical-Illustration (F04). See [ADR-0023](../architecture/decisions/ADR-0023-viewport-display-modes.md). |
| M21 | Sketch2D Feature Completion (2D Parity) | 🟦 | **[audit: ✅→🟦 pending verification — model+API+dogfood e2e strong, but full head-UI parity per tool and residual follow-ups (curve-trim, tangent/spline dims) not independently confirmed; see REPORT.md §9]** **All 11 features + every follow-up done** (PBIs 200–221). Full the reference platform 2D-sketch parity (ex-DWG) over `/api`: entities (lines/circles/arcs/points, ellipse, **elliptical arc**, splines incl. **control/fixed/offset/equation curve**, rectangles, polygons, **straight+arc slots**, **fillet/chamfer**, offset, image, **fill region**, **text**, **project geometry**), full geometric constraints (incl. **ground/offset/pattern**) + dimensional constraints (incl. **offset/3-point-angle/ellipse-radius**), constraint-status/DOF + **AutoDimension**, move/rotate/copy/mirror + **trim/extend/split**, rect/circular patterns, profiles. Discriminated `Kind`-keyed wire methods + typed `client.Sketch`; every PBI has model + dogfood e2e tests; new **Modify-panel UI tools** (offset/mirror/fillet) with e2e. lint/vet/**race** green; model cov 81%, router cov 72%. Minor residual follow-ups noted per-PBI (curve-trim, tangent/spline dims). |
| M22 | Sketch3D Feature Completion (3D Parity) | ✅ | **[audit 2026-06-04: F08/F11/F12 residuals closed → feature-complete]** All surface-curve kinds on `/api` (intersection/silhouette/onFace/projectToSurface/offset) with associative rebind verified; `model.referenceKeys` surfacing (F08); head tool-param dialogs + Sketch3DSettings (F12). Bring `Sketch3D` to full the reference platform parity across API+kernel+UI, **including** surface-derived curves (new kernel surface-intersection machinery). 12 features (F01–F12), PBIs 230–247. **Done (model+API+kernel+router, all tested):** F01 spine (`contract.Sketch3D` + wire/client + router + `Sketches3D` in the part + `SketchData3D` round-trip), F02 base entities (point/line/circle/arc), F03 conics + splines (ellipse/elliptical-arc, interpolation/control/fixed splines + equation curve), F04 helical curves (new `kernel/geom.Helix3d`, 100%), F05 geometric constraints (parallel/perp/midpoint/ground/parallel-to-axis+plane), F06 dimensions (distance/line-length/radius/point-plane/two-line-angle + drive), F07 constraint status/DOF/defer, F09 profiles & paths (chain detection by endpoint coincidence; `Profile3D`/`Paths3D` over `sketch3d.profiles`/`paths`). **Kernel: F10 DONE (both PBIs)** — `kernel/geom` surface↔curve intersection + point projection (PBI-243: `ClosestPointOnSurface`/`SignedDistanceToSurface`/`IntersectCurveSurface`) and surface↔surface intersection + silhouette (PBI-244: marching-squares tracer → `IntersectSurfaceSurface`/`Silhouette`); 100% on new files, 98.9% package. F08 edit ops (move/rotate/copy/delete over `sketch3d.transform`) **and Include** (`sketch3d.include` links part edges/vertices by reference key as associative reference geometry via the PointSource/CurveSource seam). F11 surface-derived curves (Intersection/Silhouette/ProjectToSurface/OnFace/OffsetCurve3 over the F10 kernel; **intersection + silhouette over `/api`** via `sketch3d.addSurfaceCurve` resolving part faces by reference key; recompute-derived, serialize-skipped). F12 **app-layer UI** ("3D Sketch" ribbon + edit environment + interactive Line/Point/Circle/Arc/Helix tools + Finish, e2e-tested headless). **Pending (polish):** F08 GetReferenceKey surfacing; F11 associative rebind on recompute (SurfaceSource seam) + project/on-face/offset `/api`; F12 `head/ui` ImGui dialogs + `Sketch3DSettings`. |

## PBI log (most recent first)

### M21 — Sketch2D Feature Completion (2D Parity)

The `model/sketch` layer (M06) is rich but almost entirely unexposed through `/api` (only
`sketch.create` + `sketch.rectangle`) and only partially driveable in the UI. M21 closes
that gap to full the reference platform 2D-sketch parity and fills the remaining model/kernel holes
(elliptical arc, control/fixed/offset/equation splines, slots, polygon, sketch
fillet/chamfer, fill region, text, image; ground/offset/align/pattern constraints; new
dimension kinds; trim/extend/split/move/rotate/copy/mirror; rectangular/circular sketch
patterns; `ConstraintStatus`). Each PBI ships contract+impl+UI+e2e per the DoD. Pattern
follows the `WorkPlanes` slice (discriminated `Kind`-based wire methods + typed client).

| PBI | Title | Status | Go package | Notes |
|-----|-------|:------:|------------|-------|
| 200 | Sketch contract/wire/client spine + enumeration + router | ✅ | `Oblikovati.API/{contract,wire,client}`, `addin/router`, `model/sketch` | F01. `contract.Sketch` (name/visible/entityCount/DOF) + `types.{SketchEntityKind,GeometricConstraintKind,DimensionConstraintKind}` + wire `sketch.list/get/edit/exitEdit/solve/delete/entities/constraints/dimensions` + typed `client.Sketch`. Router handlers map model entities/constraints → kind via type-switch; `var _ contract.Sketch` assertion. Dogfood e2e (create→rectangle→list/get/entities/edit/solve/delete) + client tests. lint/vet/test green. |
| 201 | Sketch properties (name/visible/color/linetype/lineweight/defer) | ✅ | `Oblikovati.API`, `model/sketch`, `addin/router` | F01. `types.SketchLineType`; `sketch.setProperty` + typed client helpers; `SketchData` now persists name + Hidden/Color/LineType/LineWeight/DeferUpdates (fixes name-not-persisted gap). Model round-trip + router setProperty/get e2e green. **F01 complete.** |
| 202 | Lines/circles/arcs/points — API + tools | ✅ | `Oblikovati.API`, `model/sketch`, `addin/router`, `app` | F02. `sketch.addEntity` (Kind+Variant discriminator) + typed client helpers; model `Circles/Arcs.AddByThreePoints` (circumcircle, collinear-rejecting). Dogfood e2e covers line/point/circle(centerRadius,threePoint)/arc(centerStartEnd,threePoint). Existing app Line/Circle/Arc/Point tools satisfy UI DoD. 3-tangent circle / tangent arc are solver-based follow-ups. **F02 complete.** |
| 203 | Ellipse & elliptical arc | ✅ | `model/sketch`, `Oblikovati.API`, `addin/router` | F03. New `EllipticalArc` entity (collection, parametric sampler, serialize round-trip, enumeration); ellipse + ellipticalArc exposed via `addEntity` (axis + unit-bearing radii/angles) + client helpers. Existing app EllipseTool satisfies UI. Model + dogfood tests. |
| 204 | Spline families & equation curves | ✅ | `model/sketch`, `Oblikovati.API`, `addin/router` | F03. Interpolation/control splines + **EquationCurve** (x(t)/y(t) via the M02 expression engine — Bind "t"/Eval, samples a unit circle), **FixedSpline** (immutable points), **OffsetSpline** (parent polyline offset). All serialize + enumerate; model + dogfood tests. **Done.** |
| 205 | Rectangles, slots, polygons | ✅ | `model/sketch`, `Oblikovati.API`, `addin/router` | F04. `composite.go`: AddRectangleByCorners/ByCenter/ByThreePoints, AddPolygon (inscribed/circumscribed), AddStraightSlot (lines + outward arc caps). `addEntity` composite kinds with multi-entity result. Rectangle/slot validate as closed profiles. Model + dogfood tests (3 regions). **Follow-up:** arc slots; auto-constraints after F06. |
| 206 | Sketch fillet & chamfer | ✅ | `model/sketch`, `Oblikovati.API`, `addin/router` | F04. `corner_blend.go`: AddFillet (tangent arc, trims lines) + AddChamfer (bevel), parallel/disjoint rejection; `Sketch.EntityByID`. `addEntity` fillet/chamfer kinds via two line EntityRefs. Model + dogfood tests. |
| 207 | Fill regions & text | ✅ | `model/sketch`, `Oblikovati.API`, `addin/router` | F04. FillRegion (seed→profile) + TextBox (anchor/text/height/justify) annotative entities + **arc slots**; serialize round-trip + dogfood. **Done.** |
| 208 | Project geometry & include | ✅ | `addin/router`, `kernel/topo`, `model/sketch` | F05. `sketch.project` resolves edge/vertex reference keys against the part's SurfaceBodies, adapts topo→CurveSource/PointSource, projects associatively. Dogfood: extrude→project a real body edge→projected curve; unknown ref→unhealthy. **Done.** |
| 209 | Offset sketch entities | ✅ | `model/sketch`, `Oblikovati.API`, `addin/router` | F05. `OffsetEntity` (parallel line / concentric circle-arc, collapse-rejecting) + `sketch.offset` + client. Model + dogfood tests. **Follow-up:** connected-chain offset + offset constraint. |
| 210 | Sketch image | ✅ | `model/sketch`, `Oblikovati.API`, `addin/router` | F05. `SketchImage` entity (ref/anchor/size/rotation/opacity) + serialize round-trip + `sketch.addImage` + client. Model round-trip + dogfood tests. |
| 211 | Expose existing geometric constraints | ✅ | `Oblikovati.API`, `addin/router` | F06. `sketch.addConstraint`/`deleteConstraint`, all ~16 existing kinds via `client.Sketch.Constrain`, ref resolution (PointByID/EntityByID). Dogfood e2e (horizontal-solves-flat, concentric, etc.). |
| 212 | Ground/offset/align/pattern constraints | ✅ | `model/sketch`, `Oblikovati.API`, `addin/router` | F06. GroundConstraint (fix entity), OffsetConstraint (parallel-at-distance), PatternConstraint (seed→member link); solver + serialize + dogfood. Horizontal/Vertical already provide align. **Done.** |
| 213 | Expose linear/angular/radial dimensions | ✅ | `Oblikovati.API`, `addin/router` | F07. `sketch.addDimension`/`driveDimension`, all 5 existing kinds via `client.Sketch.Dimension`, drive/driven/limits. Dogfood e2e (radius drives circle to 2 cm then 3 cm). |
| 214 | Advanced dimension kinds | ✅ | `model/sketch`, `Oblikovati.API`, `addin/router` | F07. OffsetDim (point↔line), ThreePointAngle, EllipseRadius (drives major radius); new DimKinds + serialize + dogfood. TangentDistance/SplineFitPoint remain minor follow-ups. **Done.** |
| 215 | Constraint status, DOF, defer/solve | ✅ | `Oblikovati.API`, `addin/router` | F08. `types.ConstraintStatus`; `sketch.constraintStatus` (non-mutating DOF analysis). DeferUpdates is the F01 property; **AutoDimension** is a follow-up. |
| 216 | Move / rotate / copy | ✅ | `model/sketch`, `Oblikovati.API`, `addin/router` | F09. `edit_ops.go` affine2 + MoveEntities/RotateEntities/CopyEntities; `sketch.transform`. Model + dogfood tests. |
| 217 | Trim / extend / split | ✅ | `model/sketch`, `Oblikovati.API` | F09. SplitLine/TrimLine (remove picked segment between crossings)/ExtendLine via line–line intersection; `sketch.transform` ops + dogfood. Curve (arc/circle) trim is a minor follow-up. **Done.** |
| 218 | Mirror & delete | ✅ | `model/sketch`, `Oblikovati.API`, `addin/router` | F09. `MirrorEntities` (reflection across a line). Model test (mirror across Y) + dogfood. Delete via existing entity handle. |
| 219 | Rectangular sketch pattern | ✅ | `model/sketch`, `Oblikovati.API`, `addin/router` | F10. `RectangularPattern` (count1×count2 grid) + `sketch.addPattern`. Model + dogfood tests. |
| 220 | Circular sketch pattern + pattern constraint | ✅ | `model/sketch`, `Oblikovati.API`, `addin/router` | F10. `CircularPattern` (count over total angle). Model + dogfood tests. **Follow-up:** PatternConstraint binding (212). |
| 221 | Profiles & regions over /api | ✅ | `model/sketch`, `Oblikovati.API`, `addin/router` | F11. `Profile.Area` + `contract.Profile` + `sketch.profiles`. Model (rectangle=12, annulus=64) + dogfood tests. |

### M20 — Feature Completion & Geometry Parity

**Definition of Done (CONVENTIONS.md):** a feature is done only with UI exposure (ribbon
button + property window) **and** end-to-end UI tests, not model+serialize alone. The
rows below marked "model-complete, UI pending" satisfy the model layer but still need
their ribbon command, property window, and e2e UI tests to count as done. **Revolve** is
the first brought fully to the new DoD (ribbon button `R`, `head/ui` property window,
app-layer e2e tests driving command→tool→OK→validated solid).

| PBI | Title | Status | Go package | Notes |
|-----|-------|:------:|------------|-------|
| 175-F03 | Fillet — real rolling-ball blend (cylinder faces) + UI | ✅ | `kernel/ops`, `model/feature`, `app`, `head/ui` | `ops.FilletEdges` (PR #43): per convex straight edge between two planar faces, solves the rolling-ball geometry (centre = corner − r(nA+nB)/(1+nA·nB), tangent points, quarter-arc each end), replaces the edge with a **real cylinder face** tangent to both, retrims the two faces to the tangent lines, arcs each end face's corner. All edges solved on the original body then one rebuild → the box's 4 verticals (sharing side faces) give 4 cylinder faces, vol 7.5708 (the F03 acceptance). New curved-loop assembler `assembleBody` (`assemble_curved.go`) welds faces carrying per-face surfaces (plane/cylinder) + per-edge curves (arcs). Real `FilletFeature`; DoD UI (`FilletTool`, ribbon alias `F`, `head/ui/fillet_dialog.go`, `fillet.svg`). Non-convex edges error; validates the whole curved-B-rep stack end-to-end. Follow-ups: edge chains, fillet-fillet corner blends (sphere/torus), variable radius, concave edges. |
| 177/178-F04 | Local face ops — COMPLETE (shell, move/offset, draft, delete+heal, replace, thicken) | ✅ | `kernel/ops`, `model/feature`, `app`, `head/ui` | Two shared kernel primitives: **`ops.rebuildWithPlanes`** (`retopo.go`, clones topology + swaps face planes + re-solves vertices at 3×3 plane intersections) and **`buildSolidFromLoops`** (`delete_face.go`, welds healed point-ring loops, drops degenerate edges). **Shell** (PR #33): cavity = kept faces offset inward + removed left flush → `solid−cavity` via coplanar boolean. **Move/Offset Face** (#34): `ops.MoveFaces`/`OffsetFaces` (Offset Face UI'd). **Draft** (#35, `ops.DraftFaces`): rotate about hinge at neutral plane ⟂ pull; PullDir default +Z. **Delete Face + heal** (#37, `ops.DeleteFaces`): each deleted-face vertex slides along its two surviving neighbour planes to the nearest other neighbour, coincident weld, loop collapses — chamfer→delete restores the sharp edge. **Replace Face** (#38, `ops.ReplaceFaces`): set selected faces to a target face's plane (2-stage UI). **Thicken** (#39, `ops.Thicken`): sheet→slab (offset faces ±t/2 + side walls on free edges). Each: real geometry + ribbon button + `head/ui` dialog + e2e (DoD). Retrim keys don't survive (new lineage). Follow-ups: vector move-face tool; curved/creased-sheet thicken; multi-face delete heal. |
| curved-brep | Curved B-rep tessellation stack (ParamAt, shared edge discretization, trimmed faces) | ✅ | `kernel/geom`, `kernel/ops` | Kernel prerequisites for a proper fillet & real curved faces. **`Surface.ParamAt`** (PR #30): point→(u,v) inversion, closed-form analytic + numeric NURBS. **Shared curved-edge discretization** (PR #31): an edge's chord polyline is derived only from the edge, so two faces sampling a curved edge match (no T-junction cracks); planar faces with arc edges now follow the arc. **Trimmed curved-face tessellation** (PR #32): meshes a curved face's trim region (boundary→UV via ParamAt) as a structured iso-grid for iso-rectangle trims (fillet/blend faces, cylinder walls) with correct curved area, outward-wound; full-domain grid fallback otherwise. |
| 171-brep | Planar B-rep boolean (sound under chaining) + coplanar faces | ✅ | `kernel/brep`, `kernel/ops` | New **B-rep boolean** (imprint→split→classify→stitch over `kernel/brep`'s 2D arrangement) replaces the BSP CSG as the primary engine in `ops.Boolean` for planar operands (CSG kept as the non-planar fallback). Produces watertight, low-face-count solids and — unlike the triangle soup — stays **exact under chaining** (a boolean's output fed back as input). **Coplanar (ON/ON)** faces handled by set-membership rules (Mantyla §12): flush union/pocket/intersection + chamfer-wedge faces on the box. Also fixed `kernel/ops` planar tessellation to **honor holes** (Eberly bridge + ear-clip) — it ignored inner loops, over-counting divergence-theorem volume on any frame face off the origin plane. Fixes the **multi-edge chamfer** (four chained wedge cuts → exact octagonal prism, V=7). Tests: brep union/diff/intersect/chained/coplanar; ops annulus-area; feature four-edge chamfer. (#28) |
| 100-real | Chamfer feature — real geometry + UI | ✅ | `model/feature`, `app`, `head/ui` | `ChamferFeature` bevels each (convex) edge by cutting a triangular wedge tool along it via the boolean (`chamfer.go`: build all wedges from the original body up front — keys don't survive a cut — then Cut each; `interiorDir` sets back along each adjacent face). Full DoD UI: `ChamferTool` (pick edges, set distance), `Modify.Chamfer` ribbon command, `head/ui/chamfer_dialog.go`, `chamfer.svg`. E2e: `TestChamferToolEndToEnd` (bevel a box edge 0.5 → exact 7.75), `TestChamferViaRibbonCommand`, `TestChamferToolNeedsEdge`; `TestChamferBevelsEdgeForReal` (model). Concave edges (which add material) and distance-angle/two-distance modes are follow-ups. |
| 102-real | Hole feature — real geometry + UI | ✅ | `model/feature`, `app`, `head/ui`, `kernel/ops` | `HoleFeature` drills a faceted cylinder (32-gon) at the placement-face centroid along the inward normal and subtracts it via the boolean (`cylinder_tool.go`: `drillTool`/`regularPolygon`/`planePerp`). Full DoD UI: `HoleTool` (pick a planar face, set Ø/depth), `Modify.Hole` ribbon command (alias `H`), `head/ui/hole_dialog.go`, `hole.svg`. E2e: `TestHoleToolEndToEnd` (drill a Ø2 through hole in a 4×4×2 block → exact remaining volume), `TestHoleViaRibbonCommand`, `TestHoleToolNeedsFace`; `TestHoleDrillsThroughForReal` (model). CSG robustness hardened for faceted tools: `dedupTriangles` cancels coplanar duplicates; T-junction removal loosened+iterated (faceted cylinder cut now welds watertight). Point placement (vs. centroid) and counterbore/countersink profiles are follow-ups; reference keys do not survive a boolean yet. |
| 171 | Face-splitting solid/solid boolean | ✅ | `kernel/ops` | General **intersecting boolean** via a BSP-tree CSG (csg.js-style) over each operand's tessellated triangles, welded back into a watertight B-rep (vertex weld + T-junction split → `cageToBody`). Unblocked by the tessellation-winding fix (#23). Two overlapping 2×2×2 boxes: **Join=12, Cut=4, Intersect=4**; a tool passing through a bar (through-hole) leaves the right volume; cavity cut (contained tool) hollows the solid. Wired into `ops.Boolean` (Join/Cut/Intersect intersecting + cavity cases). This makes overlapping **Combine** and Extrude's **Cut/Join/Intersect** operations produce real subtractive geometry through the existing UI (`TestCombineCutOverlappingForReal`). Curved-face booleans need NURBS intersection (later). |
| 174-UI-sweep | Sweep UI (ribbon + property window + e2e) | ✅ | `app`, `head/ui`, `model/sketch` | `SweepTool` (pick a profile region + a path) + `Create.Sweep` ribbon command + `head/ui/sweep_dialog.go` (output/twist) + `sweep.svg`. New `sketch.Path.Points()` (ordered chain) + app `PathHandle`/`SelectPath` so a picked sketch path resolves to the 3D rail via its sketch plane. E2e: `TestSweepToolEndToEnd` (click profile + path → OK → valid solid V=20), `TestSweepViaRibbonCommand`, `TestSweepToolNeedsProfileAndPath`. **All solid-creation tools (Extrude/Revolve/Coil/Loft/Sweep) now meet the DoD.** Solid commands split into `profileSolidCommands`/`sweptSolidCommands`. |
| 174-UI-loft | Loft UI (ribbon + property window + e2e) | ✅ | `app`, `head/ui` | `LoftTool` (each picked region → a `LoftSection`) + `Create.Loft` ribbon command + `head/ui/loft_dialog.go` (section count / output / closed-loop) + `loft.svg`. E2e: `TestLoftToolEndToEnd` (click two sections → OK → validated frustum V=140/3), `TestLoftViaRibbonCommand`, `TestLoftToolNeedsTwoSections`. New `seqPicker` test helper drives multi-section clicks. |
| 173-UI-coil | Coil UI (ribbon + property window + e2e) | ✅ | `app`, `head/ui` | `CoilTool` + `Create.Coil` ribbon command + `head/ui/coil_dialog.go` property window (output/axis/pitch/revolutions) + `coil.svg`. E2e: `TestCoilToolEndToEnd` (click profile → pitch 2 × 3 revs → OK → valid helix climbing ≈7), `TestCoilViaRibbonCommand`, `TestCoilToolNeedsProfileAndRevolutions`. Solid-feature commands split into `solidFeatureCommands()`. |
| 173-UI | Revolve UI (ribbon + property window + e2e) | ✅ | `app`, `head/ui`, `model/feature` | `RevolveTool` + `Create.Revolve` ribbon command (alias `R`) + `head/ui/revolve_dialog.go` property window (output/axis/angle) + `revolve.svg`. E2e: `TestRevolveToolEndToEnd` (click profile → full revolution → OK → validated 24π washer), `TestRevolveViaCommandAlias` (alias `R` → 90° quarter washer), `TestRevolveToolNeedsProfile`. `WorkGeometry.AxisByRef` added. **Reference for retrofitting the other M20 features' UI.** |
| 174 | Sweep & loft bodies | ✅ | `model/feature` | `SweepFeature` (profile placed along a 3D path, oriented to the local tangent, optional twist) and `LoftFeature` (blend through ordered `(sketch,profile)` sections resampled to a common point count, optional closed) generate real faceted solids via the shared `sweptSolid` primitive. A 2×2 square swept along an L-path is a valid elbow; a 4×4→2×2 square loft is an exact frustum (V=140/3). New `SweepFeatures`/`LoftFeatures` collections + `SweepData`/`LoftData` `.obk` codecs (path as 3D points, sections as sketch+profile indices). `SweepDefinition.Path` is now a `Path3D`; `LoftDefinition` carries `LoftSection{Sketch,ProfileIndex}`. Guide rails, path-frame torsion-minimization and section alignment are follow-ups. **Fix 2026-06-05:** a **twisted** loft/sweep makes warped (non-planar) quad side faces; emitted as single faces they broke the planar boolean (imprint offset → loop won't close → non-manifold). `sweptSolid` now triangulates a non-coplanar side quad (`sideQuad`/`quadPlanar`), restoring the planar-faceted invariant; regression `TestTwistedLoftUnionStaysManifold`. |
| 173 | Revolve & coil surfaces of revolution | ✅ | `model/feature` | `RevolveFeature` and `CoilFeature` now generate **real faceted solids** via a shared `sweptSolid` primitive (cross-section loops → `subd.ToBody` cage → B-rep, re-oriented outward by signed volume). Revolve: full (closed, no caps) or partial (capped) about a `WorkAxis`; a square x∈[2,4]×y∈[0,2] revolved 360° about Y is a validated washer of volume 24π (±1%). Coil: helical placement (pitch/revolutions, taper recorded) capped at both ends; climbs pitch·revs + profile height. New `CoilFeatures` collection + `CoilData` `.obk` codec (axis by WorkRef like revolve). Curved profile edges and exact analytic surfaces are a later refinement; profiles must not touch the axis (pole handling pending). |
| 189 | Decal, Reference, Client, Mark & Finish | ✅ | `model/feature` | Five cosmetic/reference parity features (`DecalFeature`/`ReferenceFeature`/`ClientFeature`/`MarkFeature`/`FinishFeature` + `*Definition` + `CosmeticFeatures` collection). Pass-through recompute (no geometry change) carrying their payload — decal image+face, reference label+source key, add-in id+attributes, mark faces+text, finish faces+spec — all round-tripping through `.obk`. Completes 5 more the reference platform `*Feature` types toward parity. |
| 191 | Pattern & mirror real duplication | ✅ | `model/feature`, `math` | Rectangular/Circular/SketchDriven patterns and Mirror now emit **real placed copies** via `ops.TransformBody` (distinct lineage per copy → independent reference keys), appended as validated solids; per-element suppression drops only that copy. Definitions carry the occurrence geometry (grid `StepX`/`StepY`, axis point+dir, sketch points, mirror plane origin+normal) and round-trip through `.obk`. `math.Reflection4(origin, normal)` (det −1) added for the mirror. A 1×3 pattern of a unit cube → 3 solids at x={0,2,4}; mirror across x=0 → reflected solid in x∈[−1,0]. **Source-only replication DONE 2026-06-05:** `ToolFeature`/`OperationalFeature` + `Input.SourceTool` make a pattern re-apply the source feature's cut/join per occurrence (one body with N holes/blades, not N copies); all boolean tool features incl. Hole carry the contract; a deferred source (Boss) replicates nothing. (See PBI-191 notes.) |
| 190 | Body transform op & Move feature | ✅ | `kernel/geom`, `kernel/ops`, `model/feature`, `math` | `geom.TransformCurve`/`TransformSurface` (similarity-transform dispatchers over the analytic curve/surface types) + `ops.TransformBody` (clones a body's combinatorial topology, maps geometry, reverses winding on reflection so normals stay outward → stays a valid manifold under translate/rotate/reflect/uniform-scale; rejects non-uniform scale). A caller `derive` either preserves reference keys (in-place Move) or makes distinct-key copies (pattern/mirror). `MoveFeature`/`MoveDefinition` relocates a running body in place (keys survive); `math.Matrix4FromCells` persists the transform as 16 cells → `.obk` round-trips. The shared enabler for pattern/mirror geometry (PBI-191). |
| 171 | Face-splitting solid/solid boolean | 🟡 | `kernel/brep`, `kernel/ops` | Planar imprint→split→classify→stitch pipeline works for **clean through-overlaps** (low-face-count, chain-exact, K1a key survival). **Robustness gap (2026-06-05):** partial penetrations / concave faceted-wall crossings leave dangling imprint segments the 2D arrangement won't cut → non-manifold / inverted normals (the lofted-fan deformity). Outstanding work tracked as **PBI-199**; precondition (twisted-loft warped quads) fixed in PBI-174. |
| 199 | Boolean robustness — partial penetration & concave-wall crossing | ⬜ | `kernel/brep` | Planned (XL) — assemble imprint segments into closed loops across face boundaries, handle T-vertices / dangling edges so a tool poking part-way in or crossing a re-entrant faceted wall cuts correctly. Acceptance = the skipped fan e2e (`TestBladeJoinBooleanIsTheDefect`, `TestFanBodyStaysManifold`). Dead-end: post-stitch orientation repair does NOT fix it. |

### M10 — Surfacing & Freeform Modeling

| PBI | Title | Status | Go package | Notes |
|-----|-------|:------:|------------|-------|
| 115 | Mesh features & mesh topology | ✅ | `model/feature` | `MeshGeometry` (welded verts + facets) imported via an **ASCII STL parser** (`ParseSTL`, coincidence-welds shared corners, names malformed tokens). `MeshFeature(s)` wraps it as reference geometry (passes the running solid through) with selectable `MeshFace`/`MeshEdge`/`MeshVertex` handles (facet centroid for measure/selection); `MeshFeatureSet` groups them. A tetra STL → a mesh feature with 4 facets / 4 verts / 6 edges. Binary STL decoder + mesh→B-rep conversion are follow-ups. |
| 116 | Mold core/cavity tooling | ✅ | `model/feature`, `kernel/subd` | `CoreCavityFeature(s)` splits the running tooling block by a **planar parting** (`PartingAxis` X/Y/Z + position) into a **core** (below) and **cavity** (above) solid — built from the block's range box via `subd.Box`/`ToBody`, both validated solids that meet at the parting plane (10³ block parted at z=4 → core z∈[0,4], cavity z∈[4,10]); `Shrinkage` allowance recorded. Sick when the parting falls outside the block. The part-shaped pocket (block − part) and silhouette parting surfaces are general solid–solid booleans (phase C). |
| 113 | Freeform (sub-D) bodies & primitives | ✅ | `kernel/subd`, `model/feature` | New **`kernel/subd`** sub-D kernel: a `Mesh` control cage (verts + polygon faces + per-edge creases), `Box`/`Plane`/`QuadBall` primitives, **Catmull–Clark `Subdivide`/`SubdivideN`** (face/edge/vertex rules with crease + boundary handling), and `ToBody` → a B-rep body (one shared vertex/edge per cage element, planar face per cage face via Newell normal; closed cage → solid, open → surface). `FreeformFeature(s)`/`FreeformBody`/`FreeformFace`·`Edge`·`Vertex`/`FreeformBodies` recompute the limit surface from the cage at the current level. A box primitive → a validated solid; the limit-surface (bicubic) approximation is a NURBS phase. |
| 114 | Freeform edit operations | ✅ | `model/feature` | Cage editing on `FreeformBody`: `MoveVertices`/per-vertex `Move` (selection transforms) and `CreaseEdges`/per-edge `Crease`, plus `SetLevel`. Recompute reflects edits in the B-rep — moving a cage corner grows the body, creasing the three edges at a corner keeps it a **sharp fixed point** while a smooth box rounds inward (verified). `AliasFreeformFeature(s)` wraps an imported Alias sub-D cage (M17) as the same editable feature. Subdivide/bridge/symmetry beyond move+crease+level are the same cage ops, extensible. |
| 111 | Trim & extend surfaces | ✅ | `kernel/ops`, `model/feature` | `ops.TrimByPlane` clips a planar surface body's boundary polygon against a cutting plane (**Sutherland–Hodgman** half-space clip, edge–plane intersections inserted) and rebuilds the kept patch; `TrimFeature(s)` trims the running surface (sick if nothing remains). `ExtendFeature(s)` validates a target surface then defers the edge-to-target geometry (`ErrDeferred`→Warning, phase C). Curved/multi-face trims → `NotYetImplemented`. |
| 112 | Mid-surface & offset | ✅ | `kernel/ops`, `model/feature` | `ops.MidSurfaces` pairs a solid's **antiparallel planar faces** within a thickness threshold (the thin walls), emits a mid-plane patch per pair and records the separation; `MidSurfaceFeature(s)` + `MidSurfaceThickness(es)` extract them from the running solid (a 4×4×1 plate → one mid-surface, thickness 1, for FEA M18). `ops.OffsetSurface` translates a planar patch along its normal; `SurfaceOffsetFeature(s)` offsets the running surface (the M09 direct-edit `FaceOffsetFeature` is a distinct deferred op). Curved offset/mid-surface → phase C. |
| 109 | Boundary patch & ruled surface | ✅ | `model/feature` | `BoundaryPatchFeature(s)`/`Definition`/`BoundaryPatchLoop(s)` fill a closed planar profile (outer loop + inner-loop holes) into a real **surface body** (one trimmed planar face); per-loop `PatchCondition` (Free/Tangent/Curvature = G0/G1/G2) carried — an isolated planar loop satisfies any condition vacuously, the curved-blend-to-adjacent-face case is NURBS phase B. `RuledSurfaceFeature(s)` rules a closed profile by a distance: `RuledNormal` builds the real band (one planar quad per edge, open quilt) along the plane normal; `RuledTangent`/`RuledPerpendicular` resolve inputs then defer (`ErrDeferred`→Warning, need adjacent-face/reference-plane data). Open profile → sick. |
| 110 | Sculpt & knit/stitch | ✅ | `kernel/ops`, `model/feature` | New `ops.Stitch` — **exact-coincidence weld** of independently-built surface bodies: vertices quantized to a tolerance grid become one, shared boundary edges merge, and when every edge ends used by exactly two faces the quilt is **closed → a solid** (deterministic sorted lineage so reference keys are stable; tolerant near-gap matching stays phase-D `Sew`). `StitchFeature(s)` welds the running surface bodies (closed quilt → solid unless `MaintainAsSurface`); `KnitFeatures` = alias. `SculptFeature(s)` requires the bounding surfaces to enclose a volume → fills the solid, else sick. Verified: 6 oriented cube-face surface bodies → one watertight, manifold, validated solid. |

### M05 — UI, Commands, Interaction & Add-in Platform

| PBI | Title | Status | Go package | Notes |
|-----|-------|:------:|------------|-------|
| 056 | Control definitions | ✅ | `app` | `CommandDefinition` (id/name/category/`ControlKind`/alias/tooltip/enable predicate) with a fluent builder. |
| 057 | Command manager | ✅ | `app` | `CommandManager` registry (by id/alias/category, dup-rejecting); `Session.Execute`/`Invoke` (alias) fire `CommandStarted`/`CommandEnded` events; disabled commands refuse. |
| 061 | Select set | ✅ | `app` | `Selection` (ordered, filtered) + typed `Selectable` handles (face/edge/vertex/body/feature/profile/sketch-entity) + `SelectionFilter` (the reference platform SelectionFilterEnum). |
| 062 | Interaction events | ✅ | `app` | `PointerEvent`/`KeyEvent`/`Modifier`; `Session.Click`/`Pointer`/`PressKey` route input per the reference platform mouse/keys (LMB select, plain-click replace / Shift add, alias keys, Esc/Enter). Interactive `Tool` framework (Start/Pick/CanCommit/Commit/Cancel) + OK/Cancel flow. |
| 063 | Hit-test & filters | ✅ | `app` | `Picker` interface (viewport ID-buffer in prod; **stub picker in tests = "click on geometry" headlessly**). **End-to-end `ExtrudeTool`**: synthetic click picks a sketch profile → set distance → OK → real watertight prism in the active part (`compdef` now owns the feature engine + `Recompute`). Real `RayPicker` = camera ray (`scene.Camera.RayThrough`) vs per-face tessellation (`ops.RayCastFaces`, Möller–Trumbore). |
| 058 | Ribbon & UI model | ✅ | `app` | `BuildRibbon` generates the reference platform's two-level ribbon from the command registry (tab → panel → button) each frame with live enabled state; `BuildBrowser` walks the active part into a parameter/sketch/feature tree; `BuildStatus` drives the status-bar prompt + selection count. Pure models that Dear ImGui renders (core/09); a new/add-in command appears as a button (with hover tooltip) under its tab with no UI-code edit. |
| 060 | Add-in platform | ✅ | `app` | `AddIn` interface (ApplicationAddInServer: `ID`/`Activate`/`Deactivate`) + `AddInManager` (register/activate/deactivate, idempotent, dup-rejecting). **Exit criterion test**: a sample add-in registers an interactive command on `Activate` → it shows as a ribbon button → executing it starts the tool that uses selection + preview graphics. |
| 064 | Client graphics | ✅ | `app`/`renderer`/`scene` | `scene.Camera` (eye/target/fov, `RayThrough`); `renderer.DrawList`/`DrawItem` + `BuildDrawList` (per-body surface + wireframe items, front-of-camera cull, object-id tagging); `Backend` interface + `NullBackend` (records frames, no GPU). `Session.AddOverlay`/`Overlays` = persistent client graphics. Metamorphic oracle: translation-invariance of draw counts (ADR-0014). |
| 065 | Interaction graphics | ✅ | `app` | `Previewable` tool capability + `Session.RenderFrame`: assembles active-part bodies + overlays + the active tool's transient preview into one draw list and submits to a backend. `ExtrudeTool.Preview` = live wireframe of the prism (bottom/top loops + vertical connectors) before OK — the reference platform's in-canvas preview, asserted headlessly via the null backend. |

### M09 — Part: Dress-up & Pattern Features

| PBI | Title | Status | Go package | Notes |
|-----|-------|:------:|------------|-------|
| 102 | Hole feature | 🟦 | `model/feature` | `HoleFeature`/`Definition` (placement face key, diameter/depth, `HoleType`, `HoleTapInfo`), `AddDrilled`/`AddTapped`; placement face resolved each recompute, cut geometry deferred (phase C boolean). |
| 103 | Boss feature | 🟦 | `model/feature` | `BossFeature` (placement face + diameter/height); resolve-then-defer. |
| 104 | Feature patterns | ✅ | `model/feature` | `RectangularPatternFeature`/`CircularPatternFeature` + `PatternElement` with **real** parameter-driven element count and per-element suppression (`ActiveCount`/`SetElementSuppressed`); geometry duplication deferred to the M11 body-transform op. |
| 105 | Sketch-driven & mirror | ✅ | `model/feature` | `SketchDrivenPatternFeature` (count = sketch points), `MirrorFeature` (one reflected element); same element model. `PatternFeatures` collection (patterns depend on their source features). |
| 106 | Combine & split | 🟦 | `model/feature` | `CombineFeature` is **real** — booleans two running bodies via `ops.Boolean` (Join/Cut/Intersect), valid for Phase-A cases (disjoint join verified manifold). `SplitFeature` defers (phase C). |
| 107 | Face edits | 🟦 | `model/feature` | `MoveFace`/`FaceOffset`/`DeleteFace`/`ReplaceFace` features: face-key resolution + deferred geometry (phase C). |
| 108 | Thicken & direct-edit | 🟦 | `model/feature` | `ThickenFeature` + the `ModifyFeatures` collection; deferred. |
| 099 | Fillet feature | 🟦 | `model/feature` | `FilletFeature`/`Definition` over picked edge **reference keys**, re-resolved against the running body via the topo rebind (`FindEdgeByKey`); a resolved input defers geometry (`ErrDeferred`→Warning), a lost edge → Sick. Rolling-ball geometry = kernel phase B. |
| 100 | Chamfer feature | 🟦 | `model/feature` | `ChamferFeature` (edge keys + distance); same resolve-then-defer + health. |
| 101 | Shell/draft/thread | 🟦 | `model/feature` | `ShellFeature` (removed faces + thickness), `FaceDraftFeature` (faces + angle), `ThreadFeature` (cylindrical face + designation); face-key resolution + Warning/Sick. `DressUpFeatures` collection. Added engine `ErrDeferred`→`health.Warning` (inputs valid, geometry pending). |

### M08 — Part: Sketched & Work Features

| PBI | Title | Status | Go package | Notes |
|-----|-------|:------:|------------|-------|
| 097 | Derived parts/components | ✅ | `model/feature` | `DerivedPartComponent` pulls a source part's bodies **associatively** (re-reads each recompute; `SourceVersion` tracks change). Source is the consumer-side `BodySource` interface — `compdef.PartComponentDefinition` satisfies it structurally, so feature doesn't import compdef (no cycle). Scale/Mirror recorded; geometric transform deferred to the M11 body-transform op. |
| 098 | Base / imported features | ✅ | `model/feature` | `NonParametricBaseFeature` wraps frozen imported bodies (from translation, M17) as a feature-tree participant, so downstream parametric features consume them. `BaseFeatures`/`DerivedComponents` collections add into the engine. |
| 092 | Extrude feature (full triangle + extents) | ✅ | `model/feature` | `ExtrudeDefinition`/`ExtrudeFeature`/`ExtrudeFeatures`. `AddByDistanceExtent` generates a **real watertight prism** B-rep from a closed profile (bottom/top caps + indexed side walls, lineage on each), validated manifold; recomputes when its driving parameter changes; goes sick on a missing/open profile; combines with running material via `ops.Boolean`. `Extent`/`ExtentType`/`ExtentDirection` enums. |
| 093 | Revolve feature | 🟦 | `model/feature` | `RevolveDefinition`/`RevolveFeature` (profile/axis/angle/op) + `Add`; generation → `NotYetImplemented` (kernel phase A analytic surfaces of revolution). |
| 094 | Sweep feature | 🟦 | `model/feature` | `SweepDefinition`/`SweepFeature` (profile/path/twist) + triangle; generation deferred (phase B NURBS). |
| 095 | Loft feature | 🟦 | `model/feature` | `LoftDefinition`/`LoftFeature` (sections/closed); generation deferred (phase B). |
| 096 | Coil & rib features | 🟦 | `model/feature` | `CoilDefinition`/`CoilFeature` (pitch/revolutions/taper), `RibDefinition`/`RibFeature` (open profile/thickness); generation deferred (phase B). |
| 090 | Work planes/axes/points by relationship | ✅ | `model/feature` | Parametric datums recomputing from a definition closure: `WorkPlanes.AddByPlaneAndOffset` (moves with its driving parameter), `AddByThreePoints`; `WorkAxes.AddByTwoPoints`/`AddByPlaneIntersection`; `WorkPoints.AddByPoint`/`AddByPlaneAndAxisIntersection`. Degenerate definitions → health-sick. Datum planes serve directly as sketch planes. |
| 091 | User coordinate systems | ✅ | `model/feature` | `UserCoordinateSystem` (origin + X/Y/Z triad, `XYPlane` sketch host) + collection; `AddByPlane` aligns a frame to a plane. |
| 087 | Feature-history recompute engine | ✅ | `model/feature` | `PartFeatures` rollback-replay (ADR-0010): `Recompute` finds the earliest dirty feature, reuses the cached body state before it, replays the tail to the EOP marker. `Feature` interface (pure `Recompute(Input)→Output`); inputs are `Ref` reference keys. Editing an early feature recomputes only it + tail (prefix reused, verified by recompute counts); a failing feature goes `health.Sick` and poisons dependents without aborting the rebuild. |
| 088 | Suppression, conditional suppression & health | ✅ | `model/feature` | `SetSuppressed` (passes body state through); `SetSuppressionCondition(param, ComparisonType, threshold)` toggles as the driving parameter crosses the threshold; health states (ok/sick/suppressed) propagate along the dependency edges. |
| 089 | Feature reorder, rename & EOP moves | ✅ | `model/feature` | `Reorder` validates against dependencies (rejects moving before a dependency), re-evaluates the affected range; id-stable `SetName`; `SetEndOfPart`/`RollToEnd` exclude/include trailing features. |

### M07 — B-Rep Kernel & Topology

| PBI | Title | Status | Go package | Notes |
|-----|-------|:------:|------------|-------|
| 085 | PartComponentDefinition container | ✅ | `model/compdef` | `PartComponentDefinition` owns `topo.SurfaceBodies` + `param.Parameters` + `sketch.Sketches`; `RangeBox`/`PreciseRangeBox`/`OrientedMinimumRangeBox`; `ModelGeometryVersion` (changes every edit via `MarkChanged`). Implements `doc.Content` (compdef→doc one-way); wired onto a `PartDocument` via new `doc.Document.SetContent`. |
| 086 | Rollback / End-of-Part state | ✅ | `model/compdef` | EOP marker (`EndOfPartPosition`/`SetEndOfPart`/`RollToEnd`, `IsRolledBack`); moving it bumps the geometry version to request re-evaluation. Evaluate-up-to-marker semantics consumed by the feature engine (M08). |
| 082 | Boolean operations (join/cut/intersect/new-body) | 🟦 | `kernel/ops`, `math/predicate` | `PartFeatureOperation` enum + `Boolean`. **Exact predicates** (`math/predicate`: orient2D/orient3D/incircle, float64 fast path + `big.Rat` exact fallback — the robustness foundation, 100% cov). `PointInsideBody` (ray-cast classification), relationship classifier; **Phase-A** handles disjoint/containment (valid manifold results, key lineage flows through). General intersecting booleans (face splitting) → `NotYetImplemented`, Phase C. |
| 083 | Tessellation & display faceting | ✅ | `kernel/ops` | `Mesh` (positions/normals/indices) + edge polylines. Adaptive chordal-tolerance sampling (recursive midpoint) honored on edges & curved faces; planar faces ear-clipped (exact `Orient2D`) → watertight per face. `TessellateBody`/`Face`/`Edge`, `Quality`. |
| 084 | Geometry healing & validation | 🟦 | `kernel/ops` | `Validate` (manifold = every solid edge used by exactly 2 faces with opposite orientation; closed; reports each offending edge precisely) + `BoundaryEdges`. Tolerant `Sew` (stitching) → `NotYetImplemented` (phase D); the "reported precisely" branch is satisfied now. |
| 081 | Topology evaluators (point/normal/tangent/curvature) | ✅ | `kernel/topo` | `CurveEvaluator`/`EdgeEvaluator` (point, unit tangent, curvature via FD, Simpson arc length, golden-section closest-param/point); `SurfaceEvaluator` (point, normal, partials, grid+projected-Gauss–Newton closest-point); `FaceEvaluator` (planar Newell area, planar point-in-loop containment incl. holes). Outputs match analytic refs within tolerance (circle κ=1/r, length 2πr; segment foot; sphere closest along radius; triangle area 0.5). |
| 079 | B-rep topology: bodies/faces/edges/vertices/loops | ✅ | `kernel/topo` | Full Body→Shell→Face→Loop→EdgeUse→Edge→Vertex graph + adjacency (Edge.Faces, Face.Edges/Vertices, Vertex.Edges, …) via a consistency-preserving `Builder`. Every entity carries a `Lineage` (generative history) → `ReferenceKey()` bytes; `Body.FindFaceByKey`/`FindEdgeByKey` **rebind by lineage after a recompute** (verified against a rebuilt body). Layering: topo is below model/, so it owns lineage/key bytes and does NOT import `model/identity` (the KeyManager binds at the feature layer, M08). |
| 080 | Topology↔geometry binding & containers | ✅ | `kernel/topo` | `Face.Geometry()` → `geom.Surface` (planar face → `geom.Plane`), `Edge.Geometry()` → `geom.Curve3` (straight → `LineSegment`, circular → `geom.Circle`). Per-entity `RangeBox` and `Body.RangeBox`; `SurfaceBodies` container (Add/ByID/Remove); solid/surface + shell organization. |

### M06 — Sketching & Constraint Solver

| PBI | Title | Status | Go package | Notes |
|-----|-------|:------:|------------|-------|
| 077 | Profile & region detection | ✅ | `model/sketch` | `Sketch.Profiles()` walks segment connectivity into closed loops + standalone circles, classifies inner/outer by even–odd nesting (all-vertices containment), groups into `Profile{outer, inner}`. Multi-region; nested hole → outer+inner; open chain → open profile (`IsClosed` lets the feature reject for solids). Construction geometry excluded. `Profile`/`Profiles`/`Loop`/`ProfileEntity`. |
| 078 | Paths for sweeps & lofts | ✅ | `model/sketch` | `Sketch.Paths()` returns every maximal connected chain (open + closed) as `Path` for sweep/loft rails; `Path3D` built from an ordered 3D point chain. |
| 075 | 2D/3D constraint solver core | ✅ | `model/sketch` | Whole-system Newton/Gauss–Newton with Levenberg–Marquardt damping over the residual vector (FD Jacobian), warm-started from current values → an edit re-solves stably; capped iterations, non-convergence → health-sick (no hang/NaN). Dimension-agnostic (`[]*Scalar`), so 2D & 3D solve through the same `Solve`. Pure-Go dense linalg (Gaussian elimination + rank). Decomposition layer deferred behind the same API (ADR-0009). |
| 076 | DOF analysis & over/under-constraint reporting | ✅ | `model/sketch` | `DOFAnalysis` from Jacobian rank: `DOF = vars−rank`, `Redundant = eqs−rank`; `SolveStatus` well/under/over. `Sketch.DegreesOfFreedom`/`AnalyzeConstraints`/`Solve`. Fully-constrained → 0 DOF; redundant constraint flagged (over-constrained); conflicting system → sick. Auto-dimension deferred (heuristic layer). |
| 073 | Dimensional constraints backed by parameters | ✅ | `model/sketch` | `DimensionConstraint` owns a `param.Parameter` (M02 DAG): residual = `measure − param.ModelValue()`. Distance/Radius/Diameter/Angle/ArcLength `Add*` (auto-named model params). Editing the param expression changes the target (drives geometry via solver). Driven mode → no residual/variables, reports `Measured()`; `Sketch.Constraints()` aggregates geometric + driving dims. |
| 074 | Constraint limits & 3D dimensions | ✅ | `model/sketch` | `ConstraintLimits` (min/max) + `Drive(v)` clamps for drive/animation. `DimensionConstraint3D`/`DimensionConstraints3D` (distance) on the same `Constraint` interface for the dimension-agnostic solver. |
| 070 | Geometric constraint set (2D) | ✅ | `model/sketch` | `Constraint` interface (`Residuals []float64` + `Variables []*Scalar` → solver is dimension-agnostic). Coincident/PointOnLine/Midpoint/PointOnCircle/Horizontal/Vertical/Parallel/Perpendicular/Collinear/Concentric/EqualLength/EqualRadius/Tangent/CircularTangent/Symmetry/Smooth/Fix, each verified residual-zero iff satisfied. **Polymorphic over a sealed `CircularCurve` (Circle or Arc):** Concentric/PointOnCircle/EqualRadius/Tangent accept an arc anywhere a circle is accepted, and `CircularTangent` does curve-to-curve tangency (external/internal mode auto-picked from placement). **`SmoothConstraint` (G2)** joins two curves curvature-continuous at an endpoint via orientation-free curvature *vectors* (G0+G1+G2); sealed `SmoothCurve` over Line/Arc/Spline, and — like the reference platform — needs a spline (the adjustable-curvature side). `GeometricConstraints` Add*/Delete/enumerate. (App tools in `app/sketch_constraint_tools.go` expose all of these, incl. Symmetric and Smooth, as tool-first ribbon commands.) |
| 071 | Constraint inference during sketching | ✅ | `model/sketch` | Pure ranked-heuristic `Inference`: `InferSegment` (near-axis → horizontal/vertical w/ priority), `InferSnap` (nearest point → coincident). Apply-on-commit verified. Glyph display deferred (UI/overlay). |
| 072 | 3D sketch constraints | ✅ | `model/sketch` | `Point3D` (3 vars) + `Coincident3D`/`Collinear3D`/`Concentric3D`/`Equal3D`/`CustomConstraint3D`; same `Constraint` interface so the F05 Newton core solves 3D unchanged. |
| 068 | Sketch lines/arcs/circles/ellipses | ✅ | `model/sketch` | Entities on shared constrainable `*Point`s (a shared endpoint *is* a coincidence): `Line`/`Arc`/`Circle`/`Ellipse`/`Point`, construction flag. Typed factory collections `Lines`/`Arcs`/`Circles`/`Ellipses`/`Points` (Add* + Count/Item); `AllPoints` exposes every solver variable. (Names drop the COM `Sketch` prefix per the Go design — `sketch.Line`, not `SketchLine`.) |
| 069 | Sketch splines & blocks | ✅ | `model/sketch` | `Spline` (fit vs control points, closed) via `AddByPoints`/`AddByControlPoints`. `BlockDefinition`/`BlockInstance` + `Blocks`: an instance reads its definition live (transform via `math.Matrix3`), so definition edits reflect in instances. |
| 066 | PlanarSketch/Sketch3D containers & collections | ✅ | `model/sketch` | `Plane` (origin + orthonormal axes, normal = x×y) with `ToModel`/`ToSketch` 2D↔3D mapping (round-trips; off-plane points orthogonally projected) + standard `XYPlane`/`XZPlane`/`YZPlane`. Shared `base` (id/name/edit/visible/health). `Sketch`+`Sketches`, `Sketch3D`+`Sketches3D`, `DrawingSketch`+`DrawingSketches`. |
| 067 | Project geometry & reference into sketch | ✅ | `model/sketch` | Projection via a kernel **seam** (`PointSource`/`CurveSource`, implemented by topo in M07): `ProjectPoint`/`ProjectCurve`/`ProjectCutEdges` create associative reference geometry; `Update` re-projects when the source moves; `BreakLink` freezes it. Verified a projected point/edge follows its source and freezes on break-link. |

### M04 — Transactions, Undo & Events

| PBI | Title | Status | Go package | Notes |
|-----|-------|:------:|------------|-------|
| 052 | Application & Document event sets | ✅ | `model/doc` | Typed events on the `Workspace` bus: `DocumentCreated`/`DocumentOpened`/`DocumentSave`/`DocumentClose`/`DocumentActivate`/`ApplicationQuit` (stable TypeIDs). Lifecycle ops (Add/Open/Save/Close/SetActive/Quit) emit Before (vetoable, `VetoError`) + After. A dirty-close or quit can be vetoed; the document/session stays. |
| 053 | ModelingEvents & ChangeManager/ChangeProcessor | ✅ | `model/doc` | `ModelChanged` event + `ChangeDefinition`/`ChangeKind`. `ChangeManager` subscribes to `ModelChanged` (After) and dispatches to registered `ChangeProcessor`s; `Registration` is the process control (enable/disable/unregister). `Workspace.NotifyModelChanged` emits Before (vetoable) + After. Composes from command+event — no separate framework (core/06). |
| 050 | Object/sink event composition & subscription | ✅ | `event` | Typed bus per core/06 (replaces COM connection points). `Subscribe[E](bus, phase, handler)` + `Emit[E]`/`EmitContext[E]` (generic funcs; methods can't be generic). Dispatch keyed by Go type+phase (reflect); multicast; `Subscription.Cancel`. Handlers may (un)subscribe mid-emit (snapshot). `Event.EventID() TypeID` = wire identity. |
| 051 | Before/after timing & HandlingCode veto | ✅ | `event` | `Phase` (Before/After = EventTimingEnum). `Outcome`/`HandlingCode` (NotHandled/Handled/Abort): `Continue`/`Handle`/`Veto(reason)`; `Emit` aggregates strongest disposition keeping the first veto reason → a Before veto cancels. Typed event structs replace `NameValueMap`; `Context` carries phase + `context.Context` (add-in veto deadline). |
| 048 | Undo/redo stacks with recompute restore | ✅ | `command` | `History.Undo`/`Redo` (done/undone stacks) revert/re-apply a committed `Batch`, restoring model state and firing one coalesced `OnChange` (the recompute hook). Undo→Redo returns to identical state; a new edit clears the redo stack; `UndoLabels`/`RedoLabels` = enumerators; guarded against an open transaction. |
| 049 | Checkpoints (SetCheckPoint/GoToCheckPoint) | ✅ | `command` | `CheckPoint` = remembered history depth + label (no geometry snapshot). `GoToCheckPoint` undoes/redoes to that depth with one coalesced update; unreachable depth errors. `CheckPoints`/`ReleaseCheckPoint` enumerate & dispose. |
| 046 | Transaction lifecycle (start/end/abort, nested, global) | ✅ | `command` | Command pattern per core/06 (not a literal COM TransactionManager). `Command` (Label/Apply/Revert, self-contained → cross-doc batches), `Func` (closure cmd), `Batch` (composite, atomic Apply with prefix-rollback). `History.Begin`→`Transaction.Commit`/`Abort`; nesting folds child batch into parent; mid-transaction `History.Do` joins the open txn. Abort reverts in reverse → exact pre-state. Labels = undo menu. |
| 047 | Transaction merge & suppression of change notifications | ✅ | `command` | `Transaction.MergeWithPrevious` combines two committed steps into one undo. `History.SuppressNotifications`+coalesced `OnChange`: a recording transaction fires exactly one update at Commit (1000-edit batch → 1 update); suppress/resume gates bare edits. |

### M03 — Documents, Persistence & Identity

| PBI | Title | Status | Go package | Notes |
|-----|-------|:------:|------------|-------|
| 044 | AttributeSets/Attributes on any model object | ✅ | `model/attr` | `Value` typed tagged union (bool/int/double/string/bytes — `ValueTypeEnum`, no `any`). `AttributeSet`/`AttributeSets` CRUD. `AttributeManager` anchors sets by `identity.RefKey` (serialized) so a face's attributes survive recompute (equal re-minted key re-anchors) and reload (`Encode`/`DecodeAttributes` = attributes.bin). `FindAttributes` query by set/name. Add-in private data round-trips. |
| 045 | iProperties / PropertySets | ✅ | `model/attr` | `Property`/`PropertySet`/`PropertySets` with the four standard sets (Summary/DocumentSummary/DesignTracking/UserDefined). `EncodeProperties`/`DecodeProperties` persist + queryable. `ExposeParameter(*param.Parameter)` bridge promotes a parameter into a custom property (provenance via `ExposedFromParameter`), surviving round-trip. |
| 042 | ReferenceKeyManager: keys, contexts, bind/can-bind | ✅ | `model/identity` | `RefKey` = opaque, serializable value encoding an entity's **generative lineage** (not an address). `KeyManager`: `CreateKeyContext`/`ReleaseContext`/`RebindSource`, `GetReferenceKey`, `BindKeyToObject`/`CanBindKeyToObject` (→`MatchType` exact/none), `SaveContextToArray`/`LoadContextToArray` (ids preserved), `KeyToString`/`StringToKey`. Built against a topology **seam** (`Entity`/`Lineage`/`EntitySource`) — kernel/topo implements it in M07. Key rebinds after recompute recreates the face; CanBind false when topology vanishes; keys survive save→reopen. |
| 043 | Reference-loss propagation policy | ✅ | `model/identity`, `model/health` | `model/health` = canonical `Status` vocabulary (ok/warning/sick/suppressed, modernizes HealthStatusEnum). `KeyManager.Resolve` is the one place the loss policy lives: a bound entity → healthy; a lost reference → `health.Sick` + `ErrReferenceLost` reason, fixable by re-selection, never fatal. (Consumers = features/dimensions/mates from M08+.) |
| 040 | Document reference graph & descriptors | ✅ | `model/doc` | `RefGraph` (owned by `Workspace`, shared by all docs): `Document.ReferencedDocuments`/`ReferencingDocuments`/`AllReferencedDocuments` (transitive), `AddReference`/`RemoveReference`/`References`. `DocumentDescriptor` (full name, needs-update, reference-key placeholder for F05, broken flag). Lazy resolution: open-set first, else load via store; **broken refs flagged, never fatal**. Drives `referencedBy` for unreferenced-only close. |
| 041 | FileManager, project paths & file locations | ✅ | `model/doc` | `FileManager.Resolve` (workspace-then-library search paths), `Relativize`, `TemplateFile(type)` (`<type>.obk` in templates dir). `DesignProject` (workspace + library roots + `FileLocations`), `DesignProjectManager` (active project). Portable path config — no registry/monikers (core/05). |
| 037 | Structured storage container & streams | ✅ | `persistence` | `.obk` = portable ZIP package (replaces OLE structured storage). `Package` = ordered named byte streams + typed `Manifest`; `OpenPackage`/`Save`. **Atomic save** stage→fsync→`commit` (rename); interrupted save leaves the prior file byte-intact. `StreamStat` (modernizes `tagSTATSTG`). Streams round-trip byte-identical. |
| 038 | DataIO stream I/O & attribute/data persistence | ✅ | `persistence` | `DataIO` bound to a `Package` (`WriteData`/`ReadData`) + file-level `WriteDataToFile`/`ReadDataFromFile` that preserve other streams. Arbitrary client/add-in data persists & reloads. |
| 039 | Compaction & version migration | ✅ | `persistence` | `Compact` drops regenerable `cache/` streams losslessly (smaller archive, recipe intact). `Migrate` pipeline: manifest `schemaVersion`, ordered steps keyed by from-version (`v0→v1` renames `model/params.bin`→`model/parameters.bin`), newer-than-supported rejected. `OnMigrateDocument` hook → M04. `PackageStore` implements `doc.Store`: documents save/reopen as real files. |
| 035 | Documents collection & create-from-template | ✅ | `model/doc` | `Workspace` (modernized `Documents` collection, core/02/05): `Add` create-from-template (empty default content, dirty until saved), `Count`/`LoadedCount`/`VisibleDocuments`/`Documents` snapshot, `ByID`/`ByName`, active document. Visible vs hidden-open. Typed-view helpers `AsPartDocument`/etc. |
| 036 | Open/OpenWithOptions/Save/Close lifecycle | ✅ | `model/doc` | `Open`/`OpenWithOptions` (typed `OpenOptions`, not `NameValueMap`; `DeferContent`→reference stub w/o store hit), `Save`/`SaveAs`, `Close`/`CloseAll(unreferencedOnly, skipSave)` with dirty-on-close handling. Persistence behind an injected `Store` seam (real zip backend = F03); tested via `fakeStore`. Saved doc reopens identically in a fresh workspace. |
| 033 | Document base: identity, dirty, lifecycle | ✅ | `model/doc` | `Document`: session `ID` (not persisted), `FullDocumentName`/`FullFileName`/`DisplayName` (derived), `Dirty`/`MarkDirty`/`ClearDirty`, `Open`/`Compacted`. `NewReference` → reference stub (identity known, content nil, not open). |
| 034 | Document specializations & content exposure | ✅ | `model/doc` | `DocumentType` enum (stable 0–4) + `PartDocument`/`AssemblyDocument`/`DrawingDocument`/`PresentationDocument` embedding the base; each exposes a typed content stub (`*PartComponentDefinition` M07, `*AssemblyComponentDefinition` M11, `*DrawingContent` M14, `*PresentationContent` M16) via the `Content` interface. Type discrimination verified end-to-end. |

### M02 — Units, Parameters & Expressions

| PBI | Title | Status | Go package | Notes |
|-----|-------|:------:|------------|-------|
| 023 | UnitsTypeEnum & quantity model | ✅ | `model/param` | `Unit` enum (stable ids) + dimension signatures; `Quantity` (db units) with dimensional arithmetic. |
| 024 | UnitsOfMeasure: conversion & format | ✅ | `model/param` | Named-unit registry; `Parse`/`Format` at the boundary; switching prefs changes display not storage. |
| 025 | Expression parser & AST | ✅ | `model/param` | Lexer + recursive-descent parser; AST with unit literals, refs, calls; positioned errors; constant folding. |
| 026 | Unit-aware evaluator + functions | ✅ | `model/param` | Dimensional eval; function library (trig/sqrt/min/max/…); refs bound by stable `ID`. `sin(30 deg)=0.5`, `1mm+1deg` errors. |
| 027 | Parameter base (triad) | ✅ | `model/param` | `Parameter`: Expression→Value→ModelValue; SetExpression/SetValue; health (dimensional error → sick). |
| 028 | Parameter types & collections | ✅ | `model/param` | `ParameterKind` (model/user/reference/derived/table); `Parameters` Add*/ByName/ByID/Rename/Delete; read-only enforcement; name uniqueness. |
| 029 | Tolerance, precision, format | ✅ | `model/param` | `Tolerance`+`ModelValueType` (nominal/upper/lower/median); `Precision`/`ParameterDisplayFormat` display-only. |
| 030 | Dependency edges | ✅ | `model/param` | `DrivenBy`/`Dependents` from bound refs; edges by id; rename preserves edges + rewrites dependent display text. |
| 031 | Cycle detection & health | ✅ | `model/param` | Cycle rejected at edit time (`CycleError`, param sick); undefined ref → sick, not panic. |
| 032 | Dirty-propagation recompute | ✅ | `model/param` | Topo recompute of exactly the transitive dependents; independents untouched. |

### M01 — Math & Transient Geometry

| PBI | Title | Status | Go package | Notes |
|-----|-------|:------:|------------|-------|
| 014 | Point/Vector/UnitVector (2D & 3D) | ✅ | `math/` | Immutable value types; ops return new values (not COM mutation). `Point3/Vector3/UnitVector3` + 2D. |
| 015 | Matrix & Matrix2d transforms | ✅ | `math/` | `Matrix4` (3D affine), `Matrix3` (2D affine): compose/invert/determinant/rigid + rotation/CS constructors. |
| 016 | Lines, arcs, circles | ✅ | `kernel/geom/` | Line(2d)/LineSegment(2d)/Circle(2d)/Arc3d/Arc2d + 3-point arc/circle constructors; `Curve2`/`Curve3` eval interfaces. |
| 017 | Ellipses, polylines | ✅ | `kernel/geom/` | EllipseFull(2d)/EllipticalArc(2d)/Polyline(2d); ratio + sweep verified. |
| 018 | Analytic surfaces | ✅ | `kernel/geom/` | Plane/Cylinder/Cone/Sphere/Torus + `Surface` eval interface (Point/Derivatives/Normal/Domains). |
| 019 | NURBS curves & surfaces | ✅ | `kernel/geom/` | BSplineCurve/BSplineSurface; Cox–de Boor + rational quotient-rule derivatives. Loft/sweep round-trip deferred to M10 (no generator yet). |
| 020 | TransientGeometry factory | ✅ | `math/`+`kernel/geom/` | No COM factory (architecture core/03); package surface = construction point. Verified allocation-free via `testing.AllocsPerRun`. PointTolerance = `math.DefaultTolerance`. |
| 021 | Boxes (AABB/OBB) | ✅ | `math/` | `Box`/`Box2d` (extend/contains/intersect/union) + `OrientedBox` (contains/corners/ToAABB) — per core/01 these live in `math/`. |
| 022 | Geometry queries | ✅ | `kernel/geom/` | Closed-form: closest-point/distance (line/segment/plane), line-plane, line-line closest+intersection, 2D line-line. General numeric curve/surface intersection deferred to kernel numeric phase (M06/M07). |

## Session log

- **2026-06-10:** **Browser timeline, edit scope & work-plane redefine (issue #132, PRs #133–#135).**
  The model browser is now one **chronological tree**: a process-wide creation clock (new
  `model/seq`) stamps every sketch, part feature, and user work plane/axis/point (persisted in
  the recipes, clock bumped on load; origin frame stamps 0), so sketches/work features/features
  interleave by creation order instead of a separate Sketches branch. A sketch consumed by
  exactly one feature **nests under that feature** (`SketchConsumer`); a **Shared** sketch (new
  Share/Unshare Sketch menu, the reference platform `PlanarSketch.Shared`) stays top-level. **Double-click
  edits** dispatch by node type (feature → parameter editor, sketch → sketch environment, work
  plane → edit dialog) with **edit-scope visibility**: everything created after the edited node
  hides — the feature engine rolls back via its end-of-part marker and the head overlays skip
  stamps past the edit cutoff — restored on commit/cancel/finish. User work planes are now
  **pickable in the viewport** (`Session.PickableWorkPlanes` feeds the ray picker). **Redefine:**
  the model exposes each plane's `RedefineSlots` (re-pickable references) + `EditableScalars`
  (offset/angle); `WorkPlaneEditTool` arms a slot's pick filter and re-points it live (Cancel
  restores the snapshotted definition); the head dialog renders scalar fields + reference rows.
  Over the wire: **`workPlanes.redefine`** + a self-describing `workPlanes.list` (kind, scalars,
  slots — API #31), **`workPoints.create`** (API #32), and the **`redefine_work_plane`** MCP
  tool (AddIns #23). Full suite + head + SPDX green.
- **2026-06-05:** **Sketch construction lines + centerlines.** Construction geometry already
  existed in the model (entity flag, excluded from `normalGeometry`/profiles, serialized) but had
  no UI. Added **centerlines** (new): `entityBase.centerline` (implies construction, so it never
  closes a profile), `Line.Axis3D(plane)` (model-space axis for revolve/mirror), `Sketch.Centerlines()`,
  and serialization (`EntityData.Centerline`). UI: a Sketch-tab **Format panel** with `Sketch.
  Construction` and `Sketch.Centerline` toggle commands (`Session.ToggleConstruction/ToggleCenterline`
  over the sketch selection) — amended the canonical ribbon doc to add the Format panel it was
  missing. The sketch **Mirror** tool already picks a `*Line` axis, so a centerline drives it with
  no change. Tests: model (centerline excluded from profiles vs. a normal midline that splits;
  axis derivation; mirror across a centerline; round-trip) + app e2e (toggle construction/centerline
  on a selected midline → region count drops 2→1; ribbon Execute). Full suite + head + SPDX green.
- **2026-06-05:** **Revolve centerline picker — the reference platform-accurate selection.** Reworked the axis
  selection to match the reference platform: a separate centerline picker with smart **pre-selection** and
  profile→axis **auto-advance**. `preselectCenterline` (pure, 5 cases tested): one centerline in
  the profile's sketch → pre-select; several in it → user must pick; none in it but one visible
  overall → that one; else none. On profile pick the tool auto-advances (filter → SketchEntity)
  and pre-selects; the user can click any centerline to override. Model now revolves about a
  **specific** centerline (`RevolveDefinition.AxisCenterline` + `AddAboutCenterlineLine`),
  serialized by (sketch,line) index (1-based; empty = profile-sketch auto centerline; a WorkRef =
  explicit work axis). The **ray picker** gained sketch-curve picking (`nearestSketchCurve`,
  reusing `raySegmentDistance`) so a centerline is selectable in the part view. Head: profile-area
  + centerline **hover highlights** (candidate colour) and picked-profile + chosen-centerline
  highlights (selected colour), mirroring Extrude; "About sketch centerline" dialog checkbox.
  Tests: preselect rules, tool pre-select + manual-pick flows, raypicker centerline pick, model
  washer (= explicit-Y 24π), serialize round-trip. Full suite + head + SPDX green. Sketch mirror
  already consumed centerlines; revolve now matches the reference platform's picker UX.
- **2026-06-05:** **Extend (Surface) — real geometry + full DoD (was G⬜).** `ops.ExtendByEdge`
  grows a planar surface by sliding a boundary edge's endpoints outward (in-plane, perpendicular
  to the edge, away from the face). Realigned the model `ExtendFeature` from the unused identity
  `Ref` to a topo `[]byte` edge key (the proven chamfer pattern; re-resolved via FindEdgeByKey each
  recompute) — so the deferred ErrDeferred path is gone. `ExtendTool` (pick a boundary edge +
  distance) + `Surface.Extend` ribbon command. Tested kernel (grow + lost-edge), model (grow + sick
  on lost edge, key survives recompute), app e2e (pick edge → y∈[-2,4]; ribbon). Multi-face and
  curved-surface extend remain phase-C (NURBS). Full suite + head + SPDX green.
- **2026-06-05:** **K5 — multi-face surface ops (Trim + Offset).** Generalized the surface-edit
  kernel beyond single planar faces: `ops.TrimByPlane` now clips EACH planar face by the cutting
  plane and welds the kept faces back into one sheet (folded sheets / stitched quilts trim
  correctly), and `ops.OffsetSurface` offsets a multi-face **coplanar** quilt (translate each face
  + weld). Shared a new `buildSheet` welder (coincident-vertex merge, one edge per pair); migrated
  MidSurfaces to it; replaced the old single-face `buildPlanarBody`. The model Trim feature and
  `SurfaceTrimTool` get multi-face for free. Tested (multi-face trim → 2 faces, x∈[1,4]; coplanar
  offset → 2 faces flat). **Honest scope:** this completes the MULTI-FACE half of K5; the CURVED
  half — NURBS surface–surface trim and parallel-surface offset of curved/folded faces — is
  genuine NURBS work that remains (it needs a curved surface-trim engine, not a planar clip).
  Full suite + head + SPDX green.
- **2026-06-05:** **Decal — UI completion; Create panel now fully populated.** `DecalTool` places
  an image (a resource id) on a picked face — cosmetic, no solid change. `Create.Decal` ribbon
  command (the last canonical Create-panel button). `DecalFeature` already existed; adds the UI +
  e2e (pick face → set image → decal recorded, body unchanged; ribbon Execute). **The canonical
  3D-Model Create panel [Extrude, Revolve, Sweep, Loft, Coil, Rib, Emboss, Decal] is now all
  wired.** Full suite + head + SPDX green.
- **2026-06-05:** **Sculpt (Surface panel) — UI completion (geometry was already real).**
  `SculptTool` fills the volume bounded by the running surface bodies into a solid (parameter-only:
  closing tolerance). `Surface.Sculpt` ribbon command. `SculptFeature` already existed; adds the
  UI + e2e (six cube-face sheets → unit-volume solid; ribbon Execute). The output operation
  (Join/Cut vs new body) is stored but geometry-unwired in the model, so the tool defaults to a
  new body and does not expose it (would mislead) — wiring it is a model refinement. Full suite +
  head + SPDX green. **Surface panel UI now done for Patch/Trim/Stitch/Sculpt; the remaining
  Surface tools (Extend, Rule Fillet) need kernel geometry (K5), not just UI.**
- **2026-06-05:** **Stitch (Surface panel) — UI completion (geometry was already real).**
  `StitchTool` welds the running surface bodies into one quilt (closed → solid unless "keep as
  surface"); parameter-only (tolerance + keep-surface), no pick. `Surface.Stitch` ribbon command.
  `StitchFeature`/`ops.Stitch` already existed; adds the UI + e2e (two adjacent patches → one
  2-face quilt; ribbon Execute). Full suite + head + SPDX green. Tolerant (gap-band) sew is the
  geometry refinement (K3); Extend/Sculpt/Rule-Fillet are the remaining Surface tools.
- **2026-06-05:** **Surface Trim — UI completion (geometry was already real).** `SurfaceTrimTool`
  cuts the running surface body with a picked work plane, keeping one side (ParameterizedTool: a
  "keep positive side" bool). `Surface.Trim` ribbon command (Surface panel). `TrimFeature` +
  `ops.TrimByPlane` geometry/serialize already existed; adds the UI + e2e (patch a 4×4 surface,
  trim by an x=2 work plane → x∈[2,4]; ribbon Execute). Full suite + head + SPDX green. Curved/
  multi-face trim (K5) is the geometry refinement; Extend/Stitch/Sculpt are the next Surface tools.
- **2026-06-05:** **Patch (Surface panel) — UI completion (geometry was already real).** First of
  the Surface-panel tools: `PatchTool` fills a closed sketch region with a boundary-patch surface,
  auto-committing on the region pick (one-click surfacing). `Surface.Patch` ribbon command (the
  canonical Surface panel, finally created in the 3D Model tab). `BoundaryPatchFeature` geometry +
  serialization already existed; this adds the missing UI + e2e (surface sheet body, 1 face, not a
  solid; ribbon Execute). Full suite + head + SPDX green. Next Surface tools: Trim, Extend, Stitch.
- **2026-06-05:** **Emboss — new Create feature, full DoD.** `EmbossFeature` raises (Join) or
  engraves (Cut) a closed sketch profile on the part: the region is extruded a shallow depth along
  the sketch-plane normal. Built on the shared `buildPrism`+`combine` machinery — extracted two
  package helpers (`resolveClosedProfiles`, `buildProfilePrisms`) now shared by Extrude and Emboss
  (refactor of the touched extrude code; its tests confirm no regression). `EmbossFeatures.Add`,
  serialize round-trip (`EmbossData`), `EmbossTool` (ParameterizedTool: Depth + Engrave, multi-
  region pick) + `Create.Emboss` ribbon (canonical Create panel; generic dialog, no bespoke cgo).
  Tests: model (raise → +vol, engrave → −vol, error), serialize round-trip, app e2e (raise/engrave/
  ribbon). Full suite + head + SPDX green. Wrapping onto a curved face is the remaining refinement.
- **2026-06-05:** **Split Solid / Trim Solid — new feature, full DoD (was G⬜).** `ops.SplitSolidByPlane`
  divides a solid by an infinite plane into the pieces on each side (intersect with a large
  half-space box per side, so each piece is capped by a clean planar cross-section). New
  `SplitSolidFeature` (`AddSplitSolid(plane, keep)`, `SplitBoth`/`Positive`/`Negative` for split
  vs trim) referencing a cutting **work plane**; serialize round-trip (`SplitSolidData`). UI:
  `SplitTool` (pick a work plane, choose split/trim) + `Modify.Split` ribbon command (canonical
  Modify panel) + head dialog. Tests at kernel (mid/oblique/miss, valid solids, volumes sum),
  model (divide + trim + round-trip), app e2e (pick plane → OK → 2 bodies; trim → 1; ribbon).
  Two bugs found & fixed building the half-space box WITHOUT subd (subd is unreachable from ops —
  `subd_test`→`ops` would cycle): edges must be stored canonically (min→max) to match the
  Reversed convention, and the negative-side box must swap U/V to stay right-handed (else the
  canonical rings wind inward → inside-out box → broken boolean). Full suite + head + SPDX green.
- **2026-06-04:** **Rib feature completed to full DoD (UI + persistence; geometry was already real).**
  Rib had real geometry (`RibFeature.Recompute` thickens an open profile into a wall) but no
  collection method, no serialization, and no UI — so it wasn't DoD-complete. Added: `RibFeatures.
  Add` (named, engine-bound), `RibData` + serialize/restore wired into the recipe registry, an
  interactive **`RibTool`** (a `ParameterizedTool` — resolves the open profile from the active or
  most-recent open-path sketch at Start, exposes Thickness + Depth, joins the wall to the part),
  and a **Create-panel ribbon command** `Create.Rib` (matching the canonical the reference platform Create panel
  [Extrude…Coil, **Rib**, Emboss, Decal]; the head renders the generic property dialog, no bespoke
  cgo). Tests: model collection (named + vol), serialize round-trip, app e2e (tool flow + ribbon
  Execute). Full suite + head + SPDX green. Rib's "to-next" part-bounding remains the only
  refinement. **This closes the last Phase-3 UI gap for a real-geometry Create feature.**
- **2026-06-04:** **Conical drill point — Hole feature reaches the reference platform parity (step 4/4 DONE).**
  `brep.CutBlindConicalHole`: a cylinder bore closed by a true CONE tip (the twist-drill shape;
  118° included = 59° half-angle default). Needed a new tessellation primitive: **cone-apex
  (pole) fan** — `ops.coneApexFan` meshes a cone whose only boundary is a single rim circle by
  fanning to the apex pole (a geometric tip, not a topology vertex). It's tried before the band
  path and guarded to fire only for a single-circle (constant-v) cone boundary, so a countersink
  frustum (two rim circles) is still gridded as a band. Subtlety found & fixed: a lone full-
  circle loop de-duplicates to a span just under 2π, so `toUVLoops` wrongly succeeded into a
  degenerate iso-rectangle — the apex fan now runs first. Wired into the Hole feature via
  `HoleDefinition.PointAngle` (0 = flat); the HoleTool defaults to 118° (the reference platform's default), the
  head dialog exposes the angle (0 = flat). Full DoD: kernel (1 cone + 1 cylinder, volume =
  cylinder + ⅓cone, apex-exits rejection), ops (apex-fan area = π·r²/sin), model, app e2e, and
  serialize round-trip. Full suite + head + SPDX green. **Hole feature now has all four profiles
  (drilled flat/point, through, blind, counterbore, countersink) with real curved geometry.**
- **2026-06-04:** **Cone-face support + countersink holes (Hole parity steps 2-3/4).** Confirmed
  the periodic-band tessellator handles a CONE frustum (varying radius) — `ops` test, lateral
  area π(r1+r2)·slant — the foundation. Then `brep.CutCountersinkHole`: a true cone frustum
  recess widening to the sink Ø at the surface, sharing a transition circle directly with the
  cylinder bore wall (no shoulder), built in one assembly. The cone widens toward the surface
  (axis points back out, apex deep). Wired into the Hole feature as **exact-only** (a wrong
  faceted cone is worse than a clear error for the rare unsupported shape): `HoleDefinition`
  gained `CounterAngle` (included angle) reusing `CounterDiameter` for the sink Ø;
  `AddCountersink`. Full DoD: Hole UI countersink toggle (mutually exclusive with counterbore,
  sink Ø + angle in degrees, head dialog), persistence (`HoleData.CounterAngle`), and tests at
  kernel (1 cone + 1 cylinder wall, volume = frustum + bore), model, app e2e, and serialize
  round-trip. Full suite + head + SPDX green. **Remaining (step 4):** the 118° conical drill
  point — needs cone-APEX (pole) tessellation, distinct from the band case.
- **2026-06-04:** **Counterbore holes — exact stepped geometry, full DoD (Hole parity step 1/4).**
  `brep.CutCounterboreHole`: a shallow recess (counterRadius × counterDepth) stepping via a flat
  **annular shoulder** to a bore (through or blind), built in ONE assembly from the planar slab —
  two true cylinder walls + the shoulder. Built directly (not by chaining two curved cuts, which
  would feed a curved body to the planar-only boolean and produce a non-manifold mess — that's
  the curved-input boolean, a later slice). `HoleFeature.cutCounterbore` prefers it, falling back
  to sequential planar prism cuts (which stay planar, so they chain) for unsupported shapes. Full
  DoD: model (`HoleDefinition.CounterDiameter/Depth` + `AddCounterbore`), Hole UI (counterbore
  toggle + recess Ø/depth, head dialog fields), persistence (`HoleData.Counter*`), and tests at
  kernel (through+blind, 2 cyl faces, volume), model, app e2e, and serialize round-trip levels.
  Full suite + head + SPDX green. **Next (Hole parity 2-4):** cone-face support → countersink →
  118° conical drill point.
- **2026-06-04:** **Blind holes now drill an EXACT cylinder wall + flat bottom (K1b).**
  `brep.CutBlindCylindricalHole`: a cylinder entering one planar face and stopping at depth
  inside the material → entry face holed + true cylinder wall + a flat circular bottom disk
  (facing the opening), watertight, vol = part − inscribed bore. Uses the same face-sense
  primitive (reversed wall). Verified the pocket stays inside via `insideSolid` (else error).
  `HoleFeature.drill` now prefers it for any blind hole, falling back to the faceted boolean
  only for unsupported shapes (bore clips a face, depth exits) — so existing depth≥thickness
  "through via depth" tests keep the faceted path unchanged. Tested at kernel (validity/
  watertight/face-count 1 cyl + 7 planar/volume, through-depth rejection) and model (blind
  depth<thickness → 1 cylinder face + flat bottom) levels. Full suite + head + SPDX green.
  Follow-up: conical drill point (the reference platform's default 118° tip).
- **2026-06-04:** **Hole feature now drills a TRUE cylinder wall (K1b consumed end-to-end).**
  Wired `brep.CutCylindricalHole` into the actual Hole feature as a full vertical slice:
  `HoleDefinition.ThroughAll` + `HoleFeatures.AddDrilledThrough`; `HoleFeature.drill` routes a
  Through-All hole on a planar slab to the exact curved boolean (one cylinder face, not a
  32-gon prism), falling back to the faceted boolean (drilled `throughDepth`) for blind holes
  and unsupported shapes. UI: `HoleTool` gained a Through-All option (depth not required when
  set) and the head hole dialog a "Through All" checkbox (disables the depth field). Persisted
  (`HoleData.ThroughAll`, round-trips). Tested across all layers — model (through hole → 1
  cylinder face, vol = slab − π·r²·h), app e2e (pick face → tick Through All → OK → true wall),
  serialize round-trip. Full headless suite + head build + SPDX green. Blind exact cylinders
  and the general (non-slab) curved boolean remain later K1b slices.
- **2026-06-04:** **K1b slice 3 — first end-to-end curved boolean: a clean drilled hole.**
  `brep.CutCylindricalHole(slab, base, axis, radius)` drills a cylindrical through-hole in a
  planar-faced slab: the two pierced faces gain a circular hole, a single TRUE cylinder face
  forms the wall, and the result is a watertight solid with the correct volume (slab − inscribed
  bore). This needed a new B-rep primitive — **face sense**: `topo.Face.Reversed` +
  `topo.Builder.AddReversedFace`, honored by `ops.TessellateFace` (negates the mesh normals and
  flips triangle winding). A Difference cut wall's surface normal points INTO the removed
  material, so its face is "reversed"; the planar boolean fakes this by building a flipped
  `Plane`, but a cylinder's normal is intrinsically outward-radial and can't be flipped — the
  sense flag is the general fix (the planar path can adopt it later). Because copied + pierced
  faces keep their source lineage, **every original face's reference key survives the curved
  cut** (extends K1a to curved booleans). Partial/blind holes error (later slices). Tested at
  topo (sense flag), ops (reversed mesh flip), and brep (validity/watertight/face-count, volume
  = slab − π·r²·h, key survival, oversize rejection) levels. Full headless suite + head build
  green. **Next:** slice 4 — sphere/cone + the general arrangement-based curved boolean.
- **2026-06-04:** **K1b slices 2 + 2b — first analytic curved solid, correctly tessellated.**
  Confirmed the topo model already supports analytic curved faces: `topo.AddEdge` allows
  **closed-circle edges** (start==end) and `ops.Validate` counts edge *uses*, so a **periodic
  cylinder side face** (seam edge with two opposite uses) is a valid manifold solid. Built
  `brep.SolidCylinder` — one true cylinder face + two planar caps sharing two closed circles +
  a seam. It validates AND is watertight. **Closed the tessellation gap (2b):** a full-2π side
  loop can't be unwrapped, so it used to fall back to the surface's whole *unbounded* UV domain
  → wrong area/volume (183.7 vs 62.8). New `ops.periodicBandGrid` grids a full-seam-wrap face
  over its true trim — the whole period (reusing the boundary's own circle-edge samples so it
  stays watertight with the caps) × the boundary's bounded range. Now face area = 2π·r·h and
  solid volume = π·r²·h (inscribed, a hair under). Tested at ops + brep levels; ops/brep suites
  green. **Next:** slice 3 — wire `SolidCylinder` as the cut tool → a clean drilled hole.
- **2026-06-04:** **K1b started — curved-face boolean, slice 1: exact analytic surface
  intersections.** New [ADR-0027](../architecture/decisions/ADR-0027-curved-face-boolean.md)
  lays out the curved-boolean architecture (generalize the planar pipeline's face model to
  `geom.Surface` + `geom.Curve3` loops; exact-curve imprints; param-space split; reuse K1a
  lineage) and a 5-slice plan. Slice 1 landed: `geom.IntersectSurfacesAnalytic` returns
  EXACT intersection curves for the pairs the boolean needs — plane∩plane → line, plane∩
  cylinder → circle (⟂) / **ellipse** (oblique), plane∩sphere → circle, plane∩**cone** →
  circle (⟂) — and reports `handled=false` for line-pair / curved-curved pairs so the
  caller falls back to the numeric tracer. 10 tests (radius/center exactness incl. the
  oblique ellipse minor=r / major=r√2, miss/parallel/tangent → no curve, order-
  independence, parallel-cylinder defers). Core suite green. **Next slices:** curved face
  model → plane∩cylinder boolean end-to-end (clean hole) → edge-key survival + retire BSP.
- **2026-06-04:** **K1a unlock realized — boolean-output references proven at the feature
  level.** `TestCombineFaceKeySurvivesIntoResultAndRecompute`: a face picked on box A keeps
  its reference key after Combining with an overlapping box B (planar boolean carries the
  lineage) AND after a height-driven recompute — so a downstream fillet/dimension/sketch on
  a combined solid stays bound across edits. Reviewed the rest: the deferred boolean
  features need MORE than K1a — **Split** needs a definition redesign (its `faceKeys` shape
  doesn't model split-by-plane), and **Hole/Fillet** survival + exact chaining need the XL
  curved-face boolean (**K1b**). So K1a's unlock = parametric robustness on the planar
  Combine/Cut path, now validated end-to-end.
- **2026-06-04:** **K1a — face reference-key survival through the planar boolean (G-09).**
  The planar B-rep boolean re-synthesized lineage, so picks on Combine/Cut output were lost
  on edit. Now the source face's lineage is threaded `planarFace → subFace → builtFace →
  AddFace`: a face surviving whole keeps its **exact** reference key; a face split into
  several pieces gets distinct child tokens (`…/brep:split#k`). Tested: Union and Cut both
  keep an untouched face's key; disjoint union still valid (vol 16); no boolean-correctness
  regressions. **Remaining K1:** edge/vertex key survival; the XL curved-face boolean (K1b)
  replacing the BSP-CSG fallback so Hole/Fillet faces survive + chain exactly.
- **2026-06-04:** **Phase 4 start — Rib geometry made real (was NotYetImplemented, G-01).**
  `RibFeature.Recompute` now builds a wall: the open sketch path is offset ±Thickness/2 into
  a closed band (`thickenPath` + vertex-normal averaging + CCW orient) and extruded by a
  finite `Depth` along the plane normal via the shared `buildPrism`, then `combine`d with the
  running body. Added `Depth` to `RibDefinition`; removed the NYI. Tests: straight rib vol=8
  (4×1×2), L-path → one solid, needs-depth→sick. ("To-next" bounding + a ribbon tool are
  follow-ups; core suite + vet green.)
- **2026-06-04:** **Phase 3 start — feature Pattern & Mirror UI (closes finding U-01).**
  The flagship gap: rectangular/circular/sketch-driven patterns + mirror had real geometry
  (M20-191) but no ribbon tool. Added `app/feature_pattern_tools.go` — `FeatureRectPattern`/
  `FeatureCircPattern`/`FeatureMirror` tools (select source features as `FeatureHandle`
  picks, set params, commit via `feature.NewPatternFeatures(part.Features()).Add*` +
  Recompute) + a **Pattern panel** in the 3D Model tab (canonical placement) + `Params()`
  so they render in the generic property dialog. e2e: extrude→select feature→3×1 rect
  pattern = 3 bodies; mirror = 2; circular > 1; needs-source guard; registration. Core
  suite + head + vet green. (Sketch-driven pattern tool = follow-up.)
- **2026-06-04:** **Phase 3 cont. — Combine / Move Face / Move Bodies UI (closes U-02).**
  `app/feature_modify_tools.go` — `CombineTool` (pick two bodies + Join/Cut/Intersect),
  `MoveFaceTool` (translate picked faces), `MoveBodyTool` (relocate a body); all on the
  3D Model **Modify** panel via `directEditCommands()` + `Params()` dialogs. e2e: combine
  join 2→1 body; move-face +Z grows volume; move-body shifts min.X by 10; registration.
  Core suite + head + vet green. **Phase 3 UI-only features done** (patterns/mirror/combine/
  move-face/move); remaining: Sketch-Driven pattern tool + Phase 4 deferred-geometry
  features (Rib/Split/Emboss/Thread).
- **2026-06-04:** **Phase 2 (Sketch3D) start — onFace surface curve over /api (M22-F11).**
  The router exposed only intersection/silhouette surface curves; added **onFace** (a curve
  in a referenced face's parameter space) end-to-end: `api/wire` UV field, new model
  `Sketch3D.AddOnFaceCurve3DRef` (associative over a `SurfaceSource`), router dispatch with
  a flat-UV→points parser (request-shape validated before ref-resolve), and typed client
  `AddOnFaceCurve`. Dogfood router test (extrude→onFace on a face by ref key→healthy; lost
  ref→unhealthy; bad UV→error) + client + model tests. **Then projectToSurface + offset**:
  new `Sketch3D.SourceCurve3` resolver (sketch entity id → `geom.Curve3` for line/circle/
  arc/ellipse/elliptical-arc/helix) + `AddProjectToSurfaceCurve3DRef` (associative surface)
  + wire `SourceEntityID`/`OffsetDistance`/`Normal` + router project/offset cases + client
  `AddProjectToSurfaceCurve`/`AddOffsetCurve`. Dogfood: extrude→add a 3D line→project it on
  a face + offset it in-plane (both healthy); unknown source id→error. **All five surface-
  curve kinds (intersection/silhouette/onFace/projectToSurface/offset) are now on /api —
  the F11 /api gap is closed.** api module + core suite + vet + SPDX green.
  **F11 associative rebind verified:** `TestSketch3DSurfaceCurveRebindsOnRecompute` — an
  onFace curve bound to the top face by reference key follows it from z=5cm→8cm when the
  extrude is grown and the part recomputes (the `FaceRefSource` re-resolves the regenerated
  surface via `FindFaceByKey`). **M22-F11 is effectively complete** (all surface-curve
  kinds + associativity); remaining Sketch3D residuals: F08 GetReferenceKey surfacing, F12
  head dialogs/Sketch3DSettings.
- **2026-06-04:** **Sketch3D F08 — reference-key surfacing (`model.referenceKeys`).** There
  was no way over /api to *obtain* a part face/edge/vertex reference key (handlers only
  *consumed* them), so the include/surface-curve/project workflow was unusable headlessly.
  Added `model.referenceKeys` (wire `TopologyRef`/`BodyTopology`/`ReferenceKeysResult` +
  router enumerating the part's bodies → faces/edges/vertices with key + representative
  point + typed client `Model.ReferenceKeys`). Round-trip test: extrude→referenceKeys
  returns a box's exact 6/12/8 topology with keys→a surfaced face key is consumable by
  addSurfaceCurve. **M22-F08 done.** Remaining Sketch3D residual: F12 head dialogs +
  Sketch3DSettings. api module + core suite + vet + SPDX green.
- **2026-06-04:** **Sketch3D F12 — head dialogs + Sketch3DSettings → Phase 2 complete.**
  Extended the generic tool-param framework with `BoolParam` (checkbox) and gave the
  parameterized 3D tools `Params()`: Circle3D (radius), Helix3D (radius/pitch/turns +
  clockwise), Arc3D (counter-clockwise) — so they render through the same
  `head/ui/tool_params_dialog.go` (no bespoke cgo per tool). Added a **`Sketch3DSettings`**
  head window (visible / show-dimensions / defer-updates over the active sketch's
  accessors). Param labels + float/bool wiring tested headlessly; head builds + ui tests
  green; core suite green. **M22 (Sketch3D) feature-complete (F01–F12) — PARTDOC-PLAN Phase
  2 done.** (Head in-window e2e for the 3D dialogs runs under the new CI head job.)
- **2026-06-04:** **Stretch tool + generic tool-parameter dialog (head DoD layer).**
  (a) **Stretch** — new model `MovePoints` (moves only the listed vertices, deforming
  shared geometry) + `SketchStretchTool` (picks vertices, sets a vector) → the last Sketch
  Modify button; **the 2D Sketch tab is now at full button parity** with the canonical
  ribbon. (b) **Generic parameter dialog** — an `app.ParameterizedTool` interface
  (`tool_params.go`: Float/Int/Text param descriptors with get/set closures bound to the
  tool's fields) implemented by all 11 param-bearing sketch tools (Move/Copy/Rotate/Scale/
  Stretch, Rect/Circ pattern, Slot/Chamfer/Text/Fillet/Offset), rendered by ONE cgo dialog
  (`head/ui/tool_params_dialog.go`) instead of 11 bespoke ones (core/09 reflection-driven
  editing). Param wiring + labels tested headlessly; head builds + ui tests green; angles
  surfaced in degrees. Core suite green.
- **2026-06-04:** **Sketch Create tools completed — Slot, Chamfer, Text.** Added the
  remaining Create-panel tools over existing model ops (`AddStraightSlot`/`AddChamfer`/
  `TextBoxes.Add`): Slot (two centre clicks + width, auto-commits), Chamfer (two-line
  bevel, sibling of Fillet), Text (anchor click + string, commits on OK). App e2e for each
  + registration; Create-panel parity test bumped to 12 buttons. **The 2D Sketch tab is now
  at button parity with the canonical ribbon** except Modify **Stretch** (deferred:
  vertex-window selection). Core suite + head green.
- **2026-06-04:** **Sketch Modify/Pattern tools built + commands refactored.** Added the
  missing Sketch tools: **Move, Copy, Rotate, Scale** (Modify panel) and **Rectangular,
  Circular** patterns (Pattern panel), each an app `Tool` over the existing model ops
  (`MoveEntities`/`RotateEntities`/`CopyEntities`/`RectangularPattern`/`CircularPattern`)
  plus new model **`ScaleEntities`** (scales points *and* circle/ellipse radii — the affine
  path doesn't). Tools gather a selection (PickSnap) + parameters (setters) and apply on
  `OK`; behavior tested headlessly (model + app e2e). **Refactor (per action plan):** split
  the 2D-sketch ribbon out of the 507-line `commands_standard.go` into `commands_sketch.go`
  (now 385/126 lines, both <500) and de-duplicated the three near-identical command-table
  loops into one `buildToolCommands` helper. Panel placement locked by
  `TestSketchTabPanelsMatchInventor`. Deferred: **Stretch** (vertex-window selection) and
  the head ImGui param dialogs (thin layer over the setters). Core suite + head green.
- **2026-06-04:** **Canonical the reference platform ribbon structure documented (user direction).** Our
  ribbon panels deviated from the reference platform's; captured the authoritative tab→panel→button tree
  in [`../architecture/mapping/inventor-ribbon-structure.md`](../architecture/mapping/inventor-ribbon-structure.md)
  (2026.1 Default) as the source of truth, linked from CONVENTIONS (DoD) and core/09-ui.
  Then aligned the whole **Sketch tab** to it: Dimension + Auto Dimension → **Constrain**
  (no phantom "Dimension" panel), **Mirror → Pattern** (new panel), **Fillet → Create**;
  Modify now = Offset/Trim/Extend/Split. Locked by `app.TestSketchTabPanelsMatchInventor`.
  Remaining (missing *tools*, not misplacements): Create Slot/Chamfer/Text, Modify Move/
  Copy/Rotate/Scale/Stretch, Pattern Rectangular/Circular; non-canonical `Draw` panel
  (Project Geometry) under review. Part Pattern/Surface panels reserved for upcoming UI.
  Core suite + head green.
- **2026-06-04:** **Audit + PartDocument-completion kickoff (REPORT.md / PARTDOC-PLAN.md).**
  Full code-vs-docs audit (REPORT.md at repo root); reconciled this tracker, README totals
  (24 milestones / 135 features / 262 PBIs), and backfilled M19's missing feature/PBI
  files; added the three-axis status model (Model/Geometry/UI+e2e) to CONVENTIONS.md and a
  deferral ledger (DEFERRALS.md). Added a `head` cgo CI job (lavapipe+xvfb) so the e2e UI
  tests finally run. **Phase 0 enablers:** fixed **T-01** (`TestAddInExtendsRibbonAndActsOnClick`
  — a stale test driving a ZeroDoc session against a Part-ribbon-defaulted add-in button,
  not a product bug) and **K2** (`geom.TransformCurve/Surface` now handle NURBS via
  control-point transform — exact for affine; weights/knots/degree invariant; 5 tests).
  **Phase 1 (Sketch2D) — curve trim/extend, two horizontal slices landed:**
  (1) *kernel-shared* **2D analytic intersection** primitives — `LineCircle2dIntersection`,
  `SegmentCircle2dIntersection`, `Circle2dCircle2dIntersection`, `Arc2d.ContainsAngle/
  ContainsPoint` (`kernel/geom`, 100% on the intersection funcs);
  (2) *model-domain* — `model/sketch` line **trim & extend now cut against circles and
  arcs** (honoring arc sweep), via `edit_trim_curves.go` adapters + a generalized
  crossing/extension engine, closing the line-only follow-up noted in `edit_trim.go`;
  (3) *ui-domain* — **Trim / Extend / Split sketch tools** (`app/sketch_trim_tools.go`) +
  ribbon commands (`Sketch.Trim`/`Extend`/`Split`, aliases X/EX/SX, Modify panel) + 6
  headless e2e tests driving pick→commit→validated edit. **Curve-trim/extend now meets the
  DoD** (model+geometry+UI+e2e) at the same bar as the offset/mirror tools (click tools,
  no dialog); (4) *enhancement* — trimming a **circle or arc as the target**
  (`edit_trim_curve_targets.go`): a trimmed circle becomes its complementary arc, a trimmed
  arc keeps the remaining arc(s); Trim tool + router `sketch.transform` dispatch on
  line/circle/arc. Also fixed a latent bug where `removeEntity` left a deleted curve in its
  typed collection — new `deleteEntity` prunes both. **Curve trim/extend/split is now
  complete across line/circle/arc.** K1 (boolean robustness) parked per user direction.
  Core suite + head green.
- **2026-06-04:** **Sketch2D residual — AutoDimension rewritten to be well-constrained +
  given a ribbon command.** The old `AutoDimension` grounded whole *entities* until DOF=0,
  which double-fixed shared points and left the sketch **over-constrained** (redundant). It
  now greedily adds candidates — length/radius **dimensions** first, then per-**point**
  grounds — each accepted only if it lowers DOF *without* raising redundancy (rank-guarded
  via `AnalyzeConstraints`), so the result is **WellConstrained, 0 redundancy**, and
  dimensions are real/editable + placed at current values (geometry-preserving). Added
  `DimensionConstraints.Delete` (prunes the dim + its parameter) for the trials. UI:
  `Sketch.AutoDimension` ribbon command (Dimension panel) + e2e; the `sketch.autoDimension`
  wire method already existed. Tests assert well-constrained/no-redundancy/real-dims/
  geometry-preserved. Core suite + head green.
- **2026-06-02:** **Usable Extrude in the head: distance dialog + profile-pick feedback
  (M05 UI, user direction).** Even with finished sketches now pickable, the head had no
  way to *complete* an extrude — picking a profile stored it on the tool with no highlight,
  and there was no distance field, so OK stayed disabled (distance 0) and extrude looked
  like "can't select." Added: `ExtrudeTool.PickedProfile`/`Distance` getters + Session
  bridges (`ActiveExtrude`, `ExtrudeDistanceDisplay`/`SetExtrudeDistanceDisplay` converting
  the document length unit ↔ database units, `LengthUnitName`); a head **Extrude dialog**
  (distance field in doc units + OK/Cancel, syncing the tool each frame) and a viewport
  **profile highlight** of the picked region (outer + holes) plus the live prism **preview**
  (the tool's Previewable wireframe, now drawn outside the sketch env). Verified headlessly:
  click a profile → set 40 mm via the dialog path → OK → a healthy solid; the mm↔cm
  round-trip is unit-tested. Core suite + vet + fmt green; head builds, `make smoke` 5
  frames zero validation errors, `head/ui` in-window tests pass. (Known lim: a circle drawn
  *inside* a rectangle is one region with a hole — the inner disk isn't independently
  selectable yet; multi-region overlapping profiles are a model-level follow-up.)
- **2026-06-02:** **Finished sketches stay visible and are pickable for extrude (M05 UI,
  user direction).** After Finish Sketch the sketch vanished from the 3D view and could
  not be selected, so extrude was unreachable. Two gaps: the sketch overlay only rendered
  *inside* the sketch editor, and `RayPicker` only hit-tested faces/work-planes (never
  profiles — the end-to-end extrude test used a stub picker). Now (1) the head renders the
  active part's finished, visible sketches in the 3D view (`partSketchOverlays`, skipping
  the one being edited); (2) `RayPicker.WithSketches` lets the picker resolve a click
  inside a sketch **profile region** to a `ProfileHandle` — it ray-casts each visible
  sketch's plane, maps the hit to 2D, and tests the new `sketch.Profile.Contains` (inside
  the outer loop, outside any hole); nearest-candidate selection keeps a solid in front
  winning over the sketch on its face, and profiles are only picked when the filter admits
  them (the Extrude tool sets `SelectProfile`). Verified end-to-end headlessly: a top-down
  click inside a square picks its profile and extrudes to a healthy solid; clicks outside
  miss; an open profile contains nothing. Core suite + vet + fmt green; head builds, `make
  smoke` 5 frames zero validation errors, `head/ui` in-window tests pass.
- **2026-06-02:** **Sketch editor: dimension constraints made real in the UI + ribbon
  panels laid out horizontally (M05 UI, user direction).** Two gaps closed toward a
  feature-complete sketch editor. (1) **Ribbon layout** — panels were stacking
  *vertically*, pushing the contextual Sketch tab's "Exit" panel (Finish Sketch) off
  screen; new `BeginGroup`/`EndGroup`/`SeparatorVertical` native bindings + a rewritten
  `drawRibbon`/`drawPanel` now lay each panel out as a horizontal group (button row +
  title) with vertical dividers, so every panel (incl. Exit/Finish Sketch) is visible.
  (2) **Dimensions** — the Dimension tool placed a driving dimension at the *measured*
  value with no way to set it and no on-screen display. Now the just-placed dimension is
  held as the session's **pending** dimension (`Session.PendingDimension`/
  `CommitPendingDimension`/`CancelPendingDimension`/`BeginEditDimension`); the head shows
  an **edit popup** pre-filled with the value, and committing an expression (e.g. "50 mm",
  "width/2") **drives the geometry** through the parameter DAG + solver. Dimensions now
  **render in the viewport**: `Session.SketchDimensions()` emits render-ready
  `DimensionView`s (witness/dimension lines, radial/diameter leaders, angle arcs, arc-
  length leaders + a value label) computed headlessly in sketch-plane coords; the head
  draws the lines as 3D line items and the value text at the label anchor projected to
  screen via the new pure `renderer.Project`. **Double-clicking** a dimension's label
  re-opens its editor. App-layer logic fully unit-tested (pending/commit/cancel + a view
  per kind + parallel-line angle skip); `renderer.Project` tested (center/upper-half/
  behind-camera). `CGO_ENABLED=0 go test ./...` + vet + race green; head builds, `make
  smoke` renders 5 frames with zero Vulkan validation errors and `head/ui` in-window
  tests pass.
- **2026-06-02:** **the reference platform-style origin coordinate system + work-feature persistence
  (branch `work-features`).** Modelled construction geometry like the reference platform (verified
  against `Oblikovati.Contracts.CSharp`): a part owns one `feature.WorkGeometry` holding
  the static origin coordinate system — center point, X/Y/Z axes, XY/XZ/YZ planes — as
  grounded `IsCoordinateSystemElement` members with well-known `WorkRef` keys, plus the
  user work planes/axes/points. Work features are now defined **relative to references**
  (origin well-known keys; user features by collection position), resolved at recompute,
  replacing the opaque eval closures with typed serializable definitions. Unified on one
  `WorkPlane` type (dropped `compdef.WorkPlane`; migrated ~13 app/head sites); the part
  recomputes its work geometry before the feature program. Persistence: a `workFeatures`
  recipe section serializes user features in creation order (origin regenerated);
  **revolve** serializes its axis as a `WorkRef` and re-binds it on restore. Verified:
  a work plane offset off the origin XY re-resolves to z=5 after reopen; a revolve's axis
  re-binds to `origin/axis/z`. Full suite + `head` green. Out of scope (error loudly until
  coded): face/edge/sketch-referenced work features, sweep/loft/coil/rib, mesh, freeform
  cage, non-parametric base.
- **2026-06-02:** **Feature persistence — more codecs (branch `feature-codecs`).** Added
  recipe codecs (with round-trip tests) for: **hole/boss** (placed on a face by key) +
  **combine** (bodies by index); **patterns** rectangular/circular/sketch-driven + **mirror**
  (source features recorded as program indices, re-bound on restore); **boundary-patch**
  and **ruled-surface** (sketch-based, like extrude); the six direct **face-edits**
  (split/move-face/face-offset/delete-face/replace-face/thicken, uniform via a faceEditor
  interface). Split the feature codecs into per-family files (serialize_*.go) to stay
  under the size limit. With extrude + dress-up (already merged), ~20 feature kinds now
  round-trip; uncoded kinds still error on save (no silent loss).
  **Blocked on model integration (not serialization):** user **work features**
  (WorkPlane/Axis/Point/UCS) and **revolve** — work features are standalone types not
  stored on the PartComponentDefinition, so there is nothing persistent to serialize until
  they are wired into the part (and revolve references a WorkAxis with no persistent home).
  **Deferred (geometry-data serialization):** freeform sub-D cage, mesh, non-parametric
  base (raw B-rep bodies). **Scattered numeric edits remaining:** trim-by-plane, surface
  offset, mid-surface, stitch, sculpt, core-cavity.
- **2026-06-02:** **Feature persistence — dress-up family + reference keys.** Added
  codecs for **fillet, chamfer, shell, draft, thread** to `model/feature/serialize.go`.
  These reference body edges/faces by **reference key**; the keys are lineage-derived
  bytes (`topo …ReferenceKey()`), so they serialize as base64 and **re-bind to the
  regenerated topology after recompute** via `body.FindEdgeByKey`/`FindFaceByKey` — no
  KeyManager context blob is needed (the earlier plan over-scoped this). Verified:
  extrude → pick a real edge → fillet → save → reopen → recompute → the fillet's edge
  key **re-binds** (health Warning = resolved, not Sick). Dress-up values
  (radius/distance/thickness/angle) persist as evaluated scalars like the extrude
  distance. No-silent-loss still errors on the remaining uncoded feature kinds (holes,
  patterns, work features, surfaces, mesh, …). Full suite + `go vet` green.
- **2026-06-02:** **Feature persistence — extrude (Phase 4 of the YAML recipe).** The
  feature program now serializes into the recipe via `model/feature/serialize.go`: a
  per-kind codec (switch) + a `SketchIndexer` seam so features re-bind their input
  sketch by index. **Extrude** is fully coded (sketch+profile+operation+distance extent
  +taper); on open the program is rebuilt and `Recompute()` regenerates the solid.
  Verified end-to-end: part → rectangle sketch → extrude → save → reopen → **identical
  body** (same range box, 1 body). `compdef.partRecipe.Features` wires it in; `cli new
  part --seed` now emits a real 4×3×5 block that round-trips. No-silent-loss:
  `MarshalRecipe` errors on any feature kind without a codec. **Remaining (follow-on,
  incremental):** codecs for the other ~24 feature kinds (dress-up, holes, patterns,
  work features, surfaces, …); the dress-up family additionally needs reference-key
  context persistence (`identity.SaveContextToArray` + `KeyToString`) so edge/face
  picks rebind after recompute — the machinery is scoped but lands with those codecs.
  Also captured a model gap: distance extents serialize the evaluated value (parametric
  distance expressions need the dimension-driven extent API). Full suite + race green.
- **2026-06-02:** **Sketch persistence (Phase 3 of the YAML recipe).** Sketches now
  round-trip in the `.obk` recipe: `model/sketch/serialize.go` + `serialize_restore.go`
  capture the host plane, every constrainable point, all curve entities (line/circle/
  arc/ellipse/spline + standalone points), and **all** geometric (~17 kinds) and
  dimensional (distance/radius/diameter/angle/arc-length) constraints. Points are
  recorded once and referenced by id, so a **shared corner stays one point** — a
  rectangle reopens as 4 points / 4 lines / 1 closed profile, not 8 points. Added
  point-accepting `Add` cores to the entity collections and a `refs []Entity` to
  `DimensionConstraint` (a dimension now records what it measures). No-silent-loss:
  `MarshalRecipe` errors on blocks / any uncoded entity/constraint kind. Wired into
  `compdef` `partRecipe.Sketches`. Verified: a part with a constrained sketch survives
  save→YAML→open through the real store; `cli new part --seed` now persists its sketch.
  Full suite + `go vet` green. **Remaining:** Phase 4 — the feature program + reference
  keys (the geometry-producing features), still dropped on save until then.
- **2026-06-02:** **Git-friendly YAML document format + parameter persistence (user
  direction; ADR-0020).** Replaced the ZIP `.obk` container with a **single YAML text
  file** per document so models live in git with line-level diffs. New
  `persistence/yamlcodec` is the sole importer of `gopkg.in/yaml.v3` (the GPL core's
  first third-party dep, wrapped per CLAUDE.md) and owns the on-disk shape: manifest at
  top level, recipe as a **native nested node** (not an escaped blob), binary data
  sections base64. `persistence` reworked (io/package/manifest/migration/compaction);
  schema bumped **1→2**; legacy ZIP `.obk` rejected with a clear message (no migration —
  only fixtures existed). **Recipe round-trip:** a new `doc.RecipeContent` seam + a
  content-factory registry (`doc.RegisterContentFactory`, populated by `compdef.init`)
  lets the store reconstruct a live part on open; `compdef/serialize.go` persists/restores
  **parameters** (creation order, incl. dependent expressions), **display units**, and
  the **end-of-part** marker, then `Recompute()`s. Proven: `cli new part p.obk --seed`
  → `cat` shows readable YAML → `save-as` reopens and re-emits the `width` param + units
  identically. `go vet` + full core suite green. **KNOWN GAP (next):** sketches (Phase 3)
  and the feature program + reference keys (Phase 4) are NOT yet in the recipe, so a
  part's geometry is currently dropped on save — strict no-silent-loss guard lands with
  those phases.
- **2026-06-02:** **Document open/save flow wired through the CLI and UI head (user
  direction, e2e fixtures).** The M03 model lifecycle (`Workspace.Open/Save/SaveAs`,
  `persistence.PackageStore`) existed but nothing drove it — `app.NewSession` passed a
  nil store, so Save/Open always errored. Added `app.NewSessionWithStore(doc.Store)` (app
  still depends only on the interface; the binaries inject `persistence.PackageStore`) and
  thin Session verbs `OpenDocument` / `SaveActiveDocument` (returns `ErrNeedsPath` when the
  doc has no `.obk` path yet) / `SaveActiveDocumentAs`. New **`oblikovati-cli`** fixture
  generator: `new <type> <path> [--seed]`, `open`, `info [--json]`, `save-as`, `version`,
  sharing one `doc.ParseDocumentType` with the add-in router (de-duped). The head's File
  menu gained working **Open / Save / Save As** via a pure-Go `fileDialog` state machine
  (unit-tested) + one new cgo `native.InputText` binding; `newDemoSession` now injects a
  real store. Fixed a latent wart: `doc.Restore` made every persisted name an explicit
  override, freezing a *derived* name across Save As — now a derived name stays derived and
  follows the file (regression test in `persistence/store_test.go`). **Known limit:**
  `PackageStore.Save` still persists only the manifest (type + name), so `--seed` content
  and model streams do not round-trip until stream persistence (M07+); `.obk` fixtures are
  type+name-bearing shells today. `go vet` + race suite green (golangci-lint not run — not
  installed locally).
- **2026-06-01:** **Coincident point-to-curve / midpoint (M05 UI, user direction).** Bug:
  coincident only worked end-to-end (point-to-point) — picking a point and a line/midpoint
  added nothing, because the model only had point-to-point coincident and the tool only
  accepted points. Added `AddPointOnLine`/`AddMidpoint`/`AddPointOnCircle` to
  `model/sketch` (residual + solve tested). The **Coincident tool** now accepts a point +
  a line/circle and applies the right constraint: two points → coincident; point + line →
  point-on-line (or **midpoint** when the line was picked at its midpoint snap); point +
  circle → point-on-circle. To know midpoint-vs-edge, constraint tools became
  **snap-aware** — each pick now carries the `SnapResult` (`ConstraintTool.picks` of
  `constraintPick`, `SketchEntityTool.PickSnap`), and the viewport feeds the snap under
  the cursor. Tested point-to-point / point-on-line / midpoint via the tool flow + the
  model constraints. `make ci` green (total 93.0%); head builds + `make smoke` renders 5
  frames, zero Vulkan validation errors.
- **2026-06-01:** **Snap markers (cross/triangle/square) + visible points (M05 UI, user
  direction).** A bare sketch point is 1px, so hover/snap highlighting was invisible.
  Extended snapping into a typed `SnapResult`/`SnapKind` (`app/sketch_snap.go`):
  `snapAt` now finds, by priority, an existing **endpoint**, a line **midpoint**, an
  **on-curve** point (closest point on a line segment or circle outline), then the grid;
  `SnapAt(px,py)` exposes it. Geometry-tool clicks snap to all of these. Head: while any
  sketch tool is active and the viewport is hovered, a **snap glyph** is drawn at the
  snap point — a **square** for an endpoint, a **triangle** for a midpoint, a **cross**
  for a line/circle edge (screen-constant size via `Camera.WorldPerPixel`); placed
  sketch points also get a persistent square marker so they are visible at rest. Snap
  kind/point detection is unit-tested (endpoint/midpoint/line-edge/circle-edge/grid).
  `make ci` green (app 91.9%, total 93.0%); head builds + `make smoke` renders 5 frames,
  zero Vulkan validation errors.
- **2026-06-01:** **Constraints/dimensions reworked to the reference platform's tool-first flow +
  hover candidate highlighting (M05 UI, user direction).** Constraints were immediate
  commands on the current selection; now each is an **interactive tool**: activate the
  constraint, then pick the geometry. New `ConstraintTool` (generic, table-driven over
  `constraintToolDefs`) gathers picks filtered by an `accepts` predicate (only the entity
  kinds valid for that constraint) and auto-applies when `ready`, then deactivates;
  `SketchEntityTool` marks them so `Pointer` routes sketch-entity picks to the active
  tool. The apply logic was refactored to `applyX(s, ents)` functions (taking explicit
  entities). **Hover highlight**: `Session.HoverCandidate` returns the entity under the
  cursor the active tool would accept; the head paints it **green** (valid candidate),
  picked entities **cyan**, others amber — so the user sees what's selectable for the
  current constraint. Tests cover every constraint/dimension tool through the pick flow
  + the apply functions. `make ci` green (app 91.8%, total 93.0%); head builds + `make
  smoke` renders 5 frames, zero Vulkan validation errors.
- **2026-06-01:** **Single-shot tools + Esc cancels at any point (M05 UI, user
  direction).** Changed the auto-commit geometry tools from continuous to **single-shot**:
  after a tool completes its shape it **deactivates** (no auto-restart) — `AutoCommitTool`
  is now a marker (`AutoCommits() bool`) and `sketchClick` commits without restarting.
  **Esc** now cancels the active tool at any point in its operation (or clears the
  selection when idle): `Session.PressKey` Escape → cancel-tool-else-deselect, and the
  head routes the physical Esc key each frame (`DrawChrome.handleKeyboard` + native
  `EscapePressed`). Tests updated to single-shot; added Esc-cancels-mid-operation and
  Esc-clears-selection coverage. `make ci` green (total 93.2%); head builds + `make smoke`
  renders 5 frames, zero Vulkan validation errors.
- **2026-06-01:** **Fix: sketch geometry tools now create geometry on click (auto-commit
  + live preview).** Bug report — clicking with Line/Rectangle/Circle active produced no
  visible object, because tools only created geometry on an explicit OK that the user
  never issued. Fixed: the fixed-arity tools (line/rectangle/circle/arc/ellipse/polygon/
  point) now **auto-commit** the instant they have enough clicks and **restart** for
  continuous drawing (new `AutoCommitTool.Fresh`; `sketchClick` commits + restarts).
  Spline keeps explicit OK (variable length). Added a **rubber-band preview**
  (`PreviewTool`/`ActiveToolPreview` + `CursorSketchPoint`): while drawing, the
  provisional shape (line to cursor, rectangle/circle/polygon from the first click
  through the cursor, spline through placed points) renders in the viewport following
  the mouse. Existing tests updated to the no-explicit-OK flow. `make ci` green (total
  93.2%); head builds + `make smoke` renders 5 frames, zero Vulkan validation errors.
- **2026-06-01:** **Sketch snapping + entity selection + constraint & dimension tools
  (M05 UI, user direction).** Verified the geometry tools (line/rectangle/circle/arc/
  spline/ellipse/polygon/point) all green, then added the constraint workflow. **Snap
  points** (`app/sketch_snap.go`): every geometry-tool click maps to the plane and snaps
  to a nearby existing point (or the sketch origin) then the nearest grid intersection,
  within a zoom-scaled tolerance (`scene.Camera.WorldPerPixel`), toggled by `SnapToPoints`/
  `SnapToGrid` prefs. **Sketch-entity selection** (`sketch_select.go`): in the sketch
  environment a plain click picks the nearest point/line/circle/arc (points before
  curves; Shift extends), and `IsSelectedEntity` drives a cyan highlight in the overlay.
  **Constraint commands** (`sketch_constraints.go`) act on the selection and re-solve:
  Coincident, Collinear, Parallel, Perpendicular, Horizontal, Vertical, Tangent,
  Concentric, Equal (length/radius), Fix — all verified to actually drive the geometry
  via the M06 solver (perpendicular→90°, parallel, equal radius, concentric, tangent…).
  **Dimension** infers distance/length, radius, or angle from the selection and creates a
  driving `DimensionConstraint` at the measured value in document units. Ribbon: new
  **Constrain** + **Dimension** panels on the Sketch tab. Head: Shift multi-select +
  selected-entity highlight. `make ci` green (app 93.4%, total 93.3%); head builds +
  `make smoke` renders 5 frames, zero Vulkan validation errors.
- **2026-06-01:** **Sketch grid (document units, configurable spacing) + Preferences
  window (M05 UI, user direction).** The part now carries document `Units` (default mm;
  db length unit = cm) via `compdef.PartComponentDefinition.Units`/`SetLengthUnit`, and
  `param.UnitsOfMeasure` gained `ToPreferred`/`FromPreferred`/`PreferredName`. New
  `app.GridSettings` (spacing as a length `Quantity` in db units, `Visible`,
  `MajorEvery`) on the `Session` (`Grid()`), with `GridSpacingDisplay`/`SetGridSpacingDisplay`
  presenting the spacing in the document's unit (5 mm ⇒ 0.5 cm model, 1 in ⇒ 2.54 cm).
  Head: a **sketch grid** drawn in the active sketch plane (lines parallel to the plane
  axes, **origin at the plane's 0,0**, minor/major/axis colors) when editing, plus a
  **Preferences window** (Tools ▸ Preferences) editing grid spacing (labeled with the
  doc unit), visibility and the major interval — new native `InputFloat`/`InputInt`/
  `Checkbox` widgets. `make ci` green (total 93.4%); head builds + `make smoke` renders
  5 frames, zero Vulkan validation errors.
- **2026-06-01:** **Camera swings to face the sketch plane on enter / restores on exit
  (M05 UI, user direction).** `scene.Camera.Facing(target,normal,up)` aims the view
  straight at a plane at the preserved eye–target distance, keeping the eye on its
  current side (no back-flip); `scene.Lerp` blends two views. The `Session` holds a
  short eased **camera tween** (`animateCameraTo`/`CameraAnimating`/`TickCameraAnimation`,
  smoothstep, 0.35 s): `EnterSketch` remembers the prior view and swings to face the
  sketch plane (`YAxis` as up); `ExitSketch` swings back to the remembered view. The
  head drives the tween in the viewport each frame (new native `DeltaTime`), ignoring
  user navigation/clicks while it runs. Verified headlessly: entering a sketch ends with
  the view's forward parallel to the plane normal; exiting restores the exact eye/target;
  an XY sketch leaves the top-down view unchanged. `make ci` green (app 94.6%, total
  93.5%); head builds + `make smoke` renders 5 frames, zero Vulkan validation errors.
- **2026-06-01:** **Create-Sketch plane-pick tool + work-plane hover highlight (M05
  UI, user direction).** Fixed two issues: Create 2D Sketch did not let you pick the
  plane *after* clicking the command, and planes had no hover feedback. Now
  `Sketch.Create2D` starts an interactive **`CreateSketchTool`** when no plane is
  pre-selected: it restricts the filter to work planes and, the instant a plane is
  clicked (3D view or browser), **auto-commits** — creating + opening the sketch
  (new `autoCommitter` capability + `Session.feedPick`; a pre-selected plane still
  sketches immediately). The `RayPicker` pick logic was rewritten to prefer the
  nearest *accepted* candidate, so a work-plane-only pick falls through a rejected
  face. Hover: `Session.PickAt` hit-tests without selecting; the viewport highlights
  the **plane under the cursor** in an amber hover color (selected > hovered > faint),
  and the camera is now updated before the click/hover pick each frame. `make ci`
  green (app 94.8%, total 93.5%); head builds + `make smoke` renders 5 frames, zero
  Vulkan validation errors.
- **2026-06-01:** **Origin work planes + selection + contextual Sketch tab (M05 UI,
  user direction).** Made the part's datum planes real, selectable objects so a sketch
  can be started on a chosen plane. `compdef`: `WorkPlane` (name + `sketch.Plane` +
  display half-extent) and the three **origin planes** (XY/XZ/YZ) owned by every part
  (`OriginPlanes`/`WorkPlaneByName`). `app`: `WorkPlaneHandle`/`SelectWorkPlane`;
  `RayPicker.WithPlanes` now hit-tests the origin planes (ray↔plane within the display
  square, **face wins when a solid is in front**); `Session.Select`/`SelectBrowserNode`
  + `Camera`/`SetCamera` keeps the picker's view in sync; `CreateSketchOnSelectedPlane`
  starts the sketch on the selected plane (falls back to XY); the **browser** gains an
  **Origin folder** of selectable plane nodes (`BrowserNode.Select`). Head: the ribbon
  **auto-switches to the contextual Sketch tab** on entering the sketch environment and
  back to 3D Model on finish (new `BeginTabItemSelected`); browser rows are clickable
  `Selectable`s that select the plane; the viewport **renders the origin planes** as
  finite squares (selected one highlighted) and **routes left-clicks** to pick a
  face/plane (or place sketch geometry), with the part's plane-aware `RayPicker`
  installed. So: click a plane in the 3D view or the tree → Create 2D Sketch builds on
  it. `make ci` green (app 94.4%, total 93.5%); head builds + `make smoke` renders 5
  frames, zero Vulkan validation errors.
- **2026-06-01:** **Sketch editor environment + full Create ribbon (M05 UI, user
  direction).** The head had placeholder sketch buttons and no way to start/edit a
  sketch like the reference platform. Added the **sketch environment** to `app`: `Session.CreateSketch`/
  `CreateSketchOnOrigin` (XY/XZ/YZ) enters edit mode, `FinishSketch` exits + recomputes
  the part, with `InSketch`/`CanCreateSketch` predicates. New geometry tools (interactive
  `Tool`+`PlaneClickTool`+`Prompted`, click-driven, real `model/sketch` entities):
  **Circle, Arc (3-point), Spline, Ellipse, Polygon, Point** joining Line/Rectangle —
  click routing refactored to the `PlaneClickTool` interface. `RegisterStandardCommands`
  builds the reference platform's ribbon: **3D Model** (Create 2D Sketch, Extrude), the contextual
  **Sketch** tab (8-tool Create panel + Finish Sketch, all gated on `InSketch`), and
  View; Extrude/Create-2D-Sketch gate on environment so the ribbon acts as a sketch
  editor when a sketch is open. Head wiring: new native verbs (`IsItemClicked`,
  `ItemRectMin`, `MousePos`) route viewport left-clicks to the active sketch tool
  (left-orbit suppressed while placing), and a **sketch overlay** renders the live
  sketch geometry (lines/circles/arcs/ellipses/splines sampled to wireframe) in the
  Vulkan viewport. `make ci` green (app 93.9%, total 93.4%); head builds + `make smoke`
  renders 5 frames with zero Vulkan validation errors. Constraints/dimensions/Slot/
  Fillet (selection-based) are the next sketch-editor increment.
- **2026-06-01:** **M10-F04 (Mesh, Imported Geometry & Mold) complete — M10 fully
  done (4/4 features).** PBI-115: `MeshGeometry` + an **ASCII STL importer**
  (`ParseSTL`, coincidence-welds shared corners) wrapped by `MeshFeature` as reference
  geometry with selectable `MeshFace`/`MeshEdge`/`MeshVertex` handles + `MeshFeatureSet`
  grouping (a tetra STL → 4 facets / 4 verts / 6 edges; the prior solid passes through).
  PBI-116: `CoreCavityFeature` splits the running tooling block by a planar parting
  (axis + position) into a **core** and **cavity** solid (10³ block at z=4 → core
  [0,4] / cavity [4,10], both validated), recording a shrinkage allowance; sick when
  the parting is outside the block. Part-pocket subtraction + silhouette parting are
  phase-C booleans. `make ci` green, total 93.3%. **M10 complete: surface creation,
  surface editing, sub-D freeform (real Catmull–Clark), and mesh/mold — the whole
  surfacing & freeform milestone, all headless-tested.**
- **2026-06-01:** **M10-F03 (Freeform Modeling) complete — 2/2 PBIs.** A real
  sub-D subsystem, headless. New **`kernel/subd`**: a `Mesh` control cage, `Box`/
  `Plane`/`QuadBall` primitives, **Catmull–Clark `Subdivide`** (face/edge/vertex
  rules with crease + boundary handling, creases propagating across levels), and
  `ToBody` converting a refined cage to a B-rep body (closed cage → validated solid,
  open → surface). `model/feature` `FreeformFeature`/`FreeformBody` recompute the
  limit surface from the cage; cage edits — `MoveVertices`, `CreaseEdges`, `SetLevel`
  + `FreeformFace`/`Edge`/`Vertex` handles — redeform the body. Verified: a box
  primitive → a valid solid; smooth subdivision **rounds** the cube (range box
  shrinks); creasing the three edges at a corner keeps it a **sharp fixed point**;
  a quad-ball is a rounded solid. `AliasFreeformFeature` wraps an imported Alias
  cage (M17). `make ci` green, total 93.4%.
- **2026-06-01:** **M10-F02 (Surface Editing) complete — 2/2 PBIs.** Real planar
  surface editing in `kernel/ops` + `model/feature`. PBI-111: `ops.TrimByPlane` is a
  **Sutherland–Hodgman half-space clip** of a planar surface body's boundary polygon
  against a cutting plane (edge–plane crossings inserted), rebuilding the kept patch;
  `TrimFeature` trims the running surface (sick if nothing remains). `ExtendFeature`
  validates a target then defers the edge-to-target geometry. PBI-112: `ops.MidSurfaces`
  pairs a solid's **antiparallel planar faces** within a thickness threshold, emitting
  a mid-plane patch + recorded thickness per thin-wall pair — a 4×4×1 plate yields one
  mid-surface (thickness 1) for FEA (M18); `MidSurfaceThickness(es)` carry the values.
  `ops.OffsetSurface` translates a planar patch along its normal (`SurfaceOffsetFeature`,
  named to avoid the M09 direct-edit `FaceOffsetFeature`). Curved/multi-face trims,
  offsets and mid-surfaces are honestly `NotYetImplemented` (phase C). `make ci` green,
  total 93.9%.
- **2026-06-01:** **M10-F01 (Surface Creation) complete — 2/2 PBIs.** The first
  surfacing features, all headless. PBI-109: `BoundaryPatchFeature` fills a closed
  planar sketch profile (with inner loops as holes) into a real **surface body** —
  one trimmed planar face — carrying a per-loop `PatchCondition` (G0/G1/G2);
  `RuledSurfaceFeature` rules a profile by a distance into an open band (real for the
  `RuledNormal` direction, tangent/perpendicular resolve-then-defer). PBI-110: new
  `kernel/ops.Stitch` is a real **exact-coincidence weld** — independently-built
  surface bodies whose vertices/edges coincide on a tolerance grid merge into one
  quilt, and a quilt where every edge is used twice becomes a **solid** (deterministic
  sorted lineage keeps reference keys stable; tolerant near-gap matching is still the
  deferred phase-D `Sew`). `StitchFeature`/`KnitFeatures` (alias) weld the running
  surface bodies; `SculptFeature` requires them to enclose a volume → solid, else
  sick. Headline: six oriented cube-face surface bodies stitch into one watertight,
  manifold, validated solid. `make ci` green (fmt+vet+lint+race), total 94.0%,
  feature 94.9%, ops 93.5%.
- **2026-06-01:** **M05 UI fidelity pass** (post-completion polish, user
  direction), grounded in the freshly-scraped the reference platform 2026 help corpus (32,824
  topics → condensed Markdown at `experiments/inventor-docs-scraper/out/md/`).
  Three increments on the cgo head: (1) **two-level ribbon** — `CommandDefinition`
  gains `Tab` (`WithTab`), `Category` becomes the panel within it; `BuildRibbon`
  groups tab → panel → button (empty tab ⇒ `DefaultTab` "Tools"); chrome renders an
  ImGui tab bar (new native `BeginTabBar`/`BeginTabItem` verbs). (2) **Status bar**
  — pure `BuildStatus` model + optional `Prompted` tool capability; Extrude/Line/
  Rectangle emit per-step prompts ("Select a profile to extrude" → "Specify the
  extrude distance" → "Click OK …"), shown with the selection count + OK/Cancel.
  (3) **Hover tooltips** on ribbon buttons (new native `SetItemTooltip`).
  (4) **Viewport mouse navigation** — pure immutable `scene.Camera.Dolly/Orbit/Pan`
  (turntable orbit via Rodrigues, pole-guarded; distance-clamped zoom; FOV-scaled
  pan), wired to wheel-zoom / middle-pan / Shift+middle-orbit (left-drag orbits too)
  through new native input verbs (input-capturing `InvisibleButton`, mouse delta/
  wheel/down, Shift, cursor get/set). Each ran the head 5 frames with **zero Vulkan
  validation errors**; `app` + `scene` tests green.
  (5) **Zoom All / Home view** — pure `scene.Camera.Fit` (frame a box, keep
  orientation) + `Home` (default +Y isometric, then fit); `Session.FitView`/
  `HomeView` over `modelBounds` (union of active-part body range boxes), exposed as
  View-tab / Navigate-panel commands. The viewport input→camera mapping was extracted
  into a pure `ui.ApplyNavigation(NavInput)` (untagged, unit-testable without cgo).
  **UI integration tests**: synthetic-input extrude → a real solid → `Zoom All`
  (incl. via `Session.Execute`, the ribbon-button path) reframes the model; ran the
  head 8 frames with zero validation errors.
  (6) **True in-window verification** — an input-injection seam in
  `obk_head_begin_frame` feeds synthetic pointer events into ImGui's IO *after* the
  GLFW backend's NewFrame (winning over the real cursor; trickling disabled so a
  pos+button+wheel burst lands in one frame), plus `SetNextWindowPos/Size` to pin the
  viewport to a known rect. cgo in-window tests open a real GLFW+Vulkan+ImGui window
  and drive the **production** nav path (InvisibleButton → ApplyNavigation →
  SetCamera) with injected hover/press/drag/scroll, asserting the live camera
  orbits / pans / zooms (distance-preserving where expected). All green, **zero
  Vulkan validation errors**; tests skip cleanly when no display/Vulkan is present.
- **2026-06-01:** Pivot to the **renderer/UI** with a headless synthetic-input
  integration-test harness (user direction). New `app` package — the application-
  shell logic (the reference platform's CommandManager/ControlDefinitions/SelectSet/
  InteractionEvents), **pure Go, no ImGui/Vulkan** (ADR-0014 "below the GPU line" +
  ADR-0004 "state in the model"). Delivered M05-F02 (command framework: definitions
  + manager + alias/enable + lifecycle events) and M05-F04 (selection set + typed
  handles + filters; pointer/key interaction routing per the reference platform mouse/keyboard;
  the interactive `Tool` framework with OK/Cancel). The headline result: a
  `Session` is driven by `Click`/`PressKey`/`Invoke`/`OK`, and a `Picker` stub lets
  a test "click on" geometry with no GPU — `TestExtrudeToolEndToEnd` **starts the
  Extrude tool, clicks the profile, sets a distance, hits OK, and asserts a real
  watertight prism solid** in the active part. To wire it, `compdef.
  PartComponentDefinition` now owns the feature engine + `Recompute()` (syncs the
  feature program's result bodies; one-way compdef→feature, no cycle — feature's
  derived-source test uses a local `BodySource` stub). the reference platform fidelity (aliases,
  LMB/RMB/MMB behavior, OK/Apply/Cancel) is the model; the 2026 help portal blocks
  WebFetch (503) so defaults are encoded with TODO-verify. `make ci` green; app
  93.5%. (M05 F01 add-ins, F03 ribbon, F05 client-graphics next; then the Vulkan/
  ImGui rendering backend behind the null/offscreen/vulkan interface.)
- **2026-06-01:** M05 **F05 client/interaction graphics + F03 ribbon/browser + F01
  add-in platform — all M05 logic now complete, headless.** F05: `scene.Camera`
  (ray casting) + `renderer.DrawList`/`BuildDrawList` (per-body surface+wireframe,
  front-of-camera cull, object-id tagging) + `Backend`/`NullBackend`;
  `Session.RenderFrame` assembles active-part bodies + persistent overlays + the
  active tool's transient `Preview` into one frame, asserted via the null backend
  (no GPU). `ExtrudeTool.Preview` draws a live prism wireframe before OK. Real
  `RayPicker` (camera ray vs face tessellation, `ops.RayCastFaces`) replaces the
  stub for production picking. F03: `BuildRibbon`/`BuildBrowser` are pure models
  built from live command-registry/part state each frame (core/09). F01: `AddIn`/
  `AddInManager` (the reference platform's ApplicationAddInServer). **M05 exit criterion met
  headlessly** — a sample add-in registers an interactive command on activation,
  it appears as a ribbon button, and executing it starts a tool that uses selection
  + preview graphics. `make ci` green; app 93.7%, scene 100%, renderer 95.6%, total
  94.2%. **Only the cgo "head" remains** (Vulkan viewport + Dear ImGui chrome behind
  the offscreen/vulkan backends) — per the user's gate, implemented now that the
  whole UI/renderer stack is built and end-to-end coverage is >90%.
- **2026-06-01:** **M05 cgo head COMPLETE — M05 fully done.** Added `renderer.
  ViewProjection` (GPU-facing column-major MVP, Vulkan clip conventions; headless
  metamorphic tests). Built the head as a **separate Go module** (`source/head`,
  replace `=> ../`) so its cgo/Vulkan/go1.24 toolchain never touches the go-1.22,
  lint-clean, 94%-covered core (`go test ./...` doesn't descend into it). Pivoted
  ImGui binding: cimgui-go's prebuilt `cimgui.a` uses an unreproducible generated
  imconfig (MyVec2/MyMatrix44), so we **vendor pristine Dear ImGui 1.92.8** (core +
  GLFW + Vulkan backends, MIT) with our own `imconfig.h` and thin cgo bindings
  (`internal/native`, the single cgo boundary per ADR-0008). The head: a persistent
  GLFW window + Vulkan device/swapchain + ImGui context, with **Go driving the frame
  loop AND the chrome layout** — menu bar, a ribbon panel per command category
  (buttons reflect each command's live enabled predicate), and the model browser tree
  — all read from the live `app.Session` each frame (ADR-0004/0009). The **3D Vulkan
  viewport** (ADR-0005) renders offscreen (color+depth) with two pipelines (Lambert
  triangles, flat lines sharing one pos/normal/color vertex format, push-constant MVP
  + lit flag), uploads `renderer.DrawList` flattened by the pure-Go `head/viewport`
  package (tested), and is shown in the dockable panel via an ImGui sampled texture.
  **VERIFIED running on this machine (Radeon/RADV + lavapipe fallback) with Vulkan
  validation layers enabled → ZERO validation errors** across the loop and teardown.
  `make smoke` / `make run` in source/head. Commits: 69cae15, 2728495, ecb7154,
  a83b1bd.
- **2026-06-01:** M09-F02/F03/F04 **complete — M09 fully done.** Hole/boss (F02)
  resolve their placement face by reference key then defer the cut geometry (phase
  C boolean); `HoleTapInfo` carries thread data for hole tables (M14). Patterns/
  mirror (F03) implement the **real** per-element model — `PatternElement` count
  driven by parameters (3×2 grid → 6 elements, 4×4 → 16) with per-element
  suppression (`ActiveCount`/`SetElementSuppressed`); rectangular/circular/sketch-
  driven/mirror all share it, with geometry duplication deferred to the M11
  body-transform op. Modify/direct (F04): **Combine is real** — it booleans two
  running bodies via `ops.Boolean` (a disjoint Join merges into one validated
  manifold); split/move-face/offset/delete-face/replace-face/thicken resolve their
  inputs then defer (phase C). All deferred features report Warning (inputs valid)
  vs Sick (input lost), never silently doing nothing. `make ci` green; feature
  95.6%. **M09 complete: 4/4.**
- **2026-06-01:** M09-F01 (Dress-up Features) — Phase A. Fillet/chamfer/shell/
  draft/thread in `model/feature` consume picked edges/faces as **reference keys**
  and re-resolve them against the running body each recompute via the topological-
  naming rebind (`topo.Body.FindEdgeByKey`/`FindFaceByKey`) — the load-bearing M03
  ↔ M07 seam exercised end to end through a feature. A resolved input means the
  inputs are valid but the rounding/cut geometry is kernel phase B, so the feature
  reports the new `ErrDeferred` sentinel which the engine maps to `health.Warning`
  (passthrough); a genuinely lost edge/face → `health.Sick` and surfaces for
  re-selection. `make ci` green; feature 95.6%.
- **2026-06-01:** M08-F04 (Derived & Reference Features) **complete — M08 fully
  done.** `DerivedPartComponent` pulls another part's bodies associatively
  (re-reads the source each recompute; `SourceVersion` tracks edits) — its source
  is the consumer-side `BodySource` interface that `compdef.PartComponentDefinition`
  satisfies structurally, so `feature` never imports `compdef` (no cycle, and
  compdef can later own the feature program one-way). `NonParametricBaseFeature`
  wraps frozen imported bodies (M17 translation output) as a feature-tree
  participant downstream features consume. `make ci` green; feature 95.4%. **M08
  complete: 4/4 features — the feature engine, datums, the extrude solid generator,
  and derived/base features. The Definition→Feature→geometry triangle now spans
  sketch → kernel → parameters → recompute end to end.**
- **2026-06-01:** M08-F03 (Sketched Features) — Phase A. **Extrude** is implemented
  end to end as the reference feature (PBI-092): `ExtrudeFeatures.AddByDistanceExtent`
  builds a real watertight prism B-rep from a closed sketch profile (bottom/top cap
  faces + one planar side wall per profile edge, lineage on each so reference keys
  derive), validated as a manifold solid by `ops.Validate`; it recomputes when its
  driving parameter changes (height grows), goes health-sick on a missing/open
  profile, and combines with running material via `ops.Boolean` (a disjoint Join
  merges two prisms). The full `Extent`/`ExtentType`/`ExtentDirection` surface is
  defined. Revolve/sweep/loft/coil/rib (PBI-093/094/095/096) ship their complete
  Definition/triangle objects and `Add` constructors, with B-rep generation honestly
  deferred (`NotYetImplemented`: revolve = kernel phase A, sweep/loft/coil = phase B
  NURBS), so those features go sick until their generators land. `make ci` green;
  feature 95.5%.
- **2026-06-01:** M08-F02 (Work Features / Datums) complete. Parametric construction
  geometry in `model/feature`, each holding a definition closure and recomputing
  when its inputs change: `WorkPlane` (offset-from-plane that moves with its driving
  parameter; three-point), `WorkAxis` (two-point; plane∩plane intersection),
  `WorkPoint` (explicit; axis∩plane pierce), and `UserCoordinateSystem` (origin +
  X/Y/Z triad). Degenerate definitions (collinear points, parallel planes, axis
  parallel to plane) go health-sick rather than producing garbage; a datum plane is
  directly usable as a sketch host (`sketch.Plane`). `make ci` green; feature 96.4%.
- **2026-06-01:** M08-F01 (Feature History Engine) complete — new `model/feature`,
  the heart of part modeling. `PartFeatures` is the ordered feature program and its
  **rollback-replay** engine (ADR-0010): `Recompute` finds the earliest dirty
  feature, reuses the cached body state from just before it (the clean prefix is
  never recomputed — verified by recompute counts), and replays the dirty tail to
  the end-of-part marker. A `Feature` is a pure `Recompute(Input)→Output`; inputs
  are `Ref` reference keys resolved lazily, not pointers (because replay destroys
  and recreates topology). Failure isolation: a feature whose op fails goes
  `health.Sick` and poisons its dependents (via dependency edges), but the rebuild
  continues so independent features still evaluate. Suppression (explicit +
  conditional via a parameter/`ComparisonType`/threshold) passes bodies through;
  reorder validates against dependencies and rejects illegal moves; rename is
  id-stable; `SetEndOfPart`/`RollToEnd` roll the part back/forward. `make ci`
  green; feature 95.7%.
- **2026-06-01:** M07-F04 (Part Component Definition Container) **complete — M07
  fully done.** New `model/compdef`: `PartComponentDefinition` is the part's
  modeling content and the root the feature engine (M08) operates within — it owns
  the surface bodies (`topo.SurfaceBodies`), parameters and sketches, the bounding
  boxes (`RangeBox`/`PreciseRangeBox`/`OrientedMinimumRangeBox`), a
  `ModelGeometryVersion` string that advances on every `MarkChanged`, and the
  end-of-part rollback marker (`SetEndOfPart`/`RollToEnd`/`IsRolledBack`, which
  bumps the version to request re-evaluation). It implements `doc.Content`
  (compdef→doc, one-way — doc never imports compdef) and attaches to a part
  document via the new `doc.Document.SetContent`; callers retrieve it by
  type-asserting `Content()`. `make ci` green; compdef 100%. **M07 complete: 4/4
  features — the pure-Go B-rep kernel (topology, evaluators, exact predicates,
  tessellation, validation, Phase-A booleans, part container) is in place.**
- **2026-06-01:** M07-F03 (Boolean & Modeling Operations) — Phase A. New
  `math/predicate` (the robustness foundation core/03 mandates): exact orient2D/
  orient3D/incircle with a float64 fast path and a `big.Rat` exact fallback, so the
  predicate SIGN is always correct even at near-degenerate configurations (verified
  on a point microscopically off a line); 100% covered. New `kernel/ops`:
  tessellation (`Mesh` + edge polylines; adaptive recursive-midpoint sampling that
  honors chordal tolerance on edges and curved faces; planar faces ear-clipped via
  exact `Orient2D` → watertight), validation (`Validate` checks manifold/closed/
  orientation, reports each offending edge precisely; `BoundaryEdges`), and the
  boolean framework (`PartFeatureOperation`, `PointInsideBody` ray-cast
  classification, relationship classifier; Phase-A handles disjoint and containment
  with valid manifold results). Genuinely hard parts are honestly deferred behind
  the same API: general intersecting booleans (face splitting) and tolerant `Sew`
  → `NotYetImplemented` (kernel phase C/D). `make ci` green; predicate 100%, ops 91.9%.
- **2026-06-01:** M07-F02 (Geometry Evaluators) complete — the numeric services on
  topology (`kernel/topo`). `CurveEvaluator`/`EdgeEvaluator`: point, unit tangent,
  curvature (κ=|r′×r″|/|r′|³ by finite difference), arc length (composite Simpson),
  and closest-param/point (coarse sample + golden-section). `SurfaceEvaluator`:
  point, unit normal, partials, and closest-point (coarse grid seed + projected
  Gauss–Newton, infinite domains clamped). `FaceEvaluator`: exact planar area
  (Newell), planar point-in-loop containment (dropping the normal's dominant axis,
  excluding holes). All verified against analytic references — radius-2 circle
  curvature 0.5 and length 4π, segment perpendicular foot, sphere closest along a
  radius, triangle area 0.5. `make ci` green; topo 94.3%.
- **2026-06-01:** M07-F01 (Topology Model) complete — new `kernel/topo`, the B-rep
  graph. Full Body→Shell→Face→Loop→EdgeUse→Edge→Vertex with adjacency
  (Edge.Faces, Face.Edges/Vertices, Vertex.Edges) assembled by a back-pointer-
  consistent `Builder`; faces bind to `geom.Surface` and edges to `geom.Curve3`
  (planar→Plane, circular→Circle), with per-entity and body `RangeBox` and a
  `SurfaceBodies` container. The load-bearing rule (parametric-cad §7): every
  entity records its generative `Lineage`, from which `ReferenceKey()` bytes
  derive, so `FindFaceByKey`/`FindEdgeByKey` rebind by lineage after a recompute
  destroys and recreates the body (verified: a rebuilt face is a new object yet
  the key still binds). Layering: topo sits below model/, so it is self-contained
  — it owns the lineage/key bytes and does NOT import `model/identity`; the
  `identity.KeyManager` binds those keys at the feature layer (M08). `make ci`
  green; topo 95.9%.
- **2026-06-01:** Strategy set — **build headless-testable (pure-Go) milestones
  first, defer renderer/UI/cgo work** (M05 UI, M16 visualization, and the
  view-rendering parts of M14). Order: M06→M07→M08–M13→M14 model→M15→M17→M18.
  Saved to memory (headless-first-sequencing).
- **2026-06-01:** M06-F06 (Profiles & Paths) **complete — M06 fully done.** The
  sketch→feature boundary (parametric-cad §10): `Sketch.Profiles()` walks segment
  connectivity into closed loops (plus standalone circles), classifies outer vs
  inner by even–odd nesting using an all-vertices containment test (robust when a
  hole is centered on the region centroid), and groups them into `Profile{outer,
  inner}` — multi-region and nested holes supported; a connected-but-unclosed chain
  becomes an open profile whose `IsClosed()` lets a solid feature reject it (surface
  features accept). Construction geometry is excluded. `Sketch.Paths()` returns every
  maximal connected chain (open and closed) as a `Path` sweep/loft rail, with
  `Path3D` constructible from an ordered 3D point chain. `make ci` green; sketch
  97.6%. **M06 complete: 6/6 features — `model/sketch` is the full constrained
  sketch environment, all headless-tested.**
- **2026-06-01:** M06-F05 (Constraint Solver) complete — the milestone's XL core,
  pure Go and headless. The whole-system numerical solver (ADR-0009): Newton/
  Gauss–Newton with Levenberg–Marquardt damping over the concatenated residual
  vector, finite-difference Jacobian, warm-started from the current variable values
  so an edit re-solves in a few iterations; iterations are capped and a
  non-convergent/conflicting system reports as health-sick rather than hanging or
  returning NaN (parametric-cad §2). It is dimension-agnostic — variables are
  `[]*math.Scalar`, so the same `Solve` resolves 2D and 3D. DOF analysis falls out
  of the Jacobian rank (`DOF = vars−rank`, `Redundant = eqs−rank`) giving
  well/under/over-constrained status: a fully-constrained sketch reports 0 DOF, a
  redundant constraint is flagged over-constrained, and conflicting constraints go
  sick. Small pure-Go dense linalg (Gaussian elimination + numeric rank) backs it;
  the graph decomposition into clusters is a future layer behind the same API.
  Auto-dimension deferred (a heuristic layer). `make ci` green; sketch 97.9%.
- **2026-06-01:** M06-F04 (Dimensional Constraints) complete. A `DimensionConstraint`
  owns a `param.Parameter` (the M02 DAG): its residual is `measure() −
  param.ModelValue()`, so editing the parameter's expression changes the target and
  the solver drives geometry to it (the solver and parameter DAG meet only through
  the parameter value, modeling/00). Distance/Radius/Diameter/Angle/ArcLength via
  auto-named model parameters; a driven dimension contributes no residual/variables
  and just reports `Measured()` (excluded from `Sketch.Constraints()`).
  `ConstraintLimits` + `Drive(v)` clamp for drive/animation. 3D variants
  (`DimensionConstraint3D`) share the `Constraint` interface. The sketch owns its
  own `param.Parameters` by default, swappable for the document's shared store via
  `SetParameters`. `make ci` green; sketch 97.8%.
- **2026-06-01:** M06-F03 (Geometric Constraints) complete. The `Constraint`
  interface exposes `Residuals() []float64` plus `Variables() []*math.Scalar`
  (DOFs by pointer) — so the solver is dimension-agnostic (2D points, 3D points,
  scalar radii all look alike). Full 2D set: Coincident/Horizontal/Vertical/
  Parallel/Perpendicular/Collinear/Concentric/EqualLength/EqualRadius/Tangent/
  Symmetry/Fix, each tested residual-zero exactly when satisfied; `GeometricConstraints`
  provides Add*/Delete/enumerate. Inference (PBI-071) is a pure ranked heuristic
  (`InferSegment` near-axis → H/V, `InferSnap` → nearest-point coincidence),
  separate from the solver, with apply-on-commit verified; the glyph overlay is
  UI-deferred. 3D variants (PBI-072): `Point3D` + Coincident3D/Collinear3D/
  Concentric3D/Equal3D/CustomConstraint3D share the same interface, so the F05
  Newton core will solve them unchanged. `make ci` green; sketch 94.1%.
- **2026-06-01:** M06-F02 (Sketch Entities) complete. Entities are built on shared
  constrainable `*Point`s — the solver's variable carriers — so a shared endpoint
  *is* a coincidence with no explicit constraint (modeling/00). `Line`/`Arc`/
  `Circle`/`Ellipse`/`Spline`/`Point` (construction flag on each curve), created by
  typed factory collections (`Lines`/`Arcs`/`Circles`/`Ellipses`/`Splines`/`Points`)
  bound to the sketch; `AllPoints` collects every solver variable. Splines support
  fit and control points; `BlockDefinition`/`BlockInstance`/`Blocks` give reusable
  groups whose instances track definition edits live (placed via `math.Matrix3`).
  Type names drop the redundant COM `Sketch` prefix (`sketch.Line`, not
  `SketchLine`) — the Go design in modeling/00, and it avoids the revive
  package-stutter rule. `make ci` green; sketch 98.1%.
- **2026-06-01:** M06-F01 (Sketch Infrastructure) complete — new `model/sketch`
  package. `Plane` is the planar sketch's host coordinate system (origin +
  orthonormal in-plane axes, normal = x×y) with `ToModel`/`ToSketch` mapping
  (round-trips; off-plane points orthogonally projected) and the three standard
  planes. A shared `base` carries id/name/edit-state/visibility/health for the
  three sketch kinds — planar `Sketch`, `Sketch3D`, and `DrawingSketch` — each
  with its collection. Projection (PBI-067) links model geometry through a kernel
  **seam** (`PointSource`/`CurveSource`, which topo implements in M07, same
  discipline as reference keys): `ProjectPoint`/`ProjectCurve`/`ProjectCutEdges`
  produce associative reference geometry that `Update` re-projects when the source
  moves and `BreakLink` freezes — so the solver/entity work (F02+) and the kernel
  stay decoupled. `make ci` green; sketch 96.6%.
- **2026-06-01:** M04-F04 (Core Events & Change Manager) **complete — M04 fully
  done.** The `Workspace` now owns an `event.Bus` (`Events()`) and emits the core
  event sets: lifecycle ops (Add/Open/Save/Close/SetActive/Quit) fire typed
  Before (vetoable) + After events — `DocumentCreated`/`DocumentOpened`/
  `DocumentSave`/`DocumentClose`/`DocumentActivate`/`ApplicationQuit` with stable
  TypeIDs. A Before handler returning `Veto` cancels the op (`VetoError`): a
  dirty-document close and an application quit can both be vetoed and the
  session is left intact; existing lifecycle tests are unaffected because an emit
  with no subscribers is `Continue`. ModelingEvents + change processing compose
  straight from the command+event primitives (no separate framework, core/06):
  `ModelChanged`/`ChangeDefinition` describe a committed edit batch, `Workspace.
  NotifyModelChanged` emits it (Before vetoable + After), and `ChangeManager`
  subscribes to the After phase and dispatches to registered `ChangeProcessor`s,
  with `Registration` giving per-processor enable/disable/unregister control.
  `make ci` green; doc 97.4%. **M04 complete: 4/4 features.**
- **2026-06-01:** M04-F03 (Event Infrastructure) complete — new pure `event`
  package: the typed bus that replaces COM connection points (core/06). One
  generic `Subscribe[E](bus, phase, handler)` (the XEventsObject/Sink split
  collapses to a single call) and `Emit[E]`/`EmitContext[E]`; in-proc dispatch is
  keyed by Go type + `Phase`, so a `Handler[DocumentClosing]` only ever sees that
  struct — no `VARIANT`, no 316 delegate types. `Phase` = EventTimingEnum;
  `Outcome`/`HandlingCode` (NotHandled/Handled/Abort) replace the `out
  HandlingCode` — a Before handler returns `Veto(reason)` and `Emit` aggregates
  the strongest disposition (keeping the first veto reason) so the caller cancels.
  The typed event struct replaces the `NameValueMap` context; `Context` still
  carries a `context.Context` to bound an add-in's veto reply. Concurrency-safe,
  with a handler snapshot so handlers may (un)subscribe mid-emit. `make ci` green;
  event 100%.
- **2026-06-01:** M04-F02 (Undo/Redo & Checkpoints) complete. `History.Undo`/
  `Redo` revert/re-apply committed `Batch`es over the done/undone stacks,
  restoring model state and firing one coalesced `OnChange` (the recompute seam):
  undo→redo returns to identical state, a new edit clears the redo stack,
  `UndoLabels`/`RedoLabels` are the enumerators, and both refuse to run while a
  transaction is open. Checkpoints follow core/06's "remember the history length"
  model — a `CheckPoint` is a captured depth + label, not a geometry snapshot;
  `GoToCheckPoint` undoes or redoes to that depth in one coalesced update and
  errors if the depth is unreachable; `CheckPoints`/`ReleaseCheckPoint` enumerate
  and dispose. `make ci` green; command 94.9%.
- **2026-06-01:** M04-F01 (Transaction Manager) complete — new `command` package
  implementing undo as the **command pattern** (core/06, realtime-3d §11), not a
  literal COM TransactionManager. `Command` is a self-contained reversible unit
  (Label/Apply/Revert, no document arg → a `Batch` can span documents, the COM
  "global transaction"); `Func` is a closure command (capturing prior state at
  Apply time so redo stays correct); `Batch` is the composite undo step with
  atomic Apply (a mid-batch failure rolls back the applied prefix). The COM
  vocabulary maps on: `History.Begin`/`Transaction.Commit`/`Abort` for
  start/end/abort, nested transactions fold their batch into the parent, and a
  bare `History.Do` issued while a transaction is open joins it (so history stays
  complete regardless of who edits). Abort reverts recorded commands in reverse
  for an exact pre-transaction restore; transaction labels are the undo-menu text.
  `MergeWithPrevious` combines two committed steps into one; `SuppressNotifications`
  + a single coalesced `OnChange` mean a 1000-edit transaction fires exactly one
  recompute/notification at commit (the seam the async recompute engine plugs into
  in M07+). `make ci` green; command 95.0%.
- **2026-06-01:** M03-F06 (Attributes & Metadata) **complete — M03 fully done.**
  New `model/attr`: the extensible metadata side-channel. `Value` is a typed
  tagged union (boolean/integer/double/string/bytes — the `ValueTypeEnum`, no
  `any`/`map[string]interface{}`). `AttributeSet`/`AttributeSets` give namespaced
  CRUD; the `AttributeManager` anchors an object's sets by `identity.RefKey` (its
  serialized bytes) so a face's attributes survive recompute — the rebuilt face
  re-mints an equal key and re-anchors — and reload (`Encode`/`DecodeAttributes`,
  the attributes.bin content); `FindAttributes` queries across objects by
  set/name. iProperties: `PropertySets` ships the four standard sets
  (Summary/DocumentSummary/DesignTracking/UserDefined), persist via
  `EncodeProperties`/`DecodeProperties`, and `ExposeParameter` bridges a
  `param.Parameter` into a custom property (provenance kept via
  `ExposedFromParameter`, round-trip-stable). `make ci` green; attr 93.8%.
  **M03 complete: 6/6 features.**
- **2026-06-01:** M03-F05 (Persistent Identity / Reference Keys) complete — the
  most load-bearing kernel mechanism, built early per the architecture's mandate
  (before features depend on selecting topology). New `model/identity`: a `RefKey`
  is an opaque, serializable **value** encoding an entity's *generative lineage*
  (e.g. "the top cap of Extrude#3"), never a pointer or array index — topological
  naming. `KeyManager` (=ReferenceKeyManager) mints (`GetReferenceKey`), rebinds
  (`BindKeyToObject`→`MatchType`), validates (`CanBindKeyToObject`), and persists
  key contexts (`SaveContextToArray`/`LoadContextToArray`, ids preserved) plus
  `KeyToString`/`StringToKey`. Because `kernel/topo` doesn't exist until M07, it is
  built against a topology **seam** — `Entity`/`Lineage`/`EntitySource` interfaces
  the real B-rep types implement later; today fakes exercise it. Verified: a key
  rebinds after a recompute that destroys and recreates the face (new object, same
  lineage), `CanBind` returns false when topology genuinely vanishes, and keys
  survive save→close→reopen via context save/load. PBI-043: new `model/health`
  holds the canonical `Status` vocabulary (ok/warning/sick/suppressed); the
  reference-loss policy lives once in `KeyManager.Resolve` — a lost reference
  yields `health.Sick`+`ErrReferenceLost`, fixable by re-selection, never a panic
  (wired to features/dimensions/mates from M08+). `make ci` green; identity 97.8%,
  health 100%.
- **2026-06-01:** M03-F04 (Document References) complete — added the document
  reference graph and project file resolution to `model/doc`. `RefGraph` is owned
  by the `Workspace` and shared by every document (each holds a back-pointer), so
  a document answers its own reference queries: `ReferencedDocuments` (an
  assembly's parts), `ReferencingDocuments` (the assemblies a part is used in),
  and the transitive `AllReferencedDocuments`. `DocumentDescriptor` records each
  reference by full document name with a needs-update flag and a reference-key
  placeholder (F05); resolution is lazy — already-open docs win, otherwise the
  target is loaded through the store — and a target that can't be found is
  **flagged broken, never fatal** (core/05). The graph maintains each doc's
  `referencedBy` count so `CloseAll(unreferencedOnly)` leaves referenced parts
  open. File resolution (`FileManager`) uses a portable project search-path model
  (`DesignProject` = workspace + library roots + `FileLocations`,
  `DesignProjectManager` tracks the active one): `Resolve` searches workspace then
  libraries, `Relativize` stores references portably, `TemplateFile` locates a
  `<type>.obk` template — no registry, no OLE monikers. `make ci` green; doc 97.2%.
- **2026-06-01:** M03-F03 (File Format & Storage) complete — new `persistence`
  package implementing `doc.Store`. The `.obk` document is a portable ZIP
  package (manifest + named binary streams), replacing COM's OLE structured
  storage with a cross-platform, inspectable container (core/05). `Package` is
  an ordered set of byte streams with typed `Manifest` access; `Save` is
  **atomic** — serialize in memory, `stage` to a sibling temp, fsync, then
  `commit` (rename) — so an interrupted save never corrupts the prior file
  (verified: a staged-but-uncommitted write leaves the live file byte-intact).
  `DataIO` reads/writes arbitrary client streams (add-ins, attributes →F06).
  `Migrate` runs on open: a manifest `schemaVersion` drives ordered migration
  steps keyed by from-version (`v0→v1` renames the legacy parameter stream
  losslessly), and a package from a newer build is rejected. `Compact` drops
  regenerable `cache/` streams (smaller archive, recipe untouched). `PackageStore`
  adapts all this into the `doc.Store` seam, so a workspace document now saves
  and reopens as a real file on disk. Scope note: streams are opaque blobs today;
  typed columnar `Codec[T]` serialization keyed by `TypeID` arrives with real
  model data (M07+). Exported `doc.Restore` for stores to rebuild a loaded
  document. `make ci` green; persistence 87.4%, doc 97.0%.
- **2026-06-01:** M03-F02 (Documents Collection & Open/Save) complete. Added
  `Workspace` to `model/doc` — the modernized `Documents` collection plus the
  Application's document ownership (core/02/05), holding open documents, the
  active document, and the (F04) reference-count hooks. Lifecycle: `Add`
  create-from-template (empty content, dirty until first save), `Open`/
  `OpenWithOptions` (typed `OpenOptions` replacing COM's `NameValueMap`;
  `DeferContent` registers a reference stub without touching the store),
  `Save`/`SaveAs`, `Close`/`CloseAll(unreferencedOnly, skipSave)` with
  save-or-discard dirty handling. Persistence is an injected `Store` seam —
  the workspace stays format-agnostic; the real zip-package backend is F03 —
  exercised here by a named `fakeStore`. A saved document reopens identically
  in a fresh workspace. `make ci` green; doc 97.0%.
- **2026-06-01:** M03-F01 (Document Model & Types) complete — new `model/doc`
  package. The document/content split (parametric-cad §1b): `Document` is the
  file/identity/lifecycle unit (session `ID` regenerated on load, not persisted;
  `FullDocumentName`/`FullFileName`/derived `DisplayName`; `Dirty` with
  `MarkDirty`/`ClearDirty`; `Open`/`Compacted`), the `Content` interface is the
  modeling payload. `NewReference` mints a reference stub — identity known,
  content nil, not open — so the reference graph (F04) can record a dependency
  without paging in the model. Four concrete specializations embed the base and
  expose a typed content stub (`PartComponentDefinition`/`AssemblyComponentDefinition`/
  `DrawingContent`/`PresentationContent`, filled in M07/M11/M14/M16), with a
  stable `DocumentType` discriminator (values 0–4, never renumber — persisted in
  manifests per core/05). `Compacted` is always false: the modern atomic
  write-temp-then-rename save leaves no slack to reclaim. `make ci` green; doc
  97.6%.
- **2026-06-01:** M02 (Units, Parameters & Expressions) **complete** — all 4
  features in one `model/param` package (Parameter/Expr/Graph are mutually
  dependent; splitting would cycle). F01 units+quantities (dimension signatures,
  `UnitsOfMeasure` parse/format). F02 expression engine (lexer + recursive-
  descent parser, unit-aware evaluator with a function library, constant
  folding, refs bound by stable `ID`). F03 parameter model (Expression→Value→
  ModelValue triad, kinds with read-only enforcement, `Parameters` collection,
  tolerance/precision). F04 dependency graph on `Parameters`: edges by id,
  cycle rejection at edit time (sick, not crash), topo dirty-propagation
  recomputing exactly the transitive dependents; rename preserves edges and
  rewrites dependents' display text token-aware. `make ci` green; param 90.2%.
- **2026-06-01:** M01-F04 (Geometry Utilities & Factory) complete — **M01 fully
  done.** Boxes in `math/` (per core/01): `Box`/`Box2d` (normalizing
  constructors, extend/contains/intersect/union/corners, empty-box union
  identity) and `OrientedBox` (axis-projection containment, corners, `ToAABB`).
  Closed-form queries in `kernel/geom/query.go` (plain functions, not a COM
  `GeometryUtilities` object): point→line/segment/plane closest-point & distance,
  plane projection + signed distance, line-plane intersection, line-line closest
  pair + intersection, 2D line-line intersection; added `math.IsNearZero` as the
  shared degeneracy guard. PBI-020: the COM factory is gone (core/03) — the
  package surface is the construction point, proven allocation-free for value
  types via `testing.AllocsPerRun == 0`. Deferred (documented): general numeric
  curve/surface intersection → kernel numeric phase (M06/M07). `make ci` green;
  math 95.5%, geom 93.5%.
- **2026-06-01:** M01-F03 (Transient Surfaces & Splines) complete. Added the
  `Surface` evaluation interface (`PointAt`/`DerivativesAt`/`NormalAt`/`UDomain`/
  `VDomain`) and the five analytic surfaces (Plane, Cylinder, Cone, Sphere,
  Torus) with exact normals, plus NURBS `BSplineCurve` (satisfies `Curve3`) and
  `BSplineSurface`: knot-span search + Cox–de Boor basis, first derivatives via
  the lower-degree recurrence, rational evaluation/derivatives via a homogeneous
  accumulator and the quotient rule. Constructors validate sizes/weights and
  deep-copy for immutability. Tests: analytic reference invariants
  (radius/normal/tube), metamorphic `NormalAt == normalize(∂u×∂v)`, partials vs
  finite difference for all surfaces, NURBS quarter-circle stays on the unit
  circle, bilinear patch matches its closed form. `make ci` green; geom 93.3%.
  Note: PBI-019's loft/sweep round-trip is deferred to M10 (no surface generator
  exists yet) — the evaluator correctness it depends on is covered now.
- **2026-06-01:** M01-F02 (Transient Curves) complete. New `kernel/geom/`
  package: ownerless immutable curve value types over `math/`, with `Curve2`/
  `Curve3` evaluation interfaces (`PointAt`/`TangentAt`/`Domain`). Lines &
  segments, circles & arcs (incl. by-three-points center/radius reconstruction
  in 2D and 3D via in-plane projection), full/partial ellipses (2D & 3D), and
  polylines (uniform-by-segment parameterization, vertices copied for
  immutability). Typed `CollinearPointsError`/`CollinearPoints3dError`. Key test:
  analytic `TangentAt` checked against a central finite difference for every
  curve type (catches missing chain factors). `make ci` green; geom coverage
  93.6%. Acceptance met: 3-point arc reproduces center/radius; ellipse major/
  minor ratio + axis directions honored; sweep/parameterization correct.
- **2026-06-01:** M01-F01 (Linear Algebra Primitives) complete. New `math/`
  package: `scalar.go` (Scalar alias, tolerances, helpers), `vector3/point3/
  unitvector3`, 2D counterparts, `matrix4` (+ `matrix4_transform` for rotation/
  coordinate-system/align constructors), `matrix3`, shared `mat_internal`
  (det3/invert3x3/mul3x3). All immutable; ops return new values. UnitVector
  enforces unit-length at construction and errors (with the offending magnitude)
  on a zero vector. `make ci` green: gofmt + vet + golangci-lint + race; math
  coverage 94.6%. Acceptance criteria met: ops match double-precision reference
  within tolerance; transform∘inverse = identity (TestMatrix4/3InverseRoundTrip).

## Design decisions made during implementation

- **2026-06-01:** M01-F01 lives in `math/` (not a `TransientGeometry` factory):
  the COM factory existed only to cross the interop boundary, which is gone
  (architecture core/03). Value types are plain immutable Go structs, `float64`.
- **2026-06-01:** Naming — internal Go types use idiomatic, grep-distinct names
  (`Vector3`/`Point3`/`UnitVector3`/`Matrix4`; `Vector2`/`Point2`/`UnitVector2`/
  `Matrix3`). Contract names (`Vector`/`Point`/`Matrix`/`Matrix2d`) live on the
  public gRPC surface (ADR-0006), so internal naming need not mirror COM.
- **2026-06-01:** Operations are **pure** (return new values) instead of COM-style
  in-place mutation (`TransformBy`, `Normalize`, `AddVector`), matching the
  immutable-value-type rule in architecture core/03.
