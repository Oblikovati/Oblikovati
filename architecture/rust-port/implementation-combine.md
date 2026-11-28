# Part Modify Combine Implementation

## Overview

Part Modify **Combine** (`#3918`) is implemented end-to-end across
`oblikovati-model`, `oblikovati-martensite`, `oblikovati-automation`,
`oblikovati-gui`, and `oblikovati-head` in Part program version `59` (recipe
source version `49`), supporting solid-body `Join`, `Cut`, and `Intersect`
Boolean operations across one base solid and one or more tool solids,
`Keep Toolbodies` retention or consumption in the Solid Bodies folder, browser
double-click `Edit Feature`, live replacement preview, clear/cancel recovery,
and `"Edit Combine Feature"` modal error handling.

## Architecture

- **Model (`crates/model/src/feature/combine.rs`, `feature/part/combine_evaluation.rs`, `history/hole_create.rs`)**:
  - Defines `CombineOperation` (`Join`, `Cut`, `Intersect`), `CombineRecipe`,
    `CombineDefinition`, and `ResolvedCombine`.
  - Validates same-document body references, non-empty tool list, and distinct
    tool bodies separate from the base body.
  - `PartFeatureEvaluation::evaluate_combine` resolves the base and all tool
    slots, evaluates the Boolean via `PartFeatureEvaluator::evaluate_combine`,
    replaces the base slot with the combined result, and removes/retires tool
    body slots when `keep_toolbodies` is `false` (retaining tool bodies
    untouched when `keep_toolbodies` is `true`).
  - Uses disjoint Part program version `59` and `WorkPlaneDefinitionRecipe::Combine`
    source version `49`.
  - `DocumentSnapshot::create_combine` and `DocumentSnapshot::edit_combine`
    provide transactional creation and in-place editing.
- **Martensite Adapter (`crates/martensite/src/prism/named/blend/combine.rs`, `solid/combine_evaluation.rs`)**:
  - Copies tool solid shapes and their named topology sources into detached
    `BooleanTool` inputs so kept tool bodies remain completely isolated.
  - Evaluates single-tool and multi-tool `Join`, `Cut`, and `Intersect`
    through `NamedSolid::boolean_tools` (`BRepAlgoAPIBooleanOperation`),
    preserving exact topology lineage and rejecting empty, unchanged, or
    disconnected multi-solid results cleanly.
- **Automation & Desktop Owner (`crates/automation/src/combine.rs`, `crates/head/src/owner_combine*.rs`)**:
  - `CombineInputs`, `build_combine_definition`, `prepare_combine_definition`
    (prefix `"Combine"` -> `Combine1`), and `replace_combine` share validation
    across preview and commit.
  - `owner_combine.rs` provides `SelectionKind::Body` viewport picking,
    live replacement preview, `browser.combine_edits` hydration, and
    transactional create/edit commits.
- **GUI (`crates/gui/src/combine_dialog/`, `crates/gui/src/ribbon_panel_additional/commands.rs`)**:
  - Exposed in `3D Model > Modify` (`Combine`) and browser double-click /
    `Edit Feature`.
  - Dockable `280px` `Combine` Properties panel with `Input Geometry`
    (`Basebody` and `Toolbodies` selector pills with `Select Solid` /
    `Select Solids` / `1 Solid` / `N Solids` captions and per-row `Clear`
    buttons), `Output` (`Join`, `Cut`, `Intersect` toggles and
    `Keep Toolbodies` checkbox), `OK`/`Cancel`/`+` footer, and the
    `"Edit Combine Feature"` error modal (`Details`, `Edit`, `Cancel`).

## Verification

- `oblikovati-model`: `feature::combine::tests::combine_definition_validates_distinct_bodies_and_records_reads` (pass).
- `oblikovati-martensite`: `sketch_prism combine_join_keep_toolbodies_and_multi_tool_cut_match_analytic_volumes`
  verifying exact analytic volumes for Join (`24 + 0.25π cm³`), multi-tool Cut
  without retention (`24 - π cm³`, 1 solid), single-tool Cut with retention
  (`24 - 0.5π cm³`, tool retained), and clean Intersect failure on disjoint
  two-cylinder tools.
- `oblikovati-gui`: full `--lib` suite including `combine_dialog::tests` (pass).
- `oblikovati-head` Wayland BDD (`features/solid-model/combine/combine.feature`):
  - **7/8 scenarios pass (50/50 steps)**: `combine-01`, `combine-02`,
    `combine-03`, `combine-05`, `combine-06`, `combine-07` (genuine Intersect
    failure on `three-bodies.opd` surfacing `"Edit Combine Feature"` modal and
    `Cancel` recovery), and `combine-08`.
  - **`combine-04` (`@needs_fixture @fixture_variation`)**: All step bindings
    are implemented, and `FIXTURES.md` notes that `two-bodies.opd` substitutes a
    10 mm cylinder for the original captured coil without reproducing the
    specimen-specific Cut edit failure. On `two-bodies.opd`, `Cut(Block, Cyl1)`
    is a geometrically valid through-hole subtraction (`24 - 0.5π ≈ 22.4292 cm³`),
    so Martensite legitimately succeeds rather than failing. A real failing
    specimen fixture (such as the captured coil-and-block geometry) is required
    for `combine-04` to trigger `"Edit Combine Feature"` authentically.
