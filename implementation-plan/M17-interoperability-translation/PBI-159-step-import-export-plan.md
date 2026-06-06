<!-- SPDX-License-Identifier: GPL-2.0-only -->
# Implementation Plan: STEP (ISO 10303) Import/Export — AP203 / AP214 / AP242

> Produced by the planning pass for PBI-158 (translator framework) + PBI-159 (STEP). Grounded in
> the actual kernel/model/persistence/contract code. Realizes M17-F01/F02.

## 0. Grounding summary (what already exists)

- **B-rep topology** (`kernel/topo/`): `Body` → `Shell` → `Face` → `Loop` → `EdgeUse` → `Edge` →
  `Vertex`, assembled via `topo.Builder` (`NewBuilder`, `AddVertex`, `AddEdge`, `AddFace`/
  `AddReversedFace`, `OuterLoop`/`InnerLoop`, `Fwd`/`Rev`). `Shell` carries `closed bool`; `Body`
  carries `solid bool`. Identity is `Lineage` (`LineageToken{Feature, Role, Index}`) →
  `ReferenceKey() []byte`. `FindFaceByKey/FindEdgeByKey/FindVertexByKey` rebind keys after recompute.
- **Geometry** (`kernel/geom/`): `Surface` (`PointAt/DerivativesAt/NormalAt/UDomain/VDomain/ParamAt`)
  = `Plane`, `Cylinder`, `Cone`, `Sphere`, `Torus`, `EllipticalCone/Cylinder`, `BSplineSurface`.
  `Curve3` (`PointAt/TangentAt/Domain`) = `Line`/`LineSegment`, `Circle`, `Arc3d`, `BSplineCurve`,
  `Helix`. Value-returning constructors with error (`NewCylinder`, `NewCone`, `NewBSplineSurface`).
  Analytic structs expose an explicit frame (Origin/Apex/AxisDir/Ref) ≈ STEP `AXIS2_PLACEMENT_3D`.
- **Validation/mass** (`kernel/ops/`): `Validate(b) ValidationReport{Valid,Manifold,Closed,
  OrientationOK,Issues}`; `BodyGeometryProperties`; `Mesh`/`Tessellate`. `Sew` is **stubbed**.
- **Model spine**: `model/compdef.PartComponentDefinition` owns `SurfaceBodies()`, `Recompute()`,
  `Units()`. `model/feature.DerivedPartComponent` is the precedent for a feature that injects
  pre-built `*topo.Body` — the model for an imported-body feature.
- **Persistence/IO**: `persistence/io.go` + `persistence/yamlcodec` (`.obk`, ADR-0020) is the ONLY
  codec; no foreign-format I/O yet. `model/doc.Workspace` (`Open`/`SaveAs`); CLI `cmd_*.go`.
- **Public-API seam**: `Oblikovati.API` `types`/`contract`/`wire`/`client`; host serves `wire` via
  `addin/router` (`handlers map[string]handlerFunc` keyed on `wire.Method*`). No
  `documents.import/export` yet.
- **Planning context**: `implementation-plan/M17-interoperability-translation/` scopes F01
  (TranslatorAddIn framework), F02/PBI-159 (STEP+IGES), F04 (mesh exchange).

## 1. Scope & background — STEP for a B-rep kernel

ISO 10303-21 ("Part 21") clear-text: `HEADER` (`FILE_DESCRIPTION`, `FILE_NAME`, `FILE_SCHEMA` —
names the AP: `CONFIG_CONTROL_DESIGN`=AP203, `AUTOMOTIVE_DESIGN`=AP214,
`AP242_MANAGED_MODEL_BASED_3D_ENGINEERING`) + `DATA` of `#id = ENTITY(args);` forming a ref graph.
We need the Part 21 syntax + a fixed entity subset, NOT a general EXPRESS interpreter.

B-rep import entity subset:
```
PRODUCT / PRODUCT_DEFINITION / PRODUCT_DEFINITION_SHAPE
SHAPE_REPRESENTATION / ADVANCED_BREP_SHAPE_REPRESENTATION
  MANIFOLD_SOLID_BREP → CLOSED_SHELL → ADVANCED_FACE*
    ADVANCED_FACE → FACE_OUTER_BOUND/FACE_BOUND → EDGE_LOOP → ORIENTED_EDGE → EDGE_CURVE
      EDGE_CURVE(v1,v2,curve,same_sense); VERTEX_POINT → CARTESIAN_POINT
    surfaces: PLANE, CYLINDRICAL/CONICAL/SPHERICAL/TOROIDAL_SURFACE, B_SPLINE_SURFACE_WITH_KNOTS(+RATIONAL)
    curves: LINE(+VECTOR/DIRECTION), CIRCLE, ELLIPSE, B_SPLINE_CURVE_WITH_KNOTS
  AXIS2_PLACEMENT_3D; units: GLOBAL_UNIT_ASSIGNED_CONTEXT + SI_UNIT/CONVERSION_BASED_UNIT,
  UNCERTAINTY_MEASURE_WITH_UNIT
```
The geometric↔topological correspondence (EDGE_CURVE.same_sense, ORIENTED_EDGE.orientation,
ADVANCED_FACE.same_sense) maps onto our `Edge.curve` + `EdgeUse.reversed` + `Face.reversed`.

