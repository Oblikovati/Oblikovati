# Work Features: Work Point (`#3918`)

Implements **3D Model > Work Features > Point** end-to-end across
`crates/model`, `crates/automation`, `crates/gui`, and `crates/head`.

## Architecture and Persistence

- **Model (`crates/model/src/feature/work_point.rs`)**:
  - `WorkPointConstruction` (`OnVertex`, `LoopCenter`, `Grounded`, `TwoLines`,
    `PlaneAndLine`, `ThreePlanes`), `WorkPointRecipe`, and `WorkPointDefinition`.
  - Exact analytic 3D evaluators:
    - `OnVertex` and `LoopCenter` resolve live through
      `PartFeatureEvaluator::resolve_frame_origin` (`NamedSolid::frame_origin`
      using exact vertex coordinates and analytic `CurveType::Circle` centers)
      via `DatumTopologyResolver`.
    - `TwoLines` computes the exact closest-point intersection of coplanar
      non-parallel lines, rejecting parallel and skew lines.
    - `PlaneAndLine` solves the exact plane-line intersection, rejecting
      parallel lines.
    - `ThreePlanes` solves the exact 3x3 normal system via Cramer's rule,
      rejecting singular/parallel configurations.
    - `Grounded` evaluates `[DatumLength; 3]` in document working units.
  - Disjoint persistence slot in `WorkPlaneDefinitionRecipe::WorkPoint`:
    `source_version = 56`, `minimum_program_version = 66`,
    `MAX_PROGRAM_VERSION = 66`.
  - Attributed dependency footprints (`reads()`) and
    `requires_named_part_replay()` integration in
    `crates/model/src/feature/part/sketch_recompute.rs`.

- **Automation (`crates/automation/src/datums.rs`, `rows.rs`)**:
  - Classifies `WorkGeometryDefinition::WorkPoint(_)` under
    `DatumCollection::Points` with constructor kind projection.

- **GUI (`crates/gui/src/work_point/`)**:
  - `3D Model > Work Features > Point` split button and constructor dropdown
    menu (`Point`, `Grounded Point`, `On Vertex, Sketch point, or Midpoint`,
    `Intersection of Three Planes`, `Intersection of Two Lines`,
    `Intersection of Plane/Surface and Line`,
    `Center Point of Loop of Edges`, `Center Point of Torus`,
    `Center Point of Sphere`).
  - Viewport 3D-projected Work Point markers (`GuiWorkPointMarker`) with
    browser-selection highlighting.
  - In-canvas **Grounded Point** floating HUD with editable `X`, `Y`, `Z`
    coordinates, projected 3D X/Y/Z move triad, and `OK` / `Cancel` controls.
  - Browser Origin row routing (`YZ Plane`, `XZ Plane`, `XY Plane`, `X Axis`,
    `Y Axis`, `Z Axis`, `Center Point`) during active Work Point commands.

- **Desktop Owner (`crates/head/src/owner_work_point.rs`)**:
  - `GuiWorkPointAction` command lifecycle, status prompts, `Escape`
    cancellation returning to `Ready`, analytic solid vertex and circular
    edge-loop viewport picking, browser datum picking, transactional
    `Work Point{N}` naming and commits (Undo/Redo compatible), and browser
    `"Work Point"` kind/icon reporting.

## Verification

- **Unit Tests**:
  - `crates/model/src/feature/work_point.rs`: exact intersection verification
    for two lines, plane and line, three planes, offset three planes, degenerate
    parallel/skew rejections, grounded coordinate validation, and recipe
    version 66 roundtrip.
- **Native Wayland BDD (`work-point.feature`)**:
  - `work-point-01`: `On Vertex, Sketch point, or Midpoint` on
    `docking-visible-solid-current.opd` prompts, picks model vertex, creates
    exact vertex work point, returns to `Ready`.
  - `work-point-02`: `Center Point of Loop of Edges` on
    `fillet-circular-edge-candidate.opd` picks circular end edge, creates exact
    center work point, returns to `Ready`.
  - `work-point-03`: `Grounded Point` on `docking-visible-solid-current.opd`
    picks vertex, displays X/Y/Z coordinates, 3D move triad, and `OK`/`Cancel`
    controls; accepting creates grounded point and returns to `Ready`.
  - `work-point-04`: `Intersection of Two Lines` on `blank-metric-part.opd`
    selects Origin `X Axis` and `Y Axis`, creates `[0, 0, 0]` point, returns to
    `Ready`.
  - `work-point-05`: `Intersection of Plane/Surface and Line` on
    `blank-metric-part.opd` selects `XY Plane` and `Z Axis`, creates
    `[0, 0, 0]` point, returns to `Ready`.
  - `work-point-06`: `Center Point of Torus` prompts for a torus; canceling via
    `Escape` and choosing `Center Point of Sphere` prompts for a sphere.
  - `work-point-07`: `Point` from the Point dropdown prompts to define the work
    point by selecting geometry.
  - `work-point-08`: `Intersection of Three Planes` on
    `work-point/six-points.opd` (`Work Point1`..`Work Point6`) selects `YZ`,
    `XZ`, `XY` planes, returns to `Ready`, appends `Work Point7` at the end of
    the browser, and selecting `Work Point7` highlights its browser entry.
