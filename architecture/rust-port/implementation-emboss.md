# Emboss Part Create implementation and verification

Emboss creates and edits certified raised, engraved, wrapped-to-planar-face,
and from-plane solid features across `model`, `martensite`, `automation`,
`gui`, and `head`.

## Architecture and schema

- **Model (`crates/model`)**: Added `SketchEmbossDefinition`,
  `SketchEmbossRecipe`, `EmbossType` (`EmbossFromFace`, `EngraveFromFace`,
  `EmbossEngraveFromPlane`), and `ResolvedSketchEmboss` at Part program
  version 46, wired into `PartFeatureDefinition::SketchEmboss`, replay
  resolution, `DocumentSnapshot::create_sketch_emboss`, and Undo/Redo history.
- **Martensite adapter (`crates/martensite`)**: Added `evaluate_named_emboss` in
  `crates/martensite/src/emboss.rs` on top of the certified named sketch-prism
  Boolean pipeline (`Join` for `EmbossFromFace`, `Cut` for `EngraveFromFace`,
  planar `resolve_planar_face` validation for `wrap_to_face`, and interior
  `prism_through_all_span` verification for `EmbossEngraveFromPlane`, cleanly
  failing when the sketch plane is flush with a planar face).
- **Automation (`crates/automation`)**: Added `PreparedEmboss`,
  `build_emboss_definition`, `prepare_emboss_definition`, `features.add`
  `"emboss"`, range-checked positive depth and taper angle expression parsing,
  and enabled `Create.Emboss` on the Part ribbon.
- **Desktop GUI and owner (`crates/gui`, `crates/head`)**: Built the dockable
  `Emboss` Properties panel (`crates/gui/src/emboss_dialog/`) with transient
  `Sketch Doctor` (open-loop diagnostic), `Edit Emboss Feature Failed` (with
  `Edit` return to settings), and `Emboss Warning` (out-of-range depth
  expression) modals, `Taper` field (`0,0` default on the third operation),
  `Flip Direction` direction marker reversal, `Wrap to Face` viewport face
  picking with pre-edit target face mapping, and browser double-click editing
  of `Emboss1`.

## Verification

- **Kernel and persistence tests**: `cargo test --locked -p oblikovati-martensite --test emboss`
  verifies raised emboss, engraved recess, wrap-to-planar-face, clean refusal
  of flush-plane `EmbossEngraveFromPlane`, and Part program v46 `DocumentPackage`
  encode/decode/reopen.
- **Native desktop BDD**: All 7 Emboss scenarios (`emboss-01`..`emboss-07`)
  passed on native Wayland. See the
  [native Emboss desktop receipt](receipts/2026-11-27-emboss-desktop/README.md).