Per-AP additions over AP203: **AP214** adds assemblies (`NEXT_ASSEMBLY_USAGE_OCCURRENCE` +
transforms), colours (`STYLED_ITEM`→`COLOUR_RGB`), layers. **AP242** supersedes 203/214 (same
geometry) + PMI/GD&T, tessellated geometry (`TESSELLATED_SHELL/SOLID`). **One core reader/writer
serves all three**; AP214/242 are additive passes.

## 2. Architecture & module layout

### 2.1 Decision: hand-write the Part 21 parser (no vendored EXPRESS lib)
Licensing is decisive — `Oblikovati.API` is Apache-2.0 and add-ins may be closed-source; mature
STEP stacks are GPL/LGPL or cgo-heavy (violates the cgo-free, headless-tested core, ADR-0008).
Part 21 grammar is small/stable → a tokenizer + recursive-descent parser fits the <500-line/
4–20-line budget. Keep a `StepReader`/`StepWriter` owned interface so a vendored parser could slot
behind it later for parsing only.

### 2.2 Module layout (GPL `/Oblikovati`, new tree `kernel/exchange/step/`, pure Go, headless)
```
kernel/exchange/translate.go        # owned interfaces: BodyImporter/BodyExporter, TranslationOptions
kernel/exchange/step/
  part21/{token,parser,graph,header,writer}.go   # lexer, recursive-descent, EntityGraph, HEADER, emitter
  schema/{ap,keywords}.go                          # ApProtocol enum + schema-string ↔ AP; entity keywords
  geommap/{surface,curve,placement}_{from,to}_step.go   # geom.Surface/Curve3 ↔ STEP
  topomap/{brep_from_step,brep_to_step,lineage}.go      # CLOSED_SHELL/ADVANCED_FACE ↔ topo.Builder
  units.go                                          # SI_UNIT/CONVERSION ↔ param units; scale
  reader.go / writer_facade.go                      # StepImporter/StepExporter (impl BodyImporter/Exporter)
  ap214/ap214.go  ap242/ap242.go                    # colours/assemblies; PMI/tessellation (additive)
```
Model wiring: `model/feature/imported_body.go` (`ImportedBodyFeature`, modeled on `derived.go`);
`model/exchange/{import,export}_step.go` (orchestration touching `doc.Workspace`/`compdef`); CLI
`cmd/oblikovati-cli/cmd_{import,export}.go`.

### 2.3 Contract-first split (MANDATORY, ADR-0018) — API FIRST
- `api/types/exchange_format.go` — `ExchangeFormat` enum (`FormatStepAP203/214/242`, future IGES/STL),
  `TranslationUnit`. Pure value types (no parser).
- `api/contract/translator.go` — `Translator` (`CanImport/CanExport`), `TranslationContext`.
- `api/wire/exchange.go` — `MethodDocumentsImport="documents.import"`, `MethodDocumentsExport=
  "documents.export"` + DTOs `ImportRequest/Response`, `ExportRequest/Response`.
- `api/client/exchange.go` — typed `Import/Export` over `Transport`.
- THEN GPL impl: `var _ contract.Translator = (*step.Translator)(nil)`; `addin/router/exchange.go`
  keyed on the wire constants; MCP bridge auto-exposes them.

## 3. Entity ↔ kernel mapping (both directions)

### Surfaces
| STEP | geom | notes |
|---|---|---|
| PLANE | Plane | placement → Origin/UAxis/Normal |
| CYLINDRICAL_SURFACE | Cylinder | exact |
| CONICAL_SURFACE | Cone | derive Apex from base radius + half-angle |
| SPHERICAL_SURFACE | Sphere | exact |
| TOROIDAL_SURFACE | Torus | major/minor R |
| B_SPLINE_SURFACE_WITH_KNOTS(+RATIONAL) | BSplineSurface | expand knots+mults; weights if rational |
| OFFSET/REVOLUTION/EXTRUSION/general BOUNDED | **GAP** | fallback: sample→BSpline, or tessellated face (AP242); warn |

### Curves
| STEP | geom | notes |
|---|---|---|
| LINE | Line/LineSegment | trim to edge verts |
| CIRCLE | Circle/Arc3d | full vs bounded |
| ELLIPSE | Ellipse/Arc | else bspline fallback |
| B_SPLINE_CURVE_WITH_KNOTS(+RATIONAL) | BSplineCurve | knots+mults, weights |
| POLYLINE/COMPOSITE/PCURVE | partial GAP | polyline→segments; pcurve ignored (recompute via ParamAt) |

