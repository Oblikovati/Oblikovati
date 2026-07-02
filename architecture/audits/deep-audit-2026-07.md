# Deep Code Audit — 2026-07-02

Six-dimension audit of `/Oblikovati` (~416k LOC) and `/Oblikovati.API` (~39k LOC):
architectural boundaries, algorithmic rigor vs literature, documentation debt,
generalization (generics), interface-based decoupling, and horizontal/vertical
slice consistency. Every finding below was verified by reading the code (several
numerically); each carries file:line evidence.

## 1. Executive summary

The load-bearing invariants hold: wire↔router↔client parity is 585=585=585 and
guard-enforced; kernel imports only `math`; the API module never touches the GPL
module; SPDX coverage is 100%; NURBS evaluation, point inversion, the adaptive
predicates, and the solver's QR/AD core are textbook-correct.

The debt concentrates in four systemic patterns:

1. **Silent degradation.** The kernel and model prefer degrading silently over
   failing loudly: the curved-boolean facet fallback is diagnosed but the default
   entry point discards the diagnostic (`rec = nil`); a tangency retry ships
   physically displaced geometry; sketch-copy drops 6 constraint types without a
   warning; mesh import weld failures are swallowed; a failed ear clip ships a
   hole. This is the same failure class as the M37 #1403 postmortem, and it is
   still the codebase's biggest source of latent corruption.
2. **Half-finished registry/interface migrations.** The "registry + interface
   beats twin switches" lesson was learned three times (feature codecs #1416,
   MutatingMethod #1426, feature-editor registry #1521) but never applied to
   sketch entities (~325 hand-mirrored type switches), sketch constraints (kind
   identity re-derived in 4 parallel switches — the #1574 class is structurally
   still open), or the sick-config commit gate (optional capability + hand list).
3. **Enforcement gaps around correct-today boundaries.** Add-in GPL purity,
   contract-assertion presence, wire-string re-declaration, and most inter-package
   edges (incl. an existing `model → persistence` inversion) are outside
   archguard/CI; they are clean today only by discipline.
4. **Tolerance-regime stragglers.** ADR-0042 relative tolerances are the design,
   but a producer/consumer mismatch (relative SSI tol vs absolute 1e-6 weld grid)
   cracks curved booleans on parts >10 cm, and a list of absolute epsilons
   (`revTol`, `quantCoord`, `trimBorderTol`, det/angle tests in `math/`) evade the
   tolerance-guard whitelist.

---

## 2. Top cross-cutting priorities

