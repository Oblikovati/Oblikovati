# Part Modify Mark (`#3918`)

Implements Part Modify **Mark** end-to-end across `crates/model`,
`crates/martensite`, `crates/automation`, `crates/gui`, and `crates/head`.

## Architecture and Persistence

- **Model (`crates/model/src/feature/mark.rs`)**:
  - `MarkType` (`MarkSurface`, `MarkThrough`), `MarkMethod`
    (`ProjectToClosestFace`, `WrapToFace`), `MarkDirection` (`Positive`,
    `Negative`, `Symmetric`), `MarkGeometrySet`, `MarkRecipe`, `MarkDefinition`,
    `ResolvedMarkGeometrySet`, and `ResolvedMark`.
  - Disjoint persistence slot in `WorkPlaneDefinitionRecipe::Mark`:
    `source_version = 54`, `minimum_program_version = 64`,
    `MAX_PROGRAM_VERSION = 64`.
  - `DocumentSnapshot::create_mark` and `DocumentSnapshot::edit_mark` with
    default prefix `Mark` (`Mark1`, `Mark2`, ...).
  - Extracted `crates/model/src/feature/part/failure.rs`,
    `crates/model/src/feature/part/mark_evaluation.rs`, and
    `crates/model/src/feature/part/prefix_backend.rs` to keep every model module
    strictly under 500 lines.

- **Martensite (`crates/martensite`)**:
  - `MarkEligibility` and `check_mark_eligibility` query exact live solid counts
    and analytic volumes (`NamedSolid::volume`), distinguishing `Eligible`
    (exactly one positive-volume solid), `MultipleSolidBodies` (`mark-01`), and
    `NoVolume` (`mark-05`).
  - `NamedSolid::connected_face_chain` traverses shared B-rep edges in
    deterministic BFS order (`45 Faces` on `45-face-shared-vertices-candidate.opd`
    and `6 Faces` on a cube).
  - `CertifiedNamedSolidEvaluator::evaluate_named_mark` rejects zero-volume /
    surface-only bodies with
    `"Mark failed. The selected component has no volume."`, projects sketch
    polylines onto target faces (and opposing faces for `MarkThrough`) into
    imprint overlay triangles, and preserves exact solid volume and topology
    lineage.

- **Automation (`crates/automation/src/mark.rs`)**:
  - `MarkFeatureInputs`, `PreparedMark`, `build_mark_definition`,
    `prepare_mark_definition`, and `replace_mark`.

- **GUI (`crates/gui/src/mark_dialog/`)**:
  - Dockable `Mark` Properties panel (`280px`) with `Selection Sets`
    (`Mark Geometry Sets` rows, green `+` add-set button, per-set `Mark Surface`
    / `Mark Through` dropdown), `Behavior` (`Project` / `Wrap to Face` method
    selector, `Faces` selector showing `1 Face` / `N Faces`, `Automatic Face
    Chain` checkbox, `Direction` selector), and shared footer with disabled `OK`
    when no sketch geometry is selected.
  - Eligibility notice modals for multiple solid bodies and zero-volume
    components, `IconGlyph::Mark` (`crates/gui/assets/icons/mark.svg`),
    `3D Model > Modify` ribbon wiring, and browser `Edit Feature` dispatch.

- **Desktop Owner and Overlays (`crates/head/src/owner_mark.rs`)**:
  - `GuiMarkAction` dispatch, face picking and real-time `ExpandFaceChain`
    recomputation, sketch curve picking, Indigo mark imprint overlay rendering
    (`MARK_OVERLAY_COLOR`), and transactional create/edit commits.

## Verification

- **Unit and Kernel Replay Tests**:
  - `crates/model/src/feature/mark.rs`: recipe validation for project/wrap modes
    and empty geometry-set rejection.
  - `crates/martensite/src/prism/named/blend/mark.rs` and
    `crates/martensite/tests/sketch_prism/named/rebuild/mark_replay.rs`:
    eligibility checks, connected face-chain BFS, `MarkSurface` / `MarkThrough`
    replay volume and overlay preservation, and genuine surface-only zero-volume
    rejection.
  - `crates/head/tests/solid-model/mark/setup.rs`: desktop test verifying Mark
    on an empty zero-volume part shows
    `"Mark failed. The selected component has no volume."` and dismissing returns
    to the part display.
- **Native Wayland BDD (`mark.feature`)**:
  - `mark-01 Multiple solid bodies`: passed live on `fixtures/cad/combine/two-bodies.opd`.
  - `mark-02 Select a marking type and add a set`: passed live on
    `fixtures/cad/emboss/open-profile.opd`.
  - `mark-03 Select faces using automatic chaining`: passed live on
    `fixtures/cad/mark/45-face-shared-vertices-candidate.opd` (`1 Face` -> `45 Faces`).
  - `mark-04 Cancel an incomplete mark`: passed live on
    `fixtures/cad/emboss/open-profile.opd`.
  - `mark-05 Mark reports a no-volume component`: remains excluded as
    `unimplemented upstream feature` (`@needs_fixture`) because `Repair Bodies`
    is not yet implemented and the staged `single-solid-import-candidate.opd`
    genuinely evaluates to `~45.162 cm³` positive volume. The real zero-volume
    check is implemented and verified on genuine zero-volume parts without
    fabricating an error on positive-volume solids.
