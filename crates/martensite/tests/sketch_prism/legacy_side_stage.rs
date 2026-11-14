// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchEntityId;
use oblikovati_persistence::package::{
    LegacyExtrusionCandidate, LegacyExtrusionFaceRole, LegacyPartStageVerifier,
};

const GO_SIDE_PLANE: &[u8] = include_bytes!(
    "../../../../architecture/rust-port/receipts/2026-11-14-go-dependent-face/cube-with-face-plane.obk"
);

/// This independent Go fixture is evidence for one source, not a production qualifier.
struct TestOnlyGoSideEvidence {
    rows: Vec<GoSide>,
}
impl LegacyPartStageVerifier for TestOnlyGoSideEvidence {
    fn solve(&self, snapshot: &mut DocumentSnapshot) -> Result<(), String> {
        solve_selected_sketches(snapshot);
        Ok(())
    }
    fn qualify_sketch(
        &self,
        _: &DocumentSnapshot,
        _: oblikovati_model::document_sketch::DocumentSketchId,
    ) -> Result<(), String> {
        Ok(())
    }
    fn verify(
        &self,
        _: &DocumentSnapshot,
        _: DocumentGuid,
        _: FeatureId,
        _: &[u8],
        _: LegacyExtrusionFaceRole,
    ) -> Result<(), String> {
        Err("fixture uses a side face".into())
    }
    fn source_side_curve(
        &self,
        key: &[u8],
        ordinal: u64,
        _: &LegacyExtrusionCandidate,
    ) -> Result<SketchEntityId, String> {
        let encoded: String = key.iter().map(|byte| format!("{byte:02x}")).collect();
        let mut rows = self
            .rows
            .iter()
            .filter(|row| row.side_index as u64 == ordinal && row.face_key_hex == encoded);
        let row = rows.next().ok_or("missing Go side lineage")?;
        if rows.next().is_some() {
            return Err("ambiguous Go side lineage".into());
        }
        Ok(SketchEntityId(row.source_line_id))
    }
    fn verify_side(
        &self,
        snapshot: &DocumentSnapshot,
        document: DocumentGuid,
        feature: FeatureId,
        key: &[u8],
        curve: SketchEntityId,
    ) -> Result<(), String> {
        let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids())
            .map_err(|e| e.to_string())?;
        let history = DocumentEditHistory::new(snapshot.clone(), ids).map_err(|e| e.to_string())?;
        let body =
            retain_certified_named_sketch_prism(&history, document, feature, app_resolution())?;
        let role = body.face_report(&history, document, key)?.role;
        if role != PrismFaceRole::ProfileEdge(curve.0) {
            return Err(format!("face role {role:?}: expected curve {curve:?}"));
        }
        Ok(())
    }
}

