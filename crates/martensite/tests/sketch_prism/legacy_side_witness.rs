// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchEntityId;
use oblikovati_persistence::package::LegacySideLineageWitness;

struct GoSideWitness<'a> {
    rows: &'a [GoSide],
    named: &'a oblikovati_martensite::NamedPrismSnapshot,
    history: &'a DocumentEditHistory,
}
impl LegacySideLineageWitness for GoSideWitness<'_> {
    fn source_curve(
        &self,
        legacy_key: &[u8],
        ordinal: u64,
        _: &oblikovati_persistence::package::LegacyExtrusionCandidate,
    ) -> Result<SketchEntityId, String> {
        let mut rows = self
            .rows
            .iter()
            .filter(|row| row.side_index as u64 == ordinal);
        let row = rows.next().ok_or("missing Go side witness")?;
        let encoded: String = legacy_key
            .iter()
            .map(|byte| format!("{byte:02x}"))
            .collect();
        if rows.next().is_some() || encoded != row.face_key_hex {
            return Err("ambiguous or unrelated Go side witness".into());
        }
        Ok(SketchEntityId(row.source_line_id))
    }
    fn verify_target(&self, key: &[u8], curve: SketchEntityId) -> Result<(), String> {
        let actual = self
            .named
            .face_report(self.history, document_guid(), key)?
            .role;
        if actual != PrismFaceRole::ProfileEdge(curve.0) {
            return Err(format!("role {actual:?}: expected source curve {curve:?}"));
        }
        Ok(())
    }
}

fn verify_side_migration(bytes: &[u8], rows: &[GoSide], history: &DocumentEditHistory) {
    let candidate = DocumentPackage::decode(bytes)
        .unwrap()
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    let feature = history.snapshot().work_geometry.get(FeatureId(1)).unwrap();
    let named = retain_certified_named_sketch_prism(
        history,
        document_guid(),
        FeatureId(1),
        app_resolution(),
    )
    .unwrap();
    let witness = GoSideWitness {
        rows,
        named: &named,
        history,
    };
    for row in rows {
        let source = format!("\x03Extrusion1:side#{}", row.side_index);
        let migrated = candidate
            .migrate_side_face(source.as_bytes(), document_guid(), feature, &witness)
            .unwrap();
        let actual = named
            .face_report(history, document_guid(), migrated.key())
            .unwrap();
        assert_eq!(actual.role, PrismFaceRole::ProfileEdge(row.source_line_id));
    }
}

#[test]
fn go_side_names_use_external_curve_lineage_through_edit_undo_redo_and_reopen() {
    for (fixture, source) in [(GO_CUBE, ORIGINAL_WITNESS), (REORDERED, REORDERED_WITNESS)] {
        let rows: GoLineage = serde_json::from_str(source).unwrap();
        let mut history = imported_history(fixture);
        verify_side_migration(fixture, &rows.before, &history);
        let token = history.begin("size").unwrap();
        history
            .edit(&token, |snapshot, _| {
                let id = snapshot.parameters.find("size").unwrap();
                snapshot.parameters.set_expression(id, "20 mm")?;
                solve_selected_sketches(snapshot);
                Ok::<(), oblikovati_model::parameter::ParameterError<u64>>(())
            })
            .unwrap();
        history.commit(&token, false).unwrap();
        verify_side_migration(fixture, &rows.after, &history);
        history.undo().unwrap();
        verify_side_migration(fixture, &rows.before, &history);
        history.redo().unwrap();
        verify_side_migration(fixture, &rows.after, &history);
        verify_side_migration(fixture, &rows.after, &reopened_history(&history));
    }
}

struct InvalidCurveWitness;
impl LegacySideLineageWitness for InvalidCurveWitness {
    fn source_curve(
        &self,
        _: &[u8],
        _: u64,
        _: &oblikovati_persistence::package::LegacyExtrusionCandidate,
    ) -> Result<SketchEntityId, String> {
        Ok(SketchEntityId(999))
    }
    fn verify_target(&self, _: &[u8], _: SketchEntityId) -> Result<(), String> {
        panic!("a non-boundary curve must be refused before target verification")
    }
}

#[test]
fn side_migration_refuses_unwitnessed_or_nonboundary_lineage() {
    let history = imported_history(GO_CUBE);
    let candidate = DocumentPackage::decode(GO_CUBE)
        .unwrap()
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    let feature = history.snapshot().work_geometry.get(FeatureId(1)).unwrap();
    let named = retain_certified_named_sketch_prism(
        &history,
        document_guid(),
        FeatureId(1),
        app_resolution(),
    )
    .unwrap();
    let witness = GoSideWitness {
        rows: &[],
        named: &named,
        history: &history,
    };
    let missing = candidate
        .migrate_side_face(b"\x03Extrusion1:side#1", document_guid(), feature, &witness)
        .unwrap_err();
    assert!(missing.to_string().contains("missing Go side witness"));
    let invalid = candidate
        .migrate_side_face(
            b"\x03Extrusion1:side#1",
            document_guid(),
            feature,
            &InvalidCurveWitness,
        )
        .unwrap_err();
    assert!(
        invalid
            .to_string()
            .contains("expected one selected boundary use")
    );
    assert!(
        candidate
            .migrate_side_face(
                b"\x03Extrusion1:start-cap#0",
                document_guid(),
                feature,
                &witness
            )
            .is_err()
    );
}

const GO_SIDE_PLANE: &[u8] = include_bytes!(
    "../../../../architecture/rust-port/receipts/2026-11-14-go-dependent-face/cube-with-face-plane.obk"
);

fn migrated_side_plane_history() -> DocumentEditHistory {
    let base = imported_history(GO_CUBE);
    let candidate = DocumentPackage::decode(GO_SIDE_PLANE)
        .unwrap()
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    let rows: GoLineage = serde_json::from_str(ORIGINAL_WITNESS).unwrap();
    let named =
        retain_certified_named_sketch_prism(&base, document_guid(), FeatureId(1), app_resolution())
            .unwrap();
    let witness = GoSideWitness {
        rows: &rows.before,
        named: &named,
        history: &base,
    };
    let target = base.snapshot().work_geometry.get(FeatureId(1)).unwrap();
    let definition = candidate
        .migrate_side_offset(3, document_guid(), target, &witness)
        .unwrap();
    let mut snapshot = base.snapshot().clone();
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(
            FeatureId(2),
            "Go side plane",
            definition,
        ))
        .unwrap();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    DocumentEditHistory::new(snapshot, ids).unwrap()
}

#[test]
fn witnessed_go_side_plane_replays_through_edit_undo_redo_and_reopen() {
    let mut history = migrated_side_plane_history();
    assert_go_offset_frame(&history, 1.5);
    let token = history.begin("size").unwrap();
    history
        .edit(&token, |snapshot, _| {
            let id = snapshot.parameters.find("size").unwrap();
            snapshot.parameters.set_expression(id, "20 mm")?;
            solve_selected_sketches(snapshot);
            Ok::<(), oblikovati_model::parameter::ParameterError<u64>>(())
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_go_offset_frame(&history, 2.5);
    history.undo().unwrap();
    assert_go_offset_frame(&history, 1.5);
    history.redo().unwrap();
    assert_go_offset_frame(&history, 2.5);
    assert_go_offset_frame(&reopened_history(&history), 2.5);
}
