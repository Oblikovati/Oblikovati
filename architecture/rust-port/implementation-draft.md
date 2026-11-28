# Part Modify Draft Implementation

## Overview

Part Modify **Draft** (`#3918`) is implemented end-to-end across
`oblikovati-model`, `oblikovati-martensite`, `oblikovati-automation`,
`oblikovati-gui`, and `oblikovati-head` in Part program version `57` (source
recipe version `47`), supporting Fixed Edge (Face Draft), Fixed Plane, and
Parting Tool modes, Absolute/Relative draft options, live replacement preview,
creation/edit failure recovery modals (`Edit` and `Cancel`), and browser
`Edit Feature` hydration.

## Architecture

- **Model (`crates/model/src/feature/draft.rs`, `crates/model/src/history/hole_create.rs`)**:
  - Defines `DraftMode` (`FixedEdge`, `FixedPlane`, `PartingTool`),
    `DraftAngleMode` (`Absolute`, `Relative`), `DraftPartingTool`
    (`OriginPlane`, `WorkPlane`, `Face`), `DraftRecipe`, `DraftDefinition`,
    and `ResolvedDraft`.
  - Persists under `WorkPlaneDefinitionRecipe::Draft` with
    `source_version = 47`, `minimum_program_version = 57`, and
    `MAX_PROGRAM_VERSION = 57`.
  - Validates finite draft angles strictly inside `(0, 90)` degrees (rejecting
    zero and out-of-range edits with `"errors during the part update"`),
    nonempty unique face references distinct from the pull-direction face, and
    required parting geometry in `PartingTool` mode.
  - `DocumentSnapshot::create_draft` and `DocumentSnapshot::edit_draft` provide
    transactional creation and in-place editing.
- **Martensite Adapter (`crates/martensite/src/prism/named/blend/draft.rs`, `solid/draft_evaluation.rs`)**:
  - Uses `martensite_tkoffset::brep_offset_api::BRepOffsetAPIDraftAngle` to
    apply exact geometric face drafting around the resolved neutral plane
    (pull-direction face plane in `FixedEdge`/`FixedPlane`, parting plane or
    pull-normal plane anchored at the parting origin in `PartingTool`).
  - Validates the resulting solid with `require_geometry`, `BRepCheckAnalyzer`,
    and tolerance budget verification, and preserves exact 1-to-1 topology
    lineage via `BRepOffsetAPIDraftAngle::modified_shape` + `name_topology`.
  - Rejects invalid multi-loop shelled-rim face drafts cleanly so `draft-01`
    surfaces creation failure recovery without corrupting the part.
- **Automation & Desktop Owner (`crates/automation/src/draft.rs`, `crates/head/src/owner_draft*.rs`)**:
  - `prepare_draft_definition` names new features `FaceDraft1`, `FaceDraft2`,
    etc.; `build_draft_definition` and `replace_draft` share validation between
    preview and commit.
  - `owner_draft.rs` wires `GuiDraftAction`, viewport planar face picking,
    live replacement previews, `browser.draft_edits` hydration, and
    transactional create/edit commits.
- **GUI (`crates/gui/src/draft_dialog/`, `crates/gui/src/ribbon_panel_additional/commands.rs`)**:
  - Exposed in the `3D Model > Modify` additional commands area (`Draft`).
  - Dockable `280px` Properties panel positioned with `fit_viewport_properties`
    featuring the 3 Draft mode buttons (`Fixed Edge`, `Fixed Plane`,
    `Parting Tool`), `Pull Direction`/`Fixed Plane`, `Parting Tool` (consuming
    Origin planes such as `XZ Plane` from the Model Browser), `Faces` (active
    once pull direction and parting tool are selected), `Draft Angle` (default
    `2.5 deg`), collapsible `Options` (`Absolute` / `Relative`, `Automatic Blending`),
    and `Create Face Draft` / `Edit Face Draft` failure modals with expandable
    `"errors during the part update"` details and `Edit`/`Cancel` recovery.

## Verification

- `oblikovati-model`: `feature::draft::tests` unit tests.
- `oblikovati-martensite`: `sketch_prism draft_replay` exact analytic wedge
  volume verification for `2.5°` and `5°` across `FixedEdge`, `FixedPlane`, and
  `PartingTool` (`XZ`).
- `oblikovati-gui`: full library test suite (851 tests).
- `oblikovati-head`: all **6/6 Wayland BDD scenarios** in
  `features/solid-model/draft/draft.feature`.
