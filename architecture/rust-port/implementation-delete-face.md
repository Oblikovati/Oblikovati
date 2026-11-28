# Part Modify Delete Face Implementation

## Overview

Part Modify **Delete Face** (`#3918`) is implemented end-to-end across
`oblikovati-model`, `oblikovati-martensite`, `oblikovati-automation`,
`oblikovati-gui`, and `oblikovati-head` in Part program version `63` (source
recipe version `53`). It supports multi-face deletion without healing (converting
a solid body into an open surface body in the Model Browser), healed face
deletion via Martensite defeaturing, viewport face highlighting, selector clear
recovery while preserving Behavior toggles, failure recovery modals (`Create
DelFace feature failed` with `Edit` and `Cancel`), and browser `Edit Feature`
hydration.

## Architecture

- **Model (`crates/model/src/feature/delete_face.rs`, `crates/model/src/feature/part/delete_face_evaluation.rs`, `crates/model/src/history/hole_create.rs`)**:
  - Defines `DeleteFaceRecipe`, `DeleteFaceDefinition`, and `ResolvedDeleteFace`
    with target body key, unique nonempty face keys (`Vec<Vec<u8>>`),
    `heal_remaining_faces: bool`, and `select_lump_or_void: bool`.
  - Persists under `WorkPlaneDefinitionRecipe::DeleteFace` with
    `source_version = 53`, `minimum_program_version = 63`, and
    `MAX_PROGRAM_VERSION = 63`.
  - Evaluates in dedicated `delete_face_evaluation.rs` (keeping `evaluate.rs`
    and `prefix_replay.rs` under 500 lines) and provides transactional
    `DocumentSnapshot::create_delete_face` and `DocumentSnapshot::edit_delete_face`.
- **Martensite Adapter (`crates/martensite/src/prism/named/blend/delete_face.rs`, `solid/delete_face_evaluation.rs`)**:
  - **Unhealed deletion (`heal_remaining_faces = false`)**: removes the selected
    faces from a detached copy of the body and constructs an open `Shell`
    (`ShapeEnum::Shell`), transitioning solids into surface bodies
    (`BlendNamedTopology::is_surface() == true`) while preserving exact topology
    lineage for retained faces, edges, and vertices.
  - **Healed deletion (`heal_remaining_faces = true`)**: runs
    `martensite_tkbo::brep_algo_api::BRepAlgoAPIDefeaturing` with history
    enabled, validates topology with `require_geometry`, `BRepCheckAnalyzer`,
    and tolerance budgets, and returns `"Create DelFace feature failed"` when
    adjacent faces cannot extend to heal the removed face (such as the lower
    circular end cap of a free cylinder).
- **Automation & Desktop Owner (`crates/automation/src/delete_face.rs`, `crates/head/src/owner_delete_face.rs`)**:
  - `prepare_delete_face_definition` names new features `Delete Face1`,
    `Delete Face2`, etc.; `build_delete_face_definition` and
    `replace_delete_face` share validation between preview and commit.
  - `owner_delete_face.rs` wires `GuiDeleteFaceAction`, multi-face viewport
    picking, active face highlighting via `set_selected_faces`, real Martensite
    candidate validation, `browser.delete_face_edits` hydration, and
    transactional commits (updating `Solid Bodies` and `Surface Bodies`).
- **GUI (`crates/gui/src/delete_face_dialog/`, `crates/gui/src/icons/`, `crates/gui/src/ribbon_panel_additional/commands.rs`)**:
  - Adds `IconGlyph::DeleteFace` backed by `cmd-model-delete-face-32.svg` and
    exposes `Delete Face` in the `3D Model > Modify` additional commands panel.
  - Dockable `280px` Properties panel positioned with `fit_viewport_properties`,
    collapsible `Input Geometry` (`Faces` selector pill showing `Select Faces`,
    `1 Face`, or `N Faces` plus `x` clear control) and `Behavior` (`Heal
    Remaining Faces` and `Select Lump or Void` checkboxes preserved across clear
    and `Edit` recovery), plus the `"Create DelFace feature failed"` modal
    offering `Edit` and `Cancel`.

## Verification

- `oblikovati-model`: `feature::delete_face::tests` recipe validation tests.
- `oblikovati-martensite`: `sketch_prism delete_face_replay` verifying unhealed
  cylinder solid-to-surface open-shell conversion (3 faces to 2 faces),
  authentic healed-deletion failure on an unhealable cylinder end cap, and
  authentic healed fillet removal restoring unit cube volume `1.0`.
- `oblikovati-head`: all **5/5 Wayland BDD scenarios** in
  `features/solid-model/delete-face/delete-face.feature`.
