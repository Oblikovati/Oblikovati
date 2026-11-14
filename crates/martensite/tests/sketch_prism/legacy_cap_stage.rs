// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_persistence::package::{
    LegacyCapStageVerifier, LegacyExtrusionFaceRole, LegacyStagedFeatureIdentity,
};
use std::collections::BTreeMap;

struct CurrentBodyVerifier;
impl LegacyCapStageVerifier for CurrentBodyVerifier {
    fn solve(&self, snapshot: &mut DocumentSnapshot) -> Result<(), String> {
        solve_selected_sketches(snapshot);
        Ok(())
    }
    fn verify(
        &self,
        snapshot: &DocumentSnapshot,
        document: DocumentGuid,
        feature: FeatureId,
        key: &[u8],
        role: LegacyExtrusionFaceRole,
    ) -> Result<(), String> {
        let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids())
            .map_err(|e| e.to_string())?;
        let history = DocumentEditHistory::new(snapshot.clone(), ids).map_err(|e| e.to_string())?;
        let body =
            retain_certified_named_sketch_prism(&history, document, feature, app_resolution())?;
        let actual = body.face_report(&history, document, key)?.role;
        let expected = match role {
            LegacyExtrusionFaceRole::StartCap => PrismFaceRole::Start,
            LegacyExtrusionFaceRole::EndCap => PrismFaceRole::End,
            other => return Err(format!("role {other:?}: expected a cap")),
        };
        if actual != expected {
            return Err(format!("role {actual:?}: expected {expected:?}"));
        }
        Ok(())
    }
}

const CAP_PLANE: &[u8] = include_bytes!(
    "../../../../architecture/rust-port/receipts/2026-11-14-go-cap-plane/cube-with-cap-plane.obk"
);

fn candidate_and_base(
    bytes: &[u8],
) -> (
    oblikovati_persistence::package::LegacyPartCandidate,
    DocumentSnapshot,
) {
    let package = DocumentPackage::decode(bytes).unwrap();
    let candidate = package
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    let mut base = selected_sketch_document();
    base.sketches = candidate.sketches.sketches.clone();
    base.parameters = candidate.sketches.parameters.parameters.clone();
    base.work_geometry = Default::default();
    (candidate, base)
}

fn identity_mapping() -> BTreeMap<u64, LegacyStagedFeatureIdentity> {
    BTreeMap::from([
        (
            2,
            LegacyStagedFeatureIdentity {
                id: FeatureId(1),
                name: "Extrusion1".into(),
            },
        ),
        (
            3,
            LegacyStagedFeatureIdentity {
                id: FeatureId(2),
                name: "Cap plane".into(),
            },
        ),
    ])
}

#[test]
fn go_cap_fixture_stages_a_complete_ordered_replay_program() {
    let (candidate, base) = candidate_and_base(CAP_PLANE);
    let staged = candidate
        .stage_cap_program(
            &base,
            document_guid(),
            &identity_mapping(),
            &CurrentBodyVerifier,
        )
        .unwrap();
    assert_eq!(staged.work_geometry.features().len(), 2);
    assert!(base.work_geometry.features().is_empty());
    assert!(base.same_metadata_recipe(&staged));
    let ids = RecipeIdAllocator::restore(None, staged.parameters.ids()).unwrap();
    let mut history = DocumentEditHistory::new(staged, ids).unwrap();
    super::legacy_cap_witness::assert_go_cap_offset(&history, 1.);
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
    super::legacy_cap_witness::assert_go_cap_offset(&history, 4.);
    history.undo().unwrap();
    super::legacy_cap_witness::assert_go_cap_offset(&history, 1.);
    history.redo().unwrap();
    super::legacy_cap_witness::assert_go_cap_offset(&history, 4.);
    super::legacy_cap_witness::assert_go_cap_offset(&reopened_history(&history), 4.);
}

struct NoSolve;
impl LegacyCapStageVerifier for NoSolve {
    fn solve(&self, _: &mut DocumentSnapshot) -> Result<(), String> {
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
        Ok(())
    }
}

struct TamperingSolve;
impl LegacyCapStageVerifier for TamperingSolve {
    fn solve(&self, snapshot: &mut DocumentSnapshot) -> Result<(), String> {
        let id = snapshot.parameters.find("size").unwrap();
        snapshot
            .parameters
            .set_expression(id, "20 mm")
            .map_err(|e| e.to_string())
    }
    fn verify(
        &self,
        _: &DocumentSnapshot,
        _: DocumentGuid,
        _: FeatureId,
        _: &[u8],
        _: LegacyExtrusionFaceRole,
    ) -> Result<(), String> {
        Ok(())
    }
}

#[test]
fn staging_refuses_unsolved_or_mutated_source_without_changing_the_base() {
    let (candidate, base) = candidate_and_base(CAP_PLANE);
    let original = base.clone();
    for verifier in [&NoSolve as &dyn LegacyCapStageVerifier, &TamperingSolve] {
        assert!(
            candidate
                .stage_cap_program(&base, document_guid(), &identity_mapping(), verifier)
                .is_err()
        );
        assert!(base.same_metadata_recipe(&original));
        assert!(base.sketches.same_recipe(&original.sketches));
        assert!(base.parameters.same_recipe(&original.parameters));
        assert!(
            base.work_geometry
                .same_work_geometry_recipe(&original.work_geometry)
        );
    }
}

#[test]
fn staging_refuses_missing_duplicate_and_extra_feature_assignments() {
    let (candidate, base) = candidate_and_base(CAP_PLANE);
    let mut missing = identity_mapping();
    missing.remove(&3);
    let mut duplicate = identity_mapping();
    duplicate.get_mut(&3).unwrap().id = FeatureId(1);
    let mut extra = identity_mapping();
    extra.insert(
        4,
        LegacyStagedFeatureIdentity {
            id: FeatureId(4),
            name: "Extra".into(),
        },
    );
    for mapping in [missing, duplicate, extra] {
        assert!(
            candidate
                .stage_cap_program(&base, document_guid(), &mapping, &CurrentBodyVerifier)
                .is_err()
        );
    }
}

#[test]
fn staging_refuses_go_side_faces_without_source_curve_evidence() {
    let bytes = include_bytes!(
        "../../../../architecture/rust-port/receipts/2026-11-14-go-dependent-face/cube-with-face-plane.obk"
    );
    let (candidate, base) = candidate_and_base(bytes);
    let error = candidate
        .stage_cap_program(
            &base,
            document_guid(),
            &identity_mapping(),
            &CurrentBodyVerifier,
        )
        .unwrap_err();
    assert!(
        error.to_string().contains("source-curve lineage"),
        "{error}"
    );
    assert!(base.work_geometry.features().is_empty());
}