| # | Finding | Where | Class |
|---|---------|-------|-------|
| 1 | SSI extent collapses on closed/periodic surfaces (torus: 1.5e-14 vs ~120, verified numerically); tracer silently dies, degrades to marching squares | `kernel/geom/intersect_surface_trace.go:301-310` | silent kernel corruption |
| 2 | Boolean tangency retry translates operand B 1e-5 cm and ships the moved geometry | `kernel/brep/boolean.go:69-90` | wrong output coordinates |
| 3 | Planar boolean classifies by one fixed parity ray, no degeneracy handling; in-tree winding number (#1317) unused here | `kernel/brep/boolean_classify.go:15-43` | parity flips |
| 4 | Relative SSI tol vs absolute 1e-6 weld grid → cracked curved booleans >10 cm extent | `kernel/brep/boolean_split.go:93-94` | watertightness |
| 5 | `ops.Boolean` passes `rec = nil`; CSG-facet fallback diagnostic invisible to standard callers (the #1403 class) | `kernel/ops/boolean.go:47` | silent degradation |
| 6 | UI and wire paths enforce different invariants: `parameters.delete` guards in-use in the router, `Session.DeleteParameter` deletes unconditionally | `addin/router/parameters_detail.go:298` vs `app/parameters_edit.go:105-108` | divergent duplicate business rules |
| 7 | Entire graphics object model contract (~20 interfaces) unimplemented; its doc claims "compile-time asserted" in the host | `Oblikovati.API/contract/graphics_object_model.go:11` | dead public contract |
| 8 | Add-in "shipped code never imports GPL" is not checked in any add-in CI (clean today, verified) | `archguard/boundary_test.go:22-24`, add-in `ci.yml`s | license risk, unenforced |
| 9 | Sketch entities/constraints lack codec registry + kind interface; ~325 type-switch sites, runtime save-failure defaults | `model/sketch/sketch.go:45`, `serialize.go:330,389` | #1416/#1574 class |
| 10 | Point-cloud imports ignore units entirely (meshes/drawings scale, clouds don't) | `model/pointcloud/reader.go:28-38` | user-visible wrong geometry |

---

## 3. Kernel algorithms vs literature (agent: algorithm rigor)

### HIGH

- **A1. SSI extent estimate collapses on closed/periodic surfaces.**
  `kernel/geom/intersect_surface_trace.go:301-310` sizes the patch by the
  corner-to-corner distance; on a torus both domains are [0,2π] so corners
  coincide — numerically verified extent 1.49e-14 (true ≈120) for R=50/r=10 →
  step 6e-17, tol 1.5e-21, every seed fails, silent fallback to 96×96 marching
  squares. Cylinder path (`kernel/brep/curved_crossing_imprint.go:52-57`) gets
  axial-height-only extent → squat cylinders truncate loops at the 6000-step cap.
  *Standard:* size from sampled bbox / tangent bounds (OCCT `Bnd_Box`/adaptor
  resolution). *Fix:* 3×3–5×5 sample-bbox diagonal; regression test torus∩plane.
- **A2. SSI marching: no step-size control, no step-acceptance.**
  `intersect_surface_trace.go:121-147`: fixed 0.004·extent predictor, corrected
  point accepted unconditionally; branch-jumps near tangency undetected; loops
  smaller than one step untraceable; loop closure (`:137`) is
  proximity-to-start only (no tangent gate) → false closure in the #1404 pinch
  configuration. *Standard:* curvature step h≈2√(2ε/κ) with reject-and-halve
  (Bajaj–Hoffmann–Lynch; OCCT `IntWalk_PWalking` `StatusDeflection`); close on
  position AND direction.
- **A3. Planar boolean fragment classification: single sampled point + fixed ray
  `(0.5773,0.5774,0.5775)`, parity mod 2, near-parallel faces dropped, strict
  even-odd membership → parity flips near shared edges; no ray reselection.
  `kernel/brep/boolean_classify.go:15-43` (consumed `boolean.go:249-260`).
  `ops/point_in_solid.go` already has the winding-number fix (#1317).
  *Fix:* reuse winding number or per-connected-component flood-fill classification
  (Mäntylä/BOPAlgo).
- **A4. Tangency "nudge" ships displaced geometry.** `kernel/brep/boolean.go:69-90`:
  on a >2-edge-use stitch, re-run with all of B translated 0.1 µm, prefer the
  nudged result. Flush/tangent modeling silently misaligned; later features flip
  between coplanar and imprint paths → nondeterministic chains. *Standard:*
  symbolic perturbation never alters output coordinates. *Fix:* use nudged
  topology with original coordinates, or trust `resolveEdgeUses` as primary.
- **A5. "General" curved boolean is a recognizer table; outside its windows →
  permanent faceting, and the default API discards the diagnostic.**
  Gates: `kernel/brep/curved_general_boolean.go:97-106` (cone/cyl membership
  only), `:127-152` (two full-circle rims), `:231-242` (loops clear of caps);
  Steinmetz needs equal-R ⊥ axes. Everything else → `ErrNonPlanar` → tessellated
  CSG, analytic surfaces unrecoverable. `CodeBooleanCSGFallback` exists (#1407)
  but `ops.Boolean` (`ops/boolean.go:47`) passes `rec = nil`. Volume guard
  (`ops/boolean.go:211-227`) is one-sided — cut-removes-too-much passes.
  *Fix:* trim caps in their own 2D arrangement (largest decline window),
  winding-number membership, surface the fallback diagnostic at feature/API
  level, two-sided Requicha volume brackets.
- **A6. Mixed tolerance regime at the curved stitch.** Producer
  `intersect_surface_trace.go:288` (`ssiTolerance = 1e-7·extent`, ADR-0042-correct);
  consumer `kernel/brep/boolean_split.go:93-94` (`roundKey` absolute 1e-6 grid).
  Extent >10 cm → same SSI endpoint quantizes to different cells → seam fails to
  weld → non-manifold output. *Fix:* key weld on
  `geom.ResolutionForBox(operands).Weld()` (as `ops/sew.go` does).
- **A7. `revTol = 1e-7` absolute flips analytic surface TYPE**
  (`kernel/brep/revolution.go:16`, used `:127-145`): sub-µm tapers become
  cylinders; large-coordinate cylinders become cones → decline into the facet
  path (compounds A5). Not in the tolerance-guard whitelist. *Fix:* classify by
  slope ratio vs angular tolerance or `ResolutionForPoints(meridian).Plane()`.
- **A8. CDT is not a true CDT.** `kernel/ops/cdt_build.go:30-40,103-112` +
  `cdt_recovery.go:146-169`: corridor recovery flips only until the segment
  exists — no in-circle legalization afterwards; Bowyer–Watson then seeds from
  `firstBad` whose cavity may not contain the point in a folded mesh; inverted
  triangles caught only heuristically. *Standard:* Anglada/Sloan/Shewchuk —
  legalize after each `recoverSegment`, insert via constrained walk.
- **A9. Terminal curved-face fallback trusted blind.**
  `kernel/ops/tessellate_trim.go:154-180` + `earclip.go:110-113`: `earClip`
  "breaks with what we have" → partial triangulation ships a hole with no
  diagnostic. *Fix:* run `weldedFreeEdgeCount` acceptance
  (`refined_patch.go:58-87`) and emit `diag.Defect`.

### MEDIUM

- **A10. Variable-radius/G2 fillets are polyhedral strips** with C0 creases
  every ~11° (`kernel/ops/fillet.go:64-68,310`, `fillet_faces.go:34-90`);
  advertised G2 ships without G1. Cheap first fix: linear-taper on a straight
  edge IS a cone frustum — emit `geom.Cone`.
- **A11. All-pairs complexity cluster in the boolean pipeline** while the
  needed indexes exist in-tree: O(Fa·Fb) imprint with no AABB culling and the
  pairing computed 2–3× (`brep/boolean.go:60-67,133-144`); O(V·T) brute winding
  number (`ops/point_in_solid.go:23-32`); O(S²) arrangement
  (`brep/arrange2d.go:56-137`); O(m²) endpoint clustering in `ops/sew.go:70-84`
  (grid-hash built AFTER the quadratic pass). `ops/self_intersect_bvh.go` (#1411)
  unused by the boolean. *Standard:* BVH pair iterators, fast winding numbers
  (Barill 2018), Bentley–Ottmann/grid hash.
- **A12. NURBS inversion seeds from fixed 16×16 grid regardless of knot
  structure** (`kernel/geom/paramat.go:90-103`; 24 in
  `evaluator_surface_query.go:255-259`): >16-span imports converge to wrong
  foot → wrong signed distance → quadtree prune
  (`intersect_surface_seed.go:135-140`) discards cells with real crossings →
  whole SSI loop missed silently. Seed-safety 2.0 is a guess where the
  hodograph control-net bound (P&T §3.3) is cheap and rigorous. 2D curve
  intersection (`intersect2d_curve.go:21`) is 256-sample bracketing — even-
  multiplicity contacts invisible; Bézier clipping is the standard.
- **A13. Sketch solver corners vs solvespace:** redundant constraints counted,
  never identified (`solve/solve.go:240-253`; solvespace
  `FindWhichToRemoveToFixJacobian` does leave-one-out); rank via Gauss–Jordan
  with fixed 1e-7 pivot (`solve/linalg.go:121-170`; standard: CPQR/SVD);
  unscaled λ·I with ×10 steps and 8-try cap → false "stuck"/sick
  (`solve/solve.go:173-192`; standard: Marquardt scaling + Moré/Nielsen);
  FD fallback absolute h=1e-7 → phantom rank drop at large coordinates
  (`solve/solve.go:316-338`).
- **A14. Scale-blind CDT gates + guard evasion:** >256-vert area-mismatched
  earcut kept silently (`kernel/ops/planar_faithful.go:34,47,80` — cap stale
  since #1409); `conformance_repair.go:213-220` fixed 1e-6 grid (µm parts
  collapse); `tessellate_trim.go:25` `trimBorderTol=1e-6` (T-junctions on tiny
  parts); `ops/smooth_normals.go:69-76` `*1e5` reciprocal grid **evades the
  tolerance-guard regex** (matches only `1e-N` literals). *Fix:* derive from
  `ResolutionFor*`; harden `tolerance_guard_test.go:110`; extend whitelist scope
  to `topo/`, `brep/revolution.go`, `ops/wire_offset.go`, `ops/fill_*.go`.
- **A15. Math-layer numerics:** 3×3 inversion singular on |det| ≤ 1e-9 absolute
  (`math/mat_internal.go:17-27` — uniform scale 1e-3 falsely singular; standard:
  Hadamard-normalized test); acos-based angles have ~1e-8 rad noise floor vs
  `AngleTolerance` 1e-9 (`math/vector3.go:71-78`; use Kahan atan2 form); wire
  closure/planarity absolute 1e-7 (`topo/wire.go:67`, `ops/wire_offset.go:111`);
  knot removal compares geometric tol against homogeneous 4-space distance
  (`geom/nurbs_homog.go:70-74`; P&T eq. 5.30 correction).

### Verified sound (do not "fix")

NURBS evaluation core (P&T A2.1–A5.9, eq. 4.8/4.20, A9.1/A9.4 exact);
`geom/project.go` inversion (proper §6.1 Newton, both stops, backtracking);
`math/predicate` (correct Shewchuk staged filters + big.Rat fallback);
solver linear algebra (Householder QR on augmented system, forward AD
cross-checked vs FD); coplanar ON/ON booleans (Mäntylä §12 — delete the stale
"follow-up" comment at `brep/boolean.go:43`); non-manifold radial-edge azimuth
pairing; shared-edge discretization architecture (`ops/edge_discretize.go`);
Euler–Poincaré validation; `geom/resolution.go` + tolerance-guard design.

---

## 4. Architectural boundaries (agent: architecture)

- **B1 (MAJOR). Business rules in the router with divergent duplicates in app.**
  Router package doc embraces touching the model directly (`addin/router/router.go:7`),
  contradicting the thin-adapter rule. Live divergence: `parameters.delete`
  guards in-use (`addin/router/parameters_detail.go:285-307`) while
  `app.Session.DeleteParameter` (`app/parameters_edit.go:105-108`, from
  `head/ui/parameters_row.go:149-150`) deletes unconditionally ("dependents go
  sick"). More policy-in-router: `parameter_groups.go:108-112` (name-nonempty +
  direct field mutation), `assembly_features.go:82-105` (self-machining rule,
  naming policy, recompute sequencing). *Fix:* per-aggregate application
  services both paths call — or amend the rule and test that UI+wire share the
  mutation seam. Reconcile parameter-delete regardless.
- **B2 (MAJOR). Graphics object model contract dead.**
  `Oblikovati.API/contract/graphics_object_model.go:11` claims host-side
  compile-time assertions; zero implementations/assertions exist anywhere for
  its ~20 interfaces. Contract-first shipped without step 2; Apache surface
  advertises capabilities the host lacks. *Fix:* implement in `clientgraphics/`
  (which already asserts `contract.ClientGraphics`) or mark pending + guard test.
- **B3 (MAJOR). Add-in GPL purity unenforced.** `archguard/boundary_test.go:22-24`
  defers to add-in CI; MCPBridge/MotorDesigner GPL imports verified test-only
  today, but no `ci.yml` runs any purity check (MCPBridge
  `ci.yml:16-21` asserts it in a comment only). *Fix:* `go list -deps` guard per
  add-in repo.
- **B4 (MAJOR). model → persistence inversion.** `model/compdef/serialize.go:18`,
  `model/material/{catalog,store}.go`, `model/drawing/recipe.go` import
  `persistence/yamlcodec` while `persistence/` imports `model/doc` — mutual
  dependency. *Fix:* move `yamlcodec` to a neutral leaf; add "model must not
  import persistence" to archguard.
- **B5 (MAJOR). Feature-args DTOs live host-side only.** `wire.AddFeatureArgs.Args`
  is `json.RawMessage` (`Oblikovati.API/wire/features.go:26`); real schemas are
  unexported in `addin/opregistry/` (~7.5k LOC). Add-ins must hand-assemble raw
  JSON for the richest API slice — "never raw JSON" structurally impossible.
  *Fix:* promote per-kind arg structs into `api/wire` (or `wire/featureargs`);
  keep runtime schema reflection for dynamic kinds only.
- **B6 (MAJOR). init()-time global registries as implicit dependencies:**
  `model/feature/serialize_registry.go:36` (79 codecs via 7 init()s),
  `model/doc/content.go:39` contentFactories (document open depends on linkage),
  `app/feature_edit_registry.go:34`, `head/ui/dockable_panel.go:41`,
  `head/ui/theme_apply.go:22-44` (~12 mutable color arrays). *Fix:* construct
  registries in a composition root (`app.NewSession` receives codec/factory
  sets); theme cache becomes an object owned by the UI loop.
- **B7 (minor). Wire strings re-declared as literals:** router error contexts
  (`parameters_detail.go:101,115,192,262,293,321,325`,
  `parameter_groups.go:83,103,117,121`); add-in event literals
  (`oblikovati-meeting/meeting/client.go:115,122`); MCPBridge
  `bridge/resources.go:29-49` (12 literal methods → raw `Call`). *Fix:* use
  `wire` constants + CI grep.
- **B8 (minor). 35/167 contract interfaces lack assertions** — ~20 are B2;
  `ModelStates`, `Representation`, `Curve`, `Surface`, `AssemblyConstraint`,
  `AddInAutomation`, `Curve2d` implemented but only implicitly checked. *Fix:*
  add the seven assertions; archguard diff of contract vs assertion sites.
- **B9 (minor). Parallel value-type definitions** (`math.Box`/`OrientedBox` ×2 vs
  `types.Box`; `model/assembly/drive.go:28,36` vs `wire/assembly_drive.go:35,50`;
  `model/sheetmetal/settings.go:8` vs `wire/flat_pattern.go:128`) maintained by
  hand converters. *Fix:* document math↔types duality as deliberate; add
  round-trip conversion tests for the identical model/wire pairs.
- **B10 (minor). Router coupled to renderer internals** (`addin/router/lighting.go`
  imports `renderer` at :38,113,189-228; `camera.go:61-62` uses `scene.Camera`);
  root cause: `app.Session` leaks `renderer.*` in signatures. *Fix:* app-owned
  lighting/environment value types.
- **B11 (minor). Raw `gopkg.in/yaml.v3` outside the wrapper:**
  `app/bug_report_diag.go:9` (:133), `release/version.go`. *Fix:* route through
  `yamlcodec` or exempt diagnostics explicitly.
- **B12 (meta). archguard coverage gaps:** domain-purity guard covers only
  kernel/model/math/solve; unguarded: app, command, event, persistence,
  clientgraphics, script, non-router addin/*; wire-string re-declaration;
  contract-assertion presence; add-in purity. *Fix:* edge-allowlist table over
  the `go list` graph (~20 edges).

**Verified clean:** dependency direction (API never imports GPL); no re-declared
DTOs anywhere (862 wire structs checked); all 585 registrations keyed on
`wire.Method*`; kernel imports only math+build+api/types; app is NOT a god-layer
(geometry delegated to model/kernel; its math is picking/interaction); SPDX 100%
both modules; alias discipline real (32 `type X = types.X` sites).

---

## 5. Interfaces & decoupling (agent: decoupling)

House precedents to replicate (each fixed a shipped bug):
`model/feature/serialize_registry.go:7-14` (#1416), `addin/router/router.go:37-47`
`MutatingMethod` (#1426), `app/feature_edit.go:181-199` editor registry (#1521).

- **I1. Sketch entities duck-typed behind 1-method `Entity`** (`model/sketch/sketch.go:45`);
  229 `case *X:` in-package + 96 in 9 files outside (router enumerate/edit, app
  constraint/trim tools, compdef rebind). Serializer default = runtime save
  failure (`serialize.go:330`). *Fix:* entity codec registry (mirror #1416) +
  `ShapedEntity{Kind, ShapePoints}` capability; incremental, registry first.
- **I2. Constraint kind re-derived in 4+ parallel switches**
  (`model/sketch/serialize.go:345-389` default = save error;
  `addin/router/sketch_constraints.go:99,144,159,176`;
  `sketch_enumerate.go:197`; app + 3D twins). Nothing links the sides — #1574
  structurally open. *Fix:* `KindedConstraint{ConstraintKind, RelatedEntities}` +
  factory registry + one create→enumerate→save→load closure test over the enum.
- **I3. Sick-config gate hangs on optional capability** (`app/commit_gate.go:28`
  type-asserts `DraftPreviewable`; non-implementing tool skips the gate; the
  only guard is the hand list `app/feature_preview.go:31-44`). *Fix:*
  `PartFeatureTool interface { Tool; DraftPreviewable }` +
  `StartFeatureTool` — bypass-by-omission becomes impossible.
- **I4. head/ui dialog roll-call:** `head/ui/chrome.go:105-139` calls 30+
  bespoke `drawXDialog(s)`; each typed on the concrete tool
  (`chamfer_dialog.go:19,58`). New tool minus chrome.go line = headless-invisible
  (#1521 shape). Generic `tool_params_dialog.go:13-16` exists but bespoke
  dialogs bypass it. *Fix:* dialog registry; key by tool so a registered
  feature-tool without a dialog fails a startup check.
- **I5. head/ui ↔ `*app.Session` god coupling:** 582 references across 242
  files; only two narrow seams exist (`viewcube_arrows.go:164`,
  `viewport_cache.go:96`). *Fix:* policy — each touched widget declares its
  ≤6-method consumer interface (house pattern `arrowSession`); no big bang.
- **I6. `kernel/geom` closed-set classification via 77 `case geom.X:` switches**
  (geomapi/evaluators.go ×9, brep/curved_stitch.go ×9, ops/fill_opening.go ×6,
  step/geommap ×9, feature/work_plane_tangent.go ×6, …). Analytic special-casing
  is legitimate (OCCT does it) — make it checkable: `Kind()` discriminators +
  enumerating per-consumer coverage test; `map[SurfaceKind]convertFn` for pure
  translators. Prevents the #1403 silent-default class.
- **I7. `app.Selectable` is 1 method; 39 head/ui switches on concrete handles**
  (`browser_node_decor.go:101,122,144`, …). *Fix:* `BrowserDecor()`,
  `ContextActions()` capability methods on handles.
- **I8. Exchange seam adopted by only 2 formats; format routing = 2 independently
  maintained switches** (`model/exchange/dispatch.go:73-80,113-126`; PDF entry
  shape differs from DWG/DXF). *Fix:* `DrawingDecoder` interface + one
  extension registry replacing both switches.
- **I9. Contract god-interfaces:** `TransientGeometry` 30 methods
  (`contract/transient_geometry.go:20`), Document 15, DisplaySettings 15,
  SurfaceEvaluator 13, FileDescriptor 13… *Fix (semver-safe):* split into
  embedded families (`TransientPoints`/`TransientCurves`/`TransientMatrices`),
  keep fat name as union.
- **I10. Optional capabilities lack completeness checks:** `pointDefined`
  (`model/sketch/drag.go:41-48`) covers 6 kinds, silently nil for splines; same
  for `SmoothCurve`, `CircularCurve`, `sourceKinded`, `idCarrier`. Only 86
  `var _ Iface =` assertions repo-wide. *Fix:* assertion blocks for
  supposed-total capabilities; registry-driven coverage-table test for
  deliberately-partial ones.
- **I11. Work-plane redefine re-switches on definition types the capability
  interfaces already abstract** (`work_plane_redefine.go` ×5,
  `work_plane_tangent.go` ×2 vs `work_axis_point.go:15,166` interfaces).
  *Fix:* `Redefine(inputs)` on the definition interface — create and redefine
  share one dispatch.
- **I12. `Tool` methods take the whole `*Session`** (`app/tool.go:10-25`) — the
  seam is minimal but its parameter is a god type; no tool testable against a
  slim host. *Fix:* `ToolHost` (~8 methods) satisfied by `*Session`, introduce
  via `PartFeatureTool` to avoid churn.

**Already right (don't touch):** feature codec registry, MethodHandler/
MutatingMethod, feature-editor registry, `doc.Store` consumer-side, Picker/
RegionPicker, `workHost` (router), BodyImporter+assertions, neutral drawing
model, solver `Residual`/`Differentiable`.

---

## 6. Slice consistency (agent: slices)

Matrices (summary): 74 feature kinds — definition/validation ~98%, preview 100%,
undo 100% of geometry ops, persistence ~95%, provenance ~90%, **edit-after-create
~25%**, assembly counterpart ~48%. 22 constraint types — all creatable/solvable/
persisted; **sketch-copy carries only 16**. Importers table: units handled
everywhere except point clouds; warnings everywhere except meshes; progress
nowhere.

1. **Point clouds ignore units** (`model/pointcloud/reader.go:28-38`, `las.go:23-29`;
   meshes scale via `kernel/exchange/meshio/units.go:10-35`, drawings via
   `model/exchange/drawing_units.go:61-66`). PLY-as-mesh scales, PLY-as-cloud
   doesn't. *Fix:* thread target-unit scale like meshio `ImportScale`.
2. **Sketch-copy drops Smooth, Ground, Offset, PatternLink, Custom,
   TextBoxAnchor** silently (`model/sketch/copy_constraints.go:100-138`) — G2
   joins lost, DOF changes silently. *Fix:* cases (or documented skips) +
   enum-driven completeness guard.
3. **Mesh import fails silently where drawings warn** (`meshio`
   `SolidOrSurface()` weld/validation failures vs `dwg/decode.go:42-53` per-entity
   warnings). *Fix:* warnings into ImportResult.
4. **Edit-after-create raggedness:** loft = zero Editable methods; face-/full-
   round-fillet, grill = none (base fillet full — `model/feature/editable.go:222,415`);
   boss params-only (`editable_m10.go:74`) vs hole full; 15/16 sheet-metal none;
   8/10 surface features none. *Fix order:* loft, fillet variants, grill.
5. **Body color-styles session-only** (`app/style_assign.go:14-28`) — no
   persistence/event/undo; appearance work vanishes on close. Material
   assignment similar.
6. **Persisted-but-not-undoable metadata** (`model/doc/document.go:268-278`
   `SetBodyName` only MarkDirty) — body names/sketch settings/display settings
   outside the recipe snapshot. *Fix:* include doc-metadata in undo snapshot.
7. **Sick-config gate OK-button bypasses:** `head/ui/feature_edit_dialog.go:69-80`,
   `command_window.go:119-124`, `sheet_canvas_input.go:61-69` (safe at
   `Session.OK()` but inconsistent button states). *Fix:* route through
   `drawCommitCancelButtons` (`dialog_buttons.go:23-42`).
8. **No 2D-constraint coverage guard** (3D has
   `addin/router/sketch3d_constraints_coverage_test.go`; the missing 2D twin is
   exactly what would have caught #1574 and finding 2).
9. **106/167 contract interfaces unasserted** (see B8/B2; slice agent verified
   61 assertions — count differs from arch agent's 132 because arch counted
   client-side + typed-return implicit checks; the enforceable gap list is B8's).
10. **Metadata mutations emit no events** (body/sketch/feature/occurrence
    rename, suppression, sketch-settings — `addin/router/bodies.go:96`) while
    feature-lifecycle/parameters/occurrences emit. Add-ins can't observe
    renames. *Fix:* generic `object.renamed`/`property.changed` event.
11. **No post-import camera fit on ANY path** (drawings get recentering,
    `model/exchange/recenter.go:22-38`, but no ZoomAll; meshes/clouds can land
    off-screen → "import did nothing"). *Fix:* fit-view after imports adding
    visible geometry.
12. **Point clouds are a bolt-on vertical** (not in `FormatFromPath`,
    `model/exchange/dispatch.go:112-131`; all-or-nothing errors; dual `.ply`
    routing undocumented).
13. **Importer progress uniformly absent** — one shared `ProgressFunc` seam
    fixes all formats.
14. **assemblySweep not editable** while assemblyExtrude/Revolve are
    (`model/feature/assembly_editable.go:20-25`) — looks like omission, not scope.
15. **Confirmed-open persistence gaps:** origin auto-projection (#1262);
    client/interaction graphics .obk; named views workstation-local only.

**Verified healthy:** 585=585=585 wire/router/client parity guard-enforced; 64
events all relayed; constraint YAML round-trip complete; all DraftPreviewable
creation dialogs honor the gate; undo covers part+assembly docs.

---

## 7. Generalization / generics (agent: generics)

Only 37 generic functions in 416k LOC; stdlib `slices`/`maps` imported 4×.
In-tree precedents to copy: `event.Bus`, `twoRefArgs[T]`, `buildKeyIndex[T]`,
`enumFromName[E]`, `removeItem[T]`.

1. **Router/opregistry generic typed-handler adapter (~1,500–2,000 LOC):** 316
   `var in wire.X`, 406 `decode(args,&in)`, 466 `return json.Marshal`, 96
   ActivePart preambles across 283 files (e.g. `router/work_surfaces.go:36-69`).
   `typed[Args,Result]`/`typedPart[...]` → handlers become typed functions;
   helps funlen lint. Medium-low risk; land per method-group file.
2. **API client `call[Resp]` (~580 LOC):** 594 call sites × 3-line body
   (`client/work_points.go:20-23`). Very low risk — verify the MCP-bridge
   generator reads signatures, not bodies, before landing.
3. **Generic YAML `FileStore[T]` (~250–300 LOC):** six byte-identical stores
   (`persistence/dialogmemory`, `userprefs`, `viewstate`, `addinstate`,
   `app/keymap`, `app/options`) + 3 duplicated MemStore fakes.
4. **Builtins adoption (~300 LOC / ~90 funcs):** 31 hand-rolled min/max
   (Go 1.21 builtins), ~10 `abs` copies (`math.Scalar = float64`, no conversion
   needed), 47 clamps in 8 shapes → `math.Clamp[T cmp.Ordered]` + `Clamp01`.
5. **`slices.Contains` adoption (~80–100 LOC):** ~12 contains helpers incl.
   pointer-equality ones; keep semantic `containsLoop`.
6. **`math.Lerp` + Point2/Point3 `Lerp` methods (~60–80 LOC, 16 sites)** — also
   removes a+(b−a)t vs a+t(b−a) rounding inconsistency.
7. **Index-guarded read-only collection view (~120–150 LOC):**
   `model/assembly/representation_contract.go:31-69` ×4 + ~30 `Item` guards.
   Contract interfaces stay untouched.
8. **Sketch `entityList[T]` storage core (~50 LOC)** — finish the started
   thought in `entity_collections.go`.
9. **`selectables[T]` widening (~15 LOC):** `app/selecting.go:53-70`.
10. **Router list-projection `projectAll` — fold into #1.**

**Anti-recommendations (deliberate non-generics):** wire DTOs (versioned API
stability surface); contract collection interfaces (COM-style object model is
the design; genericize implementations only); `math` Vec2/Vec3/Point/Box (field
count can't be abstracted without hurting the hottest paths); `head/viewport`
float32 helpers (ADR-0026 §6 self-consistency); already-solved: event.Bus,
feature codecs, enum parsing, client fakes.

---

## 8. Documentation debt (agent: docs)

Two refactorings left the trails; all label drift, not conceptual rot (~1 day of
edits):

1. **core/05-documents-persistence-identity.md:71-86** describes retired
   `KeyManager` + dead `identity/keys.bin` as the live design → rewrite around
   `identity.RefKey` (`model/identity/refkey.go:13-40`) + tiered bind/recover.
2. **mapping/com-to-go-cheatsheet.md:31** maps ReferenceKeyManager → dead
   `KeyManager.Resolve` → update to RefKey path.
3. **`PartContent`/`AssemblyContent` don't exist** (modeling/01-feature-engine.md:11,
   core/05:20) → `PartComponentDefinition` etc.
4. **Stale `/source` paths in ~10 files:** lua-scripting-plan.md:8,60,315-320;
   core/07-extensibility.md:81,99; ADR-0018 (title + ~15 refs — the ADR
   CLAUDE.md points at!); ADR-0015:22,46; ADR-0016:12; ADR-0028:93; ADR-0006:11;
   ADR-0023:53; history/implementation-log.md:28. Fix living docs; editor's-note
   banner for ADRs.
5. **API wiki claims curved booleans are mesh-CSG fallback**
   (`Oblikovati.API/docs/wiki/Oblikovati-Architecture.md`) — pre-ADR-0027 world;
   ADR-0045 (2026-07-01) closed the EPIC.
6. **ADR-0027 status stale** ("in progress") → accepted/implemented, pointer to
   ADR-0045. (Otherwise ADR hygiene excellent: 45/45 status lines, contiguous.)
7. **API README `/source` references** (also violates the no-internal-jargon
   rule for the public repo).
8. **RELEASING.md:98** → `.goreleaser.yaml` (no `source/` prefix).
9. **implementation-conventions.md:69** "`ReferenceKeyManager`" →
   `identity.RefKey`.
10. **API packages story told 3–4×** (README ≈ wiki ≈ doc.go ≈ CLAUDE.md,
    already drifting) → wiki canonical, README shrinks to pointer+quickstart.
11. **history/implementation-log.md** → add "historical snapshot" banner.
12. **7 large packages lack doc.go** (addin/router 21k, head/ui 18.7k,
    kernel/exchange 14k, **api/wire 10k — public surface**, kernel/brep 9.5k,
    addin/opregistry 5.8k, model/drawing 5.2k) — promote existing inline
    comments.
13. **ADR-0020 schema version drift** (says v2, shipped .obk is v3) — one-line
    amendment.

**Verified NOT stale:** CLAUDE.md layout, ADR numbering, assembly/instancing doc,
lua plan phase statuses, kernel layout doc, API CHANGELOG, docs/wiki pointers.

---

## 9. Suggested sequencing

1. **Stop the silent corruption (kernel):** A1 extent fix + torus∩plane
   regression; A6 weld-key from resolution; A5 diagnostic plumbing
   (`rec` default non-nil, surface `CodeBooleanCSGFallback` at feature/API);
   A3 winding-number classification; A4 nudge → topology-only.
2. **Close the enforcement gaps (cheap, one PR each):** B3 add-in purity CI
   guard; B8 seven assertions + archguard contract diff; B12 edge-allowlist;
   2D-constraint coverage test (S8); tolerance-guard regex hardening (A14).
3. **Reconcile the live behavior bug:** B1 parameter-delete divergence.
4. **Finish the started migrations:** I1/I2 sketch entity+constraint registries;
   I3 PartFeatureTool; B6 registry injection (composition root).
5. **User-visible slice holes:** S1 point-cloud units; S2 sketch-copy
   constraints; S4 loft/fillet-variant/grill editability; S5 style persistence;
   S11 post-import fit.
6. **Scaffolding shrink (mechanical):** G1 router `typed[...]`; G2 client
   `call[Resp]`; G3 FileStore[T]; G4–G6 stdlib adoption.
7. **Docs day:** findings D1–D9.
