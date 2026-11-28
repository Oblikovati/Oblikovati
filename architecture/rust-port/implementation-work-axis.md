# Part Work Features — Work Axis (`#3918`)

Implements Part Work Features **Work Axis** end-to-end across `crates/model`,
`crates/martensite`, `crates/viewport`, `crates/gui`, and `crates/head`.

## Architecture and Persistence

- **Model (`crates/model/src/feature/work_axis/constructors.rs`, `work_geometry/evaluate/queries.rs`)**:
  - Added exact analytic constructors on `WorkAxisLine`:
    - `WorkAxisLine::plane_intersection(first, second)` (exact cross-product
      direction and incidence point, rejecting parallel or coincident planes).
    - `WorkAxisLine::normal_to_plane_through_point(plane, point)`.
    - `WorkAxisLine::parallel_to_line_through_point(axis, point)` (rejecting
      points lying on the reference axis).
  - Exposed `WorkGeometryEvaluation::axes()` to iterate evaluated user work
    axes in deterministic feature identity order.
  - Persists created axes using the existing `PartFeatureDefinition::Axis`
    (`WorkPlaneDefinitionRecipe::Axis`, source version 2, minimum program
    version 3) through the document transaction log, preserving Undo/Redo,
    save/reopen, and sequential `Work Axis1..N` naming without bumping
    `MAX_PROGRAM_VERSION`.

- **Martensite (`crates/martensite/src/prism/named/work_axis_queries.rs`)**:
  - `NamedSolid::vertex_positions` and `NamedSolid::vertex_hits_on_ray` resolve
    exact B-rep vertices (`BRepTool::pnt`) against forward viewport rays.
  - `NamedSolid::edge_axis` extracts exact analytic axes via
    `BRepAdaptorCurve`: `Straight` for `CurveType::Line` and
    `CircularOrElliptical` (center + axis normal) for `CurveType::Circle` and
    `CurveType::Ellipse`.
  - `NamedSolid::revolved_face_axis` extracts exact revolution axes via
    `BRepAdaptorSurface` for `Cylinder`, `Cone`, `Torus`, and
    `SurfaceOfRevolution`, rejecting planar and non-revolved faces.

- **Viewport (`crates/viewport/src/datum_axis.rs`, `renderer/datum_plane.rs`)**:
  - `ViewportDatumAxis` and `Viewport::set_datum_axes` render visible user work
    axes as 3D depth-tested overlays (`AspectTypeOfLine::Dotdash` unselected,
    `AspectTypeOfLine::Solid` highlighted when selected).

- **GUI (`crates/gui/src/ribbon_axis_group.rs`, `work_axis_action.rs`)**:
  - Replaced the muted `Axis` button in `3D Model > Work Features` with a split
    button and constructor dropdown menu (`Axis`, `On Line or Edge`,
    `Parallel to Line through Point`, `Through Two Points`,
    `Intersection of Two Planes`, `Normal to Plane through Point`,
    `Through Center of Circular or Elliptical Edge`,
    `Through Revolved Face or Feature`).
  - Uses status-bar prompts (`##status-bar`) for the shared in-canvas
    definition lifecycle, dispatches Origin browser picks (`XY Plane`,
    `XZ Plane`, `YZ Plane`, `X/Y/Z Axis`, `Center Point`), cancels on Escape,
    and renders projected user work axes with accent highlighting when selected.

- **Desktop Owner (`crates/head/src/owner_work_axis.rs`, `owner_work_axis_pick.rs`)**:
  - Manages interactive reference collection, analytic viewport picking
    (vertices, straight edges, circular/elliptical edges, planar faces, and
    outer/internal cylindrical revolved faces), browser feature selection
    handling (keeping the revolved-face prompt active when clicking `Hole1`),
    `Work Axis` browser kind/icon reporting, 3D + projected axis overlays, and
    transactional commits returning to `Ready`.

## Verification

- **Unit Tests**:
  - `crates/model/src/feature/work_axis/constructors.rs`: plane-plane
    intersection, normal-to-plane through point, parallel-to-line through point,
    and degenerate rejection checks.
  - `crates/martensite/src/prism/named/work_axis_queries.rs`: exact solid vertex
    ray hits, straight and circular edge axis extraction, cylindrical face axis
    extraction, and planar face rejection.
  - `crates/viewport/src/datum_axis.rs`: endpoint, width, and RGBA validation.
- **Native Wayland BDD (`work-axis.feature`)**:
  - `work-axis-01` (`Through Two Points` on `docking-visible-solid-current.opd`)
  - `work-axis-02` (`On Line or Edge` on `docking-visible-solid-current.opd`)
  - `work-axis-03` (`Intersection of Two Planes` on `blank-metric-part.opd`)
  - `work-axis-04` (`Through Center of Circular or Elliptical Edge` on `fillet-circular-edge-candidate.opd`)
  - `work-axis-05` (`Normal to Plane through Point` on `docking-visible-solid-current.opd`)
  - `work-axis-06` (`Parallel to Line through Point` on `docking-visible-solid-current.opd`)
  - `work-axis-07` (`Through Revolved Face or Feature` on `fillet-circular-edge-candidate.opd`)
  - `work-axis-08` (General `Axis` command prompt on `blank-metric-part.opd`)
  - `work-axis-09` (`Through Revolved Face or Feature` on `Hole1` in `hole-seven-axes-unique-candidate.opd`, creating and highlighting `Work Axis8`)
  - `work-axis-09a` (`Through Revolved Face or Feature` on 9 mm internal wall in `internal-wall-seven-axes.opd`, creating and highlighting `Work Axis8`)
