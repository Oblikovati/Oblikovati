# Part Surface — Replace Face implementation

Issue: `#3918`  
Canonical spec: `MCAD-Scenarios/features/solid-model/replace-face/replace-face.feature` (`replace-face-01`..`replace-face-04`)

## Architecture

### Model (`crates/model`)

- Added `ReplaceFaceTarget`, `ReplaceFaceRecipe`, `ReplaceFaceDefinition`, and `ResolvedReplaceFace` in `crates/model/src/feature/replace_face.rs`.
- Uses disjoint persistence slot `source_version = 62` and `minimum_program_version = 72` (`MAX_PROGRAM_VERSION = 72`).
- `PartFeatureEvaluation::evaluate_replace_face` (`crates/model/src/feature/part/replace_face_evaluation.rs`) resolves the target solid body and every replacement face's owning body through `resolve_body_slot`, delegates to `PartFeatureEvaluator::evaluate_replace_face`, verifies target body reference lineage, replaces the target solid slot, and retains replacement surface bodies in the part.
- Added `DocumentSnapshot::{create_replace_face, edit_replace_face}` in `crates/model/src/history/replace_face_create.rs`.

### Martensite kernel (`crates/martensite`)

- Implemented `NamedSolid::evaluate_replace_face_result` in `crates/martensite/src/prism/named/blend/replace_face.rs` with analytic helpers in `replace_face_geom.rs` and replay binding in `crates/martensite/src/prism/named/solid/replace_face_evaluation.rs`.
- Supports `Automatic Face Chain` tangent face expansion via `NamedSolid::tangent_face_chain`.
- Evaluates real B-rep planar face replacement (outward prism extension + `BRepAlgoAPIBooleanOperation::Fuse`, inward/tilted replacement via `BRepAlgoAPISplitter` against the replacement plane) and general surface splitting, validating the resulting solid with `BRepCheckAnalyzer`.
- Authentically rejects invalid replacements (e.g., replacing every face of a solid, faces without adjacent faces, disjoint/non-overlapping replacement surfaces such as a torus face replaced by a separate planar offset surface) with a typed `Create Replace Face feature failed` diagnostic.
- Added periodic torus seam naming in `crates/martensite/src/prism/named/blend/naming/revolve.rs` so full 360-degree circle revolutions produce valid single-face toroidal solids.
- Verified with replay tests in `crates/martensite/tests/sketch_prism/named/rebuild/replace_face_replay.rs` checking exact +1 mm * area volume increase on a planar face replaced by `OffsetSrf1` and named failure rejection on a disjoint torus + offset surface.

### Automation, GUI, and Desktop Owner (`crates/automation`, `crates/gui`, `crates/head`)

- Added typed automation helpers in `crates/automation/src/replace_face.rs` (`ReplaceFaceFeatureInputs`, `PreparedReplaceFace`, `prepare_replace_face_definition`, `build_replace_face_definition`, `replace_replace_face`).
- Added `IconGlyph::ReplaceFace` (`crates/gui/assets/identity/cmd-model-replace-face-32.svg`), exposed the `Surface` panel (`RibbonPanel::PartSurface`) on the `3D Model` ribbon with `Replace Face`, and built the dockable `Replace Face` Properties panel (`crates/gui/src/replace_face_dialog/{mod.rs, controls.rs, types.rs}`) with `Existing Faces` / `New Faces` selectors, `Automatic Face Chain`, `OK`/`Cancel`, and the `Create Replace Face feature failed` recovery modal (`Edit` preserving selections / `Cancel` closing).
- Added `crates/head/src/owner_replace_face.rs` supporting viewport solid & surface face picking, face highlighting, live replacement preview, transactional `Replace Face1` creation/editing, and Undo/Redo.

## Verification

- Native Wayland BDD suite in `crates/head/tests/solid-model/replace_face/{setup.rs, steps.rs, viewport.rs}` passes all 4 scenarios (`replace-face-01` through `replace-face-04`, 24/24 steps) live against the native window and canonical fixtures (`body-properties-start.opd` and `seven-bodies-torus-fixed-candidate.opd`).
