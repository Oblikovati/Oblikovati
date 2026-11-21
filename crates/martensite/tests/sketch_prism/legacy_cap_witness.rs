// SPDX-License-Identifier: GPL-2.0-only
use super::*;

struct ExactCapVerifier<'a> {
    named: &'a oblikovati_martensite::NamedPrismSnapshot,
    history: &'a DocumentEditHistory,
}
impl oblikovati_persistence::package::LegacyCapBindingVerifier for ExactCapVerifier<'_> {
    fn verify(
        &self,
        key: &[u8],
        role: oblikovati_persistence::package::LegacyExtrusionFaceRole,
    ) -> Result<(), String> {
        let actual = self
            .named
            .face_report(self.history, document_guid(), key)?
            .role;
        let expected = match role {
            oblikovati_persistence::package::LegacyExtrusionFaceRole::StartCap => {
                PrismFaceRole::Start
            }
            oblikovati_persistence::package::LegacyExtrusionFaceRole::EndCap => PrismFaceRole::End,
            other => return Err(format!("role {other:?}: expected cap")),
        };
        if actual != expected {
            return Err(format!("role {actual:?}: expected {expected:?}"));
        }
        Ok(())
    }
}

fn verify_migrated_caps(history: &DocumentEditHistory) {
    let package = DocumentPackage::decode(GO_CUBE).unwrap();
    let candidate = package
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
    let verifier = ExactCapVerifier {
        named: &named,
        history,
    };
    for (source, expected) in [
        (
            b"\x03Extrusion1:start-cap#0".as_slice(),
            PrismFaceRole::Start,
        ),
        (b"\x03Extrusion1:end-cap#0".as_slice(), PrismFaceRole::End),
    ] {
        let mapped = candidate
            .migrate_cap_face(source, document_guid(), feature, &verifier)
            .unwrap();
        assert_eq!(
            named
                .face_report(history, document_guid(), mapped.key())
                .unwrap()
                .role,
            expected
        );
    }
    assert!(
        candidate
            .migrate_cap_face(
                b"\x03Extrusion1:side#1",
                document_guid(),
                feature,
                &verifier
            )
            .is_err()
    );
}

#[test]
fn go_cap_names_migrate_through_edit_undo_redo_and_reopen() {
    let mut history = imported_history(GO_CUBE);
    verify_migrated_caps(&history);
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
    verify_migrated_caps(&history);
    history.undo().unwrap();
    verify_migrated_caps(&history);
    history.redo().unwrap();
    verify_migrated_caps(&history);
    verify_migrated_caps(&reopened_history(&history));
}

struct RejectCapVerifier;
impl oblikovati_persistence::package::LegacyCapBindingVerifier for RejectCapVerifier {
    fn verify(
        &self,
        _: &[u8],
        _: oblikovati_persistence::package::LegacyExtrusionFaceRole,
    ) -> Result<(), String> {
        Err("named face was not found".into())
    }
}

#[test]
fn cap_migration_refuses_unverified_wrong_target_and_wrong_document() {
    let history = imported_history(GO_CUBE);
    let candidate = DocumentPackage::decode(GO_CUBE)
        .unwrap()
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    let feature = history.snapshot().work_geometry.get(FeatureId(1)).unwrap();
    let key = b"\x03Extrusion1:start-cap#0";
    let error = candidate
        .migrate_cap_face(key, document_guid(), feature, &RejectCapVerifier)
        .unwrap_err();
    assert!(error.to_string().contains("named face was not found"));
    let named = retain_certified_named_sketch_prism(
        &history,
        document_guid(),
        FeatureId(1),
        app_resolution(),
    )
    .unwrap();
    let verifier = ExactCapVerifier {
        named: &named,
        history: &history,
    };
    let foreign = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    assert!(
        candidate
            .migrate_cap_face(key, foreign, feature, &verifier)
            .is_err()
    );
    let wrong = AuthoredFeature::sketch_prism(
        FeatureId(9),
        "Other",
        candidate.extrusions[0].definition.clone(),
        history.snapshot(),
    )
    .unwrap();
    assert!(
        candidate
            .migrate_cap_face(key, document_guid(), &wrong, &verifier)
            .is_err()
    );
}

#[test]
fn cap_migration_refuses_ambiguous_source_sequences() {
    use oblikovati_persistence::package::LegacyExtrusionCandidate;
    let history = imported_history(GO_CUBE);
    let mut candidate = DocumentPackage::decode(GO_CUBE)
        .unwrap()
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    let source = &candidate.extrusions[0];
    candidate.extrusions.push(LegacyExtrusionCandidate {
        name: "Another extrusion".into(),
        creation_sequence: source.creation_sequence,
        suppressed: false,
        definition: source.definition.clone(),
    });
    let feature = history.snapshot().work_geometry.get(FeatureId(1)).unwrap();
    let error = candidate
        .migrate_cap_face(
            b"\x03Extrusion1:start-cap#0",
            document_guid(),
            feature,
            &RejectCapVerifier,
        )
        .unwrap_err();
    assert!(
        error.to_string().contains("one nonzero source sequence"),
        "{error}"
    );
}

