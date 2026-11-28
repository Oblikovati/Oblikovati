# Work Plane Implementation Evidence (`#3918`)

## Scope

Implements the full Solid Model **Work Plane** feature (`solid-model/work-plane.feature`, scenarios `work-plane-01` through `work-plane-12`) across model, Martensite, automation, GUI, desktop host, and native Wayland BDD tests.

## Format & Version Ownership

- **Program format**: `v68` (`MAX_PROGRAM_VERSION = 68`)
- **Source format**: `v58` (`WorkPlaneDefinitionRecipe::WorkPlaneExtended`)
- Split `crates/model/src/feature/work_plane/recipe.rs` into `recipe/mod.rs`, `recipe/restore.rs`, and `recipe/versions.rs` to keep every file well below 500 lines.

## Key Modules

- **Model (`crates/model`)**:
  - `src/feature/work_plane/extended/{mod.rs,eval.rs}`: `WorkPlaneExtendedDefinition` covering Midplane between Two Planes (parallel and angle bisector), Three Points (with B-rep vertex references), Parallel to Plane through Point, Normal to Axis through Point, Angle to Plane around Edge, Two Coplanar Edges (with skew rejection), Midplane of Torus, and Tangent to Surface through Edge / Point / Parallel to Plane.
  - `src/feature/part/work_plane_evaluation.rs`: ordered Part replay evaluation resolving named B-rep faces, vertices, straight edges, torus surfaces, and tangent surfaces via `PartFeatureEvaluator`.
- **Martensite (`crates/martensite`)**:
  - `src/prism/named/work_plane_queries.rs`: exact analytic B-rep `vertex_hits_on_ray`, `resolve_topology_point`, `resolve_straight_edge`, `resolve_torus_midplane`, `resolve_tangent_plane_through_point`, `resolve_tangent_plane_through_line`, and `resolve_tangent_plane_parallel`.
- **Automation (`crates/automation`)**:
  - `src/datums/planes/extended.rs`: `workPlanes.create` and metadata support for `WORK_PLANE_TWO_PLANES`, `WORK_PLANE_TORUS_MID_PLANE`, tangent constructors, topology face/edge/vertex references, and angle expression normalization.
- **GUI & Desktop Host (`crates/gui`, `crates/head`)**:
  - `crates/gui/src/ribbon_plane_group.rs`: enables `Plane`, `Offset from Plane`, and all 11 reference/orientation constructors in the Plane drop-down and ungrouped ribbon.
  - `crates/gui/src/work_plane_dialog/`: interactive multi-pick Work Plane state machine, status prompts, auto-commit for reference-only constructors, and auto-focused angle input (`90 degrees` selected) with Enter-key commit and Escape cancel.
  - `crates/gui/src/offset_plane.rs`: live 3D plane preview emission, status prompts, and Enter-key commit.
  - `crates/gui/src/application/workspace-layout/browser_feature_edit.rs`: checked/unchecked `Visibility` context menu item on Work Plane browser rows.
  - `crates/head/src/owner_work_plane.rs` and `crates/head/src/owner_work_plane_pick.rs`: depth-ordered viewport picking (vertices, edges, faces, work planes), live `ViewportDatumPlane` preview overlays, creation via `workPlanes.create`, and undoable `set_datum_visible`.
- **Tests**:
  - `crates/model/tests/work_plane_extended.rs`
  - `crates/head/tests/solid-model/work-plane/{setup.rs,steps.rs}` (live native Wayland BDD for `work-plane-01`..`work-plane-12`).
