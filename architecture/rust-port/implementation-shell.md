# Part Modify Shell Implementation

## Overview

Part Modify **Shell** v2 is implemented end-to-end across `oblikovati-model`,
`oblikovati-martensite`, `oblikovati-automation`, `oblikovati-gui`, and
`oblikovati-head` in Part program version `56`, upgrading uniform inward shells
to support `Inside`, `Outside`, and `Both` directions, per-face `Thickness
Override Sets`, `Sharp` and `Rounded` (`JoinType::Arc`) shell types, `Advanced
Properties` (`Allow Approximation`, `Never too thin` / `Never too thick` /
`Mean`, `Optimized` / `Specified tolerance` percentage), rounded-face tangent
chain expansion, inline zero-thickness validation, and transactional in-place
editing while keeping v1 recipes byte-identical.

## Architecture

- **Model (`crates/model/src/feature/shell.rs`, `history/shell_create.rs`, `identity/derived.rs`)**:
  - Adds `ShellType` (`Sharp`, `Rounded`), `ShellApproximationType` (`Mean`,
    `NeverTooThin`, `NeverTooThick`), `ShellApproximationTolerance`
    (`Optimized`, `Specified`), and `ShellOptions` with `10.0` default percent.
  - Encodes `ShellRecipe` v1 (exact legacy format when `options.is_default()`)
    and v2 in Part program version `56`, plus `DocumentSnapshot::edit_shell`
    for transactional in-place edits and `DerivedReferenceName::{feature,parents}`
    accessors for edited-shell face unwrapping.
- **Martensite Adapter (`crates/martensite/src/prism/named/blend/shell.rs`, `fillet_topology.rs`, `solid/shell_evaluation.rs`)**:
  - Executes `Inside`, `Outside`, and two-pass symmetric `Both` shells via
    `BRepOffsetMakeOffset` with per-face `set_offset_on_face` overrides and
    `JoinType::Intersection` (`Sharp`) or `JoinType::Arc` (`Rounded`),
    composing exact multi-pass history records so outer, inner, and generated
    faces preserve full topology lineage.
  - Adds `NamedSolid::shell_tangent_face_chain` so clicking a curved/filleted
    face expands the removal set to its tangent face chain while planar clicks
    select a single face.
- **Automation & Desktop Owner (`crates/automation/src/shell_v2.rs`, `crates/head/src/owner_shell_v2*.rs`)**:
  - `build_shell_v2_definition`, `prepare_shell_v2`, and `replace_shell` share
    validation between preview and commit.
  - `owner_shell_v2.rs` handles cumulative `Remove` face picking (with tangent
    chain expansion and edited-shell parent face unwrapping), per-override face
    picking, live replacement preview, `browser.shell_v2_edits` hydration, and
    transactional create/edit commits.
- **GUI (`crates/gui/src/shell_panel/`, `shell_feature_dialog.rs`)**:
  - Dockable `Shell` Properties panel with `Behavior` (`Shell Type`,
    `Direction`, `Remove` pill, `Thickness` with red `!` error indicator
    disabling `OK` on zero), `Thickness Override Sets` (`+` override builder),
    and `Advanced Properties` (`Allow Approximation`, `Type`, `Tolerance`, and
    percentage field disabled when unchecked), hiding overrides and advanced
    properties when `Rounded` is active.

## Verification

- `oblikovati-model`: `part_features` (96/96).
- `oblikovati-martensite`: `sketch_prism shell_v2` verifying `Inside`,
  `Outside`, `Both`, per-face overrides, `Rounded` join type, and 3-face
  tangent-chain shelling on a filleted solid.
- `oblikovati-gui`: full library test suite including `shell_feature_dialog`
  and `shell_pointer` (850/850).
- `oblikovati-head`: all **7/7 Wayland BDD scenarios (54/54 steps)** in
  `features/solid-model/shell/shell.feature`.