const GO_CAP_PLANE: &[u8] = include_bytes!(
    "../../../../architecture/rust-port/receipts/2026-11-14-go-cap-plane/cube-with-cap-plane.obk"
);

fn migrated_go_cap_plane_history() -> DocumentEditHistory {
    let base = imported_history(GO_CAP_PLANE);
    let package = DocumentPackage::decode(GO_CAP_PLANE).unwrap();
    let candidate = package
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    let feature = base.snapshot().work_geometry.get(FeatureId(1)).unwrap();
    let named =
        retain_certified_named_sketch_prism(&base, document_guid(), FeatureId(1), app_resolution())
            .unwrap();
    let verifier = ExactCapVerifier {
        named: &named,
        history: &base,
    };
    let definition = candidate
        .migrate_cap_offset(3, document_guid(), feature, &verifier)
        .unwrap();
    let mut snapshot = base.snapshot().clone();
    snapshot
        .work_geometry
        .append(AuthoredFeature::datum(
            FeatureId(2),
            "Cap plane",
            definition,
        ))
        .unwrap();
    let ids = RecipeIdAllocator::restore(None, snapshot.parameters.ids()).unwrap();
    DocumentEditHistory::new(snapshot, ids).unwrap()
}

pub(super) fn assert_go_cap_offset(history: &DocumentEditHistory, expected_area: f64) {
    use oblikovati_martensite::CertifiedNamedSolidEvaluator;
    use oblikovati_model::feature::work_plane::WorkPlaneResolver;
    let mut evaluator =
        CertifiedNamedSolidEvaluator::new(history, document_guid(), app_resolution());
    let replay = history
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let plane = replay.plane(FeatureId(2)).unwrap().recipe();
    assert_frame_values(plane.origin, [0., 0., 1.5]);
    assert_frame_values(plane.x_axis, [0., 1., 0.]);
    assert_frame_values(plane.y_axis, [-1., 0., 0.]);
    let named = retain_certified_named_sketch_prism(
        history,
        document_guid(),
        FeatureId(1),
        app_resolution(),
    )
    .unwrap();
    assert_analytic_scalar(
        named
            .face_report(history, document_guid(), &cap_name("end"))
            .unwrap()
            .area,
        expected_area,
    );
}

#[test]
fn go_cap_plane_candidate_replays_through_edit_history_and_package_reopen() {
    let mut history = migrated_go_cap_plane_history();
    assert_go_cap_offset(&history, 1.);
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
    assert_go_cap_offset(&history, 4.);
    history.undo().unwrap();
    assert_go_cap_offset(&history, 1.);
    history.redo().unwrap();
    assert_go_cap_offset(&history, 4.);
    assert_go_cap_offset(&reopened_history(&history), 4.);
}

#[test]
fn cap_offset_migration_rejects_missing_ambiguous_and_unverified_planes() {
    use oblikovati_persistence::package::LegacyFaceOffsetCandidate;
    let base = imported_history(GO_CAP_PLANE);
    let feature = base.snapshot().work_geometry.get(FeatureId(1)).unwrap();
    let mut candidate = DocumentPackage::decode(GO_CAP_PLANE)
        .unwrap()
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    assert!(
        candidate
            .migrate_cap_offset(999, document_guid(), feature, &RejectCapVerifier)
            .is_err()
    );
    let error = candidate
        .migrate_cap_offset(3, document_guid(), feature, &RejectCapVerifier)
        .unwrap_err();
    assert!(error.to_string().contains("named face was not found"));
    let plane = &candidate.face_offsets[0];
    candidate.face_offsets.push(LegacyFaceOffsetCandidate {
        creation_sequence: plane.creation_sequence,
        face_key: plane.face_key.clone(),
        offset: plane.offset,
    });
    let error = candidate
        .migrate_cap_offset(3, document_guid(), feature, &RejectCapVerifier)
        .unwrap_err();
    assert!(error.to_string().contains("unique nonzero"));
}

#[test]
fn cap_offset_migration_refuses_side_keys_and_nonfinite_offsets() {
    let base = imported_history(GO_CAP_PLANE);
    let feature = base.snapshot().work_geometry.get(FeatureId(1)).unwrap();
    let named =
        retain_certified_named_sketch_prism(&base, document_guid(), FeatureId(1), app_resolution())
            .unwrap();
    let verifier = ExactCapVerifier {
        named: &named,
        history: &base,
    };
    let mut candidate = DocumentPackage::decode(GO_CAP_PLANE)
        .unwrap()
        .prepare_legacy_part_candidate(&GoCubeProfile)
        .unwrap();
    candidate.face_offsets[0].offset = f64::NAN;
    assert!(
        candidate
            .migrate_cap_offset(3, document_guid(), feature, &verifier)
            .unwrap_err()
            .to_string()
            .contains("finite")
    );
    candidate.face_offsets[0].offset = 0.5;
    candidate.face_offsets[0].face_key = b"\x03Extrusion1:side#1".to_vec();
    assert!(
        candidate
            .migrate_cap_offset(3, document_guid(), feature, &verifier)
            .unwrap_err()
            .to_string()
            .contains("source-curve lineage")
    );
}