#[test]
fn go_side_plane_stages_and_replays_with_test_only_witness_and_qualification() {
    let (candidate, base) = super::legacy_cap_stage::candidate_and_base(GO_SIDE_PLANE);
    let oracle: GoLineage = serde_json::from_str(ORIGINAL_WITNESS).unwrap();
    let verifier = TestOnlyGoSideEvidence {
        rows: oracle.before,
    };
    let staged = candidate
        .stage_part_program(
            &base,
            document_guid(),
            &super::legacy_cap_stage::identity_mapping(),
            &verifier,
        )
        .unwrap();
    let ids = RecipeIdAllocator::restore(None, staged.parameters.ids()).unwrap();
    let mut history = DocumentEditHistory::new(staged, ids).unwrap();
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

#[test]
fn go_side_stage_refuses_missing_ambiguous_or_wrong_source_lineage() {
    let (candidate, base) = super::legacy_cap_stage::candidate_and_base(GO_SIDE_PLANE);
    let oracle: GoLineage = serde_json::from_str(ORIGINAL_WITNESS).unwrap();
    let row = oracle
        .before
        .iter()
        .find(|row| row.side_index == 1)
        .unwrap();
    let copied = || GoSide {
        side_index: row.side_index,
        source_line_id: row.source_line_id,
        face_key_hex: row.face_key_hex.clone(),
    };
    let bad_curve = GoSide {
        source_line_id: 999,
        ..copied()
    };
    let cases = [
        (vec![], "missing Go side lineage"),
        (vec![copied(), copied()], "ambiguous Go side lineage"),
        (vec![bad_curve], "expected one selected boundary use"),
    ];
    for (rows, expected) in cases {
        let error = candidate
            .stage_part_program(
                &base,
                document_guid(),
                &super::legacy_cap_stage::identity_mapping(),
                &TestOnlyGoSideEvidence { rows },
            )
            .unwrap_err();
        assert!(error.to_string().contains(expected), "{error}");
        assert!(base.work_geometry.features().is_empty());
    }
}

struct NoPrecisionSideVerifier;
impl LegacyPartStageVerifier for NoPrecisionSideVerifier {
    fn solve(&self, snapshot: &mut DocumentSnapshot) -> Result<(), String> {
        solve_selected_sketches(snapshot);
        Ok(())
    }
    fn verify(
        &self,
        _: &DocumentSnapshot,
        _: DocumentGuid,
        _: FeatureId,
        _: &[u8],
        _: LegacyExtrusionFaceRole,
    ) -> Result<(), String> {
        panic!("face verification must not run without qualified precision")
    }
    fn source_side_curve(
        &self,
        _: &[u8],
        _: u64,
        _: &LegacyExtrusionCandidate,
    ) -> Result<SketchEntityId, String> {
        panic!("source lineage must not be requested without qualified precision")
    }
}

#[test]
fn side_stage_declines_before_lineage_without_qualified_precision() {
    let (candidate, base) = super::legacy_cap_stage::candidate_and_base(GO_SIDE_PLANE);
    let before = base.clone();
    let error = candidate
        .stage_part_program(
            &base,
            document_guid(),
            &super::legacy_cap_stage::identity_mapping(),
            &NoPrecisionSideVerifier,
        )
        .unwrap_err();
    assert!(
        error.to_string().contains("qualified spatial and angular"),
        "{error}"
    );
    assert!(base.sketches.same_recipe(&before.sketches));
    assert!(base.parameters.same_recipe(&before.parameters));
    assert!(
        base.work_geometry
            .same_work_geometry_recipe(&before.work_geometry)
    );
}

struct RejectSideTarget(TestOnlyGoSideEvidence);
impl LegacyPartStageVerifier for RejectSideTarget {
    fn solve(&self, snapshot: &mut DocumentSnapshot) -> Result<(), String> {
        self.0.solve(snapshot)
    }
    fn qualify_sketch(
        &self,
        snapshot: &DocumentSnapshot,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
    ) -> Result<(), String> {
        self.0.qualify_sketch(snapshot, sketch)
    }
    fn verify(
        &self,
        _: &DocumentSnapshot,
        _: DocumentGuid,
        _: FeatureId,
        _: &[u8],
        _: LegacyExtrusionFaceRole,
    ) -> Result<(), String> {
        Err("unexpected cap".into())
    }
    fn source_side_curve(
        &self,
        key: &[u8],
        ordinal: u64,
        source: &LegacyExtrusionCandidate,
    ) -> Result<SketchEntityId, String> {
        self.0.source_side_curve(key, ordinal, source)
    }
    fn verify_side(
        &self,
        _: &DocumentSnapshot,
        _: DocumentGuid,
        _: FeatureId,
        _: &[u8],
        _: SketchEntityId,
    ) -> Result<(), String> {
        Err("current side binding unavailable".into())
    }
}

#[test]
fn side_stage_refuses_unverified_current_target_without_mutating_the_source() {
    let (candidate, base) = super::legacy_cap_stage::candidate_and_base(GO_SIDE_PLANE);
    let before = base.clone();
    let oracle: GoLineage = serde_json::from_str(ORIGINAL_WITNESS).unwrap();
    let verifier = RejectSideTarget(TestOnlyGoSideEvidence {
        rows: oracle.before,
    });
    let error = candidate
        .stage_part_program(
            &base,
            document_guid(),
            &super::legacy_cap_stage::identity_mapping(),
            &verifier,
        )
        .unwrap_err();
    assert!(
        error
            .to_string()
            .contains("current side binding unavailable"),
        "{error}"
    );
    assert!(base.same_metadata_recipe(&before));
    assert!(base.sketches.same_recipe(&before.sketches));
    assert!(
        base.work_geometry
            .same_work_geometry_recipe(&before.work_geometry)
    );
}
