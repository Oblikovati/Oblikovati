# Part Modify Thread Implementation

## Overview

Part Modify **Thread** (`#3918`) is implemented end-to-end across
`oblikovati-model`, `oblikovati-martensite`, `oblikovati-automation`,
`oblikovati-gui`, and `oblikovati-head` in Part program version `58`
(`WorkPlaneDefinitionRecipe::Thread`, source version `48`, minimum program
version `58`).

It supports cylindrical and conical face eligibility detection, external
tapered and internal cylindrical threads, standard catalogs (`ISO Taper
External`, `JIS Pipe Threads`, `ANSI Metric M Profile`), right/left handedness,
full-depth vs. partial depth + offset (including offsets beyond face length),
cosmetic model thread display bands whose spacing responds to pitch, client and
kernel validation, browser `Edit Feature` hydration, and failure recovery.

## Architecture

- **Model (`crates/model/src/feature/thread.rs`, `crates/model/src/history/hole_create.rs`)**:
  - Defines `ThreadHandedness` (`RightHand`, `LeftHand`),
    `ThreadSpecification`, `ThreadRecipe`, `ThreadDefinition`, and
    `ResolvedThread`.
  - Validates body/face reference identity, non-empty specification fields,
    strictly positive partial depth, and finite nonnegative offset.
  - `DocumentSnapshot::create_thread` and `DocumentSnapshot::edit_thread`
    provide transactional creation (`Thread1` prefix) and in-place editing.
- **Martensite Adapter (`crates/martensite/src/prism/named/blend/thread.rs`, `solid/thread_evaluation.rs`)**:
  - `NamedSolid::has_threadable_faces` performs a fast `BRepAdaptorSurface`
    surface-type check (`Cylinder` / `Cone`) without solid classification.
  - `NamedSolid::threadable_face_info` inspects trimmed face UV bounds to
    report diameter, exact generator full-depth length (axial height for
    cylinders, slant length for cones, e.g. `49.062 mm` on the tapered revolve
    and `2.000 mm` on the 9 mm internal wall), conical flag, and internal vs.
    external orientation via `BRepClass3dSolidClassifier`.
  - `CertifiedNamedSolidEvaluator::evaluate_named_thread` rejects incompatible
    straight-pipe model display (`JIS Pipe Threads` on a conical face with
    `display_in_model = true`), preserves exact body topology lineage, and
    generates pitch-scaled helical cosmetic overlay bands attached to
    `BlendNamedTopology`.
- **Automation & Desktop Owner (`crates/automation/src/thread.rs`, `crates/head/src/owner_thread*.rs`)**:
  - `build_thread_definition`, `prepare_thread_definition`, and
    `replace_thread` share expression parsing and validation between preview
    and commit.
  - `owner_thread.rs` wires cylindrical/conical viewport picking, live preview
    and committed cosmetic thread overlays (`THREAD_OVERLAY_COLOR`),
    `browser.thread_edits` hydration, and checkpointed commits.
- **GUI (`crates/gui/src/thread_dialog/`, `crates/gui/src/ribbon_panel_additional/commands.rs`)**:
  - Exposed under `3D Model > Modify` (`Thread` with `IconGlyph::Thread`).
  - Reports `"Could not find cylindrical or conical faces to thread."` when
    opened on a part with no eligible faces (`thread-01`).
  - Dockable `280px` Properties panel with `Location` (`Face` pill,
    `Full Depth`, `Depth`, conditional `Offset`, `Display thread in model`) and
    `Specification` (`Thread Type`, `Size`, `Designation`, `Class`,
    `Right Hand` / `Left Hand`), inline red validation disabling `OK` on zero
    depth, and the `"Edit Thread Feature Failed"` recovery modal (`thread-05`).

## Verification

- `oblikovati-model`: `feature::thread` unit tests + library suite.
- `oblikovati-martensite`: `cone_face_detected_as_external_conical_and_rejects_jis_pipe_display`
  and `thread_replay` integration tests (planar block rejection, internal
  cylindrical detection, pitch density scaling, hidden/beyond-offset overlays).
- `oblikovati-gui`: `thread_dialog::catalog` tests + full library test suite.
- `oblikovati-head`: all **13/13 Wayland BDD scenarios (120/120 steps)** in
  `features/solid-model/thread/thread.feature` (`thread-01`..`thread-10a`).