### Topology
CARTESIAN_POINT→Point3 (scale by unit); VERTEX_POINT→Vertex; EDGE_CURVE→Edge (same_sense=false ⇒
reverse so Edge.curve runs start→end); ORIENTED_EDGE→Fwd/Rev use; EDGE_LOOP→Loop;
FACE_OUTER_BOUND/FACE_BOUND→OuterLoop/InnerLoop; ADVANCED_FACE→AddFace/AddReversedFace
(same_sense=false ⇒ reversed); CLOSED_SHELL→solid Shell; MANIFOLD_SOLID_BREP→Body(solid);
SHELL_BASED_SURFACE_MODEL→surface Body. **Orientation is the crux** — the STEP sense triple must
compose so every manifold edge ends with two opposite-`Reversed()` uses (the `validate.go` check).
Acceptance gate: imported solids pass `ops.Validate`.

### Persistent naming
Imported B-rep has no feature lineage → root all entities of one import at the `ImportedBodyFeature`
(`LineageToken{Feature:"import:"+stepId, Role:"step", Index:stepEntityId}`) so `ReferenceKey()` is
stable across reopen. Export does NOT preserve our keys (STEP has no slot); re-import renumbers
(documented limitation; optional name-field stash deferred).

## 4. Per-AP handling
One core + additive passes gated by detected/requested `ApProtocol`:
- **AP203 (baseline):** product/shape/brep/units. In scope, slices B/C.
- **AP214:** colours→`model/material` appearance (slice D, in scope); assemblies flattened to bodies
  first cut (full tree → M11).
- **AP242:** tessellated bodies→`ops.Mesh`-backed body (the fallback path, slice E); PMI read-
  preserve only (authoring → M14).
Deferred first cut: full assembly graph, IGES, PMI authoring, p-curve fidelity, tolerant sewing
beyond PBI-084.

## 5. Phased delivery (PBIs, smallest-shippable first)
- **PBI-A — Part 21 tokenizer + parser + entity graph.** bytes→`EntityGraph`, HEADER, FILE_SCHEMA→AP.
  Tests: lex/parse goldens (escapes, reals, enums `.T.`, nested lists, `#id`, comments); malformed→
  error w/ token+position; header round-trip.
- **PBI-B — AP203 solid import → B-rep.** geommap/topomap/units/reader; MANIFOLD_SOLID_BREP→Body.
  Tests: per surface/curve mapping; import cube/cylinder/cyl-with-hole → `Validate` valid + volume
  within tol. **AC: a STEP solid imports as a valid healed body.**
- **PBI-C — AP203 export.** brep_to_step/geommap_to_step/writer; Body→AP203. Tests: **round-trip
  invariant** import→export→import preserves validity + volume/COM/face counts within tol; emitted
  file re-imports. **AC: re-exports without degradation.**
- **PBI-D — contract + model + CLI wiring (API first).** types/contract/wire/client; ImportedBody
  Feature; model/exchange; addin/router/exchange.go; CLI import/export. Tests: client table test,
  router handler test, CLI round-trip, `.obk` persists an imported body.
- **PBI-E — AP214 colours + AP242 tessellation/PMI (additive).** colour↔appearance; tessellated-
  shell import/export; unsupported-surface fallback w/ warning. Tests: colour round-trip,
  tessellated import, graceful degrade.

## 6. Test strategy
Layered per CLAUDE.md: unit (pure, golden STEP snippets in `testdata/`), round-trip invariants (the
core gate: validity + volume + COM + counts within tol = "without degradation"), golden `.step`
fixtures (hand-authored small files + references generated via FreeCAD/OCC CLI in `test-utilities/`),
optional external validation (FreeCAD/stepcode load our emitted files — non-blocking, NOT
certification), mass-prop tolerances tied to `UNCERTAINTY_MEASURE` + chord tolerance.

## 7. Risks & open questions
Units/tolerance (handle SI_UNIT + CONVERSION_BASED_UNIT, scale on import); B-spline trimming/p-curves
(recompute params via `Surface.ParamAt`, tessellated-face fallback on closure failure); imperfect
shells/sewing (`Sew` stubbed — import as-is, `Validate`+report `BoundaryEdges`, no auto-heal first
cut; depends on PBI-084); persistent naming on round-trip (no STEP slot — documented); large
assemblies (flatten first cut; instancing → M11); conformance realism (pragmatic interop w/
SolidWorks/Inventor/FreeCAD/OCC outputs, NOT CAx-IF certification); third-party dep (adopt none
first cut; keep the owned `StepReader`/`StepWriter` seam).

## Critical files
- `kernel/topo/builder.go` — the only sanctioned `Body` assembly path (import target).
- `kernel/geom/surface.go` — `Surface`/`Curve3` interfaces + analytic constructors (mapping targets/gaps).
- `kernel/ops/validate.go` — `Validate`/`BoundaryEdges`: the manifold/closed/oriented acceptance gate.
- `model/feature/derived.go` — precedent for `ImportedBodyFeature` (inject pre-built bodies).
- `addin/router/router.go` + `Oblikovati.API/wire/methods.go` — contract-first dispatch seam for
  `documents.import/export`.
